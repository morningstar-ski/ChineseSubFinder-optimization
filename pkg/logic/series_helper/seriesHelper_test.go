package series_helper

import (
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/ifaces"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/media_info_dealers"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/unit_test_helper"
	"github.com/sirupsen/logrus"
)

func TestReadSeriesInfoFromDir(t *testing.T) {

	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())
	series := unit_test_helper.SkipIfTestDataResourceAbsent(t, []string{"series", "Loki"}, 4, false)
	dealers := media_info_dealers.NewDealers(log_helper.GetLogger4Tester(), nil)
	seriesInfo, err := ReadSeriesInfoFromDir(dealers, series, 90, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if seriesInfo == nil {
		t.Fatal("ReadSeriesInfoFromDir() returned nil")
	}
}

func TestGetSeriesListFromDirs(t *testing.T) {

	series := unit_test_helper.SkipIfTestDataResourceAbsent(t, []string{"series"}, 4, false)
	got, err := GetSeriesListFromDirs(log_helper.GetLogger4Tester(), []string{series})
	if err != nil {
		t.Fatal(err)
	}

	if got.Size() < 1 {
		t.Fatal("GetSeriesListFromDirs got len < 1")
	}
}

func TestSeriesSubtitlesCoverNeedDlEpisodes(t *testing.T) {
	seriesInfo := &series.SeriesInfo{
		NeedDlEpsKeyList: map[string]series.EpisodeInfo{
			pkg.GetEpisodeKeyName(1, 2): {Season: 1, Episode: 2},
			pkg.GetEpisodeKeyName(1, 3): {Season: 1, Episode: 3},
		},
	}

	if seriesSubtitlesCoverNeedDlEpisodes(seriesInfo, []supplier.SubInfo{
		{Season: 1, Episode: 2},
	}) {
		t.Fatalf("expected partial episode coverage to be false")
	}

	if seriesSubtitlesCoverNeedDlEpisodes(seriesInfo, []supplier.SubInfo{
		{Season: 1, Episode: 0, IsFullSeason: true},
	}) == false {
		t.Fatalf("expected full-season subtitle to cover all pending episodes")
	}

	if seriesSubtitlesCoverNeedDlEpisodes(seriesInfo, []supplier.SubInfo{
		{Season: 1, Episode: 2},
		{Season: 1, Episode: 3},
	}) == false {
		t.Fatalf("expected exact episode subtitles to cover all pending episodes")
	}
}

func TestDownloadSubtitleInAllSiteByOneSeriesStopsAfterCoverage(t *testing.T) {
	seriesInfo := &series.SeriesInfo{
		DirPath: "C:\\Media\\Series",
		NeedDlEpsKeyList: map[string]series.EpisodeInfo{
			pkg.GetEpisodeKeyName(1, 2): {Season: 1, Episode: 2},
		},
	}
	first := &seriesHelperStubSupplier{
		name: "first",
		seriesSubInfos: []supplier.SubInfo{
			{Name: "s01e02.srt", Ext: ".srt", Season: 1, Episode: 2},
		},
	}
	second := &seriesHelperStubSupplier{
		name: "second",
		seriesSubInfos: []supplier.SubInfo{
			{Name: "s01e02-alt.srt", Ext: ".srt", Season: 1, Episode: 2},
		},
	}

	got := DownloadSubtitleInAllSiteByOneSeries(log_helper.GetLogger4Tester(), []ifaces.ISupplier{first, second}, seriesInfo, 1)
	if len(got) != 1 {
		t.Fatalf("expected 1 subtitle from first supplier, got %d", len(got))
	}
	if first.seriesCalls != 1 {
		t.Fatalf("expected first supplier to be called once, got %d", first.seriesCalls)
	}
	if second.seriesCalls != 0 {
		t.Fatalf("expected second supplier to be skipped, got %d calls", second.seriesCalls)
	}
}

type seriesHelperStubSupplier struct {
	name           string
	seriesSubInfos []supplier.SubInfo
	seriesCalls    int
}

func (s *seriesHelperStubSupplier) CheckAlive() (bool, int64) { return true, 0 }
func (s *seriesHelperStubSupplier) IsAlive() bool             { return true }
func (s *seriesHelperStubSupplier) GetSupplierName() string   { return s.name }
func (s *seriesHelperStubSupplier) OverDailyDownloadLimit() bool {
	return false
}
func (s *seriesHelperStubSupplier) GetLogger() *logrus.Logger { return log_helper.GetLogger4Tester() }
func (s *seriesHelperStubSupplier) GetSubListFromFile4Movie(filePath string) ([]supplier.SubInfo, error) {
	return nil, nil
}
func (s *seriesHelperStubSupplier) GetSubListFromFile4Series(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	s.seriesCalls++
	return append([]supplier.SubInfo(nil), s.seriesSubInfos...), nil
}
func (s *seriesHelperStubSupplier) GetSubListFromFile4Anime(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	return nil, nil
}
