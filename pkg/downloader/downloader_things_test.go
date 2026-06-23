package downloader

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/llm_subtitle_fallback"
	markSystem "github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/mark_system"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/save_sub_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	formatterCommon "github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_formatter/common"
	formatterEmby "github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_formatter/emby"
	common2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/subparser"
	"github.com/sirupsen/logrus"
)

const shooterASSContent = "[Script Info]\n" +
	"Title: Shooter\n\n" +
	"[Events]\n" +
	"Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n" +
	"Dialogue: 0,0:00:01.00,0:00:02.00,Default,,0,0,0,,\u5c04\u624b\\NShooter\n"

const subhdASSContent = "[Script Info]\n" +
	"Title: SubHD\n\n" +
	"[Events]\n" +
	"Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n" +
	"Dialogue: 0,0:00:01.00,0:00:02.00,Default,,0,0,0,,\u661f\u9645\\NSubHD\n"

func makeASSContent(label string) string {
	return "[Script Info]\n" +
		"Title: " + label + "\n\n" +
		"[Events]\n" +
		"Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n" +
		"Dialogue: 0,0:00:01.00,0:00:02.00,Default,,0,0,0,,你好\\N" + label + "\n" +
		"Dialogue: 0,0:00:03.00,0:00:04.00,Default,,0,0,0,,再见\\N" + label + "\n"
}

type fallbackTranslatorStub struct {
	output string
	err    error
}

func (s fallbackTranslatorStub) Translate(req llm_subtitle_fallback.TranslateRequest) error {
	if s.err != nil {
		return s.err
	}
	return os.WriteFile(req.OutputPath, []byte(s.output), 0o644)
}

func TestOneVideoSelectBestSubPrefersSubhdAndWritesSubtitle(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.DebugMode = false
	cfg.AdvancedSettings.SaveMultiSub = false
	cfg.AdvancedSettings.FixTimeLine = false
	cfg.ExperimentalFunction.AutoChangeSubEncode.Enable = false
	cfg.ExperimentalFunction.ChsChtChanger.Enable = false

	videoDir := t.TempDir()
	videoPath := filepath.Join(videoDir, "Movie.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o600); err != nil {
		t.Fatalf("WriteFile(video) error = %v", err)
	}

	downloadDir := t.TempDir()
	shooterPath := filepath.Join(downloadDir, "[shooter]_0_test.ass")
	subhdPath := filepath.Join(downloadDir, "[subhd]_0_test.ass")
	if err := os.WriteFile(shooterPath, []byte(shooterASSContent), 0o600); err != nil {
		t.Fatalf("WriteFile(shooter) error = %v", err)
	}
	if err := os.WriteFile(subhdPath, []byte(subhdASSContent), 0o600); err != nil {
		t.Fatalf("WriteFile(subhd) error = %v", err)
	}

	log := logrus.New()
	d := &Downloader{
		log:              log,
		mk:               markSystem.NewMarkingSystem(log, common2.DefaultSubSiteSequence(), 0),
		SaveSubHelper:    save_sub_helper.NewSaveSubHelper(log, formatterEmby.NewFormatter(), nil),
		subNameFormatter: formatterCommon.Emby,
	}

	if err := d.oneVideoSelectBestSub(videoPath, []string{shooterPath, subhdPath}); err != nil {
		t.Fatalf("oneVideoSelectBestSub() error = %v", err)
	}

	entries, err := os.ReadDir(videoDir)
	if err != nil {
		t.Fatalf("ReadDir(videoDir) error = %v", err)
	}

	subFiles := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == filepath.Base(videoPath) {
			continue
		}
		subFiles = append(subFiles, entry.Name())
	}
	if len(subFiles) != 1 {
		t.Fatalf("subtitle file count = %d; want 1, files = %#v", len(subFiles), subFiles)
	}
	if strings.HasSuffix(strings.ToLower(subFiles[0]), ".default.ass") == false {
		t.Fatalf("subtitle file name = %q; want suffix .default.ass", subFiles[0])
	}

	savedPath := filepath.Join(videoDir, subFiles[0])
	savedContent, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("ReadFile(savedPath) error = %v", err)
	}
	if string(savedContent) != subhdASSContent {
		t.Fatalf("saved subtitle content did not come from subhd candidate")
	}
}

func TestDownloaderResetContextAfterCancel(t *testing.T) {
	d := &Downloader{
		log: logrus.New(),
	}
	d.ResetContext()

	oldCtx := d.currentContext()
	select {
	case <-oldCtx.Done():
		t.Fatal("old context should be active before cancel")
	default:
	}

	d.Cancel()

	select {
	case <-oldCtx.Done():
	default:
		t.Fatal("old context should be canceled after Cancel")
	}

	d.ResetContext()
	newCtx := d.currentContext()
	if newCtx == oldCtx {
		t.Fatal("ResetContext should create a new context")
	}

	select {
	case <-newCtx.Done():
		t.Fatal("new context should be active after ResetContext")
	default:
	}
}

func TestOneVideoSelectBestSubUsesCurrentDefaultSourcePriority(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.DebugMode = false
	cfg.AdvancedSettings.SaveMultiSub = false
	cfg.AdvancedSettings.FixTimeLine = false
	cfg.ExperimentalFunction.AutoChangeSubEncode.Enable = false
	cfg.ExperimentalFunction.ChsChtChanger.Enable = false

	videoDir := t.TempDir()
	videoPath := filepath.Join(videoDir, "Episode.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o600); err != nil {
		t.Fatalf("WriteFile(video) error = %v", err)
	}

	downloadDir := t.TempDir()
	candidates := []struct {
		site    string
		content string
	}{
		{site: common2.SubSiteXunLei, content: makeASSContent("xunlei")},
		{site: common2.SubSiteShooter, content: makeASSContent("shooter")},
		{site: common2.SubSiteSubHd, content: makeASSContent("subhd")},
		{site: common2.SubSiteSubDL, content: makeASSContent("subdl")},
		{site: common2.SubSiteAssrt, content: makeASSContent("assrt")},
		{site: common2.SubSiteMovieSubtitles, content: makeASSContent("moviesubtitles")},
		{site: common2.SubSiteTVSubtitles, content: makeASSContent("tvsubtitles")},
		{site: common2.SubSiteOpenSubtitles, content: makeASSContent("opensubtitles")},
		{site: common2.SubSiteOpenSubtitles, content: makeASSContent("opensubtitles-2")},
	}

	subFiles := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		path := filepath.Join(downloadDir, "["+candidate.site+"]_0_test.ass")
		if err := os.WriteFile(path, []byte(candidate.content), 0o600); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", path, err)
		}
		subFiles = append(subFiles, path)
	}

	log := logrus.New()
	d := &Downloader{
		log:              log,
		mk:               markSystem.NewMarkingSystem(log, common2.DefaultSubSiteSequence(), 0),
		SaveSubHelper:    save_sub_helper.NewSaveSubHelper(log, formatterEmby.NewFormatter(), nil),
		subNameFormatter: formatterCommon.Emby,
	}

	if err := d.oneVideoSelectBestSub(videoPath, subFiles); err != nil {
		t.Fatalf("oneVideoSelectBestSub() error = %v", err)
	}

	entries, err := os.ReadDir(videoDir)
	if err != nil {
		t.Fatalf("ReadDir(videoDir) error = %v", err)
	}

	var savedPath string
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == filepath.Base(videoPath) {
			continue
		}
		savedPath = filepath.Join(videoDir, entry.Name())
		break
	}
	if savedPath == "" {
		t.Fatal("expected one saved subtitle file")
	}

	savedContent, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("ReadFile(savedPath) error = %v", err)
	}
	if string(savedContent) != makeASSContent("assrt") {
		t.Fatalf("saved subtitle content did not come from the current default-priority source")
	}
}

func TestPendingSeasonPackEpisodesSkipsAlreadySavedEpisodes(t *testing.T) {
	seriesInfo := &series.SeriesInfo{
		EpList: []series.EpisodeInfo{
			{Season: 5, Episode: 8},
			{Season: 5, Episode: 9},
			{Season: 6, Episode: 1},
		},
		NeedDlSeasonDict: map[int]int{
			5: 5,
			6: 6,
		},
	}

	savedEpisodeKeys := map[string]struct{}{
		pkg.GetEpisodeKeyName(5, 9): {},
	}

	got := pendingSeasonPackEpisodes(seriesInfo, savedEpisodeKeys)
	if len(got) != 2 {
		t.Fatalf("pendingSeasonPackEpisodes() len = %d; want 2", len(got))
	}
	if got[0].Season != 5 || got[0].Episode != 8 {
		t.Fatalf("pendingSeasonPackEpisodes()[0] = S%02dE%02d; want S05E08", got[0].Season, got[0].Episode)
	}
	if got[1].Season != 6 || got[1].Episode != 1 {
		t.Fatalf("pendingSeasonPackEpisodes()[1] = S%02dE%02d; want S06E01", got[1].Season, got[1].Episode)
	}
}

func TestOneVideoSelectBestSubSkipsAbsurdTimelineSubtitle(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.DebugMode = false
	cfg.AdvancedSettings.SaveMultiSub = false
	cfg.AdvancedSettings.FixTimeLine = false
	cfg.ExperimentalFunction.AutoChangeSubEncode.Enable = false
	cfg.ExperimentalFunction.ChsChtChanger.Enable = false

	videoDir := t.TempDir()
	videoPath := filepath.Join(videoDir, "Episode.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o600); err != nil {
		t.Fatalf("WriteFile(video) error = %v", err)
	}

	downloadDir := t.TempDir()
	invalidPath := filepath.Join(downloadDir, "["+common2.SubSiteOpenSubtitles+"]_0_bad.srt")
	validPath := filepath.Join(downloadDir, "["+common2.SubSiteSubDL+"]_0_good.srt")

	invalidContent := strings.Join([]string{
		"1",
		"23:59:57,000 --> 23:59:58,000",
		"garbled line",
		"",
		"2",
		"00:00:05,000 --> 00:00:07,000",
		"这不是正常时间轴",
		"",
	}, "\n")
	validContent := strings.Join([]string{
		"1",
		"00:00:05,000 --> 00:00:07,000",
		"这是正常字幕",
		"",
		"2",
		"00:00:08,000 --> 00:00:10,000",
		"Second line",
		"",
	}, "\n")

	if err := os.WriteFile(invalidPath, []byte(invalidContent), 0o600); err != nil {
		t.Fatalf("WriteFile(invalid) error = %v", err)
	}
	if err := os.WriteFile(validPath, []byte(validContent), 0o600); err != nil {
		t.Fatalf("WriteFile(valid) error = %v", err)
	}

	log := logrus.New()
	d := &Downloader{
		log:              log,
		mk:               markSystem.NewMarkingSystem(log, common2.DefaultSubSiteSequence(), 0),
		SaveSubHelper:    save_sub_helper.NewSaveSubHelper(log, formatterEmby.NewFormatter(), nil),
		subNameFormatter: formatterCommon.Emby,
	}

	if err := d.oneVideoSelectBestSub(videoPath, []string{invalidPath, validPath}); err != nil {
		t.Fatalf("oneVideoSelectBestSub() error = %v", err)
	}

	entries, err := os.ReadDir(videoDir)
	if err != nil {
		t.Fatalf("ReadDir(videoDir) error = %v", err)
	}

	var savedPath string
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == filepath.Base(videoPath) {
			continue
		}
		savedPath = filepath.Join(videoDir, entry.Name())
		break
	}
	if savedPath == "" {
		t.Fatal("expected one saved subtitle file")
	}

	savedContent, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("ReadFile(savedPath) error = %v", err)
	}
	if string(savedContent) != validContent {
		t.Fatalf("saved subtitle content did not skip invalid high-priority candidate")
	}
}

func TestInvalidSubtitleReasonDetectsNonMonotonicWrappedTimeline(t *testing.T) {
	fileInfo := &subparser.FileInfo{
		Dialogues: []subparser.OneDialogue{
			{StartTime: "23:59:10,560", EndTime: "23:59:16,010", Lines: []string{"line 1"}},
			{StartTime: "00:00:25,160", EndTime: "00:00:27,700", Lines: []string{"line 2"}},
		},
	}

	reason := invalidSubtitleReason(fileInfo, 3395.296)
	if reason == "" {
		t.Fatal("expected non-monotonic wrapped timeline to be rejected")
	}
}

func TestOneVideoSelectBestSubReturnsNoSubtitleWhenAllCandidatesInvalid(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.DebugMode = false
	cfg.AdvancedSettings.SaveMultiSub = false
	cfg.AdvancedSettings.FixTimeLine = false
	cfg.ExperimentalFunction.AutoChangeSubEncode.Enable = false
	cfg.ExperimentalFunction.ChsChtChanger.Enable = false

	videoDir := t.TempDir()
	videoPath := filepath.Join(videoDir, "Episode.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o600); err != nil {
		t.Fatalf("WriteFile(video) error = %v", err)
	}

	downloadDir := t.TempDir()
	invalidPath := filepath.Join(downloadDir, "["+common2.SubSiteOpenSubtitles+"]_0_bad.srt")
	invalidContent := strings.Join([]string{
		"1",
		"00:00:05,000 --> 00:00:07,000",
		"m 34 -191 l 162 -6 l 25 191",
		"",
		"2",
		"00:00:08,000 --> 00:00:10,000",
		"m -32 -191 l -32 191 l 14 191",
		"",
		"3",
		"00:00:11,000 --> 00:00:13,000",
		"m -353 191 l -354 4 b -352 -98",
		"",
	}, "\n")
	if err := os.WriteFile(invalidPath, []byte(invalidContent), 0o600); err != nil {
		t.Fatalf("WriteFile(invalid) error = %v", err)
	}

	log := logrus.New()
	d := &Downloader{
		log:              log,
		mk:               markSystem.NewMarkingSystem(log, common2.DefaultSubSiteSequence(), 0),
		SaveSubHelper:    save_sub_helper.NewSaveSubHelper(log, formatterEmby.NewFormatter(), nil),
		subNameFormatter: formatterCommon.Emby,
	}

	err := d.oneVideoSelectBestSub(videoPath, []string{invalidPath})
	if err == nil {
		t.Fatal("expected error when all subtitle candidates are invalid")
	}
	if err != common2.AllSiteDownloadSubNotFound && !strings.Contains(err.Error(), common2.AllSiteDownloadSubNotFound.Error()) {
		t.Fatalf("unexpected error = %v", err)
	}
}

func TestOneVideoSelectBestSubDoesNotTranslateEnglishCandidateInChineseStage(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.DebugMode = false
	cfg.AdvancedSettings.SaveMultiSub = false
	cfg.AdvancedSettings.FixTimeLine = false
	cfg.ExperimentalFunction.AutoChangeSubEncode.Enable = false
	cfg.ExperimentalFunction.ChsChtChanger.Enable = false
	cfg.ExperimentalFunction.LLMSubtitleFallback.Enable = true
	cfg.ExperimentalFunction.LLMSubtitleFallback.LogDir = t.TempDir()
	cfg.ExperimentalFunction.LLMSubtitleFallback.SubflowRootDir = t.TempDir()

	videoDir := t.TempDir()
	videoPath := filepath.Join(videoDir, "Episode.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o600); err != nil {
		t.Fatalf("WriteFile(video) error = %v", err)
	}

	downloadDir := t.TempDir()
	englishPath := filepath.Join(downloadDir, "["+common2.SubSiteOpenSubtitles+"]_0_My.Show.S01E03.1080p.WEB-DL-GROUP.en.srt")
	englishBody := strings.Join([]string{
		"1",
		"00:00:01,000 --> 00:00:02,000",
		"Hello there",
		"",
		"2",
		"00:00:03,000 --> 00:00:04,000",
		"General Kenobi",
		"",
	}, "\n")
	if err := os.WriteFile(englishPath, []byte(englishBody), 0o600); err != nil {
		t.Fatalf("WriteFile(english) error = %v", err)
	}

	log := logrus.New()
	d := &Downloader{
		log:              log,
		mk:               markSystem.NewMarkingSystem(log, common2.DefaultSubSiteSequence(), 0),
		SaveSubHelper:    save_sub_helper.NewSaveSubHelper(log, formatterEmby.NewFormatter(), nil),
		subNameFormatter: formatterCommon.Emby,
		llmSubtitleFallback: llm_subtitle_fallback.NewManagerWithTranslator(log, &cfg.ExperimentalFunction.LLMSubtitleFallback, fallbackTranslatorStub{
			output: strings.Join([]string{
				"1",
				"00:00:01,000 --> 00:00:02,000",
				"你好",
				"",
				"2",
				"00:00:03,000 --> 00:00:04,000",
				"将军",
				"",
			}, "\n"),
		}),
	}

	err := d.oneVideoSelectBestSub(videoPath, []string{englishPath})
	if err == nil {
		t.Fatal("expected chinese stage to reject pure english candidate")
	}
	if errors.Is(err, errNoUsableChineseSubtitle) == false {
		t.Fatalf("unexpected error = %v", err)
	}
}

func TestTryWriteEnglishSubtitleFallbackWorksWithoutLLM(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.DebugMode = false
	cfg.AdvancedSettings.SaveMultiSub = false
	cfg.AdvancedSettings.FixTimeLine = false
	cfg.ExperimentalFunction.AutoChangeSubEncode.Enable = false
	cfg.ExperimentalFunction.ChsChtChanger.Enable = false
	cfg.ExperimentalFunction.LLMSubtitleFallback.Enable = false
	videoDir := t.TempDir()
	videoPath := filepath.Join(videoDir, "Episode.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o600); err != nil {
		t.Fatalf("WriteFile(video) error = %v", err)
	}

	downloadDir := t.TempDir()
	englishPath := filepath.Join(downloadDir, "["+common2.SubSiteSubtitleCat+"]_0_My.Show.S01E03.1080p.WEB-DL-GROUP.en.srt")
	englishBody := strings.Join([]string{
		"1",
		"00:00:01,000 --> 00:00:02,000",
		"Hello there",
		"",
		"2",
		"00:00:03,000 --> 00:00:04,000",
		"General Kenobi",
		"",
	}, "\n")
	if err := os.WriteFile(englishPath, []byte(englishBody), 0o600); err != nil {
		t.Fatalf("WriteFile(english) error = %v", err)
	}

	log := logrus.New()
	d := &Downloader{
		log:              log,
		mk:               markSystem.NewMarkingSystem(log, common2.DefaultSubSiteSequence(), 0),
		SaveSubHelper:    save_sub_helper.NewSaveSubHelper(log, formatterEmby.NewFormatter(), nil),
		subNameFormatter: formatterCommon.Emby,
	}

	if d.canTryEnglishFallback() == false {
		t.Fatal("expected english fallback to remain available without llm fallback")
	}
	if d.canTryLLMStageFallback() {
		t.Fatal("expected llm fallback to stay disabled")
	}

	if err := d.tryWriteEnglishSubtitleFallback(videoPath, []string{englishPath}); err != nil {
		t.Fatalf("tryWriteEnglishSubtitleFallback() error = %v", err)
	}

	entries, err := os.ReadDir(videoDir)
	if err != nil {
		t.Fatalf("ReadDir(videoDir) error = %v", err)
	}

	var savedPath string
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == filepath.Base(videoPath) {
			continue
		}
		savedPath = filepath.Join(videoDir, entry.Name())
	}
	if savedPath == "" {
		t.Fatal("expected english fallback subtitle to be saved")
	}

	savedContent, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("ReadFile(savedPath) error = %v", err)
	}
	if string(savedContent) != englishBody {
		t.Fatalf("saved english fallback subtitle mismatch: %q", string(savedContent))
	}
}

func TestCanTryTranslatedChineseFallbackRequiresExplicitSwitch(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.SaveMultiSub = false
	cfg.SubtitleSources.SubtitleCatSettings.EnableTranslatedChineseFallback = false

	d := &Downloader{}
	if d.canTryTranslatedChineseFallback() {
		t.Fatal("translated chinese fallback should stay off by default")
	}

	cfg.SubtitleSources.SubtitleCatSettings.EnableTranslatedChineseFallback = true
	if d.canTryTranslatedChineseFallback() == false {
		t.Fatal("translated chinese fallback should require explicit opt-in only")
	}
}

func TestOrderedSubtitleFallbackStagesPreferChineseOutputsBeforeEnglish(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.SaveMultiSub = false
	cfg.SubtitleSources.SubtitleCatSettings.EnableTranslatedChineseFallback = false
	cfg.ExperimentalFunction.LLMSubtitleFallback.Enable = true
	cfg.ExperimentalFunction.LLMSubtitleFallback.APIKey = "test-key"

	log := logrus.New()
	d := &Downloader{
		llmSubtitleFallback: llm_subtitle_fallback.NewManager(log, &cfg.ExperimentalFunction.LLMSubtitleFallback),
	}

	got := d.orderedSubtitleFallbackStages()
	want := []subtitleFallbackStage{
		subtitleFallbackStageLLM,
		subtitleFallbackStageEnglish,
	}
	if len(got) != len(want) {
		t.Fatalf("orderedSubtitleFallbackStages() len = %d; want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("orderedSubtitleFallbackStages()[%d] = %v; want %v", i, got[i], want[i])
		}
	}

	cfg.SubtitleSources.SubtitleCatSettings.EnableTranslatedChineseFallback = true
	got = d.orderedSubtitleFallbackStages()
	want = []subtitleFallbackStage{
		subtitleFallbackStageTranslatedChinese,
		subtitleFallbackStageLLM,
		subtitleFallbackStageEnglish,
	}
	if len(got) != len(want) {
		t.Fatalf("orderedSubtitleFallbackStages() with translated enabled len = %d; want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("orderedSubtitleFallbackStages() with translated enabled [%d] = %v; want %v", i, got[i], want[i])
		}
	}
}

func TestOneVideoSelectBestSubPrefersNativeChineseOverSubtitleCatTranslated(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.DebugMode = false
	cfg.AdvancedSettings.SaveMultiSub = false
	cfg.AdvancedSettings.FixTimeLine = false
	cfg.ExperimentalFunction.AutoChangeSubEncode.Enable = false
	cfg.ExperimentalFunction.ChsChtChanger.Enable = false

	videoDir := t.TempDir()
	videoPath := filepath.Join(videoDir, "Movie.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o600); err != nil {
		t.Fatalf("WriteFile(video) error = %v", err)
	}

	downloadDir := t.TempDir()
	nativePath := filepath.Join(downloadDir, "["+common2.SubSiteSubHd+"]_0_native.ass")
	translatedPath := filepath.Join(downloadDir, "["+common2.SubSiteSubtitleCatTrans+"]_0_translated.ass")
	nativeContent := makeASSContent("native-subhd")
	translatedContent := makeASSContent("translated-subtitlecat")

	if err := os.WriteFile(nativePath, []byte(nativeContent), 0o600); err != nil {
		t.Fatalf("WriteFile(native) error = %v", err)
	}
	if err := os.WriteFile(translatedPath, []byte(translatedContent), 0o600); err != nil {
		t.Fatalf("WriteFile(translated) error = %v", err)
	}

	log := logrus.New()
	d := &Downloader{
		log:              log,
		mk:               markSystem.NewMarkingSystem(log, common2.DefaultSubSiteSequence(), 0),
		SaveSubHelper:    save_sub_helper.NewSaveSubHelper(log, formatterEmby.NewFormatter(), nil),
		subNameFormatter: formatterCommon.Emby,
	}

	if err := d.oneVideoSelectBestSub(videoPath, []string{translatedPath, nativePath}); err != nil {
		t.Fatalf("oneVideoSelectBestSub() error = %v", err)
	}

	entries, err := os.ReadDir(videoDir)
	if err != nil {
		t.Fatalf("ReadDir(videoDir) error = %v", err)
	}

	var savedPath string
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == filepath.Base(videoPath) {
			continue
		}
		savedPath = filepath.Join(videoDir, entry.Name())
	}
	if savedPath == "" {
		t.Fatal("expected one saved subtitle file")
	}

	savedContent, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("ReadFile(savedPath) error = %v", err)
	}
	if string(savedContent) != nativeContent {
		t.Fatalf("expected native chinese subtitle to win over subtitlecat translated fallback")
	}
}

func TestOneVideoSelectBestSubRejectsSubtitleCatTranslatedWithoutChineseContent(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.DebugMode = false
	cfg.AdvancedSettings.SaveMultiSub = false
	cfg.AdvancedSettings.FixTimeLine = false
	cfg.ExperimentalFunction.AutoChangeSubEncode.Enable = false
	cfg.ExperimentalFunction.ChsChtChanger.Enable = false

	videoDir := t.TempDir()
	videoPath := filepath.Join(videoDir, "Episode.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o600); err != nil {
		t.Fatalf("WriteFile(video) error = %v", err)
	}

	downloadDir := t.TempDir()
	disguisedPath := filepath.Join(downloadDir, "["+common2.SubSiteSubtitleCatTrans+"]_0_translated.srt")
	disguisedBody := strings.Join([]string{
		"1",
		"00:00:01,000 --> 00:00:02,000",
		"Hello there",
		"",
		"2",
		"00:00:03,000 --> 00:00:04,000",
		"General Kenobi",
		"",
	}, "\n")
	if err := os.WriteFile(disguisedPath, []byte(disguisedBody), 0o600); err != nil {
		t.Fatalf("WriteFile(disguised) error = %v", err)
	}

	log := logrus.New()
	d := &Downloader{
		log:              log,
		mk:               markSystem.NewMarkingSystem(log, common2.DefaultSubSiteSequence(), 0),
		SaveSubHelper:    save_sub_helper.NewSaveSubHelper(log, formatterEmby.NewFormatter(), nil),
		subNameFormatter: formatterCommon.Emby,
	}

	err := d.oneVideoSelectBestSub(videoPath, []string{disguisedPath})
	if err == nil {
		t.Fatal("expected disguised translated subtitle without chinese content to be rejected")
	}
	if errors.Is(err, errNoUsableChineseSubtitle) == false {
		t.Fatalf("unexpected error = %v", err)
	}
}

func TestTryWriteLLMSubtitleFallbackTranslatesEnglishCandidateInFallbackStage(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.DebugMode = false
	cfg.AdvancedSettings.SaveMultiSub = false
	cfg.AdvancedSettings.FixTimeLine = false
	cfg.ExperimentalFunction.AutoChangeSubEncode.Enable = false
	cfg.ExperimentalFunction.ChsChtChanger.Enable = false
	cfg.ExperimentalFunction.LLMSubtitleFallback.Enable = true
	cfg.ExperimentalFunction.LLMSubtitleFallback.APIKey = "test-key"
	cfg.ExperimentalFunction.LLMSubtitleFallback.LogDir = t.TempDir()
	cfg.ExperimentalFunction.LLMSubtitleFallback.SubflowRootDir = t.TempDir()

	videoDir := t.TempDir()
	videoPath := filepath.Join(videoDir, "Episode.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o600); err != nil {
		t.Fatalf("WriteFile(video) error = %v", err)
	}

	downloadDir := t.TempDir()
	englishPath := filepath.Join(downloadDir, "["+common2.SubSiteOpenSubtitles+"]_0_My.Show.S01E03.1080p.WEB-DL-GROUP.en.srt")
	englishBody := strings.Join([]string{
		"1",
		"00:00:01,000 --> 00:00:02,000",
		"Hello there",
		"",
		"2",
		"00:00:03,000 --> 00:00:04,000",
		"General Kenobi",
		"",
	}, "\n")
	if err := os.WriteFile(englishPath, []byte(englishBody), 0o600); err != nil {
		t.Fatalf("WriteFile(english) error = %v", err)
	}

	log := logrus.New()
	translatedBody := strings.Join([]string{
		"1",
		"00:00:01,000 --> 00:00:02,000",
		"CN line 1",
		"",
		"2",
		"00:00:03,000 --> 00:00:04,000",
		"CN line 2",
		"",
	}, "\n")
	d := &Downloader{
		log:              log,
		mk:               markSystem.NewMarkingSystem(log, common2.DefaultSubSiteSequence(), 0),
		SaveSubHelper:    save_sub_helper.NewSaveSubHelper(log, formatterEmby.NewFormatter(), nil),
		subNameFormatter: formatterCommon.Emby,
		llmSubtitleFallback: llm_subtitle_fallback.NewManagerWithTranslator(log, &cfg.ExperimentalFunction.LLMSubtitleFallback, fallbackTranslatorStub{
			output: translatedBody,
		}),
	}

	if err := d.tryWriteLLMSubtitleFallback(videoPath, []string{englishPath}); err != nil {
		t.Fatalf("tryWriteLLMSubtitleFallback() error = %v", err)
	}

	entries, err := os.ReadDir(videoDir)
	if err != nil {
		t.Fatalf("ReadDir(videoDir) error = %v", err)
	}

	var savedPath string
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == filepath.Base(videoPath) {
			continue
		}
		savedPath = filepath.Join(videoDir, entry.Name())
	}
	if savedPath == "" {
		t.Fatal("expected translated subtitle to be saved")
	}

	savedContent, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("ReadFile(savedPath) error = %v", err)
	}
	if string(savedContent) != translatedBody {
		t.Fatalf("saved subtitle content mismatch: %q", string(savedContent))
	}
}

func TestTryWriteLLMSubtitleFallbackSkipsWhenAPIKeyMissing(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.DebugMode = false
	cfg.AdvancedSettings.SaveMultiSub = false
	cfg.AdvancedSettings.FixTimeLine = false
	cfg.ExperimentalFunction.AutoChangeSubEncode.Enable = false
	cfg.ExperimentalFunction.ChsChtChanger.Enable = false
	cfg.ExperimentalFunction.LLMSubtitleFallback.Enable = true
	cfg.ExperimentalFunction.LLMSubtitleFallback.APIKey = ""
	cfg.ExperimentalFunction.LLMSubtitleFallback.LogDir = t.TempDir()
	cfg.ExperimentalFunction.LLMSubtitleFallback.SubflowRootDir = t.TempDir()

	videoDir := t.TempDir()
	videoPath := filepath.Join(videoDir, "Episode.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o600); err != nil {
		t.Fatalf("WriteFile(video) error = %v", err)
	}

	downloadDir := t.TempDir()
	englishPath := filepath.Join(downloadDir, "["+common2.SubSiteOpenSubtitles+"]_0_My.Show.S01E03.1080p.WEB-DL-GROUP.en.srt")
	englishBody := strings.Join([]string{
		"1",
		"00:00:01,000 --> 00:00:02,000",
		"Hello there",
		"",
	}, "\n")
	if err := os.WriteFile(englishPath, []byte(englishBody), 0o600); err != nil {
		t.Fatalf("WriteFile(english) error = %v", err)
	}

	log := logrus.New()
	d := &Downloader{
		log:              log,
		mk:               markSystem.NewMarkingSystem(log, common2.DefaultSubSiteSequence(), 0),
		SaveSubHelper:    save_sub_helper.NewSaveSubHelper(log, formatterEmby.NewFormatter(), nil),
		subNameFormatter: formatterCommon.Emby,
		llmSubtitleFallback: llm_subtitle_fallback.NewManagerWithTranslator(log, &cfg.ExperimentalFunction.LLMSubtitleFallback, fallbackTranslatorStub{
			output: "1\n00:00:01,000 --> 00:00:02,000\nCN line 1\n\n",
		}),
	}

	err := d.tryWriteLLMSubtitleFallback(videoPath, []string{englishPath})
	if err == nil {
		t.Fatal("expected missing api key to skip llm fallback")
	}
	if errors.Is(err, common2.AllSiteDownloadSubNotFound) == false {
		t.Fatalf("unexpected error = %v", err)
	}

	entries, err := os.ReadDir(videoDir)
	if err != nil {
		t.Fatalf("ReadDir(videoDir) error = %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != filepath.Base(videoPath) {
		t.Fatalf("unexpected files after skipped llm fallback: %#v", entries)
	}
}

func TestTryWriteLLMSubtitleFallbackOpenAICompatibleEndToEnd(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.DebugMode = false
	cfg.AdvancedSettings.SaveMultiSub = false
	cfg.AdvancedSettings.FixTimeLine = false
	cfg.ExperimentalFunction.AutoChangeSubEncode.Enable = false
	cfg.ExperimentalFunction.ChsChtChanger.Enable = false
	cfg.ExperimentalFunction.LLMSubtitleFallback.Enable = true
	cfg.ExperimentalFunction.LLMSubtitleFallback.LogDir = t.TempDir()
	cfg.ExperimentalFunction.LLMSubtitleFallback.SubflowRootDir = bundledSubflowRoot(t)
	cfg.ExperimentalFunction.LLMSubtitleFallback.Provider = "openai"
	cfg.ExperimentalFunction.LLMSubtitleFallback.APIKey = "test-key"
	cfg.ExperimentalFunction.LLMSubtitleFallback.Model = "mock-model"
	cfg.ExperimentalFunction.LLMSubtitleFallback.SourceLanguage = "en"
	cfg.ExperimentalFunction.LLMSubtitleFallback.TargetLanguage = "zh"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" {
			t.Fatalf("unexpected path %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("Authorization = %q", got)
		}

		var reqBody struct {
			Model    string `json:"model"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatalf("Decode(request) error = %v", err)
		}
		if reqBody.Model != "mock-model" {
			t.Fatalf("model = %q", reqBody.Model)
		}
		if len(reqBody.Messages) < 2 || strings.Contains(reqBody.Messages[1].Content, "Hello there") == false {
			t.Fatalf("unexpected prompt payload: %#v", reqBody.Messages)
		}
		if strings.Contains(reqBody.Messages[1].Content, "Return exactly one item for every cue id below.") == false {
			t.Fatalf("prompt missing completeness rule: %q", reqBody.Messages[1].Content)
		}
		if strings.Contains(reqBody.Messages[1].Content, "Do not leave an English sentence unchanged.") == false {
			t.Fatalf("prompt missing anti-english rule: %q", reqBody.Messages[1].Content)
		}

		resp := map[string]interface{}{
			"choices": []map[string]interface{}{
				{
					"message": map[string]interface{}{
						"content": "[{\"id\":1,\"lines\":[\"\\u4f60\\u597d\"]},{\"id\":2,\"lines\":[\"\\u5c06\\u519b\"]}]",
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			t.Fatalf("Encode(response) error = %v", err)
		}
	}))
	defer server.Close()

	cfg.ExperimentalFunction.LLMSubtitleFallback.BaseURL = server.URL + "/v1"

	videoDir := t.TempDir()
	videoPath := filepath.Join(videoDir, "Episode.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o600); err != nil {
		t.Fatalf("WriteFile(video) error = %v", err)
	}

	downloadDir := t.TempDir()
	englishPath := filepath.Join(downloadDir, "["+common2.SubSiteOpenSubtitles+"]_0_My.Show.S01E03.1080p.WEB-DL-GROUP.en.srt")
	englishBody := strings.Join([]string{
		"1",
		"00:00:01,000 --> 00:00:02,000",
		"Hello there",
		"",
		"2",
		"00:00:03,000 --> 00:00:04,000",
		"General Kenobi",
		"",
	}, "\n")
	if err := os.WriteFile(englishPath, []byte(englishBody), 0o600); err != nil {
		t.Fatalf("WriteFile(english) error = %v", err)
	}

	log := logrus.New()
	d := &Downloader{
		log:                 log,
		mk:                  markSystem.NewMarkingSystem(log, common2.DefaultSubSiteSequence(), 0),
		SaveSubHelper:       save_sub_helper.NewSaveSubHelper(log, formatterEmby.NewFormatter(), nil),
		subNameFormatter:    formatterCommon.Emby,
		llmSubtitleFallback: llm_subtitle_fallback.NewManager(log, &cfg.ExperimentalFunction.LLMSubtitleFallback),
	}

	if err := d.tryWriteLLMSubtitleFallback(videoPath, []string{englishPath}); err != nil {
		t.Fatalf("tryWriteLLMSubtitleFallback() error = %v", err)
	}

	entries, err := os.ReadDir(videoDir)
	if err != nil {
		t.Fatalf("ReadDir(videoDir) error = %v", err)
	}

	var savedPath string
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == filepath.Base(videoPath) {
			continue
		}
		savedPath = filepath.Join(videoDir, entry.Name())
	}
	if savedPath == "" {
		t.Fatal("expected translated subtitle to be saved")
	}

	savedContent, err := os.ReadFile(savedPath)
	if err != nil {
		t.Fatalf("ReadFile(savedPath) error = %v", err)
	}
	savedText := string(savedContent)
	if strings.Contains(savedText, "你好") == false || strings.Contains(savedText, "将军") == false {
		t.Fatalf("translated subtitle missing chinese lines: %q", savedText)
	}
	if strings.Contains(savedText, "Hello there") {
		t.Fatalf("translated subtitle should not keep english text: %q", savedText)
	}

	taskEntries, err := os.ReadDir(cfg.ExperimentalFunction.LLMSubtitleFallback.LogDir)
	if err != nil {
		t.Fatalf("ReadDir(logDir) error = %v", err)
	}
	if len(taskEntries) != 1 || taskEntries[0].IsDir() == false {
		t.Fatalf("unexpected task dir entries: %#v", taskEntries)
	}
	taskDir := filepath.Join(cfg.ExperimentalFunction.LLMSubtitleFallback.LogDir, taskEntries[0].Name())
	for _, name := range []string{"source.en.srt", "translated.zh.srt", "translate.stdout.log"} {
		if _, err := os.Stat(filepath.Join(taskDir, name)); err != nil {
			t.Fatalf("missing task artifact %q: %v", name, err)
		}
	}
	for _, name := range []string{
		"chunk-debug/chunk-001.prompt.txt",
		"chunk-debug/chunk-001.response.raw.json",
		"chunk-debug/chunk-001.response.content.txt",
		"chunk-debug/chunk-001.response.normalized.json",
	} {
		if _, err := os.Stat(filepath.Join(taskDir, filepath.FromSlash(name))); err != nil {
			t.Fatalf("missing chunk debug artifact %q: %v", name, err)
		}
	}
	if _, err := os.Stat(filepath.Join(taskDir, "source.original.srt")); err == nil {
		t.Fatal("did not expect english source copy when keep_english_source_copy=false")
	}

	logData, err := os.ReadFile(filepath.Join(taskDir, "translate.stdout.log"))
	if err != nil {
		t.Fatalf("ReadFile(translate.stdout.log) error = %v", err)
	}
	if strings.Contains(string(logData), "\"provider\": \"openai\"") == false {
		t.Fatalf("translate stdout log missing provider marker: %q", string(logData))
	}
}

func bundledSubflowRoot(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if ok == false {
		t.Fatal("runtime.Caller() failed")
	}
	root := filepath.Join(filepath.Dir(thisFile), "..", "..", "third_party", "subflow")
	if _, err := os.Stat(filepath.Join(root, "src", "subflow", "translate_job.py")); err != nil {
		t.Fatalf("bundled subflow root invalid: %v", err)
	}
	return root
}
