package subhd

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/sirupsen/logrus"
)

func TestOverDailyDownloadLimitTreatsNegativeLimitAsUnlimited(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.SuppliersSettings.SubHD.DailyDownloadLimit = -1

	supplier := &Supplier{log: logrus.New()}
	if supplier.OverDailyDownloadLimit() {
		t.Fatal("OverDailyDownloadLimit() = true; want false for unlimited limit")
	}
}

func TestProbeRootURLsAddsStableFallbacks(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.SuppliersSettings.SubHD.RootUrl = "https://subhd.tv"

	supplier := &Supplier{log: logrus.New()}
	got := supplier.probeRootURLs()
	if len(got) < 2 {
		t.Fatalf("probeRootURLs() len = %d; want at least 2", len(got))
	}
	if got[0] != "https://subhd.me" {
		t.Fatalf("probeRootURLs()[0] = %q; want %q", got[0], "https://subhd.me")
	}
	if got[1] != "https://subhd.one" {
		t.Fatalf("probeRootURLs()[1] = %q; want %q", got[1], "https://subhd.one")
	}
}

func TestShouldKeepAliveOnProbeErrorTreatsTimeoutAsTransient(t *testing.T) {
	if shouldKeepAliveOnProbeError(errors.New("context deadline exceeded")) == false {
		t.Fatal("shouldKeepAliveOnProbeError() = false; want true for timeout")
	}
	if shouldKeepAliveOnProbeError(errors.New("connection reset by peer")) == false {
		t.Fatal("shouldKeepAliveOnProbeError() = false; want true for connection reset")
	}
	if shouldKeepAliveOnProbeError(errors.New("unexpected status")) == true {
		t.Fatal("shouldKeepAliveOnProbeError() = true; want false for non-transient error")
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

func TestSelectSearchResultURLsPrefersMatchingSeriesResults(t *testing.T) {
	searchResults := []searchResultItem{
		{Title: "Spiderwick Chronicles (2024)", URL: "/wrong"},
		{Title: "Lopez vs Lopez (2022)", URL: "/right"},
		{Title: "Lopez vs Lopez Season 1", URL: "/right-pack"},
	}

	got := selectSearchResultURLs(searchResults, []string{"Lopez vs Lopez"})
	if len(got) != 2 {
		t.Fatalf("selectSearchResultURLs() len = %d; want 2", len(got))
	}
	if got[0] != "/right" || got[1] != "/right-pack" {
		t.Fatalf("selectSearchResultURLs() = %v; want [/right /right-pack]", got)
	}
}

func TestSelectSearchResultURLsFallsBackWhenNoTitleMatches(t *testing.T) {
	searchResults := []searchResultItem{
		{Title: "Unknown Result A", URL: "/a"},
		{Title: "Unknown Result B", URL: "/b"},
	}

	got := selectSearchResultURLs(searchResults, []string{"Lopez vs Lopez"})
	if len(got) != 0 {
		t.Fatalf("selectSearchResultURLs() len = %d; want 0", len(got))
	}
}

func TestSelectSearchResultURLsKeepsAllResultsWithoutTitleCandidates(t *testing.T) {
	searchResults := []searchResultItem{
		{Title: "Unknown Result A", URL: "/a"},
		{Title: "Unknown Result B", URL: "/b"},
	}

	got := selectSearchResultURLs(searchResults, nil)
	if len(got) != 2 {
		t.Fatalf("selectSearchResultURLs() len = %d; want 2", len(got))
	}
	if got[0] != "/a" || got[1] != "/b" {
		t.Fatalf("selectSearchResultURLs() = %v; want [/a /b]", got)
	}
}

func TestParseSearchResultsKeepsPageOrderAndDeduplicatesURL(t *testing.T) {
	html := `
<html><body>
  <h4>Lopez vs Lopez 的搜索结果 <span>共 2 条 当前第 1 页</span></h4>
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

func TestParseSearchResultsTreatsZeroCountAsNoResults(t *testing.T) {
	html := `
<html><body>
  <h4>Lopez vs Lopez 的搜索结果 <span>共 0 条 当前第 1 页</span></h4>
  <a class="link-dark" target="_blank" href="/d/123">热门条目</a>
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

func TestParseSearchResultsSupportsAnchorResultsWithoutImage(t *testing.T) {
	html := `
<html><body>
  <h4>tt35522483 的搜索结果 <span>共 2 条 当前第 1 页</span></h4>
  <a class="link-dark align-middle" href="/a/right-a">尼瓦那乐队秀：电影版</a>
  <a class="link-dark align-middle" href="/a/right-b">尼瓦那乐队秀：电影版 第二条</a>
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
	if got[0].URL != "/a/right-a" || got[1].URL != "/a/right-b" {
		t.Fatalf("parseSearchResults() urls = %v; want [/a/right-a /a/right-b]", []string{got[0].URL, got[1].URL})
	}
}
