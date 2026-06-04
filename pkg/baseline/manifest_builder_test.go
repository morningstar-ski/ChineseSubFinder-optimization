package baseline

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/sirupsen/logrus"
)

func TestBuildManifest(t *testing.T) {
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())

	tempDir := t.TempDir()
	movieRoot := filepath.Join(tempDir, "movies")
	seriesRoot := filepath.Join(tempDir, "series")
	if err := os.MkdirAll(movieRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll movieRoot returned error: %v", err)
	}
	if err := os.MkdirAll(seriesRoot, 0o755); err != nil {
		t.Fatalf("MkdirAll seriesRoot returned error: %v", err)
	}

	moviePath := filepath.Join(movieRoot, "Movie.2024.1080p.mkv")
	videoBytes := bytes.Repeat([]byte("v"), 1500)
	if err := os.WriteFile(moviePath, videoBytes, 0o644); err != nil {
		t.Fatalf("WriteFile movie returned error: %v", err)
	}

	showDir := filepath.Join(seriesRoot, "Show")
	if err := os.MkdirAll(showDir, 0o755); err != nil {
		t.Fatalf("MkdirAll showDir returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(showDir, "tvshow.nfo"), []byte("<tvshow></tvshow>"), 0o644); err != nil {
		t.Fatalf("WriteFile tvshow.nfo returned error: %v", err)
	}
	episodePath := filepath.Join(showDir, "Show.S01E02.1080p.mkv")
	if err := os.WriteFile(episodePath, videoBytes, 0o644); err != nil {
		t.Fatalf("WriteFile episode returned error: %v", err)
	}

	manifest, err := BuildManifest(logrus.New(), []string{movieRoot}, []string{seriesRoot}, 1, 1)
	if err != nil {
		t.Fatalf("BuildManifest returned error: %v", err)
	}
	if len(manifest.Samples) != 2 {
		t.Fatalf("expected 2 samples, got %d", len(manifest.Samples))
	}
	if manifest.Samples[0].Kind != SampleMovie || manifest.Samples[0].VideoPath != moviePath {
		t.Fatalf("unexpected movie sample %#v", manifest.Samples[0])
	}
	if manifest.Samples[1].Kind != SampleEpisode || manifest.Samples[1].Season != 1 || manifest.Samples[1].Episode != 2 {
		t.Fatalf("unexpected episode sample %#v", manifest.Samples[1])
	}
}
