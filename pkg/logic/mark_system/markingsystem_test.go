package mark_system

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
)

const testASSContent = "[Script Info]\n" +
	"Title: Test\n\n" +
	"[Events]\n" +
	"Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n" +
	"Dialogue: 0,0:00:01.00,0:00:02.00,Default,,0,0,0,,\u4f60\u597d\\NHello\n" +
	"Dialogue: 0,0:00:03.00,0:00:04.00,Default,,0,0,0,,\u518d\u89c1\\NBye\n"

func TestSelectOneSubFileSupportsSubhdOutsidePreferredSequence(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	subFilePath := filepath.Join(tmpDir, "[subhd]_0_test.ass")
	if err := os.WriteFile(subFilePath, []byte(testASSContent), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	mk := NewMarkingSystem(log_helper.GetLogger4Tester(), []string{
		common.SubSiteAssrt,
		common.SubSiteSubDL,
		common.SubSiteShooter,
		common.SubSiteXunLei,
	}, 0)
	got := mk.SelectOneSubFile([]string{subFilePath})
	if got == nil {
		t.Fatal("SelectOneSubFile() returned nil for subhd-only candidate")
	}
	if got.FromWhereSite != common.SubSiteSubHd {
		t.Fatalf("SelectOneSubFile() FromWhereSite = %q; want %q", got.FromWhereSite, common.SubSiteSubHd)
	}
	if got.FileFullPath != subFilePath {
		t.Fatalf("SelectOneSubFile() FileFullPath = %q; want %q", got.FileFullPath, subFilePath)
	}
}

func TestSelectOneSubFilePrefersSubhdInDefaultSequence(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	shooterPath := filepath.Join(tmpDir, "[shooter]_0_test.ass")
	subhdPath := filepath.Join(tmpDir, "[subhd]_0_test.ass")

	for _, path := range []string{shooterPath, subhdPath} {
		if err := os.WriteFile(path, []byte(testASSContent), 0o600); err != nil {
			t.Fatalf("WriteFile(%q) error = %v", path, err)
		}
	}

	mk := NewMarkingSystem(log_helper.GetLogger4Tester(), common.DefaultSubSiteSequence(), 0)
	got := mk.SelectOneSubFile([]string{shooterPath, subhdPath})
	if got == nil {
		t.Fatal("SelectOneSubFile() returned nil for mixed shooter/subhd candidates")
	}
	if got.FromWhereSite != common.SubSiteSubHd {
		t.Fatalf("SelectOneSubFile() FromWhereSite = %q; want %q", got.FromWhereSite, common.SubSiteSubHd)
	}
	if got.FileFullPath != subhdPath {
		t.Fatalf("SelectOneSubFile() FileFullPath = %q; want %q", got.FileFullPath, subhdPath)
	}
}
