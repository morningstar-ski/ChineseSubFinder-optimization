package settings

import "testing"

func TestNewSubtitleSourcesDisablesMovieSubtitlesByDefault(t *testing.T) {
	sources := NewSubtitleSources()
	if sources.MoviesubtitlesSettings.Enabled {
		t.Fatal("moviesubtitles should be disabled by default")
	}
}
