package assrt

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/ranking"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/random_auth_key"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/sirupsen/logrus"
)

var assrtInstance *Supplier

func defInstance() {

	pkg.ReadCustomAuthFile(log_helper.GetLogger4Tester())

	authKey := random_auth_key.AuthKey{
		BaseKey:  pkg.BaseKey(),
		AESKey16: pkg.AESKey16(),
		AESIv16:  pkg.AESIv16(),
	}

	assrtInstance = NewSupplier(file_downloader.NewFileDownloader(
		cache_center.NewCacheCenter("test", log_helper.GetLogger4Tester()), authKey))
}

func TestSupplier_getSubListFromFile(t *testing.T) {
	t.Skip("integration test depends on local media files and assrt availability")

	//videoFPath := "X:\\电影\\失控玩家 (2021)\\失控玩家 (2021).mp4"
	//isMovie := true
	defInstance()
	//videoFPath := "X:\\连续剧\\杀死伊芙 (2018)\\Season 4\\Killing Eve - S04E08 - Hello, Losers WEBDL-1080p.mkv"
	//videoFPath := "X:\\连续剧\\Why Didn’t They Ask Evans!\\Season 1\\Why Didn’t They Ask Evans! - S01E01 - Episode 1 WEBRip-1080p.mp4"
	videoFPath := "X:\\连续剧\\Pantheon\\Season 1\\Pantheon - S01E03 - Reign of Winter WEBDL-1080p.mkv"
	//videoFPath := "X:\\连续剧\\风骚律师 (2015)\\Season 6\\Better Call Saul - S06E05 - Black and Blue WEBDL-1080p.mkv"
	isMovie := false

	got, err := assrtInstance.getSubListFromFile(videoFPath, isMovie)
	if err != nil {
		t.Error(err)
	}
	for i, info := range got {
		println(i, info.Name, info.FileUrl)
	}
}

func TestSupplier_CheckAlive(t *testing.T) {
	t.Skip("integration test depends on assrt availability")

	defInstance()
	bok, speed := assrtInstance.CheckAlive()
	println(bok, speed)

}

func TestAssrtSearchKeywordOrder(t *testing.T) {
	want := []string{"cn", "en", "org", "file"}
	if !reflect.DeepEqual(assrtSearchKeywordOrder, want) {
		t.Fatalf("assrtSearchKeywordOrder = %#v; want %#v", assrtSearchKeywordOrder, want)
	}
}

func TestBuildAssrtSearchKeywordsAddsYearlessFallback(t *testing.T) {
	got := buildAssrtSearchKeywords("Nirvana the Band the Show the Movie 2026")
	want := []string{
		"Nirvana the Band the Show the Movie 2026",
		"Nirvana the Band the Show the Movie",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildAssrtSearchKeywords() = %#v; want %#v", got, want)
	}
}

func TestBuildAssrtSearchKeywordsAddsYearlessFallbackBeforeEpisodeToken(t *testing.T) {
	got := buildAssrtSearchKeywords("Why Didnt They Ask Evans 2022 S01E01")
	want := []string{
		"Why Didnt They Ask Evans 2022 S01E01",
		"Why Didnt They Ask Evans S01E01",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildAssrtSearchKeywords() = %#v; want %#v", got, want)
	}
}

func TestBuildAssrtSearchKeywordsDedupesWhenNoYearFallback(t *testing.T) {
	got := buildAssrtSearchKeywords("Lopez vs Lopez S01E02")
	want := []string{"Lopez vs Lopez S01E02"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildAssrtSearchKeywords() = %#v; want %#v", got, want)
	}
}

func TestBuildAssrtSearchKeywordsAddsAmpersandVariant(t *testing.T) {
	got := buildAssrtSearchKeywords("Will & Harper 2024")
	want := []string{
		"Will & Harper 2024",
		"Will and Harper 2024",
		"Will & Harper",
		"Will and Harper",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("buildAssrtSearchKeywords() = %#v; want %#v", got, want)
	}
}

func TestSortAssrtSearchSubsPrefersMatchingMetadata(t *testing.T) {
	subs := []SearchSubItem{
		{
			Id:         1,
			VoteScore:  10,
			Videoname:  "My Show S01E03 720p HDTV-OTHER",
			NativeName: "My Show S01E03 720p HDTV-OTHER",
			Revision:   1,
		},
		{
			Id:         2,
			VoteScore:  8,
			Videoname:  "My Show S01E03 1080p WEB-DL-GROUP",
			NativeName: "My Show S01E03 1080p WEB-DL-GROUP",
			Revision:   1,
		},
	}

	sortAssrtSearchSubs(subs, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false)
	if subs[0].Id != 2 {
		t.Fatalf("expected metadata-matching subtitle first, got %#v", subs[0])
	}
}

func TestSortAssrtSearchSubsBreaksVoteTieWithEpisodeMetadata(t *testing.T) {
	subs := []SearchSubItem{
		{
			Id:         10,
			VoteScore:  8,
			Videoname:  "My Show S01E04 1080p WEB-DL-GROUP",
			NativeName: "My Show S01E04 1080p WEB-DL-GROUP",
			Revision:   1,
		},
		{
			Id:         11,
			VoteScore:  8,
			Videoname:  "My Show S01E03 1080p WEB-DL-GROUP",
			NativeName: "My Show S01E03 1080p WEB-DL-GROUP",
			Revision:   1,
		},
	}

	sortAssrtSearchSubs(subs, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false)
	if subs[0].Id != 11 {
		t.Fatalf("expected exact episode match first, got %#v", subs[0])
	}
}

func TestSortAssrtSearchSubsPrefersExactEpisodeOverHigherVote(t *testing.T) {
	subs := []SearchSubItem{
		{
			Id:         20,
			VoteScore:  20,
			Videoname:  "My Show S01E04 1080p WEB-DL-GROUP",
			NativeName: "My Show S01E04 1080p WEB-DL-GROUP",
			Revision:   3,
		},
		{
			Id:         21,
			VoteScore:  8,
			Videoname:  "My Show S01E03 1080p WEB-DL-GROUP",
			NativeName: "My Show S01E03 1080p WEB-DL-GROUP",
			Revision:   1,
		},
	}

	sortAssrtSearchSubs(subs, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false)
	if subs[0].Id != 21 {
		t.Fatalf("expected exact episode match to outrank higher-vote wrong episode, got %#v", subs[0])
	}
}

func TestScoreAssrtSearchSubIncludesAuthorityScore(t *testing.T) {
	matcher := ranking.NewTargetMatcher(filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false)
	sub := SearchSubItem{
		VoteScore:  8,
		Revision:   2,
		Subtype:    "bilingual subtitle",
		Videoname:  "My Show S01E03 1080p WEB-DL-GROUP",
		NativeName: "My Show S01E03 1080p WEB-DL-GROUP",
	}

	got := scoreAssrtSearchSub(sub, matcher)
	want := ranking.ScoreCandidate(matcher, assrtCandidateMetadata(sub), ranking.CandidateScoreSpec{
		IsMovie:       false,
		TargetSeason:  1,
		TargetEpisode: 3,
		EpisodeMatchWeights: &ranking.EpisodeMatchWeights{
			ExactMatch:   120,
			SeasonPack:   15,
			WrongEpisode: -120,
		},
		ReleaseMatchWeights: ranking.StandardReleaseMatchWeights,
	})
	if got != want {
		t.Fatalf("scoreAssrtSearchSub = %d, want %d", got, want)
	}
}

func TestAssrtCandidateMetadata(t *testing.T) {
	sub := SearchSubItem{
		VoteScore:  8,
		Revision:   2,
		Subtype:    "bilingual subtitle",
		Videoname:  "Video.Name",
		NativeName: "Native.Name",
	}

	metadata := assrtCandidateMetadata(sub)
	if metadata.AuthorityScore != 84 || metadata.Subtype != sub.Subtype {
		t.Fatalf("unexpected metadata %#v", metadata)
	}
	if len(metadata.ReleaseNames) != 2 || metadata.ReleaseNames[0] != sub.Videoname || metadata.ReleaseNames[1] != sub.NativeName {
		t.Fatalf("unexpected release names %#v", metadata.ReleaseNames)
	}
	if metadata.Season != 0 || metadata.Episode != 0 {
		t.Fatalf("unexpected parsed season/episode %#v", metadata)
	}
}

func TestAssrtCandidateMetadataParsesSeasonEpisode(t *testing.T) {
	sub := SearchSubItem{
		Videoname:  "My Show S01E03 1080p WEB-DL-GROUP",
		NativeName: "My Show S01E03 1080p WEB-DL-GROUP",
	}

	metadata := assrtCandidateMetadata(sub)
	if metadata.Season != 1 || metadata.Episode != 3 {
		t.Fatalf("unexpected parsed season/episode %#v", metadata)
	}
}

func TestGetSubByKeyWordAllowsArrayLanglist(t *testing.T) {
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())
	cfg := settings.Get()
	oldRootURL := cfg.AdvancedSettings.SuppliersSettings.Assrt.RootUrl
	oldToken := cfg.SubtitleSources.AssrtSettings.Token
	t.Cleanup(func() {
		cfg.AdvancedSettings.SuppliersSettings.Assrt.RootUrl = oldRootURL
		cfg.SubtitleSources.AssrtSettings.Token = oldToken
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sub":{"action":"search","subs":[{"id":"7","vote_score":"8","revision":"2","lang":{"langlist":[]}}],"result":"succeed","keyword":"bad"},"status":0}`))
	}))
	defer server.Close()

	cfg.AdvancedSettings.SuppliersSettings.Assrt.RootUrl = server.URL
	cfg.SubtitleSources.AssrtSettings.Token = "test-token"

	supplier := NewSupplier(&file_downloader.FileDownloader{Log: log_helper.GetLogger4Tester()})
	supplier.theSearchInterval = 0

	got, err := supplier.getSubByKeyWord("bad")
	if err != nil {
		t.Fatalf("expected array langlist payload to be accepted, got error: %v", err)
	}
	if got == nil || len(got.Sub.Subs) != 1 {
		t.Fatalf("unexpected result: %#v", got)
	}
	if got.Sub.Subs[0].Id != 7 || got.Sub.Subs[0].VoteScore != 8 || got.Sub.Subs[0].Revision != 2 {
		t.Fatalf("unexpected numeric fields: %#v", got.Sub.Subs[0])
	}
}

func TestGetSubInfoWithFallbackMergesNonEmptyKeywordResults(t *testing.T) {
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())
	cfg := settings.Get()
	oldRootURL := cfg.AdvancedSettings.SuppliersSettings.Assrt.RootUrl
	oldToken := cfg.SubtitleSources.AssrtSettings.Token
	t.Cleanup(func() {
		cfg.AdvancedSettings.SuppliersSettings.Assrt.RootUrl = oldRootURL
		cfg.SubtitleSources.AssrtSettings.Token = oldToken
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		query := r.URL.Query().Get("q")
		switch query {
		case "中文标题":
			_, _ = w.Write([]byte(`{"sub":{"action":"search","subs":[{"id":"7","vote_score":"8","revision":"2","videoname":"Show.S01E05.1080p.WEB-DL","native_name":"中文候选","lang":{"langlist":[]}}],"result":"succeed","keyword":"中文标题"},"status":0}`))
		case "English Title":
			_, _ = w.Write([]byte(`{"sub":{"action":"search","subs":[{"id":"9","vote_score":"7","revision":"1","videoname":"Show.S01E05.720p.WEB-DL","native_name":"English Candidate","lang":{"langlist":[]}}],"result":"succeed","keyword":"English Title"},"status":0}`))
		default:
			_, _ = w.Write([]byte(`{"sub":{"action":"search","subs":[],"result":"succeed","keyword":"` + query + `"},"status":0}`))
		}
	}))
	defer server.Close()

	cfg.AdvancedSettings.SuppliersSettings.Assrt.RootUrl = server.URL
	cfg.SubtitleSources.AssrtSettings.Token = "test-token"

	supplier := NewSupplier(&file_downloader.FileDownloader{Log: log_helper.GetLogger4Tester()})
	supplier.theSearchInterval = 0

	mediaInfo := &models.MediaInfo{
		TitleCn: "中文标题",
		TitleEn: "English Title",
	}
	got, err := supplier.getSubInfoWithFallback(mediaInfo, filepath.Join("C:\\", "Media", "Movie.2024.1080p.WEB-DL.mkv"), true)
	if err != nil {
		t.Fatalf("getSubInfoWithFallback() error = %v", err)
	}
	if got == nil || len(got.Sub.Subs) != 2 {
		t.Fatalf("getSubInfoWithFallback() got %#v; want 2 merged subs", got)
	}
	if got.Sub.Subs[0].Id != 7 || got.Sub.Subs[1].Id != 9 {
		t.Fatalf("unexpected merged ids %#v", got.Sub.Subs)
	}
}

func TestAssrtSearchSubsAllowsSingleObjectPayload(t *testing.T) {
	var got SearchSubResult
	payload := `{"sub":{"action":"search","subs":{"id":"7","vote_score":"8","revision":"2","lang":{"langlist":[]}},"result":"succeed","keyword":"bad"},"status":0}`
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(got.Sub.Subs) != 1 || got.Sub.Subs[0].Id != 7 {
		t.Fatalf("unexpected subs %#v", got.Sub.Subs)
	}
}

func TestAssrtDetailSubsAllowsSingleObjectPayload(t *testing.T) {
	var got OneSubDetail
	payload := `{"sub":{"action":"detail","subs":{"id":"9","url":"https://example.com/sub.zip","lang":{"langlist":[]}},"result":"succeed"},"status":0}`
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(got.Sub.Subs) != 1 || got.Sub.Subs[0].Id != 9 || got.Sub.Subs[0].Url != "https://example.com/sub.zip" {
		t.Fatalf("unexpected detail subs %#v", got.Sub.Subs)
	}
}

func TestAssrtDetailSubsAllowsEmptyObjectPayload(t *testing.T) {
	var got OneSubDetail
	payload := `{"errmsg":"subtitle not found","sub":{"result":"failed","subs":{},"action":"detail"},"status":20900}`
	if err := json.Unmarshal([]byte(payload), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(got.Sub.Subs) != 0 {
		t.Fatalf("unexpected detail subs %#v", got.Sub.Subs)
	}
	if got.Status != 20900 || got.Sub.Result != "failed" {
		t.Fatalf("unexpected payload %#v", got)
	}
}

func TestBuildAssrtDownloadCandidatesIncludesAllDistinctDetailURLs(t *testing.T) {
	candidates := buildAssrtDownloadCandidates(
		"Episode.mkv",
		SearchSubItem{NativeName: "Search Native", Videoname: "Search Video"},
		[]AssrtDetailSubItem{
			{Url: "https://example.com/a.zip", NativeName: "Detail Native A"},
			{Url: "https://example.com/b.zip", Filename: "Detail File B"},
			{Url: "https://example.com/a.zip", NativeName: "Duplicate"},
		},
	)

	want := []assrtDownloadCandidate{
		{url: "https://example.com/a.zip", subName: "Detail Native A"},
		{url: "https://example.com/b.zip", subName: "Detail File B"},
	}
	if !reflect.DeepEqual(candidates, want) {
		t.Fatalf("buildAssrtDownloadCandidates() = %#v; want %#v", candidates, want)
	}
}

func TestBuildAssrtDownloadCandidatesFallsBackToSearchNameAndVideoName(t *testing.T) {
	candidates := buildAssrtDownloadCandidates(
		"Episode.mkv",
		SearchSubItem{NativeName: "Search Native"},
		[]AssrtDetailSubItem{
			{Url: "https://example.com/a.zip"},
		},
	)
	if len(candidates) != 1 {
		t.Fatalf("expected one candidate, got %#v", candidates)
	}
	if candidates[0].subName != "Search Native" {
		t.Fatalf("candidate subName = %q; want %q", candidates[0].subName, "Search Native")
	}

	candidates = buildAssrtDownloadCandidates(
		"Episode.mkv",
		SearchSubItem{},
		[]AssrtDetailSubItem{
			{Url: "https://example.com/b.zip"},
		},
	)
	if len(candidates) != 1 {
		t.Fatalf("expected one candidate, got %#v", candidates)
	}
	if candidates[0].subName != "Episode.mkv" {
		t.Fatalf("candidate subName = %q; want %q", candidates[0].subName, "Episode.mkv")
	}
}

func TestFirstUsableAssrtDownloadSkipsBadArchiveCandidate(t *testing.T) {
	badInfo := supplier.NewSubInfo("assrt", 0, "bad.zip", language.ChineseSimple, "https://example.com/bad.zip", 0, 0, ".zip", mustBuildZipBytes(t, map[string]string{
		"README.txt": "not subtitle",
	}))
	goodInfo := supplier.NewSubInfo("assrt", 0, "good.zip", language.ChineseSimple, "https://example.com/good.zip", 0, 0, ".zip", mustBuildZipBytes(t, map[string]string{
		"Movie.zh.srt": strings.Repeat("1\n00:00:01,000 --> 00:00:02,000\n你好，世界\nHello world subtitle line\n\n", 40),
	}))

	got, ok, err := firstUsableAssrtDownload(
		logrus.New(),
		filepath.Join("C:\\", "Media", "Movie.2024.1080p.WEB-DL.mkv"),
		true,
		[]assrtDownloadCandidate{
			{url: "https://example.com/bad.zip", subName: "bad.zip"},
			{url: "https://example.com/good.zip", subName: "good.zip"},
		},
		nil,
		func(_ int, candidate assrtDownloadCandidate) (*supplier.SubInfo, error) {
			if candidate.url == "https://example.com/bad.zip" {
				return badInfo, nil
			}
			return goodInfo, nil
		},
	)
	if err != nil {
		t.Fatalf("firstUsableAssrtDownload() error = %v, want nil", err)
	}
	if ok == false {
		t.Fatal("expected to find a usable candidate")
	}
	if got == nil || got.FileUrl != "https://example.com/good.zip" {
		t.Fatalf("firstUsableAssrtDownload() chose %#v; want good candidate", got)
	}
}

func TestFirstUsableAssrtDownloadReturnsLastErrorWhenAllCandidatesFail(t *testing.T) {
	badInfo := supplier.NewSubInfo("assrt", 0, "bad.zip", language.ChineseSimple, "https://example.com/bad.zip", 0, 0, ".zip", mustBuildZipBytes(t, map[string]string{
		"README.txt": "not subtitle",
	}))

	got, ok, err := firstUsableAssrtDownload(
		logrus.New(),
		filepath.Join("C:\\", "Media", "Movie.2024.1080p.WEB-DL.mkv"),
		true,
		[]assrtDownloadCandidate{
			{url: "https://example.com/downloader-error.zip", subName: "downloader-error.zip"},
			{url: "https://example.com/bad.zip", subName: "bad.zip"},
		},
		nil,
		func(_ int, candidate assrtDownloadCandidate) (*supplier.SubInfo, error) {
			if strings.Contains(candidate.url, "downloader-error") {
				return nil, errors.New("invalid archive payload for https://example.com/downloader-error.zip: zip: not a valid zip file")
			}
			return badInfo, nil
		},
	)
	if ok {
		t.Fatal("expected no usable candidate")
	}
	if got != nil {
		t.Fatalf("firstUsableAssrtDownload() got %#v, want nil", got)
	}
	if err == nil {
		t.Fatal("expected last download error to be returned")
	}
	if strings.Contains(err.Error(), "invalid archive payload") == false && strings.Contains(err.Error(), "assrt unusable downloaded candidate") == false {
		t.Fatalf("unexpected error %q", err.Error())
	}
}

func TestFirstUsableAssrtDownloadRejectsEnglishOnlySubtitle(t *testing.T) {
	englishInfo := supplier.NewSubInfo("assrt", 0, "english.zip", language.ChineseSimple, "https://example.com/english.zip", 0, 0, ".zip", mustBuildZipBytes(t, map[string]string{
		"My.Show.S01E03.en.srt": "1\n00:00:01,000 --> 00:00:02,000\nHello there\n\n2\n00:00:03,000 --> 00:00:04,000\nGeneral Kenobi\n",
	}))

	got, ok, err := firstUsableAssrtDownload(
		logrus.New(),
		filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"),
		false,
		[]assrtDownloadCandidate{
			{url: "https://example.com/english.zip", subName: "english.zip"},
		},
		nil,
		func(_ int, _ assrtDownloadCandidate) (*supplier.SubInfo, error) {
			return englishInfo, nil
		},
	)
	if ok {
		t.Fatal("expected english-only subtitle to be rejected")
	}
	if got != nil {
		t.Fatalf("firstUsableAssrtDownload() got %#v, want nil", got)
	}
	if err == nil || strings.Contains(err.Error(), "assrt unusable downloaded candidate") == false {
		t.Fatalf("unexpected error %v", err)
	}
}

func TestFilterBadDownloadCandidatesSkipsRememberedURL(t *testing.T) {
	supplier := &Supplier{}
	supplier.rememberBadDownloadURL("https://example.com/bad.zip")

	got := supplier.filterBadDownloadCandidates([]assrtDownloadCandidate{
		{url: "https://example.com/bad.zip", subName: "bad.zip"},
		{url: "https://example.com/good.zip", subName: "good.zip"},
	})
	if len(got) != 1 || got[0].url != "https://example.com/good.zip" {
		t.Fatalf("filterBadDownloadCandidates() = %#v; want only good candidate", got)
	}
}

func mustBuildZipBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		writer, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip.Create(%q) error = %v", name, err)
		}
		if _, err = writer.Write([]byte(body)); err != nil {
			t.Fatalf("zip.Write(%q) error = %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip.Close() error = %v", err)
	}

	return buf.Bytes()
}
