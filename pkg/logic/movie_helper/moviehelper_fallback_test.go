package movie_helper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/ifaces"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/sirupsen/logrus"
)

type fakeMovieSupplier struct {
	name      string
	subInfos  []supplier.SubInfo
	callCount *int
}

func (f fakeMovieSupplier) CheckAlive() (bool, int64)    { return true, 1 }
func (f fakeMovieSupplier) IsAlive() bool                { return true }
func (f fakeMovieSupplier) OverDailyDownloadLimit() bool { return false }
func (f fakeMovieSupplier) GetLogger() *logrus.Logger    { return logrus.New() }
func (f fakeMovieSupplier) GetSupplierName() string      { return f.name }
func (f fakeMovieSupplier) GetSubListFromFile4Series(*series.SeriesInfo) ([]supplier.SubInfo, error) {
	return nil, nil
}
func (f fakeMovieSupplier) GetSubListFromFile4Anime(*series.SeriesInfo) ([]supplier.SubInfo, error) {
	return nil, nil
}
func (f fakeMovieSupplier) GetSubListFromFile4Movie(string) ([]supplier.SubInfo, error) {
	if f.callCount != nil {
		*f.callCount++
	}
	return append([]supplier.SubInfo(nil), f.subInfos...), nil
}

var _ ifaces.ISupplier = fakeMovieSupplier{}

func TestOneMovieDlSubInAllSiteStopsAfterUsableProvider(t *testing.T) {
	tmpRoot := t.TempDir()
	videoPath := filepath.Join(tmpRoot, "Movie.2025.1080p.WEB-DL.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}

	firstCalls := 0
	secondCalls := 0
	usableSub := *supplier.NewSubInfo(
		"subdl",
		0,
		"Movie.2025.1080p.WEB-DL",
		language.ChineseSimple,
		"https://example.com/sub.srt",
		0,
		0,
		".srt",
		[]byte("1\n00:00:01,000 --> 00:00:02,000\n你好\n"),
	)

	got := OneMovieDlSubInAllSite(logrus.New(), []ifaces.ISupplier{
		fakeMovieSupplier{name: "subdl", subInfos: []supplier.SubInfo{usableSub}, callCount: &firstCalls},
		fakeMovieSupplier{name: "subhd", subInfos: []supplier.SubInfo{usableSub}, callCount: &secondCalls},
	}, videoPath, 1, true)

	if len(got) != 1 {
		t.Fatalf("OneMovieDlSubInAllSite() len = %d; want 1", len(got))
	}
	if firstCalls != 1 {
		t.Fatalf("first supplier calls = %d; want 1", firstCalls)
	}
	if secondCalls != 0 {
		t.Fatalf("second supplier calls = %d; want 0 after early stop", secondCalls)
	}
}

func TestOneMovieDlSubInAllSiteDoesNotStopOnMismatchedChineseSubtitle(t *testing.T) {
	tmpRoot := t.TempDir()
	videoPath := filepath.Join(tmpRoot, "Movie.2025.1080p.WEB-DL.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o600); err != nil {
		t.Fatal(err)
	}

	firstCalls := 0
	secondCalls := 0
	wrongSub := *supplier.NewSubInfo(
		"assrt",
		0,
		"Different.Movie.2024.1080p.WEB-DL",
		language.ChineseSimple,
		"https://example.com/wrong.srt",
		0,
		0,
		".srt",
		[]byte("1\n00:00:01,000 --> 00:00:02,000\n你好\n"),
	)
	rightSub := *supplier.NewSubInfo(
		"subhd",
		0,
		"Movie.2025.1080p.WEB-DL",
		language.ChineseSimple,
		"https://example.com/right.srt",
		0,
		0,
		".srt",
		[]byte("1\n00:00:01,000 --> 00:00:02,000\n你好\n"),
	)

	got := OneMovieDlSubInAllSite(logrus.New(), []ifaces.ISupplier{
		fakeMovieSupplier{name: "assrt", subInfos: []supplier.SubInfo{wrongSub}, callCount: &firstCalls},
		fakeMovieSupplier{name: "subhd", subInfos: []supplier.SubInfo{rightSub}, callCount: &secondCalls},
	}, videoPath, 1, true)

	if len(got) != 2 {
		t.Fatalf("OneMovieDlSubInAllSite() len = %d; want 2 accumulated candidates", len(got))
	}
	if firstCalls != 1 {
		t.Fatalf("first supplier calls = %d; want 1", firstCalls)
	}
	if secondCalls != 1 {
		t.Fatalf("second supplier calls = %d; want 1 after mismatched subtitle", secondCalls)
	}
}
