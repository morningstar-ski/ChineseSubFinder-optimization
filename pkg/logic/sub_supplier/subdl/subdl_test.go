package subdl

import (
	"path/filepath"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"
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

	candidates := selectCandidates(results, false, 1, 3, 3)
	if len(candidates) < 2 {
		t.Fatalf("expected exact unpack file plus fallback pack, got %#v", candidates)
	}
	if candidates[0].DownloadURL != "https://dl.subdl.com/file/e03.srt" {
		t.Fatalf("expected exact unpack file first, got %#v", candidates[0])
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
