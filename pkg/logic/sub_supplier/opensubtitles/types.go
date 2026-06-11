package opensubtitles

type LoginResponse struct {
	Token string `json:"token"`
}

type SearchResponse struct {
	Data []SearchItem `json:"data"`
}

type SearchItem struct {
	ID         string              `json:"id"`
	Attributes SearchItemAttribute `json:"attributes"`
}

type SearchItemAttribute struct {
	Language        string         `json:"language"`
	ISO639          string         `json:"iso639"`
	Release         string         `json:"release"`
	MovieName       string         `json:"movie_name"`
	SubFormat       string         `json:"sub_format"`
	HearingImpaired bool           `json:"hearing_impaired"`
	DownloadCount   int            `json:"download_count"`
	Files           []SearchFile   `json:"files"`
	FeatureDetails  FeatureDetails `json:"feature_details"`
}

type SearchFile struct {
	FileID   int64  `json:"file_id"`
	FileName string `json:"file_name"`
}

type FeatureDetails struct {
	Title         string `json:"title"`
	Year          int    `json:"year"`
	SeasonNumber  int    `json:"season_number"`
	EpisodeNumber int    `json:"episode_number"`
}

type DownloadResponse struct {
	Link      string `json:"link"`
	FileName  string `json:"file_name"`
	Message   string `json:"message"`
	Remaining int    `json:"remaining"`
	Requests  int    `json:"requests"`
}

type ErrorResponse struct {
	Message string      `json:"message"`
	Errors  []ErrorItem `json:"errors"`
}

type ErrorItem struct {
	Status interface{} `json:"status"`
	Title  string      `json:"title"`
	Detail string      `json:"detail"`
}

type subtitleCandidate struct {
	FileID       int64
	Name         string
	FileName     string
	ReleaseNames []string
	FeatureTitle string
	Year         int
	Season       int
	Episode      int
	Ext          string
	HasHI        bool
}
