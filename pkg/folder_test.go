package pkg

import (
	"errors"
	"testing"
)

func TestClearIdleSubFixCacheFolder(t *testing.T) {

	//err := ClearIdleSubFixCacheFolder("W:\\CSF-SubFixCache", 24*time.Hour)
	//if err != nil {
	//	t.Fatal(err)
	//}
}

func TestIsIgnorableRodCacheCleanupError(t *testing.T) {
	if isIgnorableRodCacheCleanupError(errors.New("unlinkat x\\CrashpadMetrics-active.pma: Access is denied.")) == false {
		t.Fatal("expected access denied cleanup error to be ignorable")
	}
	if isIgnorableRodCacheCleanupError(errors.New("remove x: The process cannot access the file because it is being used by another process.")) == false {
		t.Fatal("expected in-use cleanup error to be ignorable")
	}
	if isIgnorableRodCacheCleanupError(errors.New("disk io failure")) == true {
		t.Fatal("expected unrelated cleanup error to remain non-ignorable")
	}
}
