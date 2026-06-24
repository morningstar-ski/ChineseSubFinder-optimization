package save_sub_helper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/charset"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	formatterEmby "github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_formatter/emby"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/subparser"
	"github.com/sirupsen/logrus"
)

func TestWriteSubFile2VideoPathSkipsTimelineFixWhenHelperMissing(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.FixTimeLine = true
	cfg.ExperimentalFunction.AutoChangeSubEncode.Enable = false
	cfg.ExperimentalFunction.ChsChtChanger.Enable = false

	videoDir := t.TempDir()
	videoPath := filepath.Join(videoDir, "Episode.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o600); err != nil {
		t.Fatalf("WriteFile(video) error = %v", err)
	}

	helper := NewSaveSubHelper(logrus.New(), formatterEmby.NewFormatter(), nil)
	subInfo := subparser.FileInfo{
		Name: "Episode.zh.srt",
		Ext:  ".srt",
		Lang: language.ChineseSimple,
		Data: []byte("1\n00:00:01,000 --> 00:00:02,000\n你好\n"),
	}

	if err := helper.WriteSubFile2VideoPath(videoPath, subInfo, "", true, false); err != nil {
		t.Fatalf("WriteSubFile2VideoPath() error = %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(videoDir, "*.srt"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("saved subtitle count = %d, want 1", len(matches))
	}

	got, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatalf("ReadFile(saved) error = %v", err)
	}
	if string(got) != string(subInfo.Data) {
		t.Fatalf("saved subtitle mismatch: %q", string(got))
	}
}

func TestWriteSubFile2VideoPathAppliesEncodeAndChineseVariantChain(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.FixTimeLine = false

	helper := NewSaveSubHelper(logrus.New(), formatterEmby.NewFormatter(), nil)

	t.Run("convert to gbk", func(t *testing.T) {
		cfg.ExperimentalFunction.AutoChangeSubEncode.Enable = true
		cfg.ExperimentalFunction.AutoChangeSubEncode.DesEncodeType = 1
		cfg.ExperimentalFunction.ChsChtChanger.Enable = false

		videoDir := t.TempDir()
		videoPath := filepath.Join(videoDir, "Movie.mkv")
		if err := os.WriteFile(videoPath, []byte("video"), 0o600); err != nil {
			t.Fatalf("WriteFile(video) error = %v", err)
		}

		original := "1\n00:00:01,000 --> 00:00:02,000\n开发后要回家\n"
		subInfo := subparser.FileInfo{
			Name: "Movie.zh.srt",
			Ext:  ".srt",
			Lang: language.ChineseSimple,
			Data: []byte(original),
		}

		if err := helper.WriteSubFile2VideoPath(videoPath, subInfo, "", true, false); err != nil {
			t.Fatalf("WriteSubFile2VideoPath() error = %v", err)
		}

		savedPath := mustFindSavedSubtitle(t, videoDir)
		savedBytes, err := os.ReadFile(savedPath)
		if err != nil {
			t.Fatalf("ReadFile(saved) error = %v", err)
		}
		if strings.Contains(string(savedBytes), "开发后要回家") {
			t.Fatal("expected saved subtitle bytes to be encoded as GBK, got readable UTF-8 text")
		}

		decoded, err := charset.ToUTF8("GBK", string(savedBytes))
		if err != nil {
			t.Fatalf("ToUTF8(GBK) error = %v", err)
		}
		if decoded != original {
			t.Fatalf("decoded subtitle mismatch:\nwant: %q\ngot:  %q", original, decoded)
		}
	})

	t.Run("convert to traditional chinese after utf8 normalization", func(t *testing.T) {
		cfg.ExperimentalFunction.AutoChangeSubEncode.Enable = true
		cfg.ExperimentalFunction.AutoChangeSubEncode.DesEncodeType = 0
		cfg.ExperimentalFunction.ChsChtChanger.Enable = true
		cfg.ExperimentalFunction.ChsChtChanger.DesChineseLanguageType = 1

		videoDir := t.TempDir()
		videoPath := filepath.Join(videoDir, "Episode.mkv")
		if err := os.WriteFile(videoPath, []byte("video"), 0o600); err != nil {
			t.Fatalf("WriteFile(video) error = %v", err)
		}

		subInfo := subparser.FileInfo{
			Name: "Episode.zh.srt",
			Ext:  ".srt",
			Lang: language.ChineseSimple,
			Data: []byte("1\n00:00:01,000 --> 00:00:02,000\n开发后要回家\n"),
		}

		if err := helper.WriteSubFile2VideoPath(videoPath, subInfo, "", true, false); err != nil {
			t.Fatalf("WriteSubFile2VideoPath() error = %v", err)
		}

		savedPath := mustFindSavedSubtitle(t, videoDir)
		savedBytes, err := os.ReadFile(savedPath)
		if err != nil {
			t.Fatalf("ReadFile(saved) error = %v", err)
		}
		got := string(savedBytes)
		if strings.Contains(got, "開發後要回家") == false {
			t.Fatalf("expected saved subtitle to be converted to traditional chinese, got %q", got)
		}
		if strings.Contains(got, "开发后要回家") {
			t.Fatalf("expected simplified chinese content to be replaced, got %q", got)
		}
	})
}

func mustFindSavedSubtitle(t *testing.T, videoDir string) string {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(videoDir, "*.srt"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("saved subtitle count = %d, want 1", len(matches))
	}
	return matches[0]
}
