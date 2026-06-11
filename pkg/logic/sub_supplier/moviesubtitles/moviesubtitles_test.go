package moviesubtitles

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/go-resty/resty/v2"
)

func TestParseSearchResults(t *testing.T) {
	html := `
<p class="description">Search results</p>
<ul style="margin-left:2em">
  <li><div style="width:500px"><a href="/movie-5012.html">Inception (2010)</a></div></li>
  <li><div style="width:500px"><a href="/movie-9999.html">Inferno (2016)</a></div></li>
</ul>`

	results, err := parseSearchResults(html)
	if err != nil {
		t.Fatalf("parseSearchResults() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %#v", results)
	}
	if results[0].ID != 5012 || results[0].Title != "Inception" || results[0].Year != "2010" {
		t.Fatalf("unexpected first result %#v", results[0])
	}
}

func TestSelectBestMovie(t *testing.T) {
	results := []movieSearchResult{
		{ID: 1, Title: "Inferno", Year: "2016", URL: "/movie-1.html"},
		{ID: 2, Title: "Inception", Year: "2010", URL: "/movie-2.html"},
	}
	mediaInfo := &models.MediaInfo{
		TitleEn: "Inception",
		Year:    "2010-07-16",
	}

	result := selectBestMovie(results, mediaInfo, "Inception")
	if result == nil {
		t.Fatal("selectBestMovie() returned nil")
	}
	if result.ID != 2 {
		t.Fatalf("expected movie 2, got %#v", result)
	}
}

func TestSelectBestMovieRejectsWrongTitle(t *testing.T) {
	results := []movieSearchResult{
		{ID: 1, Title: "Inferno", Year: "2016", URL: "/movie-1.html"},
		{ID: 2, Title: "Dark Matter", Year: "2024", URL: "/movie-2.html"},
	}
	mediaInfo := &models.MediaInfo{
		TitleEn:       "Lopez vs Lopez",
		OriginalTitle: "Lopez vs Lopez",
		Year:          "2024-01-01",
	}

	result := selectBestMovie(results, mediaInfo, "Lopez vs Lopez")
	if result != nil {
		t.Fatalf("selectBestMovie() = %#v; want nil", result)
	}
}

func TestBuildSearchKeywordsAddsPunctuationStrippedVariant(t *testing.T) {
	mediaInfo := &models.MediaInfo{
		TitleEn: "Will & Harper",
	}

	keywords := buildSearchKeywords(mediaInfo, filepath.Join("C:\\", "Media", "Will.&.Harper.2024.mkv"))
	if len(keywords) < 2 {
		t.Fatalf("expected keyword variants, got %#v", keywords)
	}
	if keywords[0] != "Will & Harper" {
		t.Fatalf("expected original keyword first, got %#v", keywords)
	}
	found := false
	for _, keyword := range keywords {
		if keyword == "Will Harper" {
			found = true
			break
		}
	}
	if found == false {
		t.Fatalf("expected punctuation-stripped keyword in %#v", keywords)
	}
}

func TestParseMoviePageChineseOnly(t *testing.T) {
	html := `
<table>
  <tr><th><div><span><b>English subtitles:</b></span></div></th></tr>
  <tr><td><a href="/subtitle-1.html"><div class="subtitle">
    <div><a href="/subtitle-1.html"><b>Movie english subtitles (WEB-DL)</b></a></div>
    <table><tr><td title="release">GROUP-EN</td><td title="rip">WEB-DL</td><td title="downloaded">100</td></tr></table>
  </div></a></td></tr>
  <tr><th><div><span><b>Chinese subtitles:</b></span></div></th></tr>
  <tr><td><a href="/subtitle-2.html"><div class="subtitle">
    <div><a href="/subtitle-2.html"><b>Movie chinese subtitles (BluRay-GROUP)</b></a></div>
    <table><tr><td title="release">BluRay-GROUP</td><td title="rip">BluRay</td><td title="downloaded">5600</td></tr></table>
  </div></a></td></tr>
  <tr><td><a href="/subtitle-3.html"><div class="subtitle">
    <div><a href="/subtitle-3.html"><b>Movie chinese subtitles (WEB-DL-ALT)</b></a></div>
    <table><tr><td title="release">WEB-DL-ALT</td><td title="rip">WEB-DL</td><td title="downloaded">200</td></tr></table>
  </div></a></td></tr>
</table>`

	candidates, err := parseMoviePage(html)
	if err != nil {
		t.Fatalf("parseMoviePage() error = %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("expected 2 chinese candidates, got %#v", candidates)
	}
	if candidates[0].SubtitlePageURL != "/subtitle-2.html" {
		t.Fatalf("unexpected first subtitle page %q", candidates[0].SubtitlePageURL)
	}
	if candidates[0].AuthorityScore == 0 {
		t.Fatalf("expected authority score for %#v", candidates[0])
	}
}

func TestRankCandidatesPrefersReleaseMatch(t *testing.T) {
	candidates := []subtitleCandidate{
		{
			Name:            "Inception 2010 chinese subtitles (WEB-DL-ALT)",
			ReleaseNames:    []string{"WEB-DL-ALT"},
			SubtitlePageURL: "/subtitle-3.html",
		},
		{
			Name:            "Inception 2010 chinese subtitles (BluRay-GROUP)",
			ReleaseNames:    []string{"BluRay-GROUP"},
			SubtitlePageURL: "/subtitle-2.html",
		},
	}

	rankCandidates(candidates, filepath.Join("C:\\", "Media", "Inception.2010.BluRay-GROUP.mkv"), 0)
	if candidates[0].SubtitlePageURL != "/subtitle-2.html" {
		t.Fatalf("expected bluray release first, got %#v", candidates)
	}
}

func TestParseSubtitleDetailPage(t *testing.T) {
	html := `<div class="download_container"><a href="download-59143.html" class="download_link"><h3>Download</h3></a></div>`

	href, err := parseSubtitleDetailPage(html)
	if err != nil {
		t.Fatalf("parseSubtitleDetailPage() error = %v", err)
	}
	if href != "download-59143.html" {
		t.Fatalf("href = %q, want download page", href)
	}
}

func TestFetchFinalDownloadURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/download-1.html":
			http.Redirect(w, r, "/files/movie.zh.zip", http.StatusFound)
		case "/files/movie.zh.zip":
			_, _ = w.Write([]byte("zip"))
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := newTestHTTPClient(t)
	supplier := &Supplier{}

	got, err := supplier.fetchFinalDownloadURL(client, server.URL+"/download-1.html")
	if err != nil {
		t.Fatalf("fetchFinalDownloadURL() error = %v", err)
	}
	if got != server.URL+"/files/movie.zh.zip" {
		t.Fatalf("fetchFinalDownloadURL() = %q", got)
	}
}

func TestSeriesUnsupported(t *testing.T) {
	supplier := &Supplier{}
	subInfos, err := supplier.GetSubListFromFile4Series(nil)
	if err != nil {
		t.Fatalf("GetSubListFromFile4Series() error = %v", err)
	}
	if len(subInfos) != 0 {
		t.Fatalf("expected empty result, got %#v", subInfos)
	}
}

func newTestHTTPClient(t *testing.T) *resty.Client {
	t.Helper()

	client, err := pkg.NewHttpClient()
	if err != nil {
		t.Fatalf("NewHttpClient() error = %v", err)
	}
	return client
}

func TestSearchMoviesRetriesEOFUntilSuccess(t *testing.T) {
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())
	cfg := settings.Get()
	oldRootURL := cfg.AdvancedSettings.SuppliersSettings.MovieSubtitles.RootUrl
	oldSearchURL := cfg.AdvancedSettings.SuppliersSettings.MovieSubtitles.SearchUrl
	t.Cleanup(func() {
		cfg.AdvancedSettings.SuppliersSettings.MovieSubtitles.RootUrl = oldRootURL
		cfg.AdvancedSettings.SuppliersSettings.MovieSubtitles.SearchUrl = oldSearchURL
	})

	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			hijacker, ok := w.(http.Hijacker)
			if ok == false {
				t.Fatalf("response writer does not support hijacking")
			}
			conn, _, err := hijacker.Hijack()
			if err != nil {
				t.Fatalf("Hijack() error = %v", err)
			}
			_ = conn.Close()
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<p class="description">Search results</p><ul style="margin-left:2em"><li><div style="width:500px"><a href="/movie-5012.html">Inception (2010)</a></div></li></ul>`))
	}))
	defer server.Close()

	cfg.AdvancedSettings.SuppliersSettings.MovieSubtitles.RootUrl = server.URL
	cfg.AdvancedSettings.SuppliersSettings.MovieSubtitles.SearchUrl = "/search.php"

	client := newTestHTTPClient(t)
	client.SetRetryCount(0)

	supplier := &Supplier{}
	results, err := supplier.searchMovies(client, "Inception")
	if err != nil {
		t.Fatalf("searchMovies() error = %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	if len(results) != 1 || results[0].ID != 5012 {
		t.Fatalf("unexpected results %#v", results)
	}
	if client.RetryCount != 0 {
		t.Fatalf("client retry count should be restored, got %d", client.RetryCount)
	}
}
