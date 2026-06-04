package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/baseline"
)

func TestRun(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "manifest.json")
	fixturePath := filepath.Join(tempDir, "fixture.json")
	outputPath := filepath.Join(tempDir, "results.json")

	manifestContent := `{
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
	fixtureContent := `[
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
  },
  {
    "sample": {
      "id": "tv-001",
      "video_path": "C:\\Media\\Show\\S01E01.mkv",
      "kind": "episode",
      "season": 1,
      "episode": 1
    },
    "primary_failure": "keyword_search_miss",
    "attempts": [
      {
        "provider": "assrt",
        "hit": false,
        "downloaded": false,
        "failure_category": "keyword_search_miss"
      }
    ]
  }
]`

	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0o644); err != nil {
		t.Fatalf("WriteFile manifest returned error: %v", err)
	}
	if err := os.WriteFile(fixturePath, []byte(fixtureContent), 0o644); err != nil {
		t.Fatalf("WriteFile fixture returned error: %v", err)
	}

	if err := run(manifestPath, fixturePath, outputPath); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	results, err := baseline.LoadResults(outputPath)
	if err != nil {
		t.Fatalf("LoadResults returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[1].PrimaryFailure != baseline.FailureKeywordMiss {
		t.Fatalf("unexpected primary failure %q", results[1].PrimaryFailure)
	}
}
