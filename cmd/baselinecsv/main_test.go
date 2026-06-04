package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadResults(t *testing.T) {
	tempDir := t.TempDir()
	inputPath := filepath.Join(tempDir, "results.json")
	content := `[
  {
    "sample": {
      "id": "movie-001",
      "video_path": "C:\\Media\\Movie.mkv",
      "kind": "movie"
    },
    "attempts": [
      {
        "provider": "subdl",
        "hit": true,
        "downloaded": true
      }
    ]
  }
]`
	if err := os.WriteFile(inputPath, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	results, err := loadResults(inputPath)
	if err != nil {
		t.Fatalf("loadResults returned error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Sample.ID != "movie-001" {
		t.Fatalf("unexpected sample id %#v", results[0].Sample)
	}
	if len(results[0].Attempts) != 1 || results[0].Attempts[0].Provider != "subdl" {
		t.Fatalf("unexpected attempts %#v", results[0].Attempts)
	}
}
