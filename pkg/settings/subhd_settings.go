package settings

type SubHDSettings struct {
	Enabled             bool   `json:"enabled"`
	CaptchaSolver       string `json:"captcha_solver"`
	MaxCaptchaAttempts  int    `json:"max_captcha_attempts"`
	MaxVerifyCandidates int    `json:"max_verify_candidates"`
}

func NewSubHDSettings(enabled bool) *SubHDSettings {
	return &SubHDSettings{
		Enabled:             enabled,
		CaptchaSolver:       "glyph_ocr",
		MaxCaptchaAttempts:  4,
		MaxVerifyCandidates: 5,
	}
}
