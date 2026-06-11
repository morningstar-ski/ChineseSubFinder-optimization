package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tempDir := t.TempDir()
	beforePath := filepath.Join(tempDir, "before.json")
	afterPath := filepath.Join(tempDir, "after.json")
	outputPath := filepath.Join(tempDir, "comparison.json")
	csvPath := filepath.Join(tempDir, "comparison.csv")

	beforeContent := `[
  {
    "sample": {
      "id": "episode-001",
      "video_path": "C:\\Media\\Show\\S01E01.mkv",
      "kind": "episode",
      "season": 1,
      "episode": 1
    },
    "primary_failure": "no_provider_hit",
    "attempts": [
      {
        "provider": "assrt",
        "hit": false,
        "downloaded": false,
        "failure_category": "no_provider_hit"
      }
    ]
  }
]`
	afterContent := `[
  {
    "sample": {
      "id": "episode-001",
      "video_path": "C:\\Media\\Show\\S01E01.mkv",
      "kind": "episode",
      "season": 1,
      "episode": 1
    },
    "attempts": [
      {
        "provider": "assrt",
        "hit": true,
        "downloaded": true
      }
    ]
  }
]`

	if err := os.WriteFile(beforePath, []byte(beforeContent), 0o644); err != nil {
		t.Fatalf("WriteFile before returned error: %v", err)
	}
	if err := os.WriteFile(afterPath, []byte(afterContent), 0o644); err != nil {
		t.Fatalf("WriteFile after returned error: %v", err)
	}

	if err := run(beforePath, afterPath, outputPath, csvPath); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	outputBytes, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile output returned error: %v", err)
	}
	if strings.Contains(string(outputBytes), "\"improved_samples\": 1") == false {
		t.Fatalf("unexpected comparison output %s", string(outputBytes))
	}

	csvBytes, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("ReadFile csv returned error: %v", err)
	}
	if strings.HasPrefix(string(csvBytes), "\ufeff") == false {
		t.Fatalf("csv missing utf-8 bom %q", string(csvBytes))
	}
	if strings.Contains(string(csvBytes), "episode-001,C:\\Media\\Show\\S01E01.mkv,episode,1,1,improved") == false {
		t.Fatalf("unexpected comparison csv %s", string(csvBytes))
	}
}
