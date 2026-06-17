package mix_media_info

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestNormalizePathKeywordPrefersMovieRootTitleForFakeBDMV(t *testing.T) {
	root := t.TempDir()
	movieDir := filepath.Join(root, "The Shameless 2024 (2024)")
	if err := os.MkdirAll(filepath.Join(movieDir, "CERTIFICATE"), 0o755); err != nil {
		t.Fatalf("MkdirAll CERTIFICATE error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(movieDir, "BDMV"), 0o755); err != nil {
		t.Fatalf("MkdirAll BDMV error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(movieDir, "CERTIFICATE", "id.bdmv"), []byte("fake"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	got := normalizePathKeyword(filepath.Join(movieDir, "00000.m2ts"), true)
	if got != "The Shameless 2024" {
		t.Fatalf("normalizePathKeyword() = %q; want %q", got, "The Shameless 2024")
	}
}

func TestNormalizePathKeywordPrefersSeriesRootTitle(t *testing.T) {
	videoFPath := filepath.Join("C:\\", "Media", "Rick and Morty (2013)", "Season 5", "瑞克和莫蒂 - S05E09 - 第 9 集.mkv")

	got := normalizePathKeyword(videoFPath, false)
	if got != "Rick and Morty" {
		t.Fatalf("normalizePathKeyword() = %q; want Rick and Morty", got)
	}
}

func TestKeyWordSelectFallsBackWhenMediaInfoIsNil(t *testing.T) {
	videoFPath := filepath.Join("C:\\", "Media", "Rick and Morty (2013)", "Season 5", "瑞克和莫蒂 - S05E09 - 第 9 集.mkv")

	got, err := KeyWordSelect(nil, videoFPath, true, "cn")
	if err != nil {
		t.Fatalf("KeyWordSelect() error = %v", err)
	}
	if got != "Rick and Morty" {
		t.Fatalf("KeyWordSelect() = %q; want Rick and Morty", got)
	}
}
