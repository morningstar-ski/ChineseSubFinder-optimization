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

const testEnglishASSContent = "[Script Info]\n" +
	"Title: English\n\n" +
	"[Events]\n" +
	"Format: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\n" +
	"Dialogue: 0,0:00:01.00,0:00:02.00,Default,,0,0,0,,Hello there\n" +
	"Dialogue: 0,0:00:03.00,0:00:04.00,Default,,0,0,0,,General Kenobi\n"

const testEnglishSRTContent = "1\n00:00:01,000 --> 00:00:02,000\nHello there\n\n2\n00:00:03,000 --> 00:00:04,000\nGeneral Kenobi\n"

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

func TestSelectBestEnglishSubFilePrefersReleaseMatch(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	exactPath := filepath.Join(tmpDir, "[subdl]_0_My.Show.S01E03.1080p.WEB-DL-GROUP.en.srt")
	wrongEpisodePath := filepath.Join(tmpDir, "[subdl]_1_My.Show.S01E04.1080p.WEB-DL-GROUP.en.srt")

	if err := os.WriteFile(exactPath, []byte(testEnglishSRTContent), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", exactPath, err)
	}
	if err := os.WriteFile(wrongEpisodePath, []byte(testEnglishSRTContent), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", wrongEpisodePath, err)
	}

	mk := NewMarkingSystem(log_helper.GetLogger4Tester(), common.DefaultSubSiteSequence(), 0)
	got := mk.SelectBestEnglishSubFile([]string{wrongEpisodePath, exactPath}, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"))
	if got == nil {
		t.Fatal("SelectBestEnglishSubFile() returned nil")
	}
	if got.FileFullPath != exactPath {
		t.Fatalf("SelectBestEnglishSubFile() FileFullPath = %q; want %q", got.FileFullPath, exactPath)
	}
}

func TestSelectBestEnglishSubFilePrefersSRTOverASSWhenScoresClose(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	assPath := filepath.Join(tmpDir, "[subdl]_0_My.Show.S01E03.1080p.WEB-DL-GROUP.en.ass")
	srtPath := filepath.Join(tmpDir, "[subdl]_1_My.Show.S01E03.1080p.WEB-DL-GROUP.en.srt")

	if err := os.WriteFile(assPath, []byte(testEnglishASSContent), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", assPath, err)
	}
	if err := os.WriteFile(srtPath, []byte(testEnglishSRTContent), 0o600); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", srtPath, err)
	}

	mk := NewMarkingSystem(log_helper.GetLogger4Tester(), common.DefaultSubSiteSequence(), 0)
	got := mk.SelectBestEnglishSubFile([]string{assPath, srtPath}, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"))
	if got == nil {
		t.Fatal("SelectBestEnglishSubFile() returned nil")
	}
	if got.FileFullPath != srtPath {
		t.Fatalf("SelectBestEnglishSubFile() FileFullPath = %q; want %q", got.FileFullPath, srtPath)
	}
}
