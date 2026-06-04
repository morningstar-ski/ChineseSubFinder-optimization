package settings

type OpenSubtitlesSettings struct {
	Enabled  bool   `json:"enabled"`
	ApiKey   string `json:"api_key"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func NewOpenSubtitlesSettings(enabled bool, apiKey, username, password string) *OpenSubtitlesSettings {
	return &OpenSubtitlesSettings{
		Enabled:  enabled,
		ApiKey:   apiKey,
		Username: username,
		Password: password,
	}
}
