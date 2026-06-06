package subhd

import (
	"path/filepath"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/sirupsen/logrus"
)

func TestWhichEpisodeNeedDownloadSubFiltersWrongSeriesTitle(t *testing.T) {
	supplier := &Supplier{log: logrus.New()}
	seriesInfo := &series.SeriesInfo{
		Name: "洛佩兹一家",
		NeedDlEpsKeyList: map[string]series.EpisodeInfo{
			pkg.GetEpisodeKeyName(1, 2): {
				Season:       1,
				Episode:      2,
				FileFullPath: filepath.Join("C:\\", "Media", "Lopez.vs.Lopez.S01E02.1080p.WEB-DL.mkv"),
			},
		},
	}
	mediaInfo := &models.MediaInfo{
		TitleCn:       "洛佩兹一家",
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
		Name: "洛佩兹一家",
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
		TitleCn:       "洛佩兹一家",
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

func TestMatchSeriesTitleSupportsChineseAndEnglish(t *testing.T) {
	candidates := compactStrings("洛佩兹一家", "Lopez vs Lopez")
	if matchSeriesTitle("Lopez vs Lopez S01E01 1080p", candidates) == false {
		t.Fatal("matchSeriesTitle() should accept english title")
	}
	if matchSeriesTitle("洛佩兹一家 S01E01", candidates) == false {
		t.Fatal("matchSeriesTitle() should accept chinese title")
	}
	if matchSeriesTitle("Spiderwick Chronicles S01E01", candidates) == true {
		t.Fatal("matchSeriesTitle() should reject wrong series title")
	}
}
