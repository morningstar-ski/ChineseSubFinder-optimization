package settings

type SubtitleSources struct {
	AssrtSettings          AssrtSettings         `json:"assrt_settings"`
	SubDLSettings          ApiKeySettings        `json:"subdl_settings"`
	SubtitleBestSettings   SubtitleBestSettings  `json:"subtitle_best_settings"`
	OpenSubtitlesSettings  OpenSubtitlesSettings `json:"opensubtitles_settings"`
	TVsubtitlesSettings    EnabledSettings       `json:"tvsubtitles_settings"`
	MoviesubtitlesSettings EnabledSettings       `json:"moviesubtitles_settings"`
}

func NewSubtitleSources() *SubtitleSources {
	return &SubtitleSources{
		SubDLSettings:          *NewApiKeySettings(false, ""),
		OpenSubtitlesSettings:  *NewOpenSubtitlesSettings(false, "", "", ""),
		TVsubtitlesSettings:    *NewEnabledSettings(false),
		MoviesubtitlesSettings: *NewEnabledSettings(false),
	}
}
