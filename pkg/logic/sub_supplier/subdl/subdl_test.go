package subdl

import (
	"encoding/json"
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

func TestSearchResponseUsesCurrentSubtitlesField(t *testing.T) {
	payload := []byte(`{
		"status": true,
		"results": [
			{
				"sd_id": 21581,
				"name": "The Matrix"
			}
		],
		"subtitles": [
			{
				"release_name": "The.Matrix.1999.WEBRip.iTunes",
				"name": "the-matrix_chinese-bg-code-2873685.zip",
				"lang": "chinese bg code",
				"url": "/subtitle/3367071-2873685.zip",
				"season": 0,
				"episode": null,
				"language": "ZH",
				"hi": false
			}
		]
	}`)

	var response SearchResponse
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	response.populateLegacyResults()

	hits := response.SubtitleHits()
	if len(hits) != 1 {
		t.Fatalf("expected 1 subtitle hit, got %#v", hits)
	}
	if hits[0].ReleaseName != "The.Matrix.1999.WEBRip.iTunes" {
		t.Fatalf("unexpected release name %#v", hits[0])
	}
}

func TestSelectCandidatesUsesReleaseNameFallback(t *testing.T) {
	results := []SubtitleHit{
		{
			Name:        "the-matrix_chinese-bg-code.zip",
			URL:         "/subtitle/matrix.zip",
			Season:      0,
			Episode:     0,
			ReleaseName: "The.Matrix.1999.WEBRip.iTunes",
		},
	}

	candidates := selectCandidates(results, filepath.Join("C:\\", "Media", "The.Matrix.1999.WEBRip.iTunes.mkv"), true, 0, 0, 5)
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %#v", candidates)
	}
	if candidates[0].Name != "The.Matrix.1999.WEBRip.iTunes" {
		t.Fatalf("expected release name as candidate name, got %#v", candidates[0])
	}
	if len(candidates[0].Releases) != 1 || candidates[0].Releases[0] != "The.Matrix.1999.WEBRip.iTunes" {
		t.Fatalf("expected release fallback, got %#v", candidates[0])
	}
}
