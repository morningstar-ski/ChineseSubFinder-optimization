package reload_policy

import (
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
)

func TestNeedRestartHTTPServer(t *testing.T) {
	t.Run("nil settings", func(t *testing.T) {
		if NeedRestartHTTPServer(nil, nil) {
			t.Fatal("expected nil settings to not require restart")
		}
	})

	testCases := []struct {
		name   string
		mutate func(newSettings *settings.Settings)
		want   bool
	}{
		{
			name: "same settings",
			want: false,
		},
		{
			name: "debug mode changed",
			mutate: func(newSettings *settings.Settings) {
				newSettings.AdvancedSettings.DebugMode = true
			},
			want: true,
		},
		{
			name: "movie paths changed",
			mutate: func(newSettings *settings.Settings) {
				newSettings.CommonSettings.MoviePaths = []string{"movie-b"}
			},
			want: true,
		},
		{
			name: "series paths changed",
			mutate: func(newSettings *settings.Settings) {
				newSettings.CommonSettings.SeriesPaths = []string{"series-b"}
			},
			want: true,
		},
		{
			name: "subtitle source changed",
			mutate: func(newSettings *settings.Settings) {
				newSettings.SubtitleSources.SubDLSettings.Enabled = true
				newSettings.SubtitleSources.SubDLSettings.Key = "subdl-key"
				newSettings.SubtitleSources.SubtitleBestSettings.Enabled = true
				newSettings.SubtitleSources.SubtitleBestSettings.ApiKey = "subtitle-best-key"
			},
			want: false,
		},
		{
			name: "proxy changed",
			mutate: func(newSettings *settings.Settings) {
				newSettings.AdvancedSettings.ProxySettings.UseProxy = true
				newSettings.AdvancedSettings.ProxySettings.UseWhichProxyProtocol = "http"
				newSettings.AdvancedSettings.ProxySettings.InputProxyAddress = "127.0.0.1"
				newSettings.AdvancedSettings.ProxySettings.InputProxyPort = "1080"
			},
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			oldSettings := newSettingsForNeedRestartTest(t)
			newSettings := newSettingsForNeedRestartTest(t)
			if tc.mutate != nil {
				tc.mutate(newSettings)
			}

			got := NeedRestartHTTPServer(oldSettings, newSettings)
			if got != tc.want {
				t.Fatalf("NeedRestartHTTPServer() = %v, want %v", got, tc.want)
			}
		})
	}
}

func newSettingsForNeedRestartTest(t *testing.T) *settings.Settings {
	t.Helper()

	cfg := settings.NewSettings(t.TempDir())
	cfg.CommonSettings.MoviePaths = []string{"movie-a"}
	cfg.CommonSettings.SeriesPaths = []string{"series-a"}
	cfg.AdvancedSettings.DebugMode = false
	cfg.SubtitleSources.SubDLSettings.Enabled = false
	cfg.SubtitleSources.SubDLSettings.Key = ""
	cfg.SubtitleSources.SubtitleBestSettings.Enabled = false
	cfg.SubtitleSources.SubtitleBestSettings.ApiKey = ""

	return cfg
}
