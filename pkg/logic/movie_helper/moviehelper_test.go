package movie_helper

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/ifaces"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/imdb_helper"
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

func TestOneMovieDlSubInAllSiteCollectsLaterSuppliersAfterInitialResults(t *testing.T) {
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
	if len(got) != 2 {
		t.Fatalf("expected subtitles from both suppliers, got %d", len(got))
	}
	if first.movieCalls != 1 {
		t.Fatalf("expected first supplier to be called once, got %d", first.movieCalls)
	}
	if second.movieCalls != 1 {
		t.Fatalf("expected second supplier to be called once, got %d calls", second.movieCalls)
	}
}

func TestMovieNeedDlSubFallsBackWithoutMetadataForRecentBDMV(t *testing.T) {
	root := t.TempDir()
	movieDir := filepath.Join(root, "Police Academy 4 (1987)")
	if err := os.MkdirAll(filepath.Join(movieDir, "CERTIFICATE"), 0o755); err != nil {
		t.Fatalf("MkdirAll CERTIFICATE error = %v", err)
	}
	if err := os.MkdirAll(filepath.Join(movieDir, "BDMV"), 0o755); err != nil {
		t.Fatalf("MkdirAll BDMV error = %v", err)
	}

	idBDMVPath := filepath.Join(movieDir, "CERTIFICATE", "id.bdmv")
	if err := os.WriteFile(idBDMVPath, []byte("fake-bdmv"), 0o644); err != nil {
		t.Fatalf("WriteFile error = %v", err)
	}

	recent := time.Now().AddDate(0, 0, -1)
	if err := os.Chtimes(idBDMVPath, recent, recent); err != nil {
		t.Fatalf("Chtimes error = %v", err)
	}

	videoPath := filepath.Join(movieDir, "00000.m2ts")
	got, err := MovieNeedDlSub(log_helper.GetLogger4Tester(), videoPath, 90)
	if err != nil {
		t.Fatalf("MovieNeedDlSub() error = %v", err)
	}
	if got != true {
		t.Fatal("MovieNeedDlSub() = false, want true for recent metadata-free BDMV movie")
	}
}

func TestFallbackMovieInfoFromPathParsesTitleAndYear(t *testing.T) {
	got := imdb_helper.FallbackVideoNfoInfoFromPath(filepath.Join("C:\\Media", "Police Academy 4 (1987)", "00000.m2ts"), true)
	if got.Title == "" {
		t.Fatal("FallbackVideoNfoInfoFromPath() title is empty")
	}
	if got.GetYear() != 1987 {
		t.Fatalf("FallbackVideoNfoInfoFromPath() year = %d, want 1987", got.GetYear())
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
