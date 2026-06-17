package downloader

import (
	"errors"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/task_queue"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
)

func TestNormalizeSeriesTerminalErrorMapsNoSubtitleSignalsToNoSubFound(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{name: "nil", err: nil},
		{name: "queue no sub found", err: task_queue.ErrNoSubFound},
		{name: "all site download sub not found", err: common.AllSiteDownloadSubNotFound},
		{name: "no usable chinese subtitle", err: errNoUsableChineseSubtitle},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeSeriesTerminalError(tc.err)
			if errors.Is(got, task_queue.ErrNoSubFound) == false {
				t.Fatalf("normalizeSeriesTerminalError() = %v; want ErrNoSubFound", got)
			}
			if got.Error() != task_queue.ErrNoSubFound.Error() {
				t.Fatalf("normalizeSeriesTerminalError() error = %q; want %q", got.Error(), task_queue.ErrNoSubFound.Error())
			}
		})
	}
}

func TestNormalizeSeriesTerminalErrorKeepsRealErrors(t *testing.T) {
	realErr := errors.New("write subtitle failed")
	got := normalizeSeriesTerminalError(realErr)
	if got != realErr {
		t.Fatalf("normalizeSeriesTerminalError() = %v; want original error", got)
	}
}
