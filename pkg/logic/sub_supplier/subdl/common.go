package subdl

type SearchResponse struct {
	Status  bool          `json:"status"`
	Results []SubtitleHit `json:"results"`
}

type SubtitleHit struct {
	Name        string       `json:"name"`
	Lang        string       `json:"lang"`
	Author      string       `json:"author"`
	URL         string       `json:"url"`
	Season      int          `json:"season"`
	Episode     int          `json:"episode"`
	Hi          bool         `json:"hi"`
	Releases    []string     `json:"releases"`
	UnpackFiles []UnpackFile `json:"unpack_files"`
}

type UnpackFile struct {
	Name    string `json:"name"`
	Lang    string `json:"lang"`
	URL     string `json:"url"`
	Season  int    `json:"season"`
	Episode int    `json:"episode"`
	Hi      bool   `json:"hi"`
}

type subtitleCandidate struct {
	Name        string
	DownloadURL string
	Season      int
	Episode     int
	Hi          bool
	Releases    []string
}
