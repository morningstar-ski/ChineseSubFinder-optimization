package settings

import (
	"bytes"
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
  "timeline_fixer_settings": {"max_offset_time": 999, "min_offset": 3, "engine": "unknown"},
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
	if cfg.TimelineFixerSettings.MaxOffsetTime != 700 {
		t.Fatalf("timeline max_offset_time = %d", cfg.TimelineFixerSettings.MaxOffsetTime)
	}
	if cfg.TimelineFixerSettings.MinOffset != 0.2 {
		t.Fatalf("timeline min_offset = %v", cfg.TimelineFixerSettings.MinOffset)
	}
	if cfg.TimelineFixerSettings.Engine != TimelineFixerEngineFFSubSync {
		t.Fatalf("timeline engine = %q", cfg.TimelineFixerSettings.Engine)
	}
}

func TestSettingsReadAcceptsUTF8BOMConfig(t *testing.T) {
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, configName)
	raw := []byte{0xEF, 0xBB, 0xBF}
	raw = append(raw, []byte(`{
  "user_info": {"username":"bomuser","password":"123456"},
  "common_settings": {"movie_paths":["/media/movies"],"series_paths":["/media/series"]},
  "emby_settings": {"address_url":"http://127.0.0.1:8096/"}
}`)...)
	if err := os.WriteFile(configPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg := NewSettings(configDir)
	if err := cfg.read(); err != nil {
		t.Fatal(err)
	}
	if cfg.UserInfo.Username != "bomuser" {
		t.Fatalf("username = %q", cfg.UserInfo.Username)
	}
	if cfg.EmbySettings.AddressUrl != "http://127.0.0.1:8096" {
		t.Fatalf("emby address url = %q", cfg.EmbySettings.AddressUrl)
	}
}

func TestNewSuppliersSettingsDoesNotIncludeRemovedA4KProvider(t *testing.T) {
	suppliers := NewSuppliersSettings()

	got := map[string]*OneSupplierSettings{
		suppliers.Xunlei.Name:         suppliers.Xunlei,
		suppliers.Shooter.Name:        suppliers.Shooter,
		suppliers.Assrt.Name:          suppliers.Assrt,
		suppliers.SubDL.Name:          suppliers.SubDL,
		suppliers.OpenSubtitles.Name:  suppliers.OpenSubtitles,
		suppliers.TVSubtitles.Name:    suppliers.TVSubtitles,
		suppliers.MovieSubtitles.Name: suppliers.MovieSubtitles,
		suppliers.SubHD.Name:          suppliers.SubHD,
		suppliers.Zimuku.Name:         suppliers.Zimuku,
	}

	if _, ok := got["a4k"]; ok {
		t.Fatal("unexpected removed provider a4k in suppliers settings")
	}
}

func TestSettingsSaveNormalizesTimelineFixerSettings(t *testing.T) {
	configDir := t.TempDir()
	cfg := NewSettings(configDir)
	cfg.EmbySettings.AddressUrl = "http://127.0.0.1:8096/"
	cfg.TimelineFixerSettings.MaxOffsetTime = 999
	cfg.TimelineFixerSettings.MinOffset = -1
	cfg.TimelineFixerSettings.Engine = "unknown"

	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	reloaded := NewSettings(configDir)
	if err := reloaded.read(); err != nil {
		t.Fatal(err)
	}

	if reloaded.TimelineFixerSettings.MaxOffsetTime != 700 {
		t.Fatalf("timeline max_offset_time = %d", reloaded.TimelineFixerSettings.MaxOffsetTime)
	}
	if reloaded.TimelineFixerSettings.MinOffset != 0.2 {
		t.Fatalf("timeline min_offset = %v", reloaded.TimelineFixerSettings.MinOffset)
	}
	if reloaded.TimelineFixerSettings.Engine != TimelineFixerEngineFFSubSync {
		t.Fatalf("timeline engine = %q", reloaded.TimelineFixerSettings.Engine)
	}
}

func TestSettingsSaveAndReadPreservesChineseMediaPaths(t *testing.T) {
	configDir := t.TempDir()
	cfg := NewSettings(configDir)
	cfg.EmbySettings.AddressUrl = "http://127.0.0.1:8096/"
	cfg.CommonSettings.MoviePaths = []string{"/media/电影"}
	cfg.CommonSettings.SeriesPaths = []string{"/media/电视剧"}

	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}

	configPath := filepath.Join(configDir, configName)
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("??")) {
		t.Fatalf("config contains unexpected placeholder bytes: %s", string(raw))
	}
	if !bytes.Contains(raw, []byte(`"/media/电影"`)) {
		t.Fatalf("movie path missing from saved config: %s", string(raw))
	}
	if !bytes.Contains(raw, []byte(`"/media/电视剧"`)) {
		t.Fatalf("series path missing from saved config: %s", string(raw))
	}

	reloaded := NewSettings(configDir)
	if err := reloaded.read(); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(cfg.CommonSettings.MoviePaths, reloaded.CommonSettings.MoviePaths) {
		t.Fatalf("movie paths mismatch, want %#v got %#v", cfg.CommonSettings.MoviePaths, reloaded.CommonSettings.MoviePaths)
	}
	if !reflect.DeepEqual(cfg.CommonSettings.SeriesPaths, reloaded.CommonSettings.SeriesPaths) {
		t.Fatalf("series paths mismatch, want %#v got %#v", cfg.CommonSettings.SeriesPaths, reloaded.CommonSettings.SeriesPaths)
	}
}

func TestExperimentalFunctionEnsureDefaultsFillsLLMSubtitleFallback(t *testing.T) {
	cfg := NewSettings(t.TempDir())

	cfg.ExperimentalFunction = &ExperimentalFunction{}
	cfg.ensureDefaults()

	if cfg.ExperimentalFunction.LLMSubtitleFallback.Provider != defaultLLMSubtitleFallbackProvider {
		t.Fatalf("provider = %q", cfg.ExperimentalFunction.LLMSubtitleFallback.Provider)
	}
	if cfg.ExperimentalFunction.LLMSubtitleFallback.BaseURL != "" {
		t.Fatalf("base_url = %q", cfg.ExperimentalFunction.LLMSubtitleFallback.BaseURL)
	}
	if cfg.ExperimentalFunction.LLMSubtitleFallback.APIKey != "" {
		t.Fatalf("api_key = %q", cfg.ExperimentalFunction.LLMSubtitleFallback.APIKey)
	}
	if cfg.ExperimentalFunction.LLMSubtitleFallback.Model != defaultLLMSubtitleFallbackModel {
		t.Fatalf("model = %q", cfg.ExperimentalFunction.LLMSubtitleFallback.Model)
	}
	if cfg.ExperimentalFunction.LLMSubtitleFallback.SubflowRootDir != defaultLLMSubtitleFallbackSubflowRoot() {
		t.Fatalf("subflow_root_dir = %q", cfg.ExperimentalFunction.LLMSubtitleFallback.SubflowRootDir)
	}
	if cfg.ExperimentalFunction.LLMSubtitleFallback.LogDir != defaultLLMSubtitleFallbackLogDir() {
		t.Fatalf("log_dir = %q", cfg.ExperimentalFunction.LLMSubtitleFallback.LogDir)
	}
	if cfg.ExperimentalFunction.LLMSubtitleFallback.PythonExecutable != defaultLLMSubtitleFallbackPythonExecutable() {
		t.Fatalf("python_executable = %q", cfg.ExperimentalFunction.LLMSubtitleFallback.PythonExecutable)
	}
	if cfg.ExperimentalFunction.LLMSubtitleFallback.OnlyWhenNoChineseCandidate != true {
		t.Fatal("only_when_no_chinese_candidate should default to true")
	}
	if cfg.ExperimentalFunction.LLMSubtitleFallback.SourceLanguage != defaultLLMSubtitleFallbackSourceLang {
		t.Fatalf("source_language = %q", cfg.ExperimentalFunction.LLMSubtitleFallback.SourceLanguage)
	}
	if cfg.ExperimentalFunction.LLMSubtitleFallback.TargetLanguage != defaultLLMSubtitleFallbackTargetLang {
		t.Fatalf("target_language = %q", cfg.ExperimentalFunction.LLMSubtitleFallback.TargetLanguage)
	}
	if cfg.ExperimentalFunction.LLMSubtitleFallback.TranslateStyle != "" {
		t.Fatalf("translate_style = %q", cfg.ExperimentalFunction.LLMSubtitleFallback.TranslateStyle)
	}
	if cfg.ExperimentalFunction.LocalChromeSettings.Enabled != true {
		t.Fatal("local chrome should default to enabled")
	}
}

func TestExperimentalFunctionEnsureDefaultsMigratesLegacyLocalChromeToEnabled(t *testing.T) {
	cfg := NewSettings(t.TempDir())
	cfg.ExperimentalFunction = &ExperimentalFunction{
		LocalChromeSettings: LocalChromeSettings{},
	}

	cfg.ensureDefaults()

	if cfg.ExperimentalFunction.LocalChromeSettings.Enabled != true {
		t.Fatal("legacy local chrome settings should migrate to enabled")
	}
}

func TestExperimentalFunctionEnsureDefaultsPreservesExplicitLocalChromeDisable(t *testing.T) {
	cfg := NewSettings(t.TempDir())
	cfg.ExperimentalFunction = &ExperimentalFunction{
		LocalChromeSettings: LocalChromeSettings{
			Enabled:    false,
			Configured: true,
		},
	}

	cfg.ensureDefaults()

	if cfg.ExperimentalFunction.LocalChromeSettings.Enabled != false {
		t.Fatal("explicit local chrome disable should be preserved")
	}
}

func TestLLMSubtitleFallbackEnsureDefaultsMigratesLegacyWindowsSubflowRoot(t *testing.T) {
	cfg := NewLLMSubtitleFallbackSettings()
	cfg.SubflowRootDir = legacyLLMSubtitleFallbackSubflowRoot
	cfg.PythonExecutable = ""

	cfg.ensureDefaults()

	if cfg.SubflowRootDir == legacyLLMSubtitleFallbackSubflowRoot {
		t.Fatalf("legacy subflow root was not migrated: %q", cfg.SubflowRootDir)
	}
	if cfg.SubflowRootDir != defaultLLMSubtitleFallbackSubflowRoot() {
		t.Fatalf("subflow_root_dir = %q", cfg.SubflowRootDir)
	}
	if cfg.PythonExecutable != defaultLLMSubtitleFallbackPythonExecutable() {
		t.Fatalf("python_executable = %q", cfg.PythonExecutable)
	}
}

func TestLLMSubtitleFallbackValidateRequiresExplicitConfigWhenEnabled(t *testing.T) {
	cfg := NewLLMSubtitleFallbackSettings()
	cfg.Enable = true

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected validation error when explicit llm config is incomplete")
	}

	cfg.Provider = defaultLLMSubtitleFallbackProvider
	cfg.BaseURL = "https://api.test.local/v1"
	cfg.APIKey = "test-key"
	cfg.Model = defaultLLMSubtitleFallbackModel
	cfg.SourceLanguage = defaultLLMSubtitleFallbackSourceLang
	cfg.TargetLanguage = defaultLLMSubtitleFallbackTargetLang

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestLLMSubtitleFallbackEnsureDefaultsNormalizesBaseURL(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		input    string
		want     string
	}{
		{
			name:     "deepseek root gets scheme and v1",
			provider: "deepseek",
			input:    "api.deepseek.com",
			want:     "https://api.deepseek.com/v1",
		},
		{
			name:     "deepseek keeps v1",
			provider: "deepseek",
			input:    "https://api.deepseek.com/v1/",
			want:     "https://api.deepseek.com/v1",
		},
		{
			name:     "gemini root expands to openai endpoint",
			provider: "gemini",
			input:    "https://generativelanguage.googleapis.com",
			want:     "https://generativelanguage.googleapis.com/v1beta/openai",
		},
		{
			name:     "gemini v1 is rewritten to v1beta openai",
			provider: "gemini",
			input:    "https://generativelanguage.googleapis.com/v1",
			want:     "https://generativelanguage.googleapis.com/v1beta/openai",
		},
		{
			name:     "generic compatible endpoint keeps chat completions",
			provider: "openai",
			input:    "https://example.com/chat/completions",
			want:     "https://example.com/chat/completions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewLLMSubtitleFallbackSettings()
			cfg.Provider = tt.provider
			cfg.BaseURL = tt.input

			cfg.ensureDefaults()

			if cfg.BaseURL != tt.want {
				t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, tt.want)
			}
		})
	}
}
