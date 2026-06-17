package settings

type SubHDSettings struct {
	Enabled        bool   `json:"enabled"`
	OCRBackend     string `json:"ocr_backend"`
	ExternalOCRURL string `json:"external_ocr_url"`
}

func NewSubHDSettings(enabled bool) *SubHDSettings {
	return &SubHDSettings{
		Enabled:    enabled,
		OCRBackend: "ddddocr",
	}
}
