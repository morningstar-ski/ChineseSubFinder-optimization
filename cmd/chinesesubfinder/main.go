package main

import (
	"flag"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/backend"
	"github.com/ChineseSubFinder/ChineseSubFinder/internal/dao"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/cron_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/random_auth_key"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/sirupsen/logrus"
)

func newLog() *logrus.Logger {
	var level logrus.Level

	// Enable debug logging when the marker file exists in the config directory.
	if pkg.IsFile(filepath.Join(pkg.ConfigRootDirFPath(), log_helper.DebugFileName)) == true {
		level = logrus.DebugLevel
	} else {
		level = logrus.InfoLevel
	}

	logger := log_helper.NewLogHelper(
		log_helper.LogNameChineseSubFinder,
		pkg.ConfigRootDirFPath(),
		level, time.Duration(7*24)*time.Hour, time.Duration(24)*time.Hour,
		settings.Get().ExperimentalFunction.ExtendLog)
	logger.AddHook(log_helper.NewLoggerHub())

	return logger
}

func init() {
	// Parse flags before loading settings so path overrides are ready.
	flag.Parse()
	pkg.SetLinuxConfigPathInSelfPath(*setLinuxConfigPathInSelfPathFlag)

	// Initialize settings with the selected config root.
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())

	// Clean old logs and load the latest boot log once.
	log_helper.CleanAndLoadOnceLogs()
	loggerBase = newLog()
	AppVersion = resolveAppVersion()
	pkg.SetLiteMode(false)

	loggerBase.Infoln("ChineseSubFinder Version:", AppVersion)
	pkg.SetAppVersion(AppVersion)
	pkg.SetExtEnCode(ExtEnCode)
	if pkg.ReadCustomAuthFile(loggerBase) == false {
		pkg.SetBaseKey(BaseKey)
		pkg.SetAESKey16(AESKey16)
		pkg.SetAESIv16(AESIv16)
	}

	if pkg.OSCheck() == false {
		loggerBase.Panicln(`You should search runtime.GOOS in the project, Implement unimplemented function`)
	}

	// Record device and version info.
	dao.UpdateInfo(AppVersion, settings.Get())

	// Disable scan-on-startup.
	settings.Get().CommonSettings.RunScanAtStartUp = false
	err := settings.Get().Save()
	if err != nil {
		loggerBase.Panicln("settings.Get().Save() err:", err)
	}
}

func main() {
	// Create or remove the debug marker file based on current settings.
	if settings.Get().AdvancedSettings.DebugMode == true {
		err := log_helper.WriteDebugFile()
		if err != nil {
			loggerBase.Errorln("log_helper.WriteDebugFile " + err.Error())
		}
		loggerBase = newLog()
		loggerBase.Infoln("Reload Log Settings, level = Debug")
	} else {
		err := log_helper.DeleteDebugFile()
		if err != nil {
			loggerBase.Errorln("log_helper.DeleteDebugFile " + err.Error())
		}
		loggerBase = newLog()
		loggerBase.Infoln("Reload Log Settings, level = Info")
	}

	if pkg.LinuxConfigPathInSelfPath() != "" {
		loggerBase.Infoln("SetLinuxConfigPathInSelfPath:", pkg.LinuxConfigPathInSelfPath())

		if pkg.IsDir(pkg.LinuxConfigPathInSelfPath()) == false {
			// The configured config-root override must already exist.
			loggerBase.Panicln("LinuxConfigPathInSelfPath", pkg.LinuxConfigPathInSelfPath(), "is not dir")
		}
	}

	// Apply API token and dev-mode settings.
	if settings.Get().ExperimentalFunction.ApiKeySettings.Enabled == true {
		common.SetApiToken(settings.Get().ExperimentalFunction.ApiKeySettings.Key)
	} else {
		common.SetApiToken("")
	}

	// Always disable speed-dev mode at startup.
	settings.Get().SpeedDevMode = false
	err := settings.Get().Save()
	if err != nil {
		loggerBase.Panicln("settings.Get().Save() err:", err)
	}

	if settings.Get().SpeedDevMode == true {
		loggerBase.Infoln("Speed Dev Mode is On")
	} else {
		loggerBase.Infoln("Speed Dev Mode is Off")
	}

	// Start the HTTP server first so the UI can observe initialization progress.
	fileDownloader := file_downloader.NewFileDownloader(
		cache_center.NewCacheCenter("local_task_queue", loggerBase),
		random_auth_key.AuthKey{
			BaseKey:  pkg.BaseKey(),
			AESKey16: pkg.AESKey16(),
			AESIv16:  pkg.AESIv16(),
		},
	)

	// Scheduled task helper.
	cronHelper := cron_helper.NewCronHelper(fileDownloader)

	// Allow overriding the port from an external file.
	nowPort := pkg.ReadCustomPortFile(loggerBase)

	// Restart signal channel.
	restartSignal := make(chan interface{}, 1)
	defer close(restartSignal)

	bend := backend.NewBackEnd(loggerBase, cronHelper, nowPort, restartSignal)
	go bend.Restart()
	restartSignal <- 1

	// Block forever.
	select {}
}

// Use git tags or ldflags to describe the version passed to this variable.
func resolveAppVersion() string {
	version := strings.TrimSpace(AppVersion)
	if isPlaceholderVersion(version) == false {
		return version
	}

	info, ok := debug.ReadBuildInfo()
	if ok == false {
		return ""
	}

	revision := ""
	modified := false
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = strings.TrimSpace(setting.Value)
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	if revision == "" {
		return ""
	}
	if len(revision) > 8 {
		revision = revision[:8]
	}
	if modified {
		return "git-" + revision + "-dirty"
	}
	return "git-" + revision
}

func isPlaceholderVersion(version string) bool {
	switch strings.ToLower(strings.TrimSpace(version)) {
	case "", "unknown", "unknow", "dev", "development":
		return true
	default:
		return false
	}
}

var AppVersion = "unknown"

// go build -ldflags="-X main.AppVersion=aabb -X main.ExtEnCode=ccdd" .
var ExtEnCode = "abcdefg1234567890"

// For SPK builds that cannot write /config, use the program directory as config root.
var setLinuxConfigPathInSelfPathFlag = flag.String("setconfigselfpath", "", "for SPK builds that cannot write /config, use the current program directory as config root")

var (
	BaseKey  = "0123456789123456789" // Seed key used to derive runtime keys.
	AESKey16 = "1234567890123456"    // AES key.
	AESIv16  = "1234567890123456"    // AES IV.
)

var loggerBase *logrus.Logger
