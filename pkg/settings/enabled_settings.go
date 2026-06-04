package settings

type EnabledSettings struct {
	Enabled bool `json:"enabled"`
}

func NewEnabledSettings(enabled bool) *EnabledSettings {
	return &EnabledSettings{Enabled: enabled}
}
