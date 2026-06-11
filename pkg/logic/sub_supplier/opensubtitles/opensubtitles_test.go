package opensubtitles

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/go-resty/resty/v2"
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

func TestFinalizeDownloadAttemptsReturnsLastDownloadError(t *testing.T) {
	wantErr := errors.New("opensubtitles http 406: quota exceeded")

	got, err := finalizeDownloadAttempts(nil, wantErr)
	if err == nil {
		t.Fatal("finalizeDownloadAttempts() error = nil, want quota error")
	}
	if err.Error() != wantErr.Error() {
		t.Fatalf("finalizeDownloadAttempts() error = %q, want %q", err.Error(), wantErr.Error())
	}
	if got != nil {
		t.Fatalf("finalizeDownloadAttempts() got %#v, want nil", got)
	}
}

func TestFinalizeDownloadAttemptsPrefersDownloadedSubs(t *testing.T) {
	want := []supplier.SubInfo{{Name: "subtitle.srt"}}

	got, err := finalizeDownloadAttempts(want, errors.New("ignored"))
	if err != nil {
		t.Fatalf("finalizeDownloadAttempts() error = %v, want nil", err)
	}
	if len(got) != 1 || got[0].Name != want[0].Name {
		t.Fatalf("finalizeDownloadAttempts() got %#v, want %#v", got, want)
	}
}

func TestIsQuotaExceededOpenSubtitlesError(t *testing.T) {
	if isQuotaExceededOpenSubtitlesError(errors.New("opensubtitles http 406: You have downloaded your allowed 20 subtitles for 24h")) == false {
		t.Fatal("expected quota exceeded marker to be detected")
	}
	if isQuotaExceededOpenSubtitlesError(errors.New("opensubtitles login failed")) {
		t.Fatal("unexpected quota exceeded marker for unrelated error")
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

func TestOpenSubtitlesOrderedTitlesAddsYearlessFallback(t *testing.T) {
	mediaInfo := &models.MediaInfo{
		TitleEn:       "Nirvana the Band the Show the Movie 2026",
		OriginalTitle: "Nirvana the Band the Show the Movie (2026)",
		TitleCn:       "电影版 2026",
	}

	got := openSubtitlesOrderedTitles(mediaInfo, filepath.Join("C:\\", "Media", "Nirvana.the.Band.the.Show.the.Movie.2026.1080p.WEB-DL.mkv"))
	if len(got) < 5 {
		t.Fatalf("openSubtitlesOrderedTitles() = %#v; want yearless fallbacks", got)
	}

	want := map[string]bool{
		"Nirvana the Band the Show the Movie 2026": false,
		"Nirvana the Band the Show the Movie":      false,
		"电影版 2026":                                 false,
		"电影版":                                      false,
	}
	for _, title := range got {
		if _, ok := want[title]; ok {
			want[title] = true
		}
	}
	for title, seen := range want {
		if seen == false {
			t.Fatalf("openSubtitlesOrderedTitles() missing %q in %#v", title, got)
		}
	}
}

func TestBuildSearchQueriesDropsYearForYearlessMovieFallback(t *testing.T) {
	mediaInfo := &models.MediaInfo{
		TitleEn:       "The Gorge 2025",
		OriginalTitle: "The Gorge (2025)",
		Year:          "2025-01-01",
	}

	queries := buildSearchQueries(mediaInfo, filepath.Join("C:\\", "Media", "The.Gorge.2025.1080p.WEB-DL.mkv"), true, 0, 0)
	var foundWithYear bool
	var foundWithoutYear bool
	for _, query := range queries {
		if query["query"] != "The Gorge" {
			continue
		}
		if query["year"] == "2025" {
			foundWithYear = true
			continue
		}
		if query["year"] == "" {
			foundWithoutYear = true
		}
	}
	if foundWithYear {
		t.Fatalf("buildSearchQueries() should not keep year filter on yearless fallback: %#v", queries)
	}
	if foundWithoutYear == false {
		t.Fatalf("buildSearchQueries() missing yearless fallback query in %#v", queries)
	}
}

func TestBuildSearchQueriesSkipsTooShortNonASCIIQuery(t *testing.T) {
	mediaInfo := &models.MediaInfo{
		TitleEn: "Euphoria",
		TitleCn: "亢奋",
		Year:    "2019-01-01",
	}

	queries := buildSearchQueries(mediaInfo, filepath.Join("C:\\", "Media", "Euphoria.S01E01.1080p.WEB-DL.mkv"), false, 1, 1)
	for _, query := range queries {
		if query["query"] == "亢奋" {
			t.Fatalf("buildSearchQueries() should skip too-short non-ASCII query: %#v", queries)
		}
	}
}

func TestBuildSearchQueriesAddsAmpersandVariant(t *testing.T) {
	mediaInfo := &models.MediaInfo{
		TitleEn: "Will & Harper",
		Year:    "2024-01-01",
	}

	queries := buildSearchQueries(mediaInfo, filepath.Join("C:\\", "Media", "Will.and.Harper.2024.1080p.WEB-DL.mkv"), true, 0, 0)
	foundAndVariant := false
	for _, query := range queries {
		if query["query"] == "Will and Harper" {
			foundAndVariant = true
			break
		}
	}
	if foundAndVariant == false {
		t.Fatalf("buildSearchQueries() missing ampersand fallback in %#v", queries)
	}
}

func TestTitlesRoughlyMatchTreatsAmpersandAsAnd(t *testing.T) {
	if titlesRoughlyMatch("Will & Harper", "Will and Harper") == false {
		t.Fatal("expected ampersand and and titles to match")
	}
}

func TestIsIgnorableOpenSubtitlesSearchError(t *testing.T) {
	if isIgnorableOpenSubtitlesSearchError(errors.New(`opensubtitles http 400: {"errors":["Query is too short"],"status":400}`)) == false {
		t.Fatal("expected short-query error to be ignorable")
	}
	if isIgnorableOpenSubtitlesSearchError(errors.New("network timeout")) {
		t.Fatal("did not expect generic error to be ignorable")
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

	candidates := selectCandidates(items, nil, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false, 1, 3, 5, 0)
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

	candidates := selectCandidates(items, nil, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false, 1, 3, 5, 0)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %#v", candidates)
	}
	if candidates[0].FileID != 11 {
		t.Fatalf("expected exact episode first, got %#v", candidates[0])
	}
}

func TestSelectCandidatesRejectsWrongMovieTitle(t *testing.T) {
	mediaInfo := &models.MediaInfo{
		TitleEn:       "Nirvana the Band the Show the Movie",
		OriginalTitle: "Nirvana the Band the Show the Movie",
		Year:          "2026-01-01",
	}

	items := []SearchItem{
		{
			ID: "1",
			Attributes: SearchItemAttribute{
				Language:  "zh-cn",
				Release:   "The.Muppet.Show.2026.1080p.WEB-DL-GROUP",
				MovieName: "The Muppet Show",
				SubFormat: "srt",
				Files: []SearchFile{
					{FileID: 10, FileName: "The.Muppet.Show.2026.1080p.WEB-DL-GROUP.srt"},
				},
				FeatureDetails: FeatureDetails{Title: "The Muppet Show", Year: 2026},
			},
		},
	}

	candidates := selectCandidates(items, mediaInfo, filepath.Join("C:\\", "Media", "尼瓦那乐队秀：电影版 (2026).mkv"), true, 0, 0, 5, 0)
	if len(candidates) != 0 {
		t.Fatalf("expected wrong movie title to be rejected, got %#v", candidates)
	}
}

func TestSelectCandidatesKeepsMatchingMovieTitle(t *testing.T) {
	mediaInfo := &models.MediaInfo{
		TitleEn:       "Nirvana the Band the Show the Movie",
		OriginalTitle: "Nirvana the Band the Show the Movie",
		Year:          "2026-01-01",
	}

	items := []SearchItem{
		{
			ID: "1",
			Attributes: SearchItemAttribute{
				Language:  "zh-cn",
				Release:   "Nirvana.The.Band.The.Show.The.Movie.2026.1080p.WEB-DL-GROUP",
				MovieName: "Nirvana the Band the Show the Movie",
				SubFormat: "srt",
				Files: []SearchFile{
					{FileID: 11, FileName: "Nirvana.The.Band.The.Show.The.Movie.2026.1080p.WEB-DL-GROUP.srt"},
				},
				FeatureDetails: FeatureDetails{Title: "Nirvana the Band the Show the Movie", Year: 2026},
			},
		},
	}

	candidates := selectCandidates(items, mediaInfo, filepath.Join("C:\\", "Media", "尼瓦那乐队秀：电影版 (2026).mkv"), true, 0, 0, 5, 0)
	if len(candidates) != 1 {
		t.Fatalf("expected matching movie title to remain, got %#v", candidates)
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
