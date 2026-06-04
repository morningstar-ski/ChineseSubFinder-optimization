package unit_test_helper

import (
	"os"
	"testing"
)

func SkipIfTestDataResourceAbsent(t *testing.T, resourceFolderNames []string, goBackTimes int, userCopyData bool) string {
	t.Helper()

	rootPath := GetTestDataResourceRootPath(resourceFolderNames, goBackTimes, userCopyData)
	if _, err := os.Stat(rootPath); err != nil {
		t.Skipf("external test data not found: %s (%v)", rootPath, err)
	}

	return rootPath
}
