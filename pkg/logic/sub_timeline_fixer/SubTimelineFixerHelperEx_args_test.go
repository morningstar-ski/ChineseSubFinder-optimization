package sub_timeline_fixer

import (
	"testing"
)

func TestFFSubSyncArgsDoNotUseSuppressThresholdFlag(t *testing.T) {
	logDir := t.TempDir()
	args := buildFFSubSyncArgs("video.mkv", "input.srt", "output.tmp.srt", logDir, 700)

	for _, arg := range args {
		if arg == "--suppress-output-if-offset-less-than" {
			t.Fatalf("unexpected suppress threshold flag in args: %v", args)
		}
	}
}
