package assrt

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/ranking"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/random_auth_key"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	supplier2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
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

func TestShouldSkipAssrtCandidateForTargetSkipsWrongEpisode(t *testing.T) {
	sub := SearchSubItem{
		Id:         101,
		Videoname:  "My Show S01E01 1080p WEB-DL-GROUP",
		NativeName: "My Show S01E01 1080p WEB-DL-GROUP",
	}

	if shouldSkipAssrtCandidateForTarget(sub, nil, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false) == false {
		t.Fatalf("expected wrong episode candidate to be skipped")
	}
}

func TestShouldSkipAssrtCandidateForTargetKeepsSeasonPackAndUnknownEpisode(t *testing.T) {
	seasonPack := SearchSubItem{
		Id:         102,
		Videoname:  "My Show S01 Complete Pack 1080p WEB-DL-GROUP",
		NativeName: "My Show S01 Complete Pack 1080p WEB-DL-GROUP",
	}
	if shouldSkipAssrtCandidateForTarget(seasonPack, nil, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false) {
		t.Fatalf("expected season pack candidate to be kept")
	}

	unknown := SearchSubItem{
		Id:         103,
		Videoname:  "My Show 1080p WEB-DL-GROUP",
		NativeName: "My Show 1080p WEB-DL-GROUP",
	}
	if shouldSkipAssrtCandidateForTarget(unknown, nil, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false) {
		t.Fatalf("expected unknown-episode candidate to be kept")
	}
}

func TestShouldSkipAssrtCandidateForTargetSkipsWrongSeriesTitleEvenWithMatchingEpisode(t *testing.T) {
	videoPath := createAssrtEpisodeFixture(t, "George.Lopez.S01E02.1080p.WEB-DL-GROUP.mkv", 1, 2)
	mediaInfo := &models.MediaInfo{
		TitleCn:       "洛佩兹一家",
		TitleEn:       "George Lopez",
		OriginalTitle: "George Lopez",
	}
	sub := SearchSubItem{
		Id:         106,
		Videoname:  "Survival.of.the.Thickest.S01E02.720p.WEB.h264-EDITH",
		NativeName: "Survival.of.the.Thickest.S01E02.720p.WEB.h264-EDITH",
	}

	if shouldSkipAssrtCandidateForTarget(sub, mediaInfo, videoPath, false) == false {
		t.Fatalf("expected wrong series title candidate to be skipped")
	}
}

func TestShouldSkipAssrtCandidateForTargetKeepsMatchingSeriesTitleWithEpisode(t *testing.T) {
	videoPath := createAssrtEpisodeFixture(t, "George.Lopez.S01E02.1080p.WEB-DL-GROUP.mkv", 1, 2)
	mediaInfo := &models.MediaInfo{
		TitleCn:       "洛佩兹一家",
		TitleEn:       "George Lopez",
		OriginalTitle: "George Lopez",
	}
	sub := SearchSubItem{
		Id:         107,
		Videoname:  "George.Lopez.S01E02.720p.WEB.h264-GROUP",
		NativeName: "George.Lopez.S01E02.720p.WEB.h264-GROUP",
	}

	if shouldSkipAssrtCandidateForTarget(sub, mediaInfo, videoPath, false) {
		t.Fatalf("expected matching series title candidate to be kept")
	}
}

func TestShouldSkipAssrtCandidateForTargetSkipsWrongMovie(t *testing.T) {
	sub := SearchSubItem{
		Id:         104,
		Videoname:  "The Owl House S01E01 1080p WEB-DL-GROUP",
		NativeName: "The Owl House S01E01 1080p WEB-DL-GROUP",
	}

	if shouldSkipAssrtCandidateForTarget(sub, nil, filepath.Join("C:\\", "Media", "The.Kings.Warden.2026.1080p.WEB-DL-GROUP.mkv"), true) == false {
		t.Fatalf("expected wrong movie candidate to be skipped")
	}
}

func TestShouldSkipAssrtCandidateForTargetKeepsMatchingMovie(t *testing.T) {
	sub := SearchSubItem{
		Id:         105,
		Videoname:  "The Kings Warden 2026 1080p WEB-DL-GROUP",
		NativeName: "The Kings Warden 2026 1080p WEB-DL-GROUP",
	}

	if shouldSkipAssrtCandidateForTarget(sub, nil, filepath.Join("C:\\", "Media", "The.Kings.Warden.2026.1080p.WEB-DL-GROUP.mkv"), true) {
		t.Fatalf("expected matching movie candidate to be kept")
	}
}

func TestShouldSkipAssrtCandidateForTargetKeepsLocalizedMovieTitleVariants(t *testing.T) {
	videoPath := filepath.Join("C:\\", "Media", "夜班 (2025) - 1080p.mkv")
	mediaInfo := &models.MediaInfo{
		TitleCn:       "夜班",
		TitleEn:       "Late Shift",
		OriginalTitle: "Heldin",
	}

	english := SearchSubItem{
		Id:         106,
		Videoname:  "Late Shift 2025 1080p WEB-DL-GROUP",
		NativeName: "Late Shift 2025 1080p WEB-DL-GROUP",
	}
	if shouldSkipAssrtCandidateForTarget(english, mediaInfo, videoPath, true) {
		t.Fatalf("expected english movie title variant to be kept")
	}

	original := SearchSubItem{
		Id:         107,
		Videoname:  "Heldin 2025 1080p WEB-DL-GROUP",
		NativeName: "Heldin 2025 1080p WEB-DL-GROUP",
	}
	if shouldSkipAssrtCandidateForTarget(original, mediaInfo, videoPath, true) {
		t.Fatalf("expected original movie title variant to be kept")
	}
}

func TestSelectAssrtSearchKeywordAllowsFileFallbackWithoutMediaInfo(t *testing.T) {
	videoPath := createAssrtEpisodeFixture(t, "My.Show.1080p.WEB-DL.mkv", 1, 2)

	if _, err := selectAssrtSearchKeyword(nil, videoPath, false, "cn"); err == nil {
		t.Fatalf("expected non-file keyword to fail without media info")
	}

	got, err := selectAssrtSearchKeyword(nil, videoPath, false, "file")
	if err != nil {
		t.Fatalf("expected file keyword fallback to work, got error: %v", err)
	}
	if got != "My Show S01E02" {
		t.Fatalf("unexpected fallback keyword %q", got)
	}
}

func TestSelectAssrtSearchKeywordUsesMediaInfoWhenAvailable(t *testing.T) {
	videoPath := createAssrtEpisodeFixture(t, "Euphoria.1080p.WEB-DL.mkv", 1, 2)
	mediaInfo := &models.MediaInfo{
		TitleCn:       "亢奋",
		TitleEn:       "Euphoria",
		OriginalTitle: "Euphoria",
	}

	got, err := selectAssrtSearchKeyword(mediaInfo, videoPath, false, "cn")
	if err != nil {
		t.Fatalf("expected media-info keyword to work, got error: %v", err)
	}
	if got != "亢奋 S01E02" {
		t.Fatalf("unexpected media-info keyword %q", got)
	}
}

func TestGetSubInfoWithFallbackDeduplicatesResolvedKeywords(t *testing.T) {
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())
	cfg := settings.Get()
	oldRootURL := cfg.AdvancedSettings.SuppliersSettings.Assrt.RootUrl
	oldToken := cfg.SubtitleSources.AssrtSettings.Token
	t.Cleanup(func() {
		cfg.AdvancedSettings.SuppliersSettings.Assrt.RootUrl = oldRootURL
		cfg.SubtitleSources.AssrtSettings.Token = oldToken
	})

	var gotKeywords []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKeywords = append(gotKeywords, r.URL.Query().Get("q"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sub":{"action":"search","subs":[],"result":"succeed","keyword":"ok"},"status":0}`))
	}))
	defer server.Close()

	cfg.AdvancedSettings.SuppliersSettings.Assrt.RootUrl = server.URL
	cfg.SubtitleSources.AssrtSettings.Token = "test-token"

	supplier := NewSupplier(&file_downloader.FileDownloader{Log: log_helper.GetLogger4Tester()})
	supplier.theSearchInterval = 0

	videoPath := createAssrtEpisodeFixture(t, "George Lopez - S03E11 - Episode 11.mkv", 3, 11)
	mediaInfo := &models.MediaInfo{
		TitleCn:       "洛佩兹一家",
		TitleEn:       "George Lopez",
		OriginalTitle: "George Lopez",
	}

	got, err := supplier.getSubInfoWithFallback(mediaInfo, videoPath, false)
	if err != nil {
		t.Fatalf("getSubInfoWithFallback returned error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil result when all deduplicated searches miss, got %#v", got)
	}

	wantKeywords := []string{"洛佩兹一家 S03E11", "George Lopez S03E11"}
	if !reflect.DeepEqual(gotKeywords, wantKeywords) {
		t.Fatalf("search keywords = %#v; want %#v", gotKeywords, wantKeywords)
	}
}

func createAssrtEpisodeFixture(t *testing.T, videoName string, season int, episode int) string {
	t.Helper()

	seasonDir := filepath.Join(t.TempDir(), "Season 1")
	if err := os.MkdirAll(seasonDir, 0o755); err != nil {
		t.Fatal(err)
	}

	videoPath := filepath.Join(seasonDir, videoName)
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatal(err)
	}

	nfoPath := filepath.Join(seasonDir, strings.TrimSuffix(videoName, filepath.Ext(videoName))+".nfo")
	nfoBody := []byte(
		"<episodedetails>" +
			"<season>" + strconv.Itoa(season) + "</season>" +
			"<episode>" + strconv.Itoa(episode) + "</episode>" +
			"</episodedetails>",
	)
	if err := os.WriteFile(nfoPath, nfoBody, 0o644); err != nil {
		t.Fatal(err)
	}

	return videoPath
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

func TestWithAssrtRateLimitWaitsBeforeNextRequest(t *testing.T) {
	supplier := &Supplier{
		theSearchInterval: 40 * time.Millisecond,
		lastRequestAt:     time.Now(),
	}

	start := time.Now()
	if err := supplier.withAssrtRateLimit(func() error { return nil }); err != nil {
		t.Fatalf("withAssrtRateLimit returned error: %v", err)
	}
	if elapsed := time.Since(start); elapsed < 30*time.Millisecond {
		t.Fatalf("expected request limiter to wait, elapsed=%v", elapsed)
	}
}

func TestGetCachedSubInfoBySearchSubUsesCacheKey(t *testing.T) {
	cacheName := "assrt-cache-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	cache_center.DelDb(cacheName)
	t.Cleanup(func() {
		cache_center.DelDb(cacheName)
	})

	authKey := random_auth_key.AuthKey{
		BaseKey:  pkg.BaseKey(),
		AESKey16: pkg.AESKey16(),
		AESIv16:  pkg.AESIv16(),
	}
	fd := file_downloader.NewFileDownloader(newAssrtCacheCenterOrSkip(t, cacheName), authKey)
	supplier := NewSupplier(fd)
	supplier.theSearchInterval = 0

	searchSub := SearchSubItem{
		Id:         715414,
		NativeName: "The.Boys.S05E04",
	}
	cacheKey := assrtSearchSubCacheKey(supplier.GetSupplierName(), searchSub)
	subInfo := supplier2.NewSubInfo(
		supplier.GetSupplierName(),
		0,
		"The Boys S05E04",
		language.ChineseSimple,
		"https://example.com/the-boys-s05e04.srt",
		0,
		0,
		".srt",
		[]byte("1\n00:00:01,000 --> 00:00:02,000\nHello\n"),
	)
	subInfo.SetFileUrlSha256(cacheKey)
	if err := fd.CacheCenter.DownloadFileAdd(subInfo); err != nil {
		t.Fatalf("DownloadFileAdd returned error: %v", err)
	}

	found, got, err := supplier.getCachedSubInfoBySearchSub(cacheKey)
	if err != nil {
		t.Fatalf("getCachedSubInfoBySearchSub returned error: %v", err)
	}
	if found == false || got == nil {
		t.Fatalf("expected cached subtitle hit, found=%v got=%#v", found, got)
	}
}

func newAssrtCacheCenterOrSkip(t *testing.T, cacheName string) (cc *cache_center.CacheCenter) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprint(r)
			if strings.Contains(msg, "go-sqlite3 requires cgo to work") {
				t.Skip("skip assrt cache test: sqlite driver requires cgo in this environment")
			}
			panic(r)
		}
	}()

	return cache_center.NewCacheCenter(cacheName, log_helper.GetLogger4Tester())
}

func TestRememberBadDownloadSubIDOnlyForPermanentArchiveErrors(t *testing.T) {
	supplier := &Supplier{}
	subID := assrtFlexibleInt(42)

	if supplier.rememberBadDownloadSubID(subID, errors.New("temporary timeout")) {
		t.Fatalf("temporary error should not be remembered")
	}
	if supplier.shouldSkipBadDownloadSubID(subID) {
		t.Fatalf("temporary error should not mark subtitle as bad")
	}

	err := errors.New("invalid archive payload for https://example.com/file.zip: zip: not a valid zip file")
	if supplier.rememberBadDownloadSubID(subID, err) == false {
		t.Fatalf("invalid archive error should be remembered")
	}
	if supplier.shouldSkipBadDownloadSubID(subID) == false {
		t.Fatalf("remembered bad subtitle id should be skipped")
	}
}

func TestIsPermanentAssrtDownloadError(t *testing.T) {
	if isPermanentAssrtDownloadError(nil) {
		t.Fatalf("nil error should not be permanent")
	}
	if isPermanentAssrtDownloadError(errors.New("invalid archive payload for https://example.com/file.zip: zip: not a valid zip file")) == false {
		t.Fatalf("invalid archive payload should be treated as permanent")
	}
	if isPermanentAssrtDownloadError(errors.New("context deadline exceeded")) {
		t.Fatalf("transient timeout should not be treated as permanent")
	}
}

func TestRememberBadDownloadSubIDPersistsAcrossSupplierInstances(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "assrt_bad_download_sub_ids.json")
	err := errors.New("invalid archive payload for https://example.com/file.zip: zip: not a valid zip file")

	supplier := &Supplier{badDownloadSubIDsPath: cachePath}
	if supplier.rememberBadDownloadSubID(assrtFlexibleInt(42), err) == false {
		t.Fatalf("expected bad subtitle id to be remembered")
	}
	if supplier.shouldSkipBadDownloadSubID(assrtFlexibleInt(42)) == false {
		t.Fatalf("expected remembered subtitle id to be skipped")
	}

	other := &Supplier{badDownloadSubIDsPath: cachePath}
	other.loadPersistentBadDownloadSubIDs()
	if other.shouldSkipBadDownloadSubID(assrtFlexibleInt(42)) == false {
		t.Fatalf("expected persisted subtitle id to be skipped after reload")
	}
}

func TestLoadPersistentBadDownloadSubIDsPrunesExpiredEntries(t *testing.T) {
	cachePath := filepath.Join(t.TempDir(), "assrt_bad_download_sub_ids.json")
	entries := []persistentBadDownloadSubID{
		{ID: 11, UpdatedAt: time.Now().UTC().Add(-assrtBadDownloadSubIDTTL - time.Minute)},
		{ID: 12, UpdatedAt: time.Now().UTC().Add(-time.Minute)},
	}
	body, err := json.Marshal(entries)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(cachePath, body, 0o644); err != nil {
		t.Fatal(err)
	}

	supplier := &Supplier{badDownloadSubIDsPath: cachePath}
	supplier.loadPersistentBadDownloadSubIDs()

	if supplier.shouldSkipBadDownloadSubID(assrtFlexibleInt(11)) {
		t.Fatalf("expired subtitle id should not be skipped")
	}
	if supplier.shouldSkipBadDownloadSubID(assrtFlexibleInt(12)) == false {
		t.Fatalf("fresh subtitle id should be skipped")
	}

	prunedBody, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatal(err)
	}
	var pruned []persistentBadDownloadSubID
	if err = json.Unmarshal(prunedBody, &pruned); err != nil {
		t.Fatal(err)
	}
	if len(pruned) != 1 || pruned[0].ID != 12 {
		t.Fatalf("expected expired entries to be pruned, got %#v", pruned)
	}
}
