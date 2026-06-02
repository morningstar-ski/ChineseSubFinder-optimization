package reload_policy

import (
	"reflect"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
)

func NeedRestartHTTPServer(oldSettings *settings.Settings, newSettings *settings.Settings) bool {
	if oldSettings == nil || newSettings == nil {
		return false
	}
	if oldSettings.AdvancedSettings.DebugMode != newSettings.AdvancedSettings.DebugMode {
		return true
	}
	if reflect.DeepEqual(oldSettings.CommonSettings.MoviePaths, newSettings.CommonSettings.MoviePaths) == false {
		return true
	}
	if reflect.DeepEqual(oldSettings.CommonSettings.SeriesPaths, newSettings.CommonSettings.SeriesPaths) == false {
		return true
	}
	return false
}
