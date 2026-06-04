package settings

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/strcut_json"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
)

func TestNewSettings(t *testing.T) {
	fileName := filepath.Join(t.TempDir(), "testfile.json")

	inSettings := Settings{
		UserInfo: &UserInfo{
			Username: "abcd",
			Password: "123456",
		},
		CommonSettings: &CommonSettings{
			ScanInterval:     "12h",
			Threads:          12,
			RunScanAtStartUp: true,
			MoviePaths:       []string{"aaa", "bbb"},
			SeriesPaths:      []string{"ccc", "ddd"},
		},
		AdvancedSettings: &AdvancedSettings{
			ProxySettings: &ProxySettings{
				UseProxy:                 true,
				LocalHttpProxyServerPort: "123",
			},
			DebugMode:                  true,
			SaveFullSeasonTmpSubtitles: true,
			SubTypePriority:            1,
			SubNameFormatter:           1,
			SaveMultiSub:               true,
			CustomVideoExts:            []string{"aaa", "bbb"},
			FixTimeLine:                true,
		},
		EmbySettings: &EmbySettings{
			Enable:                true,
			AddressUrl:            "123456",
			APIKey:                "api123",
			MaxRequestVideoNumber: 1000,
			SkipWatched:           true,
			MoviePathsMapping:     map[string]string{"aa": "123", "bb": "456"},
			SeriesPathsMapping:    map[string]string{"aab": "123", "bbc": "456"},
		},
		DeveloperSettings: &DeveloperSettings{
			BarkServerAddress: "bark",
		},
	}

	err := strcut_json.ToFile(fileName, inSettings)
	if err != nil {
		t.Fatal(err)
	}

	outSettings := NewSettings(t.TempDir())
	err = strcut_json.ToStruct(fileName, &outSettings)
	if err != nil {
		t.Fatal(err)
	}

	if reflect.DeepEqual(inSettings.UserInfo, outSettings.UserInfo) == false {
		t.Fatal("inSettings Write And Read Not The Same")
	}
}

func TestSuppliersSettingsEnsureDefaultsFillsNewProviders(t *testing.T) {
	cfg := &SuppliersSettings{
		SubDL: NewOneSupplierSettings(common.SubSiteSubDL, "https://example.com", "/old-search", -1),
	}

	cfg.ensureDefaults()
	cfg.ReSetSearchUrl()

	if cfg.OpenSubtitles == nil || cfg.TVSubtitles == nil || cfg.MovieSubtitles == nil {
		t.Fatalf("expected new provider defaults to be filled, got %#v", cfg)
	}
	if cfg.OpenSubtitles.SearchUrl != common.SubOpenSubtitlesSearchUrl {
		t.Fatalf("opensubtitles search url = %q", cfg.OpenSubtitles.SearchUrl)
	}
	if cfg.TVSubtitles.SearchUrl != common.SubTVSubtitlesSearchUrl {
		t.Fatalf("tvsubtitles search url = %q", cfg.TVSubtitles.SearchUrl)
	}
	if cfg.MovieSubtitles.SearchUrl != common.SubMovieSubtitlesSearchUrl {
		t.Fatalf("moviesubtitles search url = %q", cfg.MovieSubtitles.SearchUrl)
	}
}

func TestSettingsReadResetsNewSupplierSearchURLs(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, configName)
	raw := `{
  "subtitle_sources": {},
  "advanced_settings": {
    "suppliers_settings": {
      "opensubtitles": {"name":"opensubtitles","root_url":"https://api.opensubtitles.com/api/v1","search_url":"/stale-open","daily_download_limit":-1},
      "tvsubtitles": {"name":"tvsubtitles","root_url":"https://www.tvsubtitles.net","search_url":"/stale-tv","daily_download_limit":-1},
      "moviesubtitles": {"name":"moviesubtitles","root_url":"https://www.moviesubtitles.org","search_url":"/stale-movie","daily_download_limit":-1}
    }
  },
  "emby_settings": {"address_url":"http://127.0.0.1:8096/"}
}`
	if err := os.WriteFile(configPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := NewSettings(configDir)
	if err := cfg.read(); err != nil {
		t.Fatal(err)
	}

	if cfg.AdvancedSettings.SuppliersSettings.OpenSubtitles.SearchUrl != common.SubOpenSubtitlesSearchUrl {
		t.Fatalf("opensubtitles search url = %q", cfg.AdvancedSettings.SuppliersSettings.OpenSubtitles.SearchUrl)
	}
	if cfg.AdvancedSettings.SuppliersSettings.TVSubtitles.SearchUrl != common.SubTVSubtitlesSearchUrl {
		t.Fatalf("tvsubtitles search url = %q", cfg.AdvancedSettings.SuppliersSettings.TVSubtitles.SearchUrl)
	}
	if cfg.AdvancedSettings.SuppliersSettings.MovieSubtitles.SearchUrl != common.SubMovieSubtitlesSearchUrl {
		t.Fatalf("moviesubtitles search url = %q", cfg.AdvancedSettings.SuppliersSettings.MovieSubtitles.SearchUrl)
	}
	if cfg.EmbySettings.AddressUrl != "http://127.0.0.1:8096" {
		t.Fatalf("emby address url = %q", cfg.EmbySettings.AddressUrl)
	}
}
