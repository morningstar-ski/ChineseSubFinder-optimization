package settings

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/strcut_json"
)

type Settings struct {
	SpeedDevMode          bool
	configFPath           string
	UserInfo              *UserInfo              `json:"user_info"`
	CommonSettings        *CommonSettings        `json:"common_settings"`
	SubtitleSources       *SubtitleSources       `json:"subtitle_sources"`
	AdvancedSettings      *AdvancedSettings      `json:"advanced_settings"`
	EmbySettings          *EmbySettings          `json:"emby_settings"`
	DeveloperSettings     *DeveloperSettings     `json:"developer_settings"`
	TimelineFixerSettings *TimelineFixerSettings `json:"timeline_fixer_settings"`
	ExperimentalFunction  *ExperimentalFunction  `json:"experimental_function"`
}

func Get(reloadSettings ...bool) *Settings {
	_settingsLocker.Lock()
	defer _settingsLocker.Unlock()

	if _settings == nil {
		_settingsOnce.Do(func() {
			if _configRootPath == "" {
				panic("please call SetConfigRootPath before Get")
			}

			_settings = NewSettings(_configRootPath)
			if isFile(_settings.configFPath) == false {
				err := os.MkdirAll(filepath.Dir(_settings.configFPath), os.ModePerm)
				if err != nil {
					panic("create config dir failed: " + err.Error())
				}
				err = _settings.Save()
				if err != nil {
					panic("Can't Save Config File:" + configName + " Error: " + err.Error())
				}
			} else {
				err := _settings.read()
				if err != nil {
					panic("Can't Read Config File:" + configName + " Error: " + err.Error())
				}
			}
		})

		if len(reloadSettings) >= 1 && reloadSettings[0] == true {
			err := _settings.read()
			if err != nil {
				panic("Can't Read Config File:" + configName + " Error: " + err.Error())
			}
		}
	}

	return _settings
}

func SetFullNewSettings(inSettings *Settings) error {
	_settingsLocker.Lock()
	defer _settingsLocker.Unlock()

	nowConfigFPath := _settings.configFPath
	_settings = inSettings
	_settings.configFPath = nowConfigFPath
	_settings.Check()

	return _settings.Save()
}

func SetConfigRootPath(configRootPath string) {
	_configRootPath = configRootPath
}

func NewSettings(configRootDirFPath string) *Settings {
	nowConfigFPath := filepath.Join(configRootDirFPath, configName)

	return &Settings{
		configFPath:           nowConfigFPath,
		UserInfo:              &UserInfo{},
		CommonSettings:        NewCommonSettings(),
		SubtitleSources:       NewSubtitleSources(),
		AdvancedSettings:      NewAdvancedSettings(),
		EmbySettings:          NewEmbySettings(),
		DeveloperSettings:     NewDeveloperSettings(),
		TimelineFixerSettings: NewTimelineFixerSettings(),
		ExperimentalFunction:  NewExperimentalFunction(),
	}
}

func (s *Settings) read() error {
	err := strcut_json.ToStruct(s.configFPath, s)
	if err != nil {
		return err
	}

	s.ensureDefaults()
	s.AdvancedSettings.SuppliersSettings.ReSetSearchUrl()
	s.TimelineFixerSettings.Check()

	newEmbyAddressURL := removeSuffixAddressSlash(s.EmbySettings.AddressUrl)
	_, err = url.Parse(newEmbyAddressURL)
	if err != nil {
		return err
	}
	s.EmbySettings.AddressUrl = newEmbyAddressURL

	return nil
}

func (s *Settings) Save() error {
	s.ensureDefaults()
	s.AdvancedSettings.SuppliersSettings.ReSetSearchUrl()
	s.TimelineFixerSettings.Check()

	newEmbyAddressURL := removeSuffixAddressSlash(s.EmbySettings.AddressUrl)
	_, err := url.Parse(newEmbyAddressURL)
	if err != nil {
		return err
	}
	s.EmbySettings.AddressUrl = newEmbyAddressURL

	return strcut_json.ToFile(s.configFPath, s)
}

func (s *Settings) GetNoPasswordSettings() *Settings {
	nowSettings := NewSettings(_configRootPath)
	err := nowSettings.read()
	if err != nil {
		panic("Can't Read Config File:" + configName + " Error: " + err.Error())
	}
	nowSettings.UserInfo.Password = noPassword4Show
	return nowSettings
}

func (s *Settings) Check() {
	s.ensureDefaults()
	s.AdvancedSettings.SuppliersSettings.ReSetSearchUrl()
	s.TimelineFixerSettings.Check()

	if s.AdvancedSettings.Topic < 0 || s.AdvancedSettings.Topic > 3 {
		s.AdvancedSettings.Topic = 1
	}
	if s.AdvancedSettings.DebugMode == true {
		s.CommonSettings.Threads = 1
	} else {
		if s.CommonSettings.Threads <= 0 || s.CommonSettings.Threads > 6 {
			s.CommonSettings.Threads = 6
		}
	}

	s.AdvancedSettings.TaskQueue.Check()
	s.AdvancedSettings.DownloadFileCache.Check()
}

func (s *Settings) ensureDefaults() {
	if s.UserInfo == nil {
		s.UserInfo = &UserInfo{}
	}
	if s.CommonSettings == nil {
		s.CommonSettings = NewCommonSettings()
	}
	if s.SubtitleSources == nil {
		s.SubtitleSources = NewSubtitleSources()
	}
	if s.AdvancedSettings == nil {
		s.AdvancedSettings = NewAdvancedSettings()
	} else {
		s.AdvancedSettings.ensureDefaults()
	}
	if s.EmbySettings == nil {
		s.EmbySettings = NewEmbySettings()
	}
	if s.DeveloperSettings == nil {
		s.DeveloperSettings = NewDeveloperSettings()
	}
	if s.TimelineFixerSettings == nil {
		s.TimelineFixerSettings = NewTimelineFixerSettings()
	}
	if s.ExperimentalFunction == nil {
		s.ExperimentalFunction = NewExperimentalFunction()
	}
}

func isDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func isFile(filePath string) bool {
	s, err := os.Stat(filePath)
	if err != nil {
		return false
	}
	return !s.IsDir()
}

func removeSuffixAddressSlash(orgAddressURLString string) string {
	outString := orgAddressURLString
	for {
		if strings.HasSuffix(outString, "/") == true {
			outString = outString[:len(outString)-1]
		} else {
			break
		}
	}

	return outString
}

var (
	_settings       *Settings
	_settingsLocker sync.Mutex
	_settingsOnce   sync.Once
	_configRootPath string
)

const (
	noPassword4Show = "******"
	configName      = "ChineseSubFinderSettings.json"
)
