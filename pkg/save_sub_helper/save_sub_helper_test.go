package save_sub_helper

import (
	"os"
	"path/filepath"
	"testing"

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
