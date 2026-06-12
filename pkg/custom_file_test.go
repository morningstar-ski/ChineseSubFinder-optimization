package pkg

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestReadCustomAuthFileFromConfigRoot(t *testing.T) {
	tempDir := t.TempDir()
	customAuthPath := filepath.Join(tempDir, customAuth)
	customAuthContent := "base-key@@@@1234567890abcdef@@@@fedcba0987654321"
	if err := os.WriteFile(customAuthPath, []byte(customAuthContent), 0o644); err != nil {
		t.Fatalf("WriteFile CustomAuth returned error: %v", err)
	}

	oldConfigPath := LinuxConfigPathInSelfPath()
	oldBaseKey := BaseKey()
	oldAESKey16 := AESKey16()
	oldAESIv16 := AESIv16()
	t.Cleanup(func() {
		SetLinuxConfigPathInSelfPath(oldConfigPath)
		SetBaseKey(oldBaseKey)
		SetAESKey16(oldAESKey16)
		SetAESIv16(oldAESIv16)
	})

	SetLinuxConfigPathInSelfPath(tempDir)
	SetBaseKey("")
	SetAESKey16("")
	SetAESIv16("")

	if ok := ReadCustomAuthFile(logrus.New()); ok == false {
		t.Fatal("ReadCustomAuthFile returned false")
	}
	if got := BaseKey(); got != "base-key" {
		t.Fatalf("BaseKey = %q", got)
	}
	if got := AESKey16(); got != "1234567890abcdef" {
		t.Fatalf("AESKey16 = %q", got)
	}
	if got := AESIv16(); got != "fedcba0987654321" {
		t.Fatalf("AESIv16 = %q", got)
	}
}

func TestReadCustomPortFileFromConfigRoot(t *testing.T) {
	tempDir := t.TempDir()
	customPortPath := filepath.Join(tempDir, customPort)
	if err := os.WriteFile(customPortPath, []byte("19037"), 0o644); err != nil {
		t.Fatalf("WriteFile CustomPort returned error: %v", err)
	}

	oldConfigPath := LinuxConfigPathInSelfPath()
	t.Cleanup(func() {
		SetLinuxConfigPathInSelfPath(oldConfigPath)
	})

	SetLinuxConfigPathInSelfPath(tempDir)

	if got := ReadCustomPortFile(logrus.New()); got != 19037 {
		t.Fatalf("ReadCustomPortFile = %d", got)
	}
}

func TestReadCustomPortFileTrimsTrailingWhitespace(t *testing.T) {
	tempDir := t.TempDir()
	customPortPath := filepath.Join(tempDir, customPort)
	if err := os.WriteFile(customPortPath, []byte("19045\r\n"), 0o644); err != nil {
		t.Fatalf("WriteFile CustomPort returned error: %v", err)
	}

	oldConfigPath := LinuxConfigPathInSelfPath()
	t.Cleanup(func() {
		SetLinuxConfigPathInSelfPath(oldConfigPath)
	})

	SetLinuxConfigPathInSelfPath(tempDir)

	if got := ReadCustomPortFile(logrus.New()); got != 19045 {
		t.Fatalf("ReadCustomPortFile with trailing newline = %d", got)
	}
}

func TestReadCustomPortFilePrefersConfigRootOverWorkingDirectory(t *testing.T) {
	configDir := t.TempDir()
	cwdDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(configDir, customPort), []byte("19045"), 0o644); err != nil {
		t.Fatalf("WriteFile config CustomPort returned error: %v", err)
	}
	if err := os.WriteFile(filepath.Join(cwdDir, customPort), []byte("19055"), 0o644); err != nil {
		t.Fatalf("WriteFile cwd CustomPort returned error: %v", err)
	}

	oldConfigPath := LinuxConfigPathInSelfPath()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd returned error: %v", err)
	}
	t.Cleanup(func() {
		SetLinuxConfigPathInSelfPath(oldConfigPath)
		_ = os.Chdir(oldWd)
	})

	SetLinuxConfigPathInSelfPath(configDir)
	if err := os.Chdir(cwdDir); err != nil {
		t.Fatalf("Chdir returned error: %v", err)
	}

	if got := ReadCustomPortFile(logrus.New()); got != 19045 {
		t.Fatalf("ReadCustomPortFile with config root precedence = %d", got)
	}
}

func TestSetLinuxConfigPathInSelfPathResetsConfigRootCache(t *testing.T) {
	oldConfigPath := LinuxConfigPathInSelfPath()
	t.Cleanup(func() {
		SetLinuxConfigPathInSelfPath(oldConfigPath)
	})

	_ = ConfigRootDirFPath()

	tempDir := t.TempDir()
	SetLinuxConfigPathInSelfPath(tempDir)

	if got := ConfigRootDirFPath(); got != tempDir {
		t.Fatalf("ConfigRootDirFPath = %q", got)
	}
}
