package series_helper

import (
	"path/filepath"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/ifaces"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/sirupsen/logrus"
)

type fakeSeriesSupplier struct {
	name      string
	subInfos  []supplier.SubInfo
	callCount *int
}

func (f fakeSeriesSupplier) CheckAlive() (bool, int64)    { return true, 1 }
func (f fakeSeriesSupplier) IsAlive() bool                { return true }
func (f fakeSeriesSupplier) OverDailyDownloadLimit() bool { return false }
func (f fakeSeriesSupplier) GetLogger() *logrus.Logger    { return logrus.New() }
func (f fakeSeriesSupplier) GetSupplierName() string      { return f.name }
func (f fakeSeriesSupplier) GetSubListFromFile4Movie(string) ([]supplier.SubInfo, error) {
	return nil, nil
}
func (f fakeSeriesSupplier) GetSubListFromFile4Anime(*series.SeriesInfo) ([]supplier.SubInfo, error) {
	return nil, nil
}
func (f fakeSeriesSupplier) GetSubListFromFile4Series(*series.SeriesInfo) ([]supplier.SubInfo, error) {
	if f.callCount != nil {
		*f.callCount++
	}
	return append([]supplier.SubInfo(nil), f.subInfos...), nil
}

var _ ifaces.ISupplier = fakeSeriesSupplier{}

func TestDownloadSubtitleInAllSiteByOneSeriesStopsWhenAllEpisodesCovered(t *testing.T) {
	seriesInfo := &series.SeriesInfo{
		Name:    "Show",
		DirPath: filepath.Join("C:\\", "Media", "Show"),
		NeedDlEpsKeyList: map[string]series.EpisodeInfo{
			pkg.GetEpisodeKeyName(1, 1): {Season: 1, Episode: 1, FileFullPath: filepath.Join("C:\\", "Media", "Show", "Show.S01E01.1080p.WEB-DL.mkv")},
			pkg.GetEpisodeKeyName(1, 2): {Season: 1, Episode: 2, FileFullPath: filepath.Join("C:\\", "Media", "Show", "Show.S01E02.1080p.WEB-DL.mkv")},
		},
	}

	firstCalls := 0
	secondCalls := 0
	episode1 := *supplier.NewSubInfo("subtitle_best", 0, "Show.S01E01.1080p.WEB-DL", language.ChineseSimple, "https://example.com/e01.srt", 0, 0, ".srt", []byte("1\n00:00:01,000 --> 00:00:02,000\n你好\n"))
	episode1.Season = 1
	episode1.Episode = 1
	episode2 := *supplier.NewSubInfo("subtitle_best", 0, "Show.S01E02.1080p.WEB-DL", language.ChineseSimple, "https://example.com/e02.srt", 0, 0, ".srt", []byte("1\n00:00:01,000 --> 00:00:02,000\n世界\n"))
	episode2.Season = 1
	episode2.Episode = 2

	got := DownloadSubtitleInAllSiteByOneSeries(logrus.New(), []ifaces.ISupplier{
		fakeSeriesSupplier{name: "subtitle_best", subInfos: []supplier.SubInfo{episode1, episode2}, callCount: &firstCalls},
		fakeSeriesSupplier{name: "subhd", subInfos: []supplier.SubInfo{episode1}, callCount: &secondCalls},
	}, seriesInfo, 1, true)

	if len(got) != 2 {
		t.Fatalf("DownloadSubtitleInAllSiteByOneSeries() len = %d; want 2", len(got))
	}
	if firstCalls != 1 {
		t.Fatalf("first supplier calls = %d; want 1", firstCalls)
	}
	if secondCalls != 0 {
		t.Fatalf("second supplier calls = %d; want 0 after full coverage", secondCalls)
	}
}

func TestDownloadSubtitleInAllSiteByOneSeriesDoesNotStopOnMismatchedChineseSubtitle(t *testing.T) {
	seriesInfo := &series.SeriesInfo{
		Name:    "Show",
		DirPath: filepath.Join("C:\\", "Media", "Show"),
		NeedDlEpsKeyList: map[string]series.EpisodeInfo{
			pkg.GetEpisodeKeyName(1, 1): {Season: 1, Episode: 1, FileFullPath: filepath.Join("C:\\", "Media", "Show", "Show.S01E01.1080p.WEB-DL.mkv")},
		},
	}

	firstCalls := 0
	secondCalls := 0
	wrongEpisode := *supplier.NewSubInfo("assrt", 0, "Different.Show.S01E09.1080p.WEB-DL", language.ChineseSimple, "https://example.com/wrong.srt", 0, 0, ".srt", []byte("1\n00:00:01,000 --> 00:00:02,000\n你好\n"))
	wrongEpisode.Season = 1
	wrongEpisode.Episode = 1
	rightEpisode := *supplier.NewSubInfo("subhd", 0, "Show.S01E01.1080p.WEB-DL", language.ChineseSimple, "https://example.com/right.srt", 0, 0, ".srt", []byte("1\n00:00:01,000 --> 00:00:02,000\n你好\n"))
	rightEpisode.Season = 1
	rightEpisode.Episode = 1

	got := DownloadSubtitleInAllSiteByOneSeries(logrus.New(), []ifaces.ISupplier{
		fakeSeriesSupplier{name: "assrt", subInfos: []supplier.SubInfo{wrongEpisode}, callCount: &firstCalls},
		fakeSeriesSupplier{name: "subhd", subInfos: []supplier.SubInfo{rightEpisode}, callCount: &secondCalls},
	}, seriesInfo, 1, true)

	if len(got) != 2 {
		t.Fatalf("DownloadSubtitleInAllSiteByOneSeries() len = %d; want 2 accumulated candidates", len(got))
	}
	if firstCalls != 1 {
		t.Fatalf("first supplier calls = %d; want 1", firstCalls)
	}
	if secondCalls != 1 {
		t.Fatalf("second supplier calls = %d; want 1 after mismatched subtitle", secondCalls)
	}
}
