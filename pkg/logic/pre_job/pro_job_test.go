package pre_job

import (
	"testing"

	"github.com/sirupsen/logrus"
)

func TestSkipMarksPreJobDone(t *testing.T) {
	preJob := NewPreJob(logrus.New())

	if err := preJob.Skip("SpeedDevMode"); err != nil {
		t.Fatal(err)
	}
	if preJob.IsDone() != true {
		t.Fatal("expected pre job done after skip")
	}
	if preJob.GetStageName() != "SpeedDevMode" {
		t.Fatalf("stage name = %q", preJob.GetStageName())
	}
}
