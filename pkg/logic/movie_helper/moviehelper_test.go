package movie_helper

import (
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/ifaces"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/sirupsen/logrus"
)

func TestSkipChineseMovie(t *testing.T) {
	//type args struct {
	//	videoFullPath string
	//}
	//tests := []struct {
	//	name    string
	//	args    args
	//	want    bool
	//	wantErr bool
	//}{
	//	{name: "00", args: args{
	//		videoFullPath: "smb://192.168.50.252/电影/Texas Chainsaw Massacre (2022)/Texas Chainsaw Massacre (2022) WEBDL-1080p.mkv",
	//	}, want: false, wantErr: false},
	//}
	//for _, tt := range tests {
	//	t.Run(tt.name, func(t *testing.T) {
	//		got, err := SkipChineseMovie(tt.args.videoFullPath)
	//		if (err != nil) != tt.wantErr {
	//			t.Errorf("SkipChineseMovie() error = %v, wantErr %v", err, tt.wantErr)
	//			return
	//		}
	//		if got != tt.want {
	//			t.Errorf("SkipChineseMovie() got = %v, want %v", got, tt.want)
	//		}
	//	})
	//}
}

func TestOneMovieDlSubInAllSiteStopsAfterFirstSupplierWithResults(t *testing.T) {
	first := &movieHelperStubSupplier{
		name: "first",
		movieSubInfos: []supplier.SubInfo{
			{Name: "movie.srt", Ext: ".srt"},
		},
	}
	second := &movieHelperStubSupplier{
		name: "second",
		movieSubInfos: []supplier.SubInfo{
			{Name: "movie-2.srt", Ext: ".srt"},
		},
	}

	got := OneMovieDlSubInAllSite(log_helper.GetLogger4Tester(), []ifaces.ISupplier{first, second}, "C:\\Media\\movie.mkv", 1)
	if len(got) != 1 {
		t.Fatalf("expected 1 subtitle from first supplier, got %d", len(got))
	}
	if first.movieCalls != 1 {
		t.Fatalf("expected first supplier to be called once, got %d", first.movieCalls)
	}
	if second.movieCalls != 0 {
		t.Fatalf("expected second supplier to be skipped, got %d calls", second.movieCalls)
	}
}

type movieHelperStubSupplier struct {
	name          string
	movieSubInfos []supplier.SubInfo
	movieCalls    int
}

func (s *movieHelperStubSupplier) CheckAlive() (bool, int64) { return true, 0 }
func (s *movieHelperStubSupplier) IsAlive() bool             { return true }
func (s *movieHelperStubSupplier) GetSupplierName() string   { return s.name }
func (s *movieHelperStubSupplier) OverDailyDownloadLimit() bool {
	return false
}
func (s *movieHelperStubSupplier) GetLogger() *logrus.Logger { return log_helper.GetLogger4Tester() }
func (s *movieHelperStubSupplier) GetSubListFromFile4Movie(filePath string) ([]supplier.SubInfo, error) {
	s.movieCalls++
	return append([]supplier.SubInfo(nil), s.movieSubInfos...), nil
}
func (s *movieHelperStubSupplier) GetSubListFromFile4Series(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	return nil, nil
}
func (s *movieHelperStubSupplier) GetSubListFromFile4Anime(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	return nil, nil
}
