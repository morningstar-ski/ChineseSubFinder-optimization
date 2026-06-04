package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadManifest(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "manifest.json")
	content := `{
  "samples": [
    {
      "id": "movie-001",
      "video_path": "C:\\Media\\Movie.mkv",
      "kind": "movie"
    },
    {
      "id": "tv-001",
      "video_path": "C:\\Media\\Show\\S01E01.mkv",
      "kind": "episode",
      "season": 1,
      "episode": 1
    }
  ]
}`
	if err := os.WriteFile(inputPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	manifest, err := loadManifest(inputPath)
	if err != nil {
		t.Fatalf("loadManifest returned error: %v", err)
	}
	if len(manifest.Samples) != 2 {
		t.Fatalf("expected 2 samples, got %d", len(manifest.Samples))
	}
	if manifest.Samples[1].Season != 1 || manifest.Samples[1].Episode != 1 {
		t.Fatalf("unexpected episode sample %#v", manifest.Samples[1])
	}
}
