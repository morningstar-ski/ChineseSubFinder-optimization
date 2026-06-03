package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tempDir := t.TempDir()
	movieRoot := filepath.Join(tempDir, "movies")
	seriesRoot := filepath.Join(tempDir, "series")
	outputPath := filepath.Join(tempDir, "manifest.json")

	if err := os.MkdirAll(movieRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll movieRoot returned error: %v", err)
	}
	if err := os.MkdirAll(seriesRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll seriesRoot returned error: %v", err)
	}

	videoBytes := bytes.Repeat([]byte("v"), 1500)
	if err := os.WriteFile(filepath.Join(movieRoot, "Movie.2024.1080p.mkv"), videoBytes, 0o644); err != nil {
		t.Fatalf("WriteFile movie returned error: %v", err)
	}

	showDir := filepath.Join(seriesRoot, "Show")
	if err := os.MkdirAll(showDir, 0o755); err != nil {
		t.Fatalf("MkdirAll showDir returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(showDir, "tvshow.nfo"), []byte("<tvshow></tvshow>"), 0o644); err != nil {
		t.Fatalf("WriteFile tvshow.nfo returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(showDir, "Show.S01E01.1080p.mkv"), videoBytes, 0o644); err != nil {
		t.Fatalf("WriteFile episode returned error: %v", err)
	}

	if err := run(movieRoot, seriesRoot, 1, 1, outputPath); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.Contains(string(content), "\"id\": \"movie-001\"") == false {
		t.Fatalf("unexpected manifest content %s", string(content))
	}
	if strings.Contains(string(content), "\"id\": \"episode-001\"") == false {
		t.Fatalf("unexpected manifest content %s", string(content))
	}
}
