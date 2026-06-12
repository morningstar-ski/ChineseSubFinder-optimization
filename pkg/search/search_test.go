package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/unit_test_helper"
)

func TestSearchSeriesAllEpsAndSubtitles(t *testing.T) {

	seriesDir := unit_test_helper.SkipIfTestDataResourceAbsent(t, []string{"series", "Pantheon"}, 4, false)
	seasonInfo, err := SeriesAllEpsAndSubtitles(log_helper.GetLogger4Tester(), filepath.Clean(seriesDir))
	if err != nil {
		t.Fatal(err)
	}
	println(seasonInfo.Name)
}

func TestSeriesAllEpsAndSubtitlesIncludesSmallSubtitleFile(t *testing.T) {
	dir := t.TempDir()
	settings.SetConfigRootPath(t.TempDir())
	seriesDir := filepath.Join(dir, "Audit Show")
	seasonDir := filepath.Join(seriesDir, "Season 1")
	if err := os.MkdirAll(seasonDir, 0o755); err != nil {
		t.Fatal(err)
	}

	videoPath := filepath.Join(seasonDir, "Audit Show - S01E01.mp4")
	subPath := filepath.Join(seasonDir, "Audit Show - S01E01.chinese(简,manual).default.srt")
	if err := os.WriteFile(videoPath, []byte(strings.Repeat("v", 2048)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(subPath, []byte("1\n00:00:01,000 --> 00:00:03,000\n连续剧测试字幕上传\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	seasonInfo, err := SeriesAllEpsAndSubtitles(log_helper.GetLogger4Tester(), filepath.Clean(seriesDir))
	if err != nil {
		t.Fatal(err)
	}
	if len(seasonInfo.OneVideoInfos) != 1 {
		t.Fatalf("expected one video info, got %#v", seasonInfo.OneVideoInfos)
	}
	if len(seasonInfo.OneVideoInfos[0].SubFPathList) != 1 || seasonInfo.OneVideoInfos[0].SubFPathList[0] != subPath {
		t.Fatalf("unexpected subtitle list: %#v", seasonInfo.OneVideoInfos[0].SubFPathList)
	}
}
