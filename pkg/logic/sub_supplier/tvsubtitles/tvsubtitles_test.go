package tvsubtitles

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/go-resty/resty/v2"
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

func TestIsDirectDownloadResponseZipBody(t *testing.T) {
	body := append([]byte("PK\x03\x04"), []byte("fake zip bytes")...)
	if isDirectDownloadResponse(body, "application/octet-stream", "") == false {
		t.Fatal("expected zip payload to be treated as direct download response")
	}
}

func TestIsDirectDownloadResponseRejectsCookieWarning(t *testing.T) {
	body := []byte("Your request could not be served because you have browser cookies disabled.")
	if isDirectDownloadResponse(body, "text/html; charset=utf-8", "") == true {
		t.Fatal("cookie warning page should not be treated as direct download response")
	}
}

func TestFetchFinalDownloadTargetDirectPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="episode.zip"`)
		_, _ = w.Write(append([]byte("PK\x03\x04"), []byte("fake zip payload")...))
	}))
	defer server.Close()

	supplier := &Supplier{}
	target, err := supplier.fetchFinalDownloadTarget(resty.New(), server.URL)
	if err != nil {
		t.Fatalf("fetchFinalDownloadTarget() error = %v", err)
	}
	if target.URL != server.URL {
		t.Fatalf("target.URL = %q", target.URL)
	}
	if target.DownloadFileName != "episode.zip" {
		t.Fatalf("target.DownloadFileName = %q", target.DownloadFileName)
	}
	if len(target.DirectData) == 0 {
		t.Fatal("expected direct payload bytes")
	}
}

func TestFetchFinalDownloadTargetHTMLRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`
<script>
var s1= 'files/';
var s2= 'show.cn.zip';
document.location = s1+s2;
</script>`))
	}))
	defer server.Close()

	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())
	settings.Get().AdvancedSettings.SuppliersSettings.TVSubtitles.RootUrl = server.URL

	supplier := &Supplier{}
	target, err := supplier.fetchFinalDownloadTarget(resty.New(), server.URL)
	if err != nil {
		t.Fatalf("fetchFinalDownloadTarget() error = %v", err)
	}
	if target.URL != server.URL+"/files/show.cn.zip" {
		t.Fatalf("target.URL = %q", target.URL)
	}
	if len(target.DirectData) != 0 {
		t.Fatal("expected html redirect path, not direct bytes")
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
