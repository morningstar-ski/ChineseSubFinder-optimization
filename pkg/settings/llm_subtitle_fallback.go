package settings

const (
	defaultLLMSubtitleFallbackProvider    = "gemini"
	defaultLLMSubtitleFallbackModel       = "gemini-2.5-flash"
	defaultLLMSubtitleFallbackSubflowRoot = "E:\\codex_movie\\codex-subflow"
	defaultLLMSubtitleFallbackLogDir      = "/tmp/csf-llm-subtitle-fallback"
	defaultLLMSubtitleFallbackSourceLang  = "en"
	defaultLLMSubtitleFallbackTargetLang  = "zh"
)

type LLMSubtitleFallbackSettings struct {
	Enable                     bool   `json:"enable"`
	Provider                   string `json:"provider"`
	BaseURL                    string `json:"base_url"`
	APIKey                     string `json:"api_key"`
	Model                      string `json:"model"`
	PythonExecutable           string `json:"python_executable"`
	SubflowRootDir             string `json:"subflow_root_dir"`
	TranslateStyle             string `json:"translate_style"`
	OnlyWhenNoChineseCandidate bool   `json:"only_when_no_chinese_candidate"`
	KeepEnglishSourceCopy      bool   `json:"keep_english_source_copy"`
	LogDir                     string `json:"log_dir"`
	SourceLanguage             string `json:"source_language"`
	TargetLanguage             string `json:"target_language"`
}

func NewLLMSubtitleFallbackSettings() *LLMSubtitleFallbackSettings {
	return &LLMSubtitleFallbackSettings{
		Enable:                     false,
		Provider:                   defaultLLMSubtitleFallbackProvider,
		BaseURL:                    "",
		APIKey:                     "",
		Model:                      defaultLLMSubtitleFallbackModel,
		PythonExecutable:           "",
		SubflowRootDir:             defaultLLMSubtitleFallbackSubflowRoot,
		TranslateStyle:             "",
		OnlyWhenNoChineseCandidate: true,
		KeepEnglishSourceCopy:      false,
		LogDir:                     defaultLLMSubtitleFallbackLogDir,
		SourceLanguage:             defaultLLMSubtitleFallbackSourceLang,
		TargetLanguage:             defaultLLMSubtitleFallbackTargetLang,
	}
}

func (s *LLMSubtitleFallbackSettings) ensureDefaults() {
	defaults := NewLLMSubtitleFallbackSettings()
	if s.Provider == "" {
		s.Provider = defaults.Provider
	}
	if s.Model == "" {
		s.Model = defaults.Model
	}
	if s.SubflowRootDir == "" {
		s.SubflowRootDir = defaults.SubflowRootDir
	}
	if s.LogDir == "" {
		s.LogDir = defaults.LogDir
	}
	if s.SourceLanguage == "" {
		s.SourceLanguage = defaults.SourceLanguage
	}
	if s.TargetLanguage == "" {
		s.TargetLanguage = defaults.TargetLanguage
	}
	if s.OnlyWhenNoChineseCandidate == false && s.KeepEnglishSourceCopy == false &&
		s.Provider == defaults.Provider && s.BaseURL == defaults.BaseURL &&
		s.APIKey == defaults.APIKey && s.Model == defaults.Model &&
		s.PythonExecutable == "" && s.SubflowRootDir == defaults.SubflowRootDir &&
		s.TranslateStyle == "" && s.LogDir == defaults.LogDir &&
		s.SourceLanguage == defaults.SourceLanguage && s.TargetLanguage == defaults.TargetLanguage {
		s.OnlyWhenNoChineseCandidate = defaults.OnlyWhenNoChineseCandidate
	}
}
