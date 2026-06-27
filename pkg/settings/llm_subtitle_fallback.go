package settings

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"net/url"
	"strings"
)

const (
	defaultLLMSubtitleFallbackProvider        = "gemini"
	defaultLLMSubtitleFallbackModel           = "gemini-2.5-flash"
	legacyLLMSubtitleFallbackSubflowRoot      = "E:\\codex_movie\\codex-subflow"
	defaultLLMSubtitleFallbackUnixLogDir      = "/tmp/csf-llm-subtitle-fallback"
	defaultLLMSubtitleFallbackWindowsLogDir   = "D:\\tmp\\csf-llm-subtitle-fallback"
	defaultLLMSubtitleFallbackContainerRoot   = "/opt/subflow"
	defaultLLMSubtitleFallbackContainerPython = "/opt/csf-ocr/bin/python3"
	defaultLLMSubtitleFallbackSourceLang      = "en"
	defaultLLMSubtitleFallbackTargetLang      = "zh"
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
		PythonExecutable:           defaultLLMSubtitleFallbackPythonExecutable(),
		SubflowRootDir:             defaultLLMSubtitleFallbackSubflowRoot(),
		TranslateStyle:             "",
		OnlyWhenNoChineseCandidate: true,
		KeepEnglishSourceCopy:      false,
		LogDir:                     defaultLLMSubtitleFallbackLogDir(),
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
	if s.PythonExecutable == "" {
		s.PythonExecutable = defaults.PythonExecutable
	}
	if s.SubflowRootDir == "" || sameNormalizedPath(s.SubflowRootDir, legacyLLMSubtitleFallbackSubflowRoot) {
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
	s.BaseURL = NormalizeLLMSubtitleFallbackBaseURL(s.Provider, s.BaseURL)
	if s.OnlyWhenNoChineseCandidate == false && s.KeepEnglishSourceCopy == false &&
		s.Provider == defaults.Provider && s.BaseURL == defaults.BaseURL &&
		s.APIKey == defaults.APIKey && s.Model == defaults.Model &&
		s.PythonExecutable == defaults.PythonExecutable && s.SubflowRootDir == defaults.SubflowRootDir &&
		s.TranslateStyle == "" && s.LogDir == defaults.LogDir &&
		s.SourceLanguage == defaults.SourceLanguage && s.TargetLanguage == defaults.TargetLanguage {
		s.OnlyWhenNoChineseCandidate = defaults.OnlyWhenNoChineseCandidate
	}
}

func (s *LLMSubtitleFallbackSettings) Validate() error {
	if s == nil || s.Enable == false {
		return nil
	}

	requiredFields := []struct {
		label string
		value string
	}{
		{label: "provider", value: s.Provider},
		{label: "base_url", value: s.BaseURL},
		{label: "api_key", value: s.APIKey},
		{label: "model", value: s.Model},
		{label: "source_language", value: s.SourceLanguage},
		{label: "target_language", value: s.TargetLanguage},
	}

	for _, field := range requiredFields {
		if strings.TrimSpace(field.value) == "" {
			return fmt.Errorf("llm subtitle fallback %s is required", field.label)
		}
	}

	return nil
}

func NormalizeLLMSubtitleFallbackBaseURL(provider, rawBaseURL string) string {
	rawBaseURL = strings.TrimSpace(rawBaseURL)
	if rawBaseURL == "" {
		return ""
	}

	if strings.Contains(rawBaseURL, "://") == false && strings.Contains(rawBaseURL, " ") == false {
		rawBaseURL = "https://" + rawBaseURL
	}

	parsed, err := url.Parse(rawBaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return rawBaseURL
	}

	path := strings.TrimRight(parsed.Path, "/")
	lowerProvider := strings.ToLower(strings.TrimSpace(provider))
	switch lowerProvider {
	case "gemini":
		switch {
		case strings.HasSuffix(path, "/v1beta/openai/chat/completions"):
			path = strings.TrimSuffix(path, "/chat/completions")
		case strings.HasSuffix(path, "/v1beta/openai"):
			// keep as-is
		case strings.HasSuffix(path, "/v1/responses"):
			path = strings.TrimSuffix(path, "/v1/responses") + "/v1beta/openai"
		case strings.HasSuffix(path, "/responses"):
			path = strings.TrimSuffix(path, "/responses") + "/v1beta/openai"
		case strings.HasSuffix(path, "/v1/openai"):
			path = strings.TrimSuffix(path, "/v1/openai") + "/v1beta/openai"
		case strings.HasSuffix(path, "/v1beta"):
			path += "/openai"
		case strings.HasSuffix(path, "/v1"):
			path = strings.TrimSuffix(path, "/v1") + "/v1beta/openai"
		case path == "" || path == "/":
			path = "/v1beta/openai"
		default:
			path = path + "/v1beta/openai"
		}
	default:
		switch {
		case strings.HasSuffix(path, "/v1/chat/completions"):
			// keep as-is
		case strings.HasSuffix(path, "/chat/completions"):
			// keep as-is
		case strings.HasSuffix(path, "/v1/responses"):
			path = strings.TrimSuffix(path, "/responses")
		case strings.HasSuffix(path, "/responses"):
			path = strings.TrimSuffix(path, "/responses")
		case strings.HasSuffix(path, "/v1"):
			// keep as-is
		default:
			path = strings.TrimSuffix(path, "/")
			if path == "" {
				path = "/v1"
			} else {
				path += "/v1"
			}
		}
	}

	parsed.Path = path
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func defaultLLMSubtitleFallbackPythonExecutable() string {
	for _, candidate := range []string{
		strings.TrimSpace(os.Getenv("CSF_LLM_SUBTITLE_FALLBACK_PYTHON")),
		strings.TrimSpace(os.Getenv("CSF_DDDDOCR_PYTHON")),
		defaultLLMSubtitleFallbackContainerPython,
	} {
		if candidate == "" {
			continue
		}
		if isExistingFile(candidate) {
			return filepath.Clean(filepath.FromSlash(candidate))
		}
	}
	return ""
}

func defaultLLMSubtitleFallbackSubflowRoot() string {
	candidates := make([]string, 0, 4)
	appendCandidate := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		candidates = append(candidates, filepath.Clean(filepath.FromSlash(path)))
	}

	appendCandidate(os.Getenv("CSF_LLM_SUBTITLE_FALLBACK_SUBFLOW_ROOT"))
	appendCandidate(defaultLLMSubtitleFallbackContainerRoot)

	if _, thisFile, _, ok := runtime.Caller(0); ok {
		appendCandidate(filepath.Join(filepath.Dir(thisFile), "..", "..", "third_party", "subflow"))
	}
	if cwd, err := os.Getwd(); err == nil {
		appendCandidate(filepath.Join(cwd, "third_party", "subflow"))
	}

	for _, candidate := range candidates {
		if isValidLLMSubtitleFallbackSubflowRoot(candidate) {
			return candidate
		}
	}
	return ""
}

func defaultLLMSubtitleFallbackLogDir() string {
	if runtime.GOOS == "windows" {
		return defaultLLMSubtitleFallbackWindowsLogDir
	}
	return defaultLLMSubtitleFallbackUnixLogDir
}

func isExistingFile(path string) bool {
	info, err := os.Stat(filepath.Clean(filepath.FromSlash(path)))
	return err == nil && info.IsDir() == false
}

func isValidLLMSubtitleFallbackSubflowRoot(root string) bool {
	info, err := os.Stat(root)
	if err != nil || info.IsDir() == false {
		return false
	}
	translateJobPath := filepath.Join(root, "src", "subflow", "translate_job.py")
	info, err = os.Stat(translateJobPath)
	return err == nil && info.IsDir() == false
}

func sameNormalizedPath(left string, right string) bool {
	if strings.TrimSpace(left) == "" || strings.TrimSpace(right) == "" {
		return false
	}
	return strings.EqualFold(
		filepath.Clean(filepath.FromSlash(left)),
		filepath.Clean(filepath.FromSlash(right)),
	)
}
