package sub_supplier

import (
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	backend2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/backend"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/sirupsen/logrus"
)

func TestSubSupplierHubSeparatesMovieAndSeriesSuppliers(t *testing.T) {
	movieOnly := &hubStubSupplier{name: "movie-only"}
	seriesOnly := &hubStubSupplier{name: "series-only"}
	both := &hubStubSupplier{name: "both"}

	hub := NewSubSupplierHubWithCapabilities(movieOnly, true, false)
	hub.AddSubSupplierWithCapability(seriesOnly, false, true)
	hub.AddSubSupplierWithCapability(both, true, true)

	seriesInfo := &series.SeriesInfo{
		DirPath: "C:\\Media\\Series",
		NeedDlEpsKeyList: map[string]series.EpisodeInfo{
			pkg.GetEpisodeKeyName(1, 1): {Season: 1, Episode: 1},
		},
	}

	_, err := hub.DownloadSub4Movie("C:\\Media\\Movie\\sample.mkv", 1)
	if err != nil {
		t.Fatalf("DownloadSub4Movie() error = %v", err)
	}
	if movieOnly.movieCalls != 1 {
		t.Fatalf("movie-only supplier movieCalls = %d; want 1", movieOnly.movieCalls)
	}
	if seriesOnly.movieCalls != 0 {
		t.Fatalf("series-only supplier movieCalls = %d; want 0", seriesOnly.movieCalls)
	}
	if both.movieCalls != 1 {
		t.Fatalf("both supplier movieCalls = %d; want 1", both.movieCalls)
	}

	_, err = hub.DownloadSub4Series("C:\\Media\\Series", seriesInfo, 1)
	if err != nil {
		t.Fatalf("DownloadSub4Series() error = %v", err)
	}
	if movieOnly.seriesCalls != 0 {
		t.Fatalf("movie-only supplier seriesCalls = %d; want 0", movieOnly.seriesCalls)
	}
	if seriesOnly.seriesCalls != 1 {
		t.Fatalf("series-only supplier seriesCalls = %d; want 1", seriesOnly.seriesCalls)
	}
	if both.seriesCalls != 1 {
		t.Fatalf("both supplier seriesCalls = %d; want 1", both.seriesCalls)
	}
}

func TestCheckSubSiteStatusKeepsSupplierAfterTransientProbeFailure(t *testing.T) {
	healthy := &hubStubSupplier{name: "healthy", alive: true}
	flaky := &hubStubSupplier{name: "flaky", alive: false}
	hub := NewSubSupplierHubWithCapabilities(healthy, true, true)
	hub.AddSubSupplierWithCapability(flaky, true, true)

	reply := hub.CheckSubSiteStatus()

	if len(reply.SubSiteStatus) != 2 {
		t.Fatalf("status count = %d; want 2", len(reply.SubSiteStatus))
	}
	if len(hub.Suppliers) != 2 {
		t.Fatalf("supplier count after check = %d; want 2", len(hub.Suppliers))
	}
	if len(hub.movieSuppliers) != 2 {
		t.Fatalf("movie supplier count after check = %d; want 2", len(hub.movieSuppliers))
	}
	if len(hub.seriesSuppliers) != 2 {
		t.Fatalf("series supplier count after check = %d; want 2", len(hub.seriesSuppliers))
	}
	if !containsStatus(reply.SubSiteStatus, "flaky", false) {
		t.Fatalf("expected flaky supplier status to be reported as invalid without being removed")
	}
}

func containsStatus(items []backend2.SiteStatus, name string, valid bool) bool {
	for _, item := range items {
		if item.Name == name && item.Valid == valid {
			return true
		}
	}
	return false
}

type hubStubSupplier struct {
	name        string
	movieCalls  int
	seriesCalls int
	alive       bool
	overLimit   bool
}

func (s *hubStubSupplier) CheckAlive() (bool, int64) { return s.alive, 0 }
func (s *hubStubSupplier) IsAlive() bool             { return s.alive }
func (s *hubStubSupplier) GetSupplierName() string   { return s.name }
func (s *hubStubSupplier) OverDailyDownloadLimit() bool {
	return s.overLimit
}
func (s *hubStubSupplier) GetLogger() *logrus.Logger { return log_helper.GetLogger4Tester() }
func (s *hubStubSupplier) GetSubListFromFile4Movie(filePath string) ([]supplier.SubInfo, error) {
	s.movieCalls++
	return nil, nil
}
func (s *hubStubSupplier) GetSubListFromFile4Series(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	s.seriesCalls++
	return nil, nil
}
func (s *hubStubSupplier) GetSubListFromFile4Anime(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	return nil, nil
}
