package opensubtitles

import (
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

	candidates := selectCandidates(items, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false, 1, 3, 5, 0)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %#v", candidates)
	}
	if candidates[0].FileID != 11 {
		t.Fatalf("expected exact episode first, got %#v", candidates[0])
	}
}

func newTestHTTPClient(t *testing.T) *resty.Client {
	t.Helper()

	client, err := pkg.NewHttpClient()
	if err != nil {
		t.Fatalf("NewHttpClient() error = %v", err)
	}
	return client
}
