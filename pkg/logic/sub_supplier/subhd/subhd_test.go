package subhd

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/imdb_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/sirupsen/logrus"
)

func TestOverDailyDownloadLimitTreatsNegativeLimitAsUnlimited(t *testing.T) {
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())

	oldLimit := settings.Get().AdvancedSettings.SuppliersSettings.SubHD.DailyDownloadLimit
	settings.Get().AdvancedSettings.SuppliersSettings.SubHD.DailyDownloadLimit = -1
	defer func() {
		settings.Get().AdvancedSettings.SuppliersSettings.SubHD.DailyDownloadLimit = oldLimit
	}()

	supplier := &Supplier{log: logrus.New()}
	if supplier.OverDailyDownloadLimit() {
		t.Fatal("expected negative daily download limit to mean unlimited")
	}
}

func TestNewPageHTTPClientUsesLightweightTimeoutAndNoRetry(t *testing.T) {
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())

	supplier := &Supplier{log: logrus.New()}

	client, err := supplier.newPageHTTPClient()
	if err != nil {
		t.Fatalf("newPageHTTPClient() error = %v", err)
	}

	if got := client.GetClient().Timeout; got != subHDPageRequestTimeout {
		t.Fatalf("newPageHTTPClient timeout = %v; want %v", got, subHDPageRequestTimeout)
	}
	if got := client.RetryCount; got != subHDPageRequestRetryCount {
		t.Fatalf("newPageHTTPClient retry count = %d; want %d", got, subHDPageRequestRetryCount)
	}
}

func TestShouldRetryPageFetchOnlyForTransientTransportAnd5xx(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "eof", err: fmt.Errorf("Get \"https://subhd.me/a/test\": EOF"), want: true},
		{name: "reset", err: fmt.Errorf("read tcp: connection reset by peer"), want: true},
		{name: "502", err: fmt.Errorf("unexpected http status 502 for https://subhd.me/a/test"), want: true},
		{name: "504", err: fmt.Errorf("unexpected http status 504 for https://subhd.me/a/test"), want: true},
		{name: "404", err: fmt.Errorf("unexpected http status 404 for https://subhd.me/a/test"), want: false},
		{name: "layout", err: fmt.Errorf("detail_layout_changed: html changed"), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRetryPageFetch(tc.err); got != tc.want {
				t.Fatalf("shouldRetryPageFetch() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestHTTPGetPageRetriesEOFUntilSuccess(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
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
		_, _ = w.Write([]byte("<html>ok</html>"))
	}))
	defer server.Close()

	supplier := &Supplier{log: logrus.New()}
	got, err := supplier.httpGetPage(server.URL)
	if err != nil {
		t.Fatalf("httpGetPage() error = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d; want 2", attempts)
	}
	if got != "<html>ok</html>" {
		t.Fatalf("httpGetPage() = %q", got)
	}
}

func TestDownloadSeriesSubHDItemWithRetryRetriesTransientGetExFailure(t *testing.T) {
	attempts := 0
	got, err := retrySeriesSubHDDownload(logrus.New(), "subhd", "https://subhd.me/a/RHptLe", func() (*supplier.SubInfo, error) {
		attempts++
		if attempts == 1 {
			return nil, fmt.Errorf("Get %q: EOF", "https://subhd.me/a/RHptLe")
		}
		return &supplier.SubInfo{Name: "The.Boys.S01E01.zip"}, nil
	})
	if err != nil {
		t.Fatalf("retrySeriesSubHDDownload() error = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d; want 2", attempts)
	}
	if got == nil || got.Name != "The.Boys.S01E01.zip" {
		t.Fatalf("unexpected sub info %#v", got)
	}
}

func TestShouldRetrySubHDDetailPageWithBrowserOnlyForDetailProbeFailures(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "detail layout", err: wrapReason(ReasonDetailLayoutChanged, fmt.Errorf("detail html changed")), want: true},
		{name: "probe", err: wrapReason(ReasonProbeFailed, fmt.Errorf("eof")), want: true},
		{name: "captcha", err: wrapReason(ReasonCaptchaOcrFailed, fmt.Errorf("bad captcha")), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRetrySubHDDetailPageWithBrowser(tc.err); got != tc.want {
				t.Fatalf("shouldRetrySubHDDetailPageWithBrowser() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestShouldRetrySubHDSearchPageWithBrowserOnlyForSearchProbeFailures(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "search layout", err: wrapReason(ReasonSearchLayoutChanged, fmt.Errorf("search html changed")), want: true},
		{name: "probe", err: wrapReason(ReasonProbeFailed, fmt.Errorf("eof")), want: true},
		{name: "detail layout", err: wrapReason(ReasonDetailLayoutChanged, fmt.Errorf("detail html changed")), want: false},
		{name: "captcha", err: wrapReason(ReasonCaptchaOcrFailed, fmt.Errorf("bad captcha")), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRetrySubHDSearchPageWithBrowser(tc.err); got != tc.want {
				t.Fatalf("shouldRetrySubHDSearchPageWithBrowser() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestShouldSkipDuplicateMovieFallback(t *testing.T) {
	cases := []struct {
		name     string
		base     string
		fallback string
		want     bool
	}{
		{name: "same keyword", base: "The Hours 2002", fallback: "The Hours 2002", want: true},
		{name: "trimmed same keyword", base: " The Hours 2002 ", fallback: "The Hours 2002", want: true},
		{name: "different keyword", base: "The Hours 2002", fallback: "The Hours tt0274558", want: false},
		{name: "empty base", base: "", fallback: "The Hours 2002", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldSkipDuplicateMovieFallback(tc.base, tc.fallback); got != tc.want {
				t.Fatalf("shouldSkipDuplicateMovieFallback() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestBuildSubHDMovieTitleCandidatesUsesPathFallbackForFakeBDMV(t *testing.T) {
	videoPath := filepath.Join("C:\\", "Media", "The Shameless (2024)", "00000.m2ts")
	imdbInfo := imdb_helper.FallbackVideoNfoInfoFromPath(videoPath, true)

	got := buildSubHDMovieTitleCandidates(videoPath, nil, imdbInfo)
	if len(got) == 0 {
		t.Fatal("buildSubHDMovieTitleCandidates() returned empty candidates")
	}
	if got[0] != "The Shameless" {
		t.Fatalf("buildSubHDMovieTitleCandidates()[0] = %q; want %q", got[0], "The Shameless")
	}
}

func TestParseSubtitleRowsParsesCurrentSingleSubtitleDetailLayout(t *testing.T) {
	const detailHTML = `
<div class="bg-white shadow-sm rounded-3 mt-3 mb-3">
  <div class="p-3">
    <div class="mb-2">片源版本</div>
    <div class="f16 fw-bold mb-2">耐撕侦探.The.Nice.Guys.2016.官方简繁英</div>
  </div>
  <div class="p-3 my-2 bg-light clearfix">
    <div class="float-start">
      <span class="rounded p-1 me-1 text-white" style="background-color:#00a3ff!important">官方字幕</span>
      <span class="p-1 fw-bold">简体</span><span class="p-1 fw-bold">繁体</span><span class="p-1 fw-bold">英语</span>
      <span class="p-1 text-secondary">SRT</span>
    </div>
    <div class="float-end">
      <span class="align-text-top me-3">123k</span>
      <span class="align-text-top me-3">1125</span>
      <span class="align-text-top">2025-1-9 10:31</span>
    </div>
  </div>
</div>
<div class="mb-3 clearfix">
  <div class="float-start">
    <a class="btn btn-danger down" sid="UKrdfL" href="/down/UKrdfL" target="_blank">下载字幕文件</a>
  </div>
</div>`

	got, err := parseSubtitleRows(detailHTML, "https://subhd.me", "https://subhd.me/a/UKrdfL", true, 1)
	if err != nil {
		t.Fatalf("parseSubtitleRows() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("parseSubtitleRows() len = %d; want 1", len(got))
	}
	if got[0].Title != "耐撕侦探.The.Nice.Guys.2016.官方简繁英" {
		t.Fatalf("title = %q", got[0].Title)
	}
	if got[0].Url != "https://subhd.me/a/UKrdfL" {
		t.Fatalf("url = %q", got[0].Url)
	}
	if got[0].DownCount != 1125 {
		t.Fatalf("down count = %d; want 1125", got[0].DownCount)
	}
}

func TestCheckAliveKeepsPreviousAliveStateOnTransientError(t *testing.T) {
	supplier := &Supplier{
		log:     logrus.New(),
		isAlive: true,
	}

	supplier.log.Out = bytes.NewBuffer(nil)

	oldRoot := settings.Get().AdvancedSettings.SuppliersSettings.SubHD.RootUrl
	settings.Get().AdvancedSettings.SuppliersSettings.SubHD.RootUrl = "https://127.0.0.1:1"
	defer func() {
		settings.Get().AdvancedSettings.SuppliersSettings.SubHD.RootUrl = oldRoot
	}()

	alive, _ := supplier.CheckAlive()
	if alive == false {
		t.Fatal("expected transient error to keep previous alive state")
	}
	if supplier.IsAlive() == false {
		t.Fatal("expected supplier to remain alive after transient error")
	}
}

func TestWhichEpisodeNeedDownloadSubFiltersWrongSeriesTitle(t *testing.T) {
	supplier := &Supplier{log: logrus.New()}
	seriesInfo := &series.SeriesInfo{
		Name: "Lopez vs Lopez",
		NeedDlEpsKeyList: map[string]series.EpisodeInfo{
			pkg.GetEpisodeKeyName(1, 2): {
				Season:       1,
				Episode:      2,
				FileFullPath: filepath.Join("C:\\", "Media", "Lopez.vs.Lopez.S01E02.1080p.WEB-DL.mkv"),
			},
		},
	}
	mediaInfo := &models.MediaInfo{
		TitleCn:       "Lopez vs Lopez CN",
		TitleEn:       "Lopez vs Lopez",
		OriginalTitle: "Lopez vs Lopez",
	}
	allSubList := []HdListItem{
		{Title: "Spiderwick Chronicles S01E02 1080p", Url: "/wrong", DownCount: 99},
		{Title: "Lopez vs Lopez S01E02 1080p", Url: "/right", DownCount: 1},
	}

	got := supplier.whichEpisodeNeedDownloadSub(seriesInfo, mediaInfo, allSubList)
	if len(got) != 1 {
		t.Fatalf("whichEpisodeNeedDownloadSub() len = %d; want 1", len(got))
	}
	if got[0].Url != "/right" {
		t.Fatalf("whichEpisodeNeedDownloadSub() picked %q; want %q", got[0].Url, "/right")
	}
	if got[0].Season != 1 || got[0].Episode != 2 {
		t.Fatalf("whichEpisodeNeedDownloadSub() season/episode = S%02dE%02d; want S01E02", got[0].Season, got[0].Episode)
	}
}

func TestWhichEpisodeNeedDownloadSubAllowsMatchingSeasonPackOnce(t *testing.T) {
	supplier := &Supplier{log: logrus.New()}
	seriesInfo := &series.SeriesInfo{
		Name: "Lopez vs Lopez",
		NeedDlEpsKeyList: map[string]series.EpisodeInfo{
			pkg.GetEpisodeKeyName(1, 3): {
				Season:       1,
				Episode:      3,
				FileFullPath: filepath.Join("C:\\", "Media", "Lopez.vs.Lopez.S01E03.1080p.WEB-DL.mkv"),
			},
			pkg.GetEpisodeKeyName(1, 4): {
				Season:       1,
				Episode:      4,
				FileFullPath: filepath.Join("C:\\", "Media", "Lopez.vs.Lopez.S01E04.1080p.WEB-DL.mkv"),
			},
		},
	}
	mediaInfo := &models.MediaInfo{
		TitleCn:       "Lopez vs Lopez CN",
		TitleEn:       "Lopez vs Lopez",
		OriginalTitle: "Lopez vs Lopez",
	}
	allSubList := []HdListItem{
		{Title: "Wrong Show S01 Complete", Url: "/wrong-pack", DownCount: 99},
		{Title: "Lopez vs Lopez S01 Complete", Url: "/right-pack", DownCount: 10},
	}

	got := supplier.whichEpisodeNeedDownloadSub(seriesInfo, mediaInfo, allSubList)
	if len(got) != 1 {
		t.Fatalf("whichEpisodeNeedDownloadSub() len = %d; want 1 season pack", len(got))
	}
	if got[0].Url != "/right-pack" {
		t.Fatalf("whichEpisodeNeedDownloadSub() picked %q; want %q", got[0].Url, "/right-pack")
	}
	if got[0].Season != 1 || got[0].Episode != 0 {
		t.Fatalf("whichEpisodeNeedDownloadSub() season/episode = S%02dE%02d; want S01 pack", got[0].Season, got[0].Episode)
	}
}

func TestWhichEpisodeNeedDownloadSubLopezVsLopezPrefersExactEpisodes(t *testing.T) {
	supplier := &Supplier{log: logrus.New()}
	seriesInfo := &series.SeriesInfo{
		Name: "Lopez vs Lopez",
		NeedDlEpsKeyList: map[string]series.EpisodeInfo{
			pkg.GetEpisodeKeyName(1, 1): {
				Season:       1,
				Episode:      1,
				FileFullPath: filepath.Join("C:\\", "Media", "Lopez.vs.Lopez.S01E01.1080p.WEB-DL-GROUP.mkv"),
			},
			pkg.GetEpisodeKeyName(1, 2): {
				Season:       1,
				Episode:      2,
				FileFullPath: filepath.Join("C:\\", "Media", "Lopez.vs.Lopez.S01E02.1080p.WEB-DL-GROUP.mkv"),
			},
			pkg.GetEpisodeKeyName(1, 3): {
				Season:       1,
				Episode:      3,
				FileFullPath: filepath.Join("C:\\", "Media", "Lopez.vs.Lopez.S01E03.1080p.WEB-DL-GROUP.mkv"),
			},
			pkg.GetEpisodeKeyName(1, 4): {
				Season:       1,
				Episode:      4,
				FileFullPath: filepath.Join("C:\\", "Media", "Lopez.vs.Lopez.S01E04.1080p.WEB-DL-GROUP.mkv"),
			},
		},
	}
	mediaInfo := &models.MediaInfo{
		TitleCn:       "Lopez vs Lopez CN",
		TitleEn:       "Lopez vs Lopez",
		OriginalTitle: "Lopez vs Lopez",
	}
	allSubList := []HdListItem{
		{Title: "Spiderwick Chronicles S01E01 1080p WEB-DL-GROUP", Url: "/wrong-e01", DownCount: 99},
		{Title: "Lopez vs Lopez S01E01 1080p WEB-DL-GROUP", Url: "/right-e01", DownCount: 5},
		{Title: "Wrong Show S01E02 1080p WEB-DL-GROUP", Url: "/wrong-e02", DownCount: 100},
		{Title: "Lopez vs Lopez S01E02 1080p WEB-DL-GROUP", Url: "/right-e02", DownCount: 4},
		{Title: "Lopez vs Lopez S01 Complete 1080p WEB-DL-GROUP", Url: "/season-pack", DownCount: 2},
		{Title: "Wrong Show S01 Complete 1080p WEB-DL-GROUP", Url: "/wrong-pack", DownCount: 200},
	}

	got := supplier.whichEpisodeNeedDownloadSub(seriesInfo, mediaInfo, allSubList)
	gotByURL := make(map[string]HdListItem, len(got))
	for _, item := range got {
		gotByURL[item.Url] = item
	}
	if _, ok := gotByURL["/wrong-e01"]; ok {
		t.Fatal("whichEpisodeNeedDownloadSub() should reject wrong show episode for E01")
	}
	if _, ok := gotByURL["/wrong-e02"]; ok {
		t.Fatal("whichEpisodeNeedDownloadSub() should reject wrong show episode for E02")
	}
	if _, ok := gotByURL["/wrong-pack"]; ok {
		t.Fatal("whichEpisodeNeedDownloadSub() should reject wrong show season pack")
	}
	if _, ok := gotByURL["/right-e01"]; ok == false {
		t.Fatal("whichEpisodeNeedDownloadSub() should keep exact E01 match")
	}
	if _, ok := gotByURL["/right-e02"]; ok == false {
		t.Fatal("whichEpisodeNeedDownloadSub() should keep exact E02 match")
	}
	if pack, ok := gotByURL["/season-pack"]; ok == false {
		t.Fatal("whichEpisodeNeedDownloadSub() should keep season pack as fallback for missing episodes")
	} else if pack.Season != 1 || pack.Episode != 0 {
		t.Fatalf("whichEpisodeNeedDownloadSub() season pack = S%02dE%02d; want S01 pack", pack.Season, pack.Episode)
	}
}

func TestSortMovieSubHDCandidatesPrefersBetterReleaseMatch(t *testing.T) {
	candidates := []HdListItem{
		{Title: "Unrelated.Movie.2001.1080p.BluRay.x264-GROUP", Url: "/wrong", DownCount: 99},
		{Title: "The.Hours.2002.1080p.BluRay.x264-GROUP", Url: "/best", DownCount: 3},
		{Title: "The.Hours.2002.720p.WEB-DL.x264-GROUP", Url: "/fallback", DownCount: 40},
	}

	sortMovieSubHDCandidates(candidates, filepath.Join("C:\\", "Media", "The.Hours.2002.1080p.BluRay.x264-GROUP.mkv"))

	if candidates[0].Url != "/best" {
		t.Fatalf("sortMovieSubHDCandidates() first = %q; want %q", candidates[0].Url, "/best")
	}
}

func TestSelectMovieSearchResultURLsFiltersWrongTitleAndYear(t *testing.T) {
	searchResults := []searchResultItem{
		{Title: "Time.2021.S01.1080p.WEBRip.x265-RARBG", URL: "/wrong-time"},
		{Title: "时时刻刻 官方简繁字幕 The.Hours.2002.1080p.BluRay.x264-FilmHD", URL: "/right-cn"},
		{Title: "The.Hours.2002.720p.BluRay.X264-AMIABLE", URL: "/right-en"},
		{Title: "Another.Movie.2002.1080p.BluRay.x264-GROUP", URL: "/wrong-title"},
	}

	got := selectMovieSearchResultURLs(searchResults, []string{"时时刻刻", "The Hours"}, 2002)
	if len(got) != 2 {
		t.Fatalf("selectMovieSearchResultURLs() len = %d; want 2", len(got))
	}
	gotSet := map[string]struct{}{
		got[0]: {},
		got[1]: {},
	}
	if _, ok := gotSet["/right-cn"]; ok == false {
		t.Fatalf("selectMovieSearchResultURLs() missing /right-cn: %v", got)
	}
	if _, ok := gotSet["/right-en"]; ok == false {
		t.Fatalf("selectMovieSearchResultURLs() missing /right-en: %v", got)
	}
}

func TestSelectMovieSearchResultURLsCapsFilteredResults(t *testing.T) {
	searchResults := make([]searchResultItem, 0, maxMovieSearchResultPages+3)
	searchResults = append(searchResults, searchResultItem{Title: "The.Hours.2002.1080p.BluRay.x264-GROUP", URL: "/best"})
	for i := 0; i < maxMovieSearchResultPages+3; i++ {
		searchResults = append(searchResults, searchResultItem{
			Title: fmt.Sprintf("The.Hours.2002.720p.BluRay.x264-GROUP-%d", i),
			URL:   fmt.Sprintf("/extra-%d", i),
		})
	}

	got := selectMovieSearchResultURLs(searchResults, []string{"The Hours"}, 2002)
	if len(got) != maxMovieSearchResultPages {
		t.Fatalf("selectMovieSearchResultURLs() len = %d; want %d", len(got), maxMovieSearchResultPages)
	}
	if got[0] != "/best" {
		t.Fatalf("selectMovieSearchResultURLs() first = %q; want /best", got[0])
	}
}

func TestSelectMovieSearchResultURLsFallsBackToSmallRawWindowWhenFilterEmptiesResults(t *testing.T) {
	searchResults := []searchResultItem{
		{Title: "Wrong.Movie.1997.1080p.BluRay.x264-GROUP", URL: "/year-match-1"},
		{Title: "Another.Title.1997.720p.WEB-DL.x264-GROUP", URL: "/year-match-2"},
		{Title: "Different.Movie.2001.1080p.BluRay.x264-GROUP", URL: "/wrong-year"},
	}

	got := selectMovieSearchResultURLs(searchResults, []string{"搞错人", "The Wrong Guy"}, 1997)
	if len(got) != maxMovieSearchResultFallbackPages {
		t.Fatalf("selectMovieSearchResultURLs() len = %d; want %d fallback items", len(got), maxMovieSearchResultFallbackPages)
	}
	if got[0] != "/year-match-1" || got[1] != "/year-match-2" {
		t.Fatalf("selectMovieSearchResultURLs() fallback = %v; want [/year-match-1 /year-match-2]", got)
	}
}

func TestShouldAbortMovieSubHDLoopAfterGetExOnlyForTransientGateFailures(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "probe 500", err: wrapReason(ReasonProbeFailed, fmt.Errorf("unexpected gate status 500")), want: true},
		{name: "object chain", err: wrapReason(ReasonProbeFailed, fmt.Errorf("object reference chain is too long")), want: true},
		{name: "navigation closed", err: wrapReason(ReasonProbeFailed, fmt.Errorf("probe_failed: error value: &rod.ErrNavigation{Reason:\"net::ERR_CONNECTION_CLOSED\"}")), want: true},
		{name: "download eof", err: wrapReason(ReasonProbeFailed, fmt.Errorf("probe_failed: Get \"https://subhd.me/a/test\": EOF")), want: true},
		{name: "captcha reject", err: wrapReason(ReasonCaptchaOcrFailed, fmt.Errorf("subhd captcha rejected: abcd")), want: false},
		{name: "download missing", err: wrapReason(ReasonDownloadFailed, fmt.Errorf("subhd download url missing after captcha verify")), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldAbortMovieSubHDLoopAfterGetEx(tc.err); got != tc.want {
				t.Fatalf("shouldAbortMovieSubHDLoopAfterGetEx() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestMatchSeriesTitleSupportsAliases(t *testing.T) {
	candidates := compactStrings("Lopez vs Lopez CN", "Lopez vs Lopez")
	if matchSeriesTitle("Lopez vs Lopez S01E01 1080p", candidates) == false {
		t.Fatal("matchSeriesTitle() should accept english title")
	}
	if matchSeriesTitle("Lopez vs Lopez CN S01E01", candidates) == false {
		t.Fatal("matchSeriesTitle() should accept alias title")
	}
	if matchSeriesTitle("Spiderwick Chronicles S01E01", candidates) == true {
		t.Fatal("matchSeriesTitle() should reject wrong series title")
	}
}

func TestMatchSeriesTitleRejectsBroadChinesePartialMatch(t *testing.T) {
	candidates := compactStrings("黑袍纠察队", "The Boys")
	if matchSeriesTitle("黑袍纠察队：V世代.Gen.V.S01E08.1080p.WEB.h264-ETHEL_chi&eng", candidates) == true {
		t.Fatal("matchSeriesTitle() should reject Gen V when target is The Boys")
	}
}

func TestMatchSeriesTitleKeepsBilingualExactShowMatch(t *testing.T) {
	candidates := compactStrings("拥挤的房间", "The Crowded Room")
	if matchSeriesTitle("拥挤的房间.The.Crowded.Room.S01E09.Family.1080p.ATVP.WEB-DL.DDP5.1.H.264-NTb", candidates) == false {
		t.Fatal("matchSeriesTitle() should keep bilingual same-show title")
	}
}

func TestMatchSeriesTitleRejectsShortEnglishFragment(t *testing.T) {
	candidates := compactStrings("黑袍纠察队", "The Boys")
	if matchSeriesTitle("Invisible.Boys.S01E08.1080p.WEB.h264-ETHEL", candidates) == true {
		t.Fatal("matchSeriesTitle() should reject generic Boys fragment match")
	}
	if matchSeriesTitle("The.Boys.S01.1080p.BluRay.x264-SHORTBREHD", candidates) == false {
		t.Fatal("matchSeriesTitle() should keep exact The Boys match")
	}
}

func TestSelectSearchResultURLsPrefersMatchingSeriesResults(t *testing.T) {
	searchResults := []searchResultItem{
		{Title: "Spiderwick Chronicles (2024)", URL: "/wrong"},
		{Title: "Lopez vs Lopez (2022)", URL: "/right"},
		{Title: "Lopez vs Lopez Season 1", URL: "/right-pack"},
	}

	got := selectSearchResultURLs(searchResults, []string{"Lopez vs Lopez"}, 0, 0)
	if len(got) != 2 {
		t.Fatalf("selectSearchResultURLs() len = %d; want 2", len(got))
	}
	if got[0] != "/right" || got[1] != "/right-pack" {
		t.Fatalf("selectSearchResultURLs() = %v; want [/right /right-pack]", got)
	}
}

func TestSelectSearchResultURLsMatchesReleaseStyleTitles(t *testing.T) {
	searchResults := []searchResultItem{
		{Title: "Rick.and.Morty.S05E09.Forgetting.Sarick.Mortshall.1080p.AMZN.WEB-DL", URL: "/episode"},
		{Title: "Rick.and.Morty.S05.1080p.BluRay.x264-BORDURE", URL: "/season"},
		{Title: "Spiderwick.Chronicles.S01E01.1080p.WEB-DL", URL: "/wrong"},
	}

	got := selectSearchResultURLs(searchResults, []string{"Rick and Morty"}, 0, 0)
	if len(got) != 2 {
		t.Fatalf("selectSearchResultURLs() len = %d; want 2", len(got))
	}
	if got[0] != "/episode" || got[1] != "/season" {
		t.Fatalf("selectSearchResultURLs() = %v; want [/episode /season]", got)
	}
}

func TestSelectSearchResultURLsMatchesBilingualReleaseStyleTitles(t *testing.T) {
	searchResults := []searchResultItem{
		{Title: "瑞克和莫蒂.Rick.and.Morty.S09E03.Rick.Fu.Hustle.1080p.AMZN.WEB-DL", URL: "/bilingual"},
		{Title: "蜘蛛侠.Spiderwick.Chronicles.S01E01.1080p.WEB-DL", URL: "/wrong"},
	}

	got := selectSearchResultURLs(searchResults, []string{"Rick and Morty"}, 0, 0)
	if len(got) != 1 || got[0] != "/bilingual" {
		t.Fatalf("selectSearchResultURLs() = %v; want [/bilingual]", got)
	}
}

func TestSelectSearchResultURLsSkipsWrongSeasonAndKeepsTargetSeason(t *testing.T) {
	searchResults := []searchResultItem{
		{Title: "Rick.and.Morty.Season.7.BD.REMUX.DTS-HD.MA.5.1-PB69", URL: "/season-7"},
		{Title: "Rick.and.Morty.Season.6.1080p.WEB-DL", URL: "/season-6"},
		{Title: "Rick.and.Morty.S05.1080p.BluRay.x264-BORDURE", URL: "/season-5-pack"},
		{Title: "Rick.and.Morty.S05E09.Forgetting.Sarick.Mortshall.1080p.AMZN.WEB-DL", URL: "/season-5-ep"},
	}

	got := selectSearchResultURLs(searchResults, []string{"Rick and Morty"}, 5, 0)
	if len(got) != 2 {
		t.Fatalf("selectSearchResultURLs() len = %d; want 2", len(got))
	}
	gotSet := map[string]struct{}{
		got[0]: {},
		got[1]: {},
	}
	if _, ok := gotSet["/season-5-pack"]; ok == false {
		t.Fatalf("selectSearchResultURLs() missing season pack result: %v", got)
	}
	if _, ok := gotSet["/season-5-ep"]; ok == false {
		t.Fatalf("selectSearchResultURLs() missing episode result: %v", got)
	}
}

func TestSelectSearchResultURLsReturnsEmptyWhenOnlyWrongSeasonMatchesExist(t *testing.T) {
	searchResults := []searchResultItem{
		{Title: "Rick.and.Morty.Season.7.BD.REMUX.DTS-HD.MA.5.1-PB69", URL: "/season-7"},
		{Title: "Rick.and.Morty.Season1.1080p.BluRay", URL: "/season-1"},
	}

	got := selectSearchResultURLs(searchResults, []string{"Rick and Morty"}, 5, 0)
	if len(got) != 0 {
		t.Fatalf("selectSearchResultURLs() = %v; want empty result for wrong seasons only", got)
	}
}

func TestBuildSeriesSearchKeywordsOrdersSeasonVariantsBeforeBaseTitle(t *testing.T) {
	got := buildSeriesSearchKeywords("Rick and Morty", 5)
	if len(got) < 4 {
		t.Fatalf("buildSeriesSearchKeywords() = %#v; want 4 variants", got)
	}
	if got[0] != "Rick and Morty S05" {
		t.Fatalf("unexpected first keyword %#v", got)
	}
	if got[1] != "Rick and Morty Season 5" {
		t.Fatalf("unexpected second keyword %#v", got)
	}
	if got[len(got)-1] != "Rick and Morty" {
		t.Fatalf("expected base title last, got %#v", got)
	}
}

func TestBuildSubHDSeriesTitleCandidatesIncludesNormalizedRootDir(t *testing.T) {
	seriesInfo := &series.SeriesInfo{
		Name: "瑞克和莫蒂 - S05E09 - 第 9 集",
		NeedDlEpsKeyList: map[string]series.EpisodeInfo{
			pkg.GetEpisodeKeyName(5, 9): {
				Season:       5,
				Episode:      9,
				FileFullPath: filepath.Join("C:\\", "Media", "Rick and Morty (2013)", "Season 5", "瑞克和莫蒂 - S05E09 - 第 9 集.mkv"),
			},
		},
	}

	got := buildSubHDSeriesTitleCandidates(seriesInfo, nil)
	found := false
	for _, item := range got {
		if item == "Rick and Morty" {
			found = true
			break
		}
	}
	if found == false {
		t.Fatalf("buildSubHDSeriesTitleCandidates() = %#v; want Rick and Morty candidate", got)
	}
}

func TestBuildSubHDSeriesTitleCandidatesIncludesSeriesNfoAliases(t *testing.T) {
	rootDir := t.TempDir()
	seriesDir := filepath.Join(rootDir, "拥挤的房间 (2023)")
	seasonDir := filepath.Join(seriesDir, "Season 1")
	if err := os.MkdirAll(seasonDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	tvshowNfo := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<tvshow>
  <title>拥挤的房间</title>
  <originaltitle>The Crowded Room</originaltitle>
</tvshow>`
	if err := os.WriteFile(filepath.Join(seriesDir, "tvshow.nfo"), []byte(tvshowNfo), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	videoPath := filepath.Join(seasonDir, "拥挤的房间.S01E01.mkv")
	if err := os.WriteFile(videoPath, []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile() video error = %v", err)
	}

	seriesInfo := &series.SeriesInfo{
		Name:    "拥挤的房间",
		DirPath: seriesDir,
		NeedDlEpsKeyList: map[string]series.EpisodeInfo{
			pkg.GetEpisodeKeyName(1, 1): {
				Season:       1,
				Episode:      1,
				FileFullPath: videoPath,
			},
		},
	}

	got := buildSubHDSeriesTitleCandidates(seriesInfo, nil)
	gotSet := make(map[string]struct{}, len(got))
	for _, item := range got {
		gotSet[item] = struct{}{}
	}
	if _, ok := gotSet["The Crowded Room"]; ok == false {
		t.Fatalf("buildSubHDSeriesTitleCandidates() = %#v; want The Crowded Room from tvshow.nfo", got)
	}
	if _, ok := gotSet["拥挤的房间"]; ok == false {
		t.Fatalf("buildSubHDSeriesTitleCandidates() = %#v; want chinese title", got)
	}
}

func TestSelectSearchResultURLsFallsBackWhenNoTitleMatches(t *testing.T) {
	searchResults := []searchResultItem{
		{Title: "Unknown Result A", URL: "/a"},
		{Title: "Unknown Result B", URL: "/b"},
	}

	got := selectSearchResultURLs(searchResults, []string{"Lopez vs Lopez"}, 0, 0)
	if len(got) != 0 {
		t.Fatalf("selectSearchResultURLs() len = %d; want 0", len(got))
	}
}

func TestSelectSearchResultURLsKeepsAllResultsWithoutTitleCandidates(t *testing.T) {
	searchResults := []searchResultItem{
		{Title: "Unknown Result A", URL: "/a"},
		{Title: "Unknown Result B", URL: "/b"},
	}

	got := selectSearchResultURLs(searchResults, nil, 0, 0)
	if len(got) != 2 {
		t.Fatalf("selectSearchResultURLs() len = %d; want 2", len(got))
	}
	if got[0] != "/a" || got[1] != "/b" {
		t.Fatalf("selectSearchResultURLs() = %v; want [/a /b]", got)
	}
}

func TestSelectSearchResultURLsCapsSeriesResultsAndKeepsBestMatchesFirst(t *testing.T) {
	searchResults := []searchResultItem{
		{Title: "Wrong Show Season 1", URL: "/wrong"},
		{Title: "The Boys Season 5", URL: "/season"},
		{Title: "The Boys", URL: "/exact"},
	}
	for i := 0; i < 8; i++ {
		searchResults = append(searchResults, searchResultItem{
			Title: fmt.Sprintf("The Boys Extra %d", i),
			URL:   fmt.Sprintf("/extra-%d", i),
		})
	}

	got := selectSearchResultURLs(searchResults, []string{"The Boys"}, 0, 0)
	if len(got) != maxSeriesSearchResultPages {
		t.Fatalf("selectSearchResultURLs() len = %d; want %d", len(got), maxSeriesSearchResultPages)
	}
	if got[0] != "/exact" || got[1] != "/season" {
		t.Fatalf("selectSearchResultURLs() first urls = %v; want [/exact /season ...]", got[:2])
	}
	for _, url := range got {
		if url == "/wrong" {
			t.Fatal("selectSearchResultURLs() should drop non-matching results")
		}
	}
}

func TestParseSearchResultsKeepsPageOrderAndDeduplicatesURL(t *testing.T) {
	html := `
<html><body>
  <a href="/detail-b"><img class="rounded-start" src="b.jpg" />Lopez vs Lopez (2022)</a>
  <div><a href="/detail-a"><img class="rounded-start" src="a.jpg" />Lopez vs Lopez Season 1</a></div>
  <a href="/detail-b"><img class="rounded-start" src="b2.jpg" />Lopez vs Lopez Duplicate</a>
</body></html>`

	got, count, err := parseSearchResults(html)
	if err != nil {
		t.Fatalf("parseSearchResults() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("parseSearchResults() count = %d; want 2", count)
	}
	if len(got) != 2 {
		t.Fatalf("parseSearchResults() len = %d; want 2", len(got))
	}
	if got[0].URL != "/detail-b" || got[1].URL != "/detail-a" {
		t.Fatalf("parseSearchResults() urls = %v; want [/detail-b /detail-a]", []string{got[0].URL, got[1].URL})
	}
}

func TestParseSearchResultsUsesCurrentCardLayoutTitleAndDetailLink(t *testing.T) {
	html := `
<html><body>
  <div class="bg-white shadow-sm rounded-3 mb-4">
    <div class="row">
      <div class="col-2 d-none d-lg-block">
        <a href="/d/poster-a"><div class="pics"><img class="rounded-start" src="a.jpg" /></div></a>
      </div>
      <div class="col-lg-10">
        <div class="view-text text-secondary">
          <a href="/a/detail-a" class="link-dark">Rick.and.Morty.S05E09.Forgetting.Sarick.Mortshall.1080p.AMZN.WEB-DL</a>
        </div>
      </div>
    </div>
  </div>
  <div class="bg-white shadow-sm rounded-3 mb-4">
    <div class="row">
      <div class="col-2 d-none d-lg-block">
        <a href="/d/poster-b"><div class="pics"><img class="rounded-start" src="b.jpg" /></div></a>
      </div>
      <div class="col-lg-10">
        <div class="view-text text-secondary">
          <a href="/a/detail-b" class="link-dark">Rick.and.Morty.S05.1080p.BluRay.x264-BORDURE</a>
        </div>
      </div>
    </div>
  </div>
</body></html>`

	got, count, err := parseSearchResults(html)
	if err != nil {
		t.Fatalf("parseSearchResults() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("parseSearchResults() count = %d; want 2", count)
	}
	if len(got) != 2 {
		t.Fatalf("parseSearchResults() len = %d; want 2", len(got))
	}
	if got[0].URL != "/a/detail-a" || got[1].URL != "/a/detail-b" {
		t.Fatalf("parseSearchResults() urls = %v; want [/a/detail-a /a/detail-b]", []string{got[0].URL, got[1].URL})
	}
	if got[0].Title == "" || got[1].Title == "" {
		t.Fatalf("parseSearchResults() titles should not be empty: %#v", got)
	}
}

func TestParseSearchResultsTreatsZeroResultPageAsMiss(t *testing.T) {
	html := `
<html><body>
  <h4 class="py-4">
    洛佩兹一家+第三季 的搜索结果
    <span class="f13 ps-3 text-secondary">共 0 条 当前第 1 页</span>
  </h4>
</body></html>`

	got, count, err := parseSearchResults(html)
	if err != nil {
		t.Fatalf("parseSearchResults() error = %v", err)
	}
	if count != 0 {
		t.Fatalf("parseSearchResults() count = %d; want 0", count)
	}
	if len(got) != 0 {
		t.Fatalf("parseSearchResults() len = %d; want 0", len(got))
	}
}

func TestParseSubtitleRowsSupportsCurrentSingleDetailLayout(t *testing.T) {
	html := `
<html><body>
  <div class="bg-white shadow-sm rounded-3 mt-3 mb-3">
    <div class="p-3">
      <div class="f16 fw-bold mb-2">Rick.and.Morty.S05.1080p.BluRay.x264-BORDURE</div>
    </div>
    <div class="p-3 my-2 bg-light clearfix">
      <div class="float-start">
        <span class="p-1 fw-bold">双语</span>
        <span class="p-1 fw-bold">简体</span>
        <span class="p-1 text-secondary">ASS</span>
      </div>
      <div class="float-end">
        <span class='align-text-top me-3'>211k</span>
        <span class="align-text-top me-3">2522</span>
        <span class="align-text-top">2024-7-9 23:55</span>
      </div>
    </div>
  </div>
  <div class="mb-3 clearfix">
    <div class="float-start">
      <a class="btn btn-danger down" sid="OyURuJ" href="/down/OyURuJ" target="_blank">下载字幕文件</a>
    </div>
  </div>
</body></html>`

	got, err := parseSubtitleRows(html, "https://subhd.me", "/a/OyURuJ", false, 1)
	if err != nil {
		t.Fatalf("parseSubtitleRows() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("parseSubtitleRows() len = %d; want 1", len(got))
	}
	if got[0].Url != "https://subhd.me/a/OyURuJ" {
		t.Fatalf("parseSubtitleRows() url = %q; want %q", got[0].Url, "https://subhd.me/a/OyURuJ")
	}
	if got[0].Title != "Rick.and.Morty.S05.1080p.BluRay.x264-BORDURE" {
		t.Fatalf("parseSubtitleRows() title = %q", got[0].Title)
	}
	if got[0].DownCount != 2522 {
		t.Fatalf("parseSubtitleRows() downCount = %d; want 2522", got[0].DownCount)
	}
}

func TestConfiguredCaptchaBackendDefaultsToDDDDOCR(t *testing.T) {
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())

	cfg := settings.Get()
	oldBackend := cfg.SubtitleSources.SubHDSettings.OCRBackend
	oldURL := cfg.SubtitleSources.SubHDSettings.ExternalOCRURL
	cfg.SubtitleSources.SubHDSettings.OCRBackend = ""
	cfg.SubtitleSources.SubHDSettings.ExternalOCRURL = ""
	defer func() {
		cfg.SubtitleSources.SubHDSettings.OCRBackend = oldBackend
		cfg.SubtitleSources.SubHDSettings.ExternalOCRURL = oldURL
	}()

	if got := configuredCaptchaBackend(); got != "ddddocr" {
		t.Fatalf("configuredCaptchaBackend() = %q; want ddddocr", got)
	}
}

func TestConfiguredCaptchaBackendAllowsExplicitExternal(t *testing.T) {
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())

	cfg := settings.Get()
	oldBackend := cfg.SubtitleSources.SubHDSettings.OCRBackend
	oldURL := cfg.SubtitleSources.SubHDSettings.ExternalOCRURL
	cfg.SubtitleSources.SubHDSettings.OCRBackend = "external"
	cfg.SubtitleSources.SubHDSettings.ExternalOCRURL = "http://127.0.0.1:9999/ocr"
	defer func() {
		cfg.SubtitleSources.SubHDSettings.OCRBackend = oldBackend
		cfg.SubtitleSources.SubHDSettings.ExternalOCRURL = oldURL
	}()

	if got := configuredCaptchaBackend(); got != "external" {
		t.Fatalf("configuredCaptchaBackend() = %q; want external", got)
	}
}

func TestConfiguredCaptchaBackendIgnoresExternalURLWithoutExplicitBackend(t *testing.T) {
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())

	cfg := settings.Get()
	oldBackend := cfg.SubtitleSources.SubHDSettings.OCRBackend
	oldURL := cfg.SubtitleSources.SubHDSettings.ExternalOCRURL
	cfg.SubtitleSources.SubHDSettings.OCRBackend = ""
	cfg.SubtitleSources.SubHDSettings.ExternalOCRURL = "http://127.0.0.1:9999/ocr"
	defer func() {
		cfg.SubtitleSources.SubHDSettings.OCRBackend = oldBackend
		cfg.SubtitleSources.SubHDSettings.ExternalOCRURL = oldURL
	}()

	if got := configuredCaptchaBackend(); got != "ddddocr" {
		t.Fatalf("configuredCaptchaBackend() = %q; want ddddocr when backend is not explicitly external", got)
	}
}

func TestConfiguredCaptchaBackendFallsBackToDDDDOCRForUnknownValue(t *testing.T) {
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())

	cfg := settings.Get()
	oldBackend := cfg.SubtitleSources.SubHDSettings.OCRBackend
	cfg.SubtitleSources.SubHDSettings.OCRBackend = "tesseract"
	defer func() {
		cfg.SubtitleSources.SubHDSettings.OCRBackend = oldBackend
	}()

	if got := configuredCaptchaBackend(); got != "ddddocr" {
		t.Fatalf("configuredCaptchaBackend() = %q; want ddddocr for unknown backend", got)
	}
}

func TestPrepareCaptchaPNGForOCRCropsWhitespaceAndScales(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 20, 12))
	for y := 0; y < 12; y++ {
		for x := 0; x < 20; x++ {
			img.Set(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	for y := 4; y <= 7; y++ {
		for x := 6; x <= 10; x++ {
			img.Set(x, y, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
		}
	}

	var in bytes.Buffer
	if err := png.Encode(&in, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}

	got, err := prepareCaptchaPNGForOCR(in.Bytes())
	if err != nil {
		t.Fatalf("prepareCaptchaPNGForOCR() error = %v", err)
	}

	outImg, err := png.Decode(bytes.NewReader(got))
	if err != nil {
		t.Fatalf("png.Decode() error = %v", err)
	}

	if outImg.Bounds().Dx() >= img.Bounds().Dx()*captchaScaleFactor {
		t.Fatalf("expected cropped width smaller than full scaled width, got %d", outImg.Bounds().Dx())
	}
	if outImg.Bounds().Dx() <= (10 - 6 + 1) {
		t.Fatalf("expected scaled width larger than raw foreground width, got %d", outImg.Bounds().Dx())
	}
	if outImg.Bounds().Dy() <= (7 - 4 + 1) {
		t.Fatalf("expected scaled height larger than raw foreground height, got %d", outImg.Bounds().Dy())
	}
}

func TestPrepareCaptchaPNGForOCRFallsBackWhenNoForeground(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}

	var in bytes.Buffer
	if err := png.Encode(&in, img); err != nil {
		t.Fatalf("png.Encode() error = %v", err)
	}

	got, err := prepareCaptchaPNGForOCR(in.Bytes())
	if err != nil {
		t.Fatalf("prepareCaptchaPNGForOCR() error = %v", err)
	}
	if bytes.Equal(got, in.Bytes()) == false {
		t.Fatal("expected all-white captcha to keep original bytes")
	}
}

func TestCaptchaChallengeMessageAcceptsRefreshedSVG(t *testing.T) {
	resp := &downloadGateResponse{
		Success: true,
		Pass:    false,
		Msg:     "<svg><text>abcd</text></svg>",
	}

	if got := captchaChallengeMessage(resp); got != resp.Msg {
		t.Fatalf("captchaChallengeMessage() = %q", got)
	}
}

func TestCaptchaChallengeMessageRejectsPlainErrorText(t *testing.T) {
	resp := &downloadGateResponse{
		Success: true,
		Pass:    false,
		Msg:     "captcha incorrect",
	}

	if got := captchaChallengeMessage(resp); got != "" {
		t.Fatalf("captchaChallengeMessage() = %q; want empty", got)
	}
}

func TestExtractCaptchaTextFromSVGNestedTSpan(t *testing.T) {
	svg := `<svg><text><tspan>ab</tspan><tspan>12</tspan></text></svg>`

	if got := extractCaptchaTextFromSVG(svg); got != "ab12" {
		t.Fatalf("extractCaptchaTextFromSVG() = %q; want ab12", got)
	}
}

func TestExtractCaptchaTextFromSVGDecodesEntities(t *testing.T) {
	svg := `<svg><text>a&#49;b&#50;</text></svg>`

	if got := extractCaptchaTextFromSVG(svg); got != "a1b2" {
		t.Fatalf("extractCaptchaTextFromSVG() = %q; want a1b2", got)
	}
}

func TestShouldRetryDownloadGateProbeOnlyForTransient5xx(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "500", err: fmt.Errorf("probe_failed: unexpected gate status 500"), want: true},
		{name: "503", err: fmt.Errorf("probe_failed: unexpected gate status 503"), want: true},
		{name: "navigation closed", err: fmt.Errorf("probe_failed: error value: &rod.ErrNavigation{Reason:\"net::ERR_CONNECTION_CLOSED\"}"), want: true},
		{name: "download eof", err: fmt.Errorf("probe_failed: Get \"https://subhd.me/a/test\": EOF"), want: true},
		{name: "connection reset", err: fmt.Errorf("probe_failed: read tcp 172.17.0.2:55392->104.21.48.1:443: connection reset by peer"), want: true},
		{name: "expired temporary page", err: fmt.Errorf("download_gate_changed: 时间过长本临时页面已经失效"), want: true},
		{name: "object reference chain too long", err: fmt.Errorf("probe_failed: error value: &cdp.Error{Code:-32000, Message:\"Object reference chain is too long\", Data:\"\"}"), want: true},
		{name: "captcha reject", err: fmt.Errorf("captcha_ocr_failed: subhd captcha rejected"), want: false},
		{name: "layout changed", err: fmt.Errorf("download_gate_changed: button changed"), want: false},
		{name: "nil", err: nil, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRetryDownloadGateProbe(tc.err); got != tc.want {
				t.Fatalf("shouldRetryDownloadGateProbe() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestNextDownloadGateRetryDelayUsesLinearBackoff(t *testing.T) {
	if got := nextDownloadGateRetryDelay(0); got != subHDGateRetryBaseDelay {
		t.Fatalf("nextDownloadGateRetryDelay(0) = %v; want %v", got, subHDGateRetryBaseDelay)
	}
	if got := nextDownloadGateRetryDelay(1); got != subHDGateRetryBaseDelay {
		t.Fatalf("nextDownloadGateRetryDelay(1) = %v; want %v", got, subHDGateRetryBaseDelay)
	}
	if got := nextDownloadGateRetryDelay(3); got != 3*subHDGateRetryBaseDelay {
		t.Fatalf("nextDownloadGateRetryDelay(3) = %v; want %v", got, 3*subHDGateRetryBaseDelay)
	}
}

func TestIsExpiredTemporaryDownloadGateError(t *testing.T) {
	cases := []struct {
		name string
		msg  string
		want bool
	}{
		{name: "empty", msg: "", want: false},
		{name: "cn", msg: "download_gate_changed: 时间过长本临时页面已经失效", want: true},
		{name: "garbled cn", msg: "download_gate_changed: 鏃堕棿杩囬暱鏈复鏃堕〉闈㈠凡缁忓け鏁?", want: true},
		{name: "en", msg: "download_gate_changed: temporary page expired", want: true},
		{name: "unrelated temporary", msg: "download_gate_changed: temporary challenge changed", want: false},
		{name: "unrelated expired", msg: "download_gate_changed: token expired", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isExpiredTemporaryDownloadGateError(tc.msg); got != tc.want {
				t.Fatalf("isExpiredTemporaryDownloadGateError() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestShouldRetryCaptchaOCRFailureOnlyForUnexpectedOCRShape(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "short output", err: fmt.Errorf("captcha_ocr_failed: unexpected captcha OCR output \"bxq\""), want: true},
		{name: "refreshed challenge reject", err: fmt.Errorf("captcha_ocr_failed: subhd captcha rejected after refreshed challenge: mqgk"), want: true},
		{name: "plain reject", err: fmt.Errorf("captcha_ocr_failed: subhd captcha rejected: abcd"), want: false},
		{name: "layout", err: fmt.Errorf("download_gate_changed: button changed"), want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRetryCaptchaOCRFailure(tc.err); got != tc.want {
				t.Fatalf("shouldRetryCaptchaOCRFailure() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestShouldRetryTransientPageEvalOnlyForObjectReferenceChainError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "object chain", err: fmt.Errorf("error value: &cdp.Error{Code:-32000, Message:\"Object reference chain is too long\", Data:\"\"}"), want: true},
		{name: "timeout", err: fmt.Errorf("context deadline exceeded"), want: false},
		{name: "captcha reject", err: fmt.Errorf("captcha_ocr_failed: subhd captcha rejected"), want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldRetryTransientPageEval(tc.err); got != tc.want {
				t.Fatalf("shouldRetryTransientPageEval() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestWithTransientPageEvalRetryRetriesObjectReferenceChainOnce(t *testing.T) {
	attempts := 0
	err := withTransientPageEvalRetry(func() error {
		attempts++
		if attempts == 1 {
			return fmt.Errorf("error value: &cdp.Error{Code:-32000, Message:\"Object reference chain is too long\", Data:\"\"}")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("withTransientPageEvalRetry() error = %v", err)
	}
	if attempts != 2 {
		t.Fatalf("attempts = %d; want 2", attempts)
	}
}

func TestNormalizeTransientPageEvalErrorCompactsRodStackNoise(t *testing.T) {
	rawErr := fmt.Errorf("error value: &cdp.Error{Code:-32000, Message:\"Object reference chain is too long\", Data:\"\"}\ngoroutine 1 [running]:\nruntime/debug.Stack()")

	got := normalizeTransientPageEvalError(rawErr)
	if got == nil {
		t.Fatal("normalizeTransientPageEvalError() returned nil")
	}
	if got.Error() != "object reference chain is too long" {
		t.Fatalf("normalizeTransientPageEvalError() = %q; want compact transient error", got.Error())
	}
}

func TestShouldIgnoreSubHDDownloadNavigateError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "aborted", err: fmt.Errorf("net::ERR_ABORTED"), want: true},
		{name: "object chain", err: fmt.Errorf("Object reference chain is too long"), want: true},
		{name: "timeout", err: fmt.Errorf("context deadline exceeded"), want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldIgnoreSubHDDownloadNavigateError(tc.err); got != tc.want {
				t.Fatalf("shouldIgnoreSubHDDownloadNavigateError() = %v; want %v", got, tc.want)
			}
		})
	}
}
