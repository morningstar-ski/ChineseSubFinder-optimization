package file_downloader

import "testing"

func TestBuildCacheKeyStableAndSafe(t *testing.T) {
	key1 := BuildCacheKey("subdl", "tt8772296", "0", "https://dl.subdl.com/subtitle/9n1K1x5VGn6/NFOgefnyXQ")
	key2 := BuildCacheKey("subdl", "tt8772296", "0", "https://dl.subdl.com/subtitle/9n1K1x5VGn6/NFOgefnyXQ")

	if key1 != key2 {
		t.Fatalf("expected stable cache key, got %q and %q", key1, key2)
	}
	if len(key1) != 64 {
		t.Fatalf("expected sha256 hex cache key, got %q", key1)
	}
	for _, ch := range `\/:*?"<>|` {
		if containsRune(key1, ch) {
			t.Fatalf("cache key contains unsafe rune %q: %q", ch, key1)
		}
	}
}

func containsRune(s string, want rune) bool {
	for _, ch := range s {
		if ch == want {
			return true
		}
	}
	return false
}
