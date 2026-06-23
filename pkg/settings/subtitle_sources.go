package settings

type SubtitleSources struct {
	AssrtSettings          AssrtSettings         `json:"assrt_settings"`
	SubDLSettings          ApiKeySettings        `json:"subdl_settings"`
	SubHDSettings          SubHDSettings         `json:"subhd_settings"`
	OpenSubtitlesSettings  OpenSubtitlesSettings `json:"opensubtitles_settings"`
	TVsubtitlesSettings    EnabledSettings       `json:"tvsubtitles_settings"`
	MoviesubtitlesSettings EnabledSettings       `json:"moviesubtitles_settings"`
	SubtitleCatSettings    *SubtitleCatSettings  `json:"subtitlecat_settings"`
}

func NewSubtitleSources() *SubtitleSources {
	return &SubtitleSources{
		SubHDSettings:          *NewSubHDSettings(false),
		SubDLSettings:          *NewApiKeySettings(false, ""),
		OpenSubtitlesSettings:  *NewOpenSubtitlesSettings(false, "", "", ""),
		TVsubtitlesSettings:    *NewEnabledSettings(false),
		MoviesubtitlesSettings: *NewEnabledSettings(false),
		SubtitleCatSettings:    NewSubtitleCatSettings(),
	}
}

func (s *SubtitleSources) ensureDefaults() {
	if s == nil {
		return
	}
	if s.SubtitleCatSettings == nil {
		s.SubtitleCatSettings = NewSubtitleCatSettings()
	} else {
		s.SubtitleCatSettings.ensureDefaults()
	}
}
