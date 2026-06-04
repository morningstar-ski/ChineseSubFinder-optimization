package mix_media_info

import (
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
