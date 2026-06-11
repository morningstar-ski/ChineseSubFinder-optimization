package subdl

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"
	"github.com/go-resty/resty/v2"
)

func TestBuildSearchQueries(t *testing.T) {
	supplier := &Supplier{topic: 1, api: NewApi("test-key")}
	mediaInfo := &models.MediaInfo{
		ImdbId:        "tt1234567",
		TmdbId:        "1234",
		TitleEn:       "English Title",
		OriginalTitle: "Original Title",
		TitleCn:       "中文名",
		Year:          "2024-01-01",
	}

	queries := supplier.buildSearchQueries(mediaInfo, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL.mkv"), false, 1, 3)
	if len(queries) < 4 {
		t.Fatalf("expected multiple fallback queries, got %d", len(queries))
	}
	if queries[0]["imdb_id"] != "tt1234567" {
		t.Fatalf("expected imdb_id first, got %#v", queries[0])
	}
	if queries[0]["type"] != "tv" || queries[0]["season_number"] != "1" || queries[0]["episode_number"] != "3" {
		t.Fatalf("expected tv episode params, got %#v", queries[0])
	}
}

func TestOrderedSearchTitlesAddsYearlessFallbacksAndDedupes(t *testing.T) {
	mediaInfo := &models.MediaInfo{
		TitleEn:       "The Gorge 2025",
		OriginalTitle: "The Gorge (2025)",
		TitleCn:       "The Gorge",
	}

	got := orderedSearchTitles(mediaInfo, filepath.Join("C:\\", "Media", "The.Gorge.2025.1080p.WEB-DL.mkv"))
	foundYearless := false
	for _, item := range got {
		if item == "The Gorge" {
			foundYearless = true
			break
		}
	}
	if foundYearless == false {
		t.Fatalf("orderedSearchTitles() = %#v; want yearless fallback", got)
	}
	seen := make(map[string]struct{}, len(got))
	for _, item := range got {
		if _, ok := seen[item]; ok {
			t.Fatalf("orderedSearchTitles() contains duplicate title %q in %#v", item, got)
		}
		seen[item] = struct{}{}
	}
}

func TestOrderedSearchTitlesAddsAmpersandVariant(t *testing.T) {
	mediaInfo := &models.MediaInfo{
		TitleEn: "Will & Harper",
	}

	got := orderedSearchTitles(mediaInfo, filepath.Join("C:\\", "Media", "Will.and.Harper.2024.1080p.WEB-DL.mkv"))
	foundAmpersand := false
	foundAnd := false
	for _, item := range got {
		if item == "Will & Harper" {
			foundAmpersand = true
		}
		if item == "Will and Harper" {
			foundAnd = true
		}
	}
	if foundAmpersand == false || foundAnd == false {
		t.Fatalf("orderedSearchTitles() = %#v; want both ampersand and and variants", got)
	}
}

func TestBuildSearchQueriesDropsYearForYearlessFallback(t *testing.T) {
	supplier := &Supplier{topic: 1, api: NewApi("test-key")}
	mediaInfo := &models.MediaInfo{
		TitleEn:       "The Gorge 2025",
		OriginalTitle: "The Gorge (2025)",
		Year:          "2025-01-01",
	}

	queries := supplier.buildSearchQueries(mediaInfo, filepath.Join("C:\\", "Media", "The.Gorge.2025.1080p.WEB-DL.mkv"), true, 0, 0)
	var foundWithYear bool
	var foundWithoutYear bool
	for _, query := range queries {
		if query["film_name"] != "The Gorge" {
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

func TestBuildSearchQueriesDoesNotApplyYearToIDQueries(t *testing.T) {
	supplier := &Supplier{topic: 1, api: NewApi("test-key")}
	mediaInfo := &models.MediaInfo{
		ImdbId: "tt1234567",
		TmdbId: "1234",
		Year:   "2025-01-01",
	}

	queries := supplier.buildSearchQueries(mediaInfo, filepath.Join("C:\\", "Media", "The.Gorge.2025.1080p.WEB-DL.mkv"), true, 0, 0)
	for _, query := range queries {
		if query["imdb_id"] == "" && query["tmdb_id"] == "" {
			continue
		}
		if query["year"] != "" {
			t.Fatalf("buildSearchQueries() should not attach year to ID query: %#v", query)
		}
	}
}

func TestSelectCandidatesPrefersExactUnpackFiles(t *testing.T) {
	results := []SubtitleHit{
		{
			Name:    "Season Pack",
			URL:     "/subtitle/season-pack.zip",
			Season:  1,
			Episode: 0,
			UnpackFiles: []UnpackFile{
				{Name: "Episode 2.srt", URL: "/file/e02.srt", Season: 1, Episode: 2},
				{Name: "Episode 3.srt", URL: "/file/e03.srt", Season: 1, Episode: 3},
			},
		},
	}

	candidates := selectCandidates(results, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false, 1, 3, 3)
	if len(candidates) < 2 {
		t.Fatalf("expected exact unpack file plus fallback pack, got %#v", candidates)
	}
	if candidates[0].DownloadURL != "https://dl.subdl.com/file/e03.srt" {
		t.Fatalf("expected exact unpack file first, got %#v", candidates[0])
	}
}

func TestSelectCandidatesPrefersMatchingReleaseMetadata(t *testing.T) {
	results := []SubtitleHit{
		{
			Name:     "Show subtitle 720p",
			URL:      "/subtitle/720.zip",
			Season:   1,
			Episode:  3,
			Releases: []string{"My.Show.S01E03.720p.HDTV-OTHER"},
		},
		{
			Name:     "Show subtitle 1080p",
			URL:      "/subtitle/1080.zip",
			Season:   1,
			Episode:  3,
			Releases: []string{"My.Show.S01E03.1080p.WEB-DL-GROUP"},
		},
	}

	candidates := selectCandidates(results, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false, 1, 3, 5)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %#v", candidates)
	}
	if candidates[0].DownloadURL != "https://dl.subdl.com/subtitle/1080.zip" {
		t.Fatalf("expected metadata-matching candidate first, got %#v", candidates[0])
	}
}

func TestSelectCandidatesPenalizesWrongEpisodeEvenWithBetterAuthority(t *testing.T) {
	results := []SubtitleHit{
		{
			Name:     "Wrong episode",
			URL:      "/subtitle/e04.zip",
			Season:   1,
			Episode:  4,
			Releases: []string{"My.Show.S01E04.1080p.WEB-DL-GROUP"},
		},
		{
			Name:     "Exact episode",
			URL:      "/subtitle/e03.zip",
			Season:   1,
			Episode:  3,
			Releases: []string{"My.Show.S01E03.720p.HDTV-OTHER"},
		},
	}

	candidates := selectCandidates(results, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false, 1, 3, 5)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %#v", candidates)
	}
	if candidates[0].DownloadURL != "https://dl.subdl.com/subtitle/e03.zip" {
		t.Fatalf("expected exact episode first, got %#v", candidates[0])
	}
}

func TestNormalizeDownloadURL(t *testing.T) {
	tests := map[string]string{
		"/subtitle/123.zip":           "https://dl.subdl.com/subtitle/123.zip",
		"subtitle/123.zip":            "https://dl.subdl.com/subtitle/123.zip",
		"https://example.com/123.zip": "https://example.com/123.zip",
	}

	for input, want := range tests {
		if got := normalizeDownloadURL(input); got != want {
			t.Fatalf("normalizeDownloadURL(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestSubdlCandidateMetadata(t *testing.T) {
	candidate := subtitleCandidate{
		Name:     "candidate-name",
		Season:   1,
		Episode:  3,
		Hi:       true,
		Releases: []string{"Release.A", "Release.B"},
	}

	metadata := subdlCandidateMetadata(candidate)
	if metadata.Name != candidate.Name || metadata.Season != 1 || metadata.Episode != 3 || metadata.HasHI != true {
		t.Fatalf("unexpected metadata %#v", metadata)
	}
	if len(metadata.ReleaseNames) != 2 || metadata.ReleaseNames[0] != "Release.A" || metadata.ReleaseNames[1] != "Release.B" {
		t.Fatalf("unexpected release names %#v", metadata.ReleaseNames)
	}
}

func TestSearchSubtitlesTreatsStatusFalseAsEmptyResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":false,"results":[]}`))
	}))
	defer server.Close()

	client := resty.New()
	client.SetBaseURL(server.URL)

	api := NewApi("test-key")
	resp, err := api.SearchSubtitles(client, map[string]string{
		"api_key": "test-key",
		"type":    "tv",
	})
	if err != nil {
		t.Fatalf("SearchSubtitles() error = %v", err)
	}
	if resp == nil {
		t.Fatal("SearchSubtitles() response is nil")
	}
	if len(resp.Results) != 0 {
		t.Fatalf("SearchSubtitles() results len = %d; want 0", len(resp.Results))
	}
}
