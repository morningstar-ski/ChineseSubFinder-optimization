package subdl

import (
	"errors"
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

func TestBuildSearchQueriesWithoutMediaInfoFallsBackToFilmName(t *testing.T) {
	supplier := &Supplier{topic: 1, api: NewApi("test-key")}

	queries := supplier.buildSearchQueries(nil, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL.mkv"), false, 1, 3)
	if len(queries) == 0 {
		t.Fatal("expected fallback queries without media info")
	}
	if queries[0]["film_name"] != "My Show" {
		t.Fatalf("expected file-name fallback query, got %#v", queries[0])
	}
	if queries[0]["season_number"] != "1" || queries[0]["episode_number"] != "3" {
		t.Fatalf("expected episode params in fallback query, got %#v", queries[0])
	}
}

func TestBuildSearchQueriesUsesEnglishLanguageForEnglishSupplier(t *testing.T) {
	supplier := &Supplier{topic: 1, api: NewApi("test-key"), queryLanguage: subdlEnglishLanguage}

	queries := supplier.buildSearchQueries(nil, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL.mkv"), false, 1, 3)
	if len(queries) == 0 {
		t.Fatal("expected fallback queries without media info")
	}
	if queries[0]["languages"] != subdlEnglishLanguage {
		t.Fatalf("expected english language query, got %#v", queries[0])
	}
}

func TestOrderedSearchTitlesAddsPunctuationStrippedVariant(t *testing.T) {
	mediaInfo := &models.MediaInfo{
		TitleEn: "Will & Harper",
	}

	titles := orderedSearchTitles(mediaInfo, filepath.Join("C:\\", "Media", "Will.&.Harper.2024.mkv"))
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

func TestShouldTreatCheckAliveProbeAsHealthy(t *testing.T) {
	if shouldTreatCheckAliveProbeAsHealthy(nil) == false {
		t.Fatal("nil probe error should be treated as healthy")
	}
	if shouldTreatCheckAliveProbeAsHealthy(errSubdlStatusFalse) == false {
		t.Fatal("status=false probe should be treated as healthy")
	}
	if shouldTreatCheckAliveProbeAsHealthy(errors.New("network down")) != false {
		t.Fatal("network failure should not be treated as healthy")
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
