package rod_helper

import (
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/sirupsen/logrus"
)

type BrowserOptions struct {
	Log         *logrus.Logger
	LoadAdblock bool
	Settings    *settings.Settings
	preLoadURL  string
}

func NewBrowserOptions(log *logrus.Logger, loadAdblock bool, settings *settings.Settings) *BrowserOptions {
	return &BrowserOptions{
		Log:         log,
		LoadAdblock: loadAdblock,
		Settings:    settings,
	}
}

func (o *BrowserOptions) SetPreLoadUrl(rawURL string) {
	o.preLoadURL = rawURL
}

func (o *BrowserOptions) PreLoadUrl() string {
	return o.preLoadURL
}
