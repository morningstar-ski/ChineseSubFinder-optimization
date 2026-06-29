package manual_upload_sub_2_local

import (
	"strings"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/save_sub_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_formatter/normal"
)

func TestNewManualUploadSub2Local(t *testing.T) {

	log := log_helper.GetLogger4Tester()
	saveSubHelper := save_sub_helper.NewSaveSubHelper(log, normal.NewFormatter(log), nil)
	got := NewManualUploadSub2Local(log, saveSubHelper, nil)
	if got == nil {
		t.Fatal("NewManualUploadSub2Local() returned nil")
	}
	if got.subParserHub == nil {
		t.Fatal("NewManualUploadSub2Local() did not initialize subParserHub")
	}
}

func TestFixTimelineOnlyJobStoresFailureResult(t *testing.T) {
	log := log_helper.GetLogger4Tester()
	saveSubHelper := save_sub_helper.NewSaveSubHelper(log, normal.NewFormatter(log), nil)
	queue := NewManualUploadSub2Local(log, saveSubHelper, nil)

	job := &Job{
		VideoFPath: "video.mkv",
		SubFPath:   "subtitle.srt",
		Mode:       "fix_timeline_only",
	}

	queue.Add(job)

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		result := queue.JobResult(job)
		if result == "" {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		if !strings.Contains(result, "timeline fixer helper is nil") {
			t.Fatalf("unexpected job result: %s", result)
		}
		return
	}

	t.Fatal("timed out waiting for manual upload queue result")
}
