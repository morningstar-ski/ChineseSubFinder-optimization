package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/baseline"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/random_auth_key"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/sirupsen/logrus"
)

func TestRunWithEvaluator(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "manifest.json")
	resultsPath := filepath.Join(tempDir, "results.json")
	csvPath := filepath.Join(tempDir, "baseline.csv")

	manifestContent := `{
  "samples": [
    {
      "id": "movie-001",
      "video_path": "C:\\Media\\Movie.mkv",
      "kind": "movie"
    }
  ]
}`
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0o644); err != nil {
		t.Fatalf("WriteFile manifest returned error: %v", err)
	}

	evaluator := baseline.EvaluatorFunc(func(ctx context.Context, sample baseline.Sample) (baseline.Evaluation, error) {
		return baseline.Evaluation{
			Attempts: []baseline.ProviderAttempt{
				{Provider: "subdl", Hit: true, Downloaded: true, Note: "ok"},
			},
		}, nil
	})

	if err := runWithEvaluator(manifestPath, resultsPath, csvPath, evaluator); err != nil {
		t.Fatalf("runWithEvaluator returned error: %v", err)
	}

	resultsBytes, err := os.ReadFile(resultsPath)
	if err != nil {
		t.Fatalf("ReadFile results returned error: %v", err)
	}
	if strings.Contains(string(resultsBytes), "\"provider\": \"subdl\"") == false {
		t.Fatalf("unexpected results content %s", string(resultsBytes))
	}

	csvBytes, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("ReadFile csv returned error: %v", err)
	}
	if strings.HasPrefix(string(csvBytes), "\ufeff") == false {
		t.Fatalf("csv missing utf-8 bom %q", string(csvBytes))
	}
	if strings.Contains(string(csvBytes), "movie-001,C:\\Media\\Movie.mkv,movie") == false {
		t.Fatalf("unexpected csv content %s", string(csvBytes))
	}
}

func TestResolveConfigRoot(t *testing.T) {
	t.Run("explicit override", func(t *testing.T) {
		if got := resolveConfigRoot("C:\\tmp\\config"); got != "C:\\tmp\\config" {
			t.Fatalf("resolveConfigRoot override = %s", got)
		}
	})

	t.Run("default root", func(t *testing.T) {
		if got := resolveConfigRoot(""); got != pkg.ConfigRootDirFPath() {
			t.Fatalf("resolveConfigRoot default = %s", got)
		}
	})
}

func TestResolveCacheName(t *testing.T) {
	t.Run("explicit override", func(t *testing.T) {
		if got := resolveCacheName("C:\\tmp\\config", "manual-cache"); got != "manual-cache" {
			t.Fatalf("resolveCacheName override = %s", got)
		}
	})

	t.Run("stable hash from config root", func(t *testing.T) {
		left := resolveCacheName("C:\\tmp\\config", "")
		right := resolveCacheName("C:\\tmp\\config", "")
		if left != right {
			t.Fatalf("resolveCacheName should be stable, got %q and %q", left, right)
		}
		if strings.HasPrefix(left, "baseline_live_") == false {
			t.Fatalf("resolveCacheName prefix = %q", left)
		}
		if len(left) != len("baseline_live_")+16 {
			t.Fatalf("resolveCacheName length = %d, value = %q", len(left), left)
		}
	})
}

func TestEnterWorkRoot(t *testing.T) {
	originalWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}

	workRoot := filepath.Join(t.TempDir(), "config-root")
	restoreWorkingDir, err := enterWorkRoot(workRoot)
	if err != nil {
		t.Fatalf("enterWorkRoot returned error: %v", err)
	}
	t.Cleanup(restoreWorkingDir)

	gotWorkingDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd after enterWorkRoot returned error: %v", err)
	}
	if gotWorkingDir != workRoot {
		t.Fatalf("working directory = %q, want %q", gotWorkingDir, workRoot)
	}

	restoreWorkingDir()
	gotWorkingDir, err = os.Getwd()
	if err != nil {
		t.Fatalf("Getwd after restore returned error: %v", err)
	}
	if gotWorkingDir != originalWorkingDir {
		t.Fatalf("restored working directory = %q, want %q", gotWorkingDir, originalWorkingDir)
	}
}

func TestLoadAuthKeyFallsBackToDefaults(t *testing.T) {
	oldBaseKey := pkg.BaseKey()
	oldAESKey16 := pkg.AESKey16()
	oldAESIv16 := pkg.AESIv16()
	t.Cleanup(func() {
		pkg.SetBaseKey(oldBaseKey)
		pkg.SetAESKey16(oldAESKey16)
		pkg.SetAESIv16(oldAESIv16)
	})

	pkg.SetBaseKey("")
	pkg.SetAESKey16("")
	pkg.SetAESIv16("")

	authKey := loadAuthKey(logrus.New())
	if authKey.BaseKey != random_auth_key.BaseKey {
		t.Fatalf("BaseKey = %q", authKey.BaseKey)
	}
	if authKey.AESKey16 != random_auth_key.AESKey16 {
		t.Fatalf("AESKey16 = %q", authKey.AESKey16)
	}
	if authKey.AESIv16 != random_auth_key.AESIv16 {
		t.Fatalf("AESIv16 = %q", authKey.AESIv16)
	}
}

func TestBuildTmdbHelperFromSettings(t *testing.T) {
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())
	cfg := settings.Get()
	oldTmdbSettings := cfg.AdvancedSettings.TmdbApiSettings
	t.Cleanup(func() {
		cfg.AdvancedSettings.TmdbApiSettings = oldTmdbSettings
	})

	cfg.AdvancedSettings.TmdbApiSettings = settings.TmdbApiSettings{
		Enable:              true,
		ApiKey:              "test-api-key",
		UseAlternateBaseURL: true,
	}
	helper, err := buildTmdbHelperFromSettings(logrus.New())
	if err != nil {
		t.Fatalf("buildTmdbHelperFromSettings returned error: %v", err)
	}
	if helper == nil {
		t.Fatal("buildTmdbHelperFromSettings returned nil helper")
	}

	cfg.AdvancedSettings.TmdbApiSettings = settings.TmdbApiSettings{}
	helper, err = buildTmdbHelperFromSettings(logrus.New())
	if err != nil {
		t.Fatalf("buildTmdbHelperFromSettings disabled returned error: %v", err)
	}
	if helper != nil {
		t.Fatal("buildTmdbHelperFromSettings disabled should return nil")
	}
}
