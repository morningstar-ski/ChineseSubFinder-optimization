package settings

type SubtitleCatSettings struct {
	// Enabled is kept only for backward compatibility with older saved configs.
	// SubtitleCat stays enabled by default in the English fallback chain.
	Enabled                         bool `json:"enabled"`
	EnableTranslatedChineseFallback bool `json:"enable_translated_chinese_fallback"`
}

func NewSubtitleCatSettings() *SubtitleCatSettings {
	return &SubtitleCatSettings{
		Enabled:                         true,
		EnableTranslatedChineseFallback: false,
	}
}

func (s *SubtitleCatSettings) ensureDefaults() {
	if s == nil {
		return
	}
	s.Enabled = true
}
