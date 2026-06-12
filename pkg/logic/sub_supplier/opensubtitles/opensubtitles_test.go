package opensubtitles

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/random_auth_key"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
)

func TestAPICheckAliveSuccess(t *testing.T) {
	loginCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != common.SubOpenSubtitlesLoginUrl {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		loginCalls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"token-1"}`))
	}))
	defer server.Close()

	api := NewApi(server.URL, "api-key", "user", "pass")
	client := newTestHTTPClient(t)

	if err := api.CheckAlive(client); err != nil {
		t.Fatalf("CheckAlive() error = %v", err)
	}
	if loginCalls != 1 {
		t.Fatalf("loginCalls = %d, want 1", loginCalls)
	}
	if api.token() != "token-1" {
		t.Fatalf("token = %q, want token-1", api.token())
	}
}

func TestAPICheckAliveInvalidCredentials(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errors":[{"title":"Unauthorized","detail":"invalid credentials"}]}`))
	}))
	defer server.Close()

	api := NewApi(server.URL, "api-key", "user", "wrong")
	client := newTestHTTPClient(t)

	if err := api.CheckAlive(client); err == nil {
		t.Fatal("expected invalid credentials error")
	}
}

func TestAPISearchSubtitles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case common.SubOpenSubtitlesLoginUrl:
			_, _ = w.Write([]byte(`{"token":"token-1"}`))
		case common.SubOpenSubtitlesSearchUrl:
			if got := r.URL.Query().Get("imdb_id"); got != "1234567" {
				t.Fatalf("imdb_id = %q, want 1234567", got)
			}
			_, _ = w.Write([]byte(`{"data":[{"id":"1","attributes":{"language":"zh-cn","release":"My.Show.S01E03.1080p.WEB-DL-GROUP","sub_format":"srt","files":[{"file_id":99,"file_name":"My.Show.S01E03.1080p.WEB-DL-GROUP.srt"}],"feature_details":{"season_number":1,"episode_number":3}}}]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	api := NewApi(server.URL, "api-key", "user", "pass")
	client := newTestHTTPClient(t)

	resp, err := api.SearchSubtitles(client, map[string]string{"imdb_id": "1234567"})
	if err != nil {
		t.Fatalf("SearchSubtitles() error = %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].Attributes.Files[0].FileID != 99 {
		t.Fatalf("unexpected response %#v", resp)
	}
}

func TestAPIDownloadByFileID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case common.SubOpenSubtitlesLoginUrl:
			_, _ = w.Write([]byte(`{"token":"token-1"}`))
		case common.SubOpenSubtitlesDownloadUrl:
			if r.Method != http.MethodPost {
				t.Fatalf("method = %s, want POST", r.Method)
			}
			body, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("ReadAll() error = %v", err)
			}
			if strings.TrimSpace(string(body)) != `{"file_id":99}` {
				t.Fatalf("body = %q, want file_id json", string(body))
			}
			_, _ = w.Write([]byte(`{"link":"https://cdn.example.com/subtitle.zip","file_name":"subtitle.zip"}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	api := NewApi(server.URL, "api-key", "user", "pass")
	client := newTestHTTPClient(t)

	resp, err := api.DownloadByFileID(client, 99)
	if err != nil {
		t.Fatalf("DownloadByFileID() error = %v", err)
	}
	if resp.Link != "https://cdn.example.com/subtitle.zip" {
		t.Fatalf("link = %q, want subtitle url", resp.Link)
	}
}

func TestAPIReLoginAfterUnauthorized(t *testing.T) {
	loginCalls := 0
	searchCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case common.SubOpenSubtitlesLoginUrl:
			loginCalls++
			_, _ = w.Write([]byte(`{"token":"token-` + strconv.Itoa(loginCalls) + `"}`))
		case common.SubOpenSubtitlesSearchUrl:
			searchCalls++
			if r.Header.Get("Authorization") == "Bearer token-1" {
				w.WriteHeader(http.StatusUnauthorized)
				_, _ = w.Write([]byte(`{"errors":[{"title":"Unauthorized","detail":"expired token"}]}`))
				return
			}
			_, _ = w.Write([]byte(`{"data":[]}`))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	api := NewApi(server.URL, "api-key", "user", "pass")
	client := newTestHTTPClient(t)

	_, err := api.SearchSubtitles(client, map[string]string{"query": "My Show"})
	if err != nil {
		t.Fatalf("SearchSubtitles() error = %v", err)
	}
	if loginCalls != 2 {
		t.Fatalf("loginCalls = %d, want 2", loginCalls)
	}
	if searchCalls != 2 {
		t.Fatalf("searchCalls = %d, want 2", searchCalls)
	}
}

func TestBuildSearchQueriesOrder(t *testing.T) {
	mediaInfo := &models.MediaInfo{
		ImdbId:        "tt1234567",
		TmdbId:        "5678",
		TitleEn:       "English Title",
		OriginalTitle: "Original Title",
		Year:          "2024-02-01",
	}

	queries := buildSearchQueries(mediaInfo, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false, 1, 3)
	if len(queries) < 4 {
		t.Fatalf("expected multiple queries, got %d", len(queries))
	}
	if queries[0]["imdb_id"] != "1234567" {
		t.Fatalf("expected imdb_id first, got %#v", queries[0])
	}
	if queries[1]["tmdb_id"] != "5678" {
		t.Fatalf("expected tmdb_id second, got %#v", queries[1])
	}
	if queries[2]["query"] != "English Title" {
		t.Fatalf("expected title query third, got %#v", queries[2])
	}
}

func TestBuildSearchQueriesWithoutMediaInfoFallsBackToQueryTitle(t *testing.T) {
	queries := buildSearchQueries(nil, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false, 1, 3)
	if len(queries) == 0 {
		t.Fatal("expected fallback query without media info")
	}
	if queries[0]["query"] != "My Show" {
		t.Fatalf("expected normalized title fallback, got %#v", queries[0])
	}
	if queries[0]["season_number"] != "1" || queries[0]["episode_number"] != "3" {
		t.Fatalf("expected episode params in fallback query, got %#v", queries[0])
	}
}

func TestOpenSubtitlesOrderedTitlesAddsPunctuationStrippedVariant(t *testing.T) {
	mediaInfo := &models.MediaInfo{
		TitleEn: "Will & Harper",
	}

	titles := openSubtitlesOrderedTitles(mediaInfo, filepath.Join("C:\\", "Media", "Will.&.Harper.2024.mkv"))
	if len(titles) < 2 {
		t.Fatalf("expected variants, got %#v", titles)
	}
	if titles[0] != "Will & Harper" {
		t.Fatalf("expected original title first, got %#v", titles)
	}
	found := false
	for _, title := range titles {
		if title == "Will Harper" {
			found = true
			break
		}
	}
	if found == false {
		t.Fatalf("expected punctuation-stripped variant in %#v", titles)
	}
}

func TestBuildSearchQueriesIncludesPunctuationStrippedMovieVariant(t *testing.T) {
	mediaInfo := &models.MediaInfo{
		TitleEn: "The King's Warden",
		Year:    "2026-01-01",
	}

	queries := buildSearchQueries(mediaInfo, filepath.Join("C:\\", "Media", "The.Kings.Warden.2026.1080p.WEB-DL-GROUP.mkv"), true, 0, 0)
	found := false
	for _, query := range queries {
		if query["query"] == "The Kings Warden" && query["year"] == "2026" {
			found = true
			break
		}
	}
	if found == false {
		t.Fatalf("expected punctuation-stripped movie query in %#v", queries)
	}
}

func TestShouldSkipOpenSubtitlesMovieCandidateSkipsWrongMovie(t *testing.T) {
	candidate := subtitleCandidate{
		FileID:       12537139,
		Name:         "The Drama",
		ReleaseNames: []string{"The Drama (2026)"},
	}
	mediaInfo := &models.MediaInfo{
		TitleEn:       "The King's Warden",
		OriginalTitle: "The King's Warden",
	}

	if shouldSkipOpenSubtitlesMovieCandidate(candidate, mediaInfo, filepath.Join("C:\\", "Media", "The.Kings.Warden.2026.1080p.WEB-DL-GROUP.mkv")) == false {
		t.Fatalf("expected wrong movie candidate to be skipped")
	}
}

func TestShouldSkipOpenSubtitlesMovieCandidateKeepsLocalizedVariant(t *testing.T) {
	candidate := subtitleCandidate{
		FileID:       668861,
		Name:         "Late Shift",
		ReleaseNames: []string{"Late.Shift.2025.GER.BluRay.720p.x264.DD.5.1-CMCT"},
	}
	mediaInfo := &models.MediaInfo{
		TitleCn:       "夜班",
		TitleEn:       "Late Shift",
		OriginalTitle: "Heldin",
	}

	if shouldSkipOpenSubtitlesMovieCandidate(candidate, mediaInfo, filepath.Join("C:\\", "Media", "夜班 (2025) - 1080p.mkv")) {
		t.Fatalf("expected localized movie candidate to be kept")
	}
}

func TestBuildSearchQueriesSkipsTooShortNonASCIIQuery(t *testing.T) {
	mediaInfo := &models.MediaInfo{
		ImdbId:        "tt8772296",
		TmdbId:        "85552",
		TitleEn:       "Euphoria",
		OriginalTitle: "Euphoria",
	}

	queries := buildSearchQueries(mediaInfo, filepath.Join("C:\\", "Media", "亢奋 - S01E01 - 第 1 集.mkv"), false, 1, 1)
	for _, query := range queries {
		if query["query"] == "亢奋" {
			t.Fatalf("unexpected short non-ascii query in %#v", queries)
		}
	}
}

func TestIsOpenSubtitlesQuotaExceeded(t *testing.T) {
	if isOpenSubtitlesQuotaExceeded(nil) != false {
		t.Fatal("nil error should not be quota exhaustion")
	}
	if isOpenSubtitlesQuotaExceeded(errors.New("opensubtitles http 406: quota exceeded")) == false {
		t.Fatal("http 406 should be treated as quota exhaustion")
	}
	if isOpenSubtitlesQuotaExceeded(errors.New("opensubtitles http 403: blocked")) != false {
		t.Fatal("http 403 should not be treated as quota exhaustion")
	}
}

func TestOverDailyDownloadLimitTreatsQuotaExhaustedAsExceeded(t *testing.T) {
	supplierInstance := &Supplier{
		log:            logrus.New(),
		quotaExhausted: true,
	}

	if supplierInstance.OverDailyDownloadLimit() == false {
		t.Fatal("expected quotaExhausted supplier to report daily limit reached")
	}
}

func TestGetSubListFromFileSkipsSearchWhenQuotaExhausted(t *testing.T) {
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())
	oldApiKey := settings.Get().SubtitleSources.OpenSubtitlesSettings.ApiKey
	oldUsername := settings.Get().SubtitleSources.OpenSubtitlesSettings.Username
	oldPassword := settings.Get().SubtitleSources.OpenSubtitlesSettings.Password
	settings.Get().SubtitleSources.OpenSubtitlesSettings.ApiKey = "api-key"
	settings.Get().SubtitleSources.OpenSubtitlesSettings.Username = "user"
	settings.Get().SubtitleSources.OpenSubtitlesSettings.Password = "pass"
	defer func() {
		settings.Get().SubtitleSources.OpenSubtitlesSettings.ApiKey = oldApiKey
		settings.Get().SubtitleSources.OpenSubtitlesSettings.Username = oldUsername
		settings.Get().SubtitleSources.OpenSubtitlesSettings.Password = oldPassword
	}()

	searchCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		searchCalls++
		t.Fatalf("unexpected http call for quota exhausted supplier: %s", r.URL.Path)
	}))
	defer server.Close()

	supplierInstance := &Supplier{
		log:            logrus.New(),
		quotaExhausted: true,
		api:            NewApi(server.URL, "api-key", "user", "pass"),
	}

	subInfos, err := supplierInstance.getSubListFromFile(filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false, 1, 3)
	if err != nil {
		t.Fatalf("getSubListFromFile() error = %v", err)
	}
	if subInfos != nil {
		t.Fatalf("expected nil subtitles when quota exhausted, got %#v", subInfos)
	}
	if searchCalls != 0 {
		t.Fatalf("searchCalls = %d; want 0", searchCalls)
	}
}

func TestSelectCandidatesPrefersExactEpisode(t *testing.T) {
	items := []SearchItem{
		{
			ID: "1",
			Attributes: SearchItemAttribute{
				Language:  "zh-cn",
				Release:   "My.Show.S01.1080p.WEB-DL-GROUP",
				SubFormat: "srt",
				Files: []SearchFile{
					{FileID: 10, FileName: "My.Show.S01.1080p.WEB-DL-GROUP.srt"},
				},
				FeatureDetails: FeatureDetails{SeasonNumber: 1},
			},
		},
		{
			ID: "2",
			Attributes: SearchItemAttribute{
				Language:  "zh-cn",
				Release:   "My.Show.S01E03.1080p.WEB-DL-GROUP",
				SubFormat: "srt",
				Files: []SearchFile{
					{FileID: 11, FileName: "My.Show.S01E03.1080p.WEB-DL-GROUP.srt"},
				},
				FeatureDetails: FeatureDetails{SeasonNumber: 1, EpisodeNumber: 3},
			},
		},
	}

	candidates := selectCandidates(items, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false, 1, 3, 5, 0, subtitleLanguageChinese)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %#v", candidates)
	}
	if candidates[0].FileID != 11 {
		t.Fatalf("expected exact episode first, got %#v", candidates[0])
	}
}

func TestSelectCandidatesRejectsWrongEpisodeDespiteCloserRelease(t *testing.T) {
	items := []SearchItem{
		{
			ID: "1",
			Attributes: SearchItemAttribute{
				Language:  "zh-cn",
				Release:   "My.Show.S01E04.1080p.WEB-DL-GROUP",
				SubFormat: "srt",
				Files: []SearchFile{
					{FileID: 10, FileName: "My.Show.S01E04.1080p.WEB-DL-GROUP.srt"},
				},
				FeatureDetails: FeatureDetails{SeasonNumber: 1, EpisodeNumber: 4},
			},
		},
		{
			ID: "2",
			Attributes: SearchItemAttribute{
				Language:  "zh-cn",
				Release:   "My.Show.S01E03.720p.HDTV-OTHER",
				SubFormat: "srt",
				Files: []SearchFile{
					{FileID: 11, FileName: "My.Show.S01E03.720p.HDTV-OTHER.srt"},
				},
				FeatureDetails: FeatureDetails{SeasonNumber: 1, EpisodeNumber: 3},
			},
		},
	}

	candidates := selectCandidates(items, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false, 1, 3, 5, 0, subtitleLanguageChinese)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %#v", candidates)
	}
	if candidates[0].FileID != 11 {
		t.Fatalf("expected exact episode first, got %#v", candidates[0])
	}
}

func TestSelectCandidatesCanFilterEnglishFallback(t *testing.T) {
	items := []SearchItem{
		{
			ID: "1",
			Attributes: SearchItemAttribute{
				Language:  "zh-cn",
				Release:   "My.Show.S01E03.1080p.WEB-DL-GROUP",
				SubFormat: "srt",
				Files: []SearchFile{
					{FileID: 10, FileName: "My.Show.S01E03.1080p.WEB-DL-GROUP.zh.srt"},
				},
				FeatureDetails: FeatureDetails{SeasonNumber: 1, EpisodeNumber: 3},
			},
		},
		{
			ID: "2",
			Attributes: SearchItemAttribute{
				Language:  "en",
				Release:   "My.Show.S01E03.1080p.WEB-DL-GROUP",
				SubFormat: "srt",
				Files: []SearchFile{
					{FileID: 11, FileName: "My.Show.S01E03.1080p.WEB-DL-GROUP.en.srt"},
				},
				FeatureDetails: FeatureDetails{SeasonNumber: 1, EpisodeNumber: 3},
			},
		},
	}

	candidates := selectCandidates(items, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false, 1, 3, 5, 0, subtitleLanguageEnglish)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 english candidate, got %#v", candidates)
	}
	if candidates[0].FileID != 11 {
		t.Fatalf("expected english candidate first, got %#v", candidates[0])
	}
}

func TestGetCachedSubInfoReturnsCachedSubtitle(t *testing.T) {
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())
	cacheName := "test_opensubtitles_cache_" + time.Now().Format("20060102150405.000000000")
	cacheCenter := newOpenSubtitlesCacheCenterOrSkip(t, cacheName)
	t.Cleanup(func() {
		cacheCenter.Close()
		cache_center.DelDb(cacheName)
	})

	downloader := file_downloader.NewFileDownloader(cacheCenter, random_auth_key.AuthKey{})
	cachedSubInfo := supplier.NewSubInfo(
		common.SubSiteOpenSubtitles,
		0,
		"cached.srt",
		language.ChineseSimple,
		"https://cdn.example.com/cached.srt",
		0,
		0,
		".srt",
		[]byte("1\n00:00:00,000 --> 00:00:01,000\ncached\n"),
	)
	cachedSubInfo.SetFileUrlSha256("opensubtitles-99")
	if err := cacheCenter.DownloadFileAdd(cachedSubInfo); err != nil {
		t.Fatalf("DownloadFileAdd() error = %v", err)
	}

	supplierInstance := &Supplier{fileDownloader: downloader, log: logrus.New()}
	got, found, err := supplierInstance.getCachedSubInfo("opensubtitles-99")
	if err != nil {
		t.Fatalf("getCachedSubInfo() error = %v", err)
	}
	if found == false {
		t.Fatal("expected cached subtitle hit")
	}
	if got == nil || got.FileUrl != cachedSubInfo.FileUrl {
		t.Fatalf("unexpected cached subtitle %#v", got)
	}
}

func newTestHTTPClient(t *testing.T) *resty.Client {
	t.Helper()

	client, err := pkg.NewHttpClient()
	if err != nil {
		t.Fatalf("NewHttpClient() error = %v", err)
	}
	client.SetRetryCount(0)
	return client
}

func newOpenSubtitlesCacheCenterOrSkip(t *testing.T, cacheName string) *cache_center.CacheCenter {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprint(r)
			if strings.Contains(msg, "go-sqlite3 requires cgo to work") {
				t.Skip("skip opensubtitles cache test: sqlite driver requires cgo in this environment")
			}
			panic(r)
		}
	}()

	cache_center.DelDb(cacheName)
	return cache_center.NewCacheCenter(cacheName, logrus.New())
}
