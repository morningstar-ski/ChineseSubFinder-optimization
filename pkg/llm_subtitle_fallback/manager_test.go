package llm_subtitle_fallback

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/subparser"
	"github.com/sirupsen/logrus"
)

type stubTranslator struct {
	output string
	err    error
}

func (s stubTranslator) Translate(req TranslateRequest) error {
	if s.err != nil {
		return s.err
	}
	return os.WriteFile(req.OutputPath, []byte(s.output), 0o644)
}

func TestWriteCandidateAsSRTRendersASSDialogues(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "source.en.srt")
	info := subparser.FileInfo{
		Name: "candidate.ass",
		Ext:  common.SubExtASS,
		Dialogues: []subparser.OneDialogue{
			{StartTime: "0:00:01.00", EndTime: "0:00:02.00", Lines: []string{"Hello\\NWorld"}},
			{StartTime: "0:00:03.00", EndTime: "0:00:04.00", Lines: []string{"Bye"}},
		},
	}

	if err := writeCandidateAsSRT(info, dest); err != nil {
		t.Fatalf("writeCandidateAsSRT() error = %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(data)
	if !strings.Contains(got, "0:00:01,000 --> 0:00:02,000") {
		t.Fatalf("rendered SRT missing normalized timecode: %q", got)
	}
	if !strings.Contains(got, "Hello\nWorld") {
		t.Fatalf("rendered SRT missing line breaks: %q", got)
	}
}

func TestBuildChineseSubtitleFromEnglishParsesTranslatedSRT(t *testing.T) {
	cfg := settings.NewLLMSubtitleFallbackSettings()
	cfg.Enable = true
	cfg.APIKey = "test-key"
	cfg.LogDir = t.TempDir()
	cfg.SubflowRootDir = t.TempDir()

	sourcePath := filepath.Join(t.TempDir(), "candidate.en.srt")
	sourceBody := "1\n00:00:01,000 --> 00:00:02,000\nHello\n\n"
	if err := os.WriteFile(sourcePath, []byte(sourceBody), 0o644); err != nil {
		t.Fatalf("WriteFile(source) error = %v", err)
	}

	manager := NewManagerWithTranslator(logrus.New(), cfg, stubTranslator{
		output: "1\n00:00:01,000 --> 00:00:02,000\n你好\n\n",
	})

	info, err := manager.BuildChineseSubtitleFromEnglish(filepath.Join("C:\\", "Media", "Movie.mkv"), &subparser.FileInfo{
		Name:         "candidate.en.srt",
		Ext:          common.SubExtSRT,
		FileFullPath: sourcePath,
		Data:         []byte(sourceBody),
	})
	if err != nil {
		t.Fatalf("BuildChineseSubtitleFromEnglish() error = %v", err)
	}
	if info == nil {
		t.Fatal("BuildChineseSubtitleFromEnglish() returned nil")
	}
	if info.FromWhereSite != FallbackSourceSite {
		t.Fatalf("FromWhereSite = %q", info.FromWhereSite)
	}
	if strings.Contains(info.Name, ".llm.zh.srt") == false {
		t.Fatalf("Name = %q", info.Name)
	}
	if strings.Contains(string(info.Data), "你好") == false {
		t.Fatalf("translated subtitle content missing: %q", string(info.Data))
	}
}

func TestManagerReadyRequiresAPIKey(t *testing.T) {
	cfg := settings.NewLLMSubtitleFallbackSettings()
	cfg.Enable = true

	manager := NewManagerWithTranslator(logrus.New(), cfg, stubTranslator{})
	if manager.Ready() {
		t.Fatal("expected manager to be not ready without api key")
	}

	cfg.APIKey = "test-key"
	manager = NewManagerWithTranslator(logrus.New(), cfg, stubTranslator{})
	if manager.Ready() == false {
		t.Fatal("expected manager to be ready with api key")
	}
}
