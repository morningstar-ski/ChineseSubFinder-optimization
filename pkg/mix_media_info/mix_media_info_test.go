package mix_media_info

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"
)

func TestNormalizeFileKeyword(t *testing.T) {
	videoFPath := filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL.mkv")

	got := normalizeFileKeyword(videoFPath)
	if got == "" {
		t.Fatal("expected non-empty keyword")
	}
	if !strings.Contains(got, "My Show") {
		t.Fatalf("expected normalized title in %q", got)
	}
	if strings.ContainsAny(got, "._-") {
		t.Fatalf("expected normalized keyword without punctuation: %q", got)
	}
}

func TestExpandSearchKeywordsAddsAmpersandVariants(t *testing.T) {
	got := ExpandSearchKeywords("Will & Harper", "Will and Harper")
	want := []string{
		"Will & Harper",
		"Will and Harper",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("ExpandSearchKeywords() = %#v; want %#v", got, want)
	}
}

func TestNormalizeComparableTitleCanonicalizesAmpersand(t *testing.T) {
	left := NormalizeComparableTitle("Will & Harper")
	right := NormalizeComparableTitle("Will and Harper")
	if left != right {
		t.Fatalf("NormalizeComparableTitle() mismatch: %q vs %q", left, right)
	}
}

func TestBuildFallbackMediaInfoUsesLocalNfo(t *testing.T) {
	tempDir := t.TempDir()
	videoPath := filepath.Join(tempDir, "Interstellar (2014).mkv")
	nfoPath := filepath.Join(tempDir, "Interstellar (2014).nfo")

	if err := os.WriteFile(videoPath, []byte("placeholder"), 0o644); err != nil {
		t.Fatalf("WriteFile video returned error: %v", err)
	}
	if err := os.WriteFile(nfoPath, []byte("<movie><title>Interstellar</title><imdbid>tt0816692</imdbid><releasedate>2014-11-07</releasedate></movie>"), 0o644); err != nil {
		t.Fatalf("WriteFile nfo returned error: %v", err)
	}

	got := buildFallbackMediaInfo(videoPath, true, &models.IMDBInfo{IMDBID: "tt0816692"})
	if got.ImdbId != "tt0816692" {
		t.Fatalf("ImdbId = %q", got.ImdbId)
	}
	if got.TitleEn != "Interstellar" {
		t.Fatalf("TitleEn = %q", got.TitleEn)
	}
	if got.OriginalTitle != "Interstellar" {
		t.Fatalf("OriginalTitle = %q", got.OriginalTitle)
	}
	if got.Year != "2014-11-07" {
		t.Fatalf("Year = %q", got.Year)
	}
}
