package settings

type SubHDSettings struct {
	Enabled bool `json:"enabled"`
}

func NewSubHDSettings(enabled bool) *SubHDSettings {
	return &SubHDSettings{Enabled: enabled}
}
