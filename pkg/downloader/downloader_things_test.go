package downloader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	markSystem "github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/mark_system"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/save_sub_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	formatterCommon "github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_formatter/common"
	formatterEmby "github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_formatter/emby"
	common2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
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
		{site: common2.SubSiteSubtitleBest, content: makeASSContent("subtitle_best")},
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
	if string(savedContent) != makeASSContent("subtitle_best") {
		t.Fatalf("saved subtitle content did not come from highest-priority current source")
	}
}
