package sub_timeline_fixer

import (
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/subparser"
)

func TestInvalidTimelineFixedSubtitleReasonRejectsWrappedTimeline(t *testing.T) {
	fileInfo := &subparser.FileInfo{
		Dialogues: []subparser.OneDialogue{
			{StartTime: "23:59:10,560", EndTime: "23:59:16,010", Lines: []string{"line 1"}},
			{StartTime: "00:00:25,160", EndTime: "00:00:27,700", Lines: []string{"line 2"}},
		},
	}

	reason := invalidTimelineFixedSubtitleReason(fileInfo, 3395.296)
	if reason == "" {
		t.Fatal("expected wrapped timeline to be rejected")
	}
}

func TestInvalidTimelineFixedSubtitleReasonAllowsLongMovieEndTimeWithinTolerance(t *testing.T) {
	fileInfo := &subparser.FileInfo{
		Dialogues: []subparser.OneDialogue{
			{StartTime: "00:00:45,690", EndTime: "00:00:48,290", Lines: []string{"（英国，萨塞克斯）"}},
			{StartTime: "00:48:17,893", EndTime: "00:48:18,693", Lines: []string{"谁？"}},
			{StartTime: "01:45:32,296", EndTime: "01:45:34,537", Lines: []string{"生命中的时时刻刻"}},
		},
	}

	if end := pkg.Time2SecondNumber(fileInfo.GetEndTime()); end <= 3600 {
		t.Fatalf("fixture end time = %v; want > 3600", end)
	}
	if reason := invalidTimelineFixedSubtitleReason(fileInfo, 6886.881); reason != "" {
		t.Fatalf("invalidTimelineFixedSubtitleReason() = %q; want empty", reason)
	}
}
