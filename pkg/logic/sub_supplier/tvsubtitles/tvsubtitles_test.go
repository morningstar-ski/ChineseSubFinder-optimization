package tvsubtitles

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
)

func TestParseSearchResults(t *testing.T) {
	html := `
<p class="description">Search results</p>
<ul>
  <li><div><a href="/tvshow-2434.html">From (2022-2026)</a></div></li>
  <li><div><a href="/tvshow-79.html">John from Cincinnati (2007-2007)</a></div></li>
</ul>`

	results, err := parseSearchResults(html)
	if err != nil {
		t.Fatalf("parseSearchResults() error = %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %#v", results)
	}
	if results[0].ID != 2434 || results[0].Title != "From" {
		t.Fatalf("unexpected first result %#v", results[0])
	}
}

func TestParseSeasonPagePrefersExactEpisode(t *testing.T) {
	html := `
<table>
<tr><td>1x03</td><td><a href="episode-69977.html"><b>Choosing Day</b></a></td><td>4</td><td><nobr><a href="subtitle-319826.html"><img src="images/flags/en.gif" alt="en"></a><a href="subtitle-319827-cn.html"><img src="images/flags/cn.gif" alt="cn"></a></nobr></td></tr>
<tr><td></td><td><a href="episode-2434-1.html"><b>All episodes</b></a></td><td>5</td><td><nobr><a href="subtitle-2434-1-en.html"><img src="images/flags/en.gif" alt="en"></a><a href="subtitle-2434-1-cn.html"><img src="images/flags/cn.gif" alt="cn"></a></nobr></td></tr>
</table>`

	plan, err := parseSeasonPage(html)
	if err != nil {
		t.Fatalf("parseSeasonPage() error = %v", err)
	}
	if plan.EpisodeSubtitlePages[3] != "subtitle-319827-cn.html" {
		t.Fatalf("unexpected episode page %#v", plan.EpisodeSubtitlePages)
	}
	if plan.AllEpisodesPage != "subtitle-2434-1-cn.html" {
		t.Fatalf("unexpected all episodes page %q", plan.AllEpisodesPage)
	}
}

func TestParseSeasonPageFallsBackToAllEpisodes(t *testing.T) {
	html := `
<table>
<tr><td>1x03</td><td><a href="episode-69977.html"><b>Choosing Day</b></a></td><td>1</td><td><nobr><a href="subtitle-319826.html"><img src="images/flags/en.gif" alt="en"></a></nobr></td></tr>
<tr><td></td><td><a href="episode-2434-1.html"><b>All episodes</b></a></td><td>2</td><td><nobr><a href="subtitle-2434-1-cn.html"><img src="images/flags/cn.gif" alt="cn"></a></nobr></td></tr>
</table>`

	plan, err := parseSeasonPage(html)
	if err != nil {
		t.Fatalf("parseSeasonPage() error = %v", err)
	}
	if plan.EpisodeSubtitlePages[3] != "" {
		t.Fatalf("expected no exact chinese episode, got %#v", plan.EpisodeSubtitlePages)
	}
	if plan.AllEpisodesPage != "subtitle-2434-1-cn.html" {
		t.Fatalf("unexpected all episodes page %q", plan.AllEpisodesPage)
	}
}

func TestParseSubtitleDetailPage(t *testing.T) {
	html := `<div class="subtitle1"><a href="download-319833.html"><h3>Download</h3></a></div>`

	href, err := parseSubtitleDetailPage(html)
	if err != nil {
		t.Fatalf("parseSubtitleDetailPage() error = %v", err)
	}
	if href != "download-319833.html" {
		t.Fatalf("href = %q, want download page", href)
	}
}

func TestParseDownloadPage(t *testing.T) {
	html := `
<script>
var s1= 'fil';
var s2= 'es/F';
var s3= 'ro';
var s4= 'm_1x10_WEB.AMZN.cn.zip';
document.location = s1+s2+s3+s4;
</script>`

	path, err := parseDownloadPage(html)
	if err != nil {
		t.Fatalf("parseDownloadPage() error = %v", err)
	}
	if path != "files/From_1x10_WEB.AMZN.cn.zip" {
		t.Fatalf("path = %q", path)
	}
}

func TestMovieUnsupported(t *testing.T) {
	supplier := &Supplier{}
	subInfos, err := supplier.GetSubListFromFile4Movie("movie.mkv")
	if err != nil {
		t.Fatalf("GetSubListFromFile4Movie() error = %v", err)
	}
	if len(subInfos) != 0 {
		t.Fatalf("expected empty result, got %#v", subInfos)
	}
}

func TestSelectBestShowRejectsWrongSeries(t *testing.T) {
	results := []showSearchResult{
		{ID: 1, Title: "Spiderwick Chronicles"},
		{ID: 2, Title: "Dark Matter"},
	}
	mediaInfo := &models.MediaInfo{
		TitleEn:       "Lopez vs Lopez",
		OriginalTitle: "Lopez vs Lopez",
	}

	result := selectBestShow(results, mediaInfo, "Lopez vs Lopez")
	if result != nil {
		t.Fatalf("selectBestShow() = %#v; want nil", result)
	}
}

func TestSearchShowsRetriesEOFUntilSuccess(t *testing.T) {
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())
	cfg := settings.Get()
	oldRootURL := cfg.AdvancedSettings.SuppliersSettings.TVSubtitles.RootUrl
	oldSearchURL := cfg.AdvancedSettings.SuppliersSettings.TVSubtitles.SearchUrl
	t.Cleanup(func() {
		cfg.AdvancedSettings.SuppliersSettings.TVSubtitles.RootUrl = oldRootURL
		cfg.AdvancedSettings.SuppliersSettings.TVSubtitles.SearchUrl = oldSearchURL
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
		_, _ = w.Write([]byte(`<p class="description">Search results</p><ul><li><div><a href="/tvshow-2434.html">From (2022-2026)</a></div></li></ul>`))
	}))
	defer server.Close()

	cfg.AdvancedSettings.SuppliersSettings.TVSubtitles.RootUrl = server.URL
	cfg.AdvancedSettings.SuppliersSettings.TVSubtitles.SearchUrl = "/search1.php"

	client, err := pkg.NewHttpClient()
	if err != nil {
		t.Fatalf("NewHttpClient() error = %v", err)
	}
	client.SetRetryCount(0)

	supplier := &Supplier{}
	results, err := supplier.searchShows(client, "From")
	if err != nil {
		t.Fatalf("searchShows() error = %v", err)
	}
	if attempts != 3 {
		t.Fatalf("attempts = %d, want 3", attempts)
	}
	if len(results) != 1 || results[0].ID != 2434 {
		t.Fatalf("unexpected results %#v", results)
	}
	if client.RetryCount != 0 {
		t.Fatalf("client retry count should be restored, got %d", client.RetryCount)
	}
}
