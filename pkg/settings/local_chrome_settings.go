package settings

type LocalChromeSettings struct {
	Enabled             bool   `json:"enabled"`
	Configured          bool   `json:"configured"`
	LocalChromeExeFPath string `json:"local_chrome_exe_f_path"`
}

func NewLocalChromeSettings() LocalChromeSettings {
	return LocalChromeSettings{
		Enabled: true,
	}
}

func (s *LocalChromeSettings) ensureDefaults() {
	if s.Configured == false {
		s.Enabled = true
	}
}
