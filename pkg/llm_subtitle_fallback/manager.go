package llm_subtitle_fallback

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/ass"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/srt"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_parser_hub"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/subparser"
	"github.com/sirupsen/logrus"
)

const (
	FallbackSourceSite = "llm-fallback"
	defaultPythonExe   = "python"
)

type translator interface {
	Translate(req TranslateRequest) error
}

type TranslateRequest struct {
	InputPath        string
	OutputPath       string
	Provider         string
	BaseURL          string
	APIKey           string
	Model            string
	SourceLanguage   string
	TargetLanguage   string
	Style            string
	PythonExecutable string
	SubflowRootDir   string
	TaskDir          string
}

type Manager struct {
	log        *logrus.Logger
	settings   *settings.LLMSubtitleFallbackSettings
	parserHub  *sub_parser_hub.SubParserHub
	translator translator
}

func NewManager(log *logrus.Logger, cfg *settings.LLMSubtitleFallbackSettings) *Manager {
	return NewManagerWithTranslator(log, cfg, subflowTranslator{})
}

func NewManagerWithTranslator(log *logrus.Logger, cfg *settings.LLMSubtitleFallbackSettings, translator translator) *Manager {
	if cfg == nil {
		cfg = settings.NewLLMSubtitleFallbackSettings()
	}
	cfgCopy := *cfg
	fillDefaultSettings(&cfgCopy)
	if translator == nil {
		translator = subflowTranslator{}
	}

	return &Manager{
		log:        log,
		settings:   &cfgCopy,
		parserHub:  sub_parser_hub.NewSubParserHub(log, ass.NewParser(log), srt.NewParser(log)),
		translator: translator,
	}
}

func (m *Manager) Enabled() bool {
	return m != nil && m.settings != nil && m.settings.Enable
}

func (m *Manager) BuildChineseSubtitleFromEnglish(videoPath string, englishCandidate *subparser.FileInfo) (*subparser.FileInfo, error) {
	if m == nil || m.Enabled() == false {
		return nil, fmt.Errorf("llm subtitle fallback disabled")
	}
	if englishCandidate == nil {
		return nil, fmt.Errorf("english subtitle candidate is nil")
	}

	taskDir, err := m.createTaskDir(videoPath)
	if err != nil {
		return nil, err
	}

	sourcePath := filepath.Join(taskDir, "source.en"+common.SubExtSRT)
	if err := writeCandidateAsSRT(*englishCandidate, sourcePath); err != nil {
		return nil, err
	}

	if m.settings.KeepEnglishSourceCopy {
		if err := m.keepOriginalEnglishCopy(taskDir, *englishCandidate); err != nil {
			m.log.Warningln("llm_subtitle_fallback.keepOriginalEnglishCopy", err)
		}
	}

	outputPath := filepath.Join(taskDir, "translated.zh"+common.SubExtSRT)
	err = m.translator.Translate(TranslateRequest{
		InputPath:        sourcePath,
		OutputPath:       outputPath,
		Provider:         m.settings.Provider,
		BaseURL:          m.settings.BaseURL,
		APIKey:           m.settings.APIKey,
		Model:            m.settings.Model,
		SourceLanguage:   m.settings.SourceLanguage,
		TargetLanguage:   m.settings.TargetLanguage,
		Style:            m.settings.TranslateStyle,
		PythonExecutable: m.settings.PythonExecutable,
		SubflowRootDir:   m.settings.SubflowRootDir,
		TaskDir:          taskDir,
	})
	if err != nil {
		return nil, err
	}

	found, fileInfo, err := m.parserHub.DetermineFileTypeFromFile(outputPath)
	if err != nil {
		return nil, fmt.Errorf("parse translated subtitle: %w", err)
	}
	if found == false || fileInfo == nil {
		return nil, fmt.Errorf("translated subtitle is not a supported subtitle file: %s", outputPath)
	}

	fileInfo.FromWhereSite = FallbackSourceSite
	if englishCandidate.Name != "" {
		fileInfo.Name = englishCandidate.Name + ".llm.zh" + common.SubExtSRT
	} else {
		fileInfo.Name = filepath.Base(outputPath)
	}
	fileInfo.FileFullPath = outputPath

	return fileInfo, nil
}

func (m *Manager) createTaskDir(videoPath string) (string, error) {
	root := filepath.FromSlash(m.settings.LogDir)
	if root == "" {
		root = filepath.FromSlash(settings.NewLLMSubtitleFallbackSettings().LogDir)
	}
	taskDir := filepath.Join(root, sanitizeTaskName(videoPath)+"-"+fmt.Sprintf("%d", time.Now().UnixNano()))
	if err := os.MkdirAll(taskDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("create llm subtitle fallback task dir: %w", err)
	}
	return taskDir, nil
}

func (m *Manager) keepOriginalEnglishCopy(taskDir string, englishCandidate subparser.FileInfo) error {
	if englishCandidate.FileFullPath == "" {
		return nil
	}
	data, err := os.ReadFile(englishCandidate.FileFullPath)
	if err != nil {
		return err
	}
	name := filepath.Base(englishCandidate.FileFullPath)
	if name == "" || name == "." {
		name = "source.original" + englishCandidate.Ext
	}
	return os.WriteFile(filepath.Join(taskDir, name), data, 0o644)
}

var taskNameCleaner = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func sanitizeTaskName(videoPath string) string {
	name := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
	name = taskNameCleaner.ReplaceAllString(name, "_")
	name = strings.Trim(name, "._-")
	if name == "" {
		return "task"
	}
	return name
}

func fillDefaultSettings(cfg *settings.LLMSubtitleFallbackSettings) {
	if cfg == nil {
		return
	}
	defaults := settings.NewLLMSubtitleFallbackSettings()
	if cfg.Provider == "" {
		cfg.Provider = defaults.Provider
	}
	if cfg.Model == "" {
		cfg.Model = defaults.Model
	}
	if cfg.SubflowRootDir == "" {
		cfg.SubflowRootDir = defaults.SubflowRootDir
	}
	if cfg.LogDir == "" {
		cfg.LogDir = defaults.LogDir
	}
	if cfg.SourceLanguage == "" {
		cfg.SourceLanguage = defaults.SourceLanguage
	}
	if cfg.TargetLanguage == "" {
		cfg.TargetLanguage = defaults.TargetLanguage
	}
}
