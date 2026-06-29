package sub_timeline_fixer

import (
	"errors"
	"strings"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/ffmpeg_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/sirupsen/logrus"
)

func TestProcessReturnsErrorWhenDependenciesAreNotReady(t *testing.T) {
	originalFFmpegProbe := ffmpegVersionProbe
	originalFFSubSyncProbe := ffsubsyncVersionProbe
	t.Cleanup(func() {
		ffmpegVersionProbe = originalFFmpegProbe
		ffsubsyncVersionProbe = originalFFSubSyncProbe
	})

	ffmpegVersionProbe = func(_ *ffmpeg_helper.FFMPEGHelper) (string, error) {
		return "", errors.New("ffmpeg missing")
	}
	ffsubsyncVersionProbe = func() (string, error) {
		return "ffsubsync 0.5.0", nil
	}

	helper := NewSubTimelineFixerHelperEx(logrus.New(), *settings.NewTimelineFixerSettings())
	err := helper.Process("video.mkv", "subtitle.srt")
	if err == nil {
		t.Fatal("expected Process() to return error when dependencies are unavailable")
	}
	if !strings.Contains(err.Error(), "ffmpeg/ffprobe not ready") {
		t.Fatalf("unexpected error: %v", err)
	}
}
