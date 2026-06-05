package subdl

import "encoding/json"

type SearchResponse struct {
	Status        bool            `json:"status"`
	Results       json.RawMessage `json:"results"`
	Subtitles     []SubtitleHit   `json:"subtitles"`
	TotalPages    int             `json:"totalPages"`
	CurrentPage   int             `json:"currentPage"`
	legacyResults []SubtitleHit
}

type SubtitleHit struct {
	Name         string       `json:"name"`
	Lang         string       `json:"lang"`
	Language     string       `json:"language"`
	Author       string       `json:"author"`
	URL          string       `json:"url"`
	SubtitlePage string       `json:"subtitlePage"`
	Season       int          `json:"season"`
	Episode      int          `json:"episode"`
	Hi           bool         `json:"hi"`
	ReleaseName  string       `json:"release_name"`
	Releases     []string     `json:"releases"`
	UnpackFiles  []UnpackFile `json:"unpack_files"`
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

func (s *SearchResponse) SubtitleHits() []SubtitleHit {
	if len(s.Subtitles) > 0 {
		return s.Subtitles
	}
	return s.legacyResults
}

func (s *SearchResponse) populateLegacyResults() {
	if len(s.Results) == 0 || string(s.Results) == "null" {
		return
	}
	var hits []SubtitleHit
	if err := json.Unmarshal(s.Results, &hits); err == nil {
		s.legacyResults = hits
	}
}

func (h SubtitleHit) ReleaseNames() []string {
	if len(h.Releases) > 0 {
		return append([]string(nil), h.Releases...)
	}
	if h.ReleaseName != "" {
		return []string{h.ReleaseName}
	}
	return nil
}
