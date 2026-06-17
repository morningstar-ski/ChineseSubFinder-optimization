package settings

import "testing"

func TestNewSubtitleSourcesDisablesMovieSubtitlesByDefault(t *testing.T) {
	sources := NewSubtitleSources()
	if sources.MoviesubtitlesSettings.Enabled {
		t.Fatal("moviesubtitles should be disabled by default")
	}
}

func TestNewSubtitleSourcesKeepsSubtitleCatEnglishFallbackEnabledByDefault(t *testing.T) {
	sources := NewSubtitleSources()
	if sources.SubtitleCatSettings == nil {
		t.Fatal("subtitlecat settings should not be nil")
	}
	if sources.SubtitleCatSettings.Enabled == false {
		t.Fatal("subtitlecat english fallback should be enabled by default")
	}
	if sources.SubtitleCatSettings.EnableTranslatedChineseFallback {
		t.Fatal("subtitlecat translated chinese fallback should be disabled by default")
	}
}

func TestSubtitleCatSettingsEnsureDefaultsForcesEnglishFallbackEnabled(t *testing.T) {
	cfg := &SubtitleCatSettings{
		Enabled:                         false,
		EnableTranslatedChineseFallback: true,
	}

	cfg.ensureDefaults()

	if cfg.Enabled == false {
		t.Fatal("subtitlecat english fallback should be forced on")
	}
	if cfg.EnableTranslatedChineseFallback == false {
		t.Fatal("translated chinese fallback switch should stay user-controlled")
	}
}
