package pre_download_process

import (
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/ifaces"
	subSupplier "github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/sirupsen/logrus"
)

func TestBuildSubSupplierHubOrdersSuppliersByDefaultSequence(t *testing.T) {
	plans := map[string]supplierPlan{
		common.SubSiteOpenSubtitles: {
			siteName:        common.SubSiteOpenSubtitles,
			supplierFactory: newFakeSupplierFactory(common.SubSiteOpenSubtitles),
		},
		common.SubSiteTVSubtitles: {
			siteName:        common.SubSiteTVSubtitles,
			supplierFactory: newFakeSupplierFactory(common.SubSiteTVSubtitles),
		},
		common.SubSiteMovieSubtitles: {
			siteName:        common.SubSiteMovieSubtitles,
			supplierFactory: newFakeSupplierFactory(common.SubSiteMovieSubtitles),
		},
		common.SubSiteXunLei: {
			siteName:        common.SubSiteXunLei,
			supplierFactory: newFakeSupplierFactory(common.SubSiteXunLei),
		},
		common.SubSiteShooter: {
			siteName:        common.SubSiteShooter,
			supplierFactory: newFakeSupplierFactory(common.SubSiteShooter),
		},
		common.SubSiteAssrt: {
			siteName:        common.SubSiteAssrt,
			supplierFactory: newFakeSupplierFactory(common.SubSiteAssrt),
		},
		common.SubSiteSubDL: {
			siteName:        common.SubSiteSubDL,
			supplierFactory: newFakeSupplierFactory(common.SubSiteSubDL),
		},
	}

	hub := buildSubSupplierHub(plans)
	if hub == nil {
		t.Fatal("buildSubSupplierHub() returned nil")
	}

	got := make([]string, 0, len(hub.Suppliers))
	for _, supplier := range hub.Suppliers {
		got = append(got, supplier.GetSupplierName())
	}

	want := []string{
		common.SubSiteAssrt,
		common.SubSiteSubDL,
		common.SubSiteShooter,
		common.SubSiteXunLei,
		common.SubSiteOpenSubtitles,
		common.SubSiteTVSubtitles,
		common.SubSiteMovieSubtitles,
	}

	if len(got) != len(want) {
		t.Fatalf("len(hub.Suppliers) = %d; want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("hub.Suppliers[%d] = %q; want %q", i, got[i], want[i])
		}
	}
}

func TestBuildSubSupplierHubPreservesEnglishFallbacks(t *testing.T) {
	plans := map[string]supplierPlan{
		common.SubSiteSubDL: {
			siteName:        common.SubSiteSubDL,
			supplierFactory: newFakeSupplierFactory(common.SubSiteSubDL),
			addEnglishFallbacks: func(hub *subSupplier.SubSupplierHub) {
				hub.AddEnglishFallbackSupplier(newFakeSupplierFactory("subdl-en")(), true, true)
			},
		},
	}

	hub := buildSubSupplierHub(plans)
	if hub == nil {
		t.Fatal("buildSubSupplierHub() returned nil")
	}
	if hub.HasEnglishFallbackMovieSuppliers() == false {
		t.Fatal("HasEnglishFallbackMovieSuppliers() = false; want true")
	}
	if hub.HasEnglishFallbackSeriesSuppliers() == false {
		t.Fatal("HasEnglishFallbackSeriesSuppliers() = false; want true")
	}
}

func TestBuildSubSupplierHubKeepsSubtitleCatFallbackOutOfPrimarySuppliers(t *testing.T) {
	plans := map[string]supplierPlan{
		common.SubSiteXunLei: {
			siteName:        common.SubSiteXunLei,
			supplierFactory: newFakeSupplierFactory(common.SubSiteXunLei),
			supportMovie:    true,
			supportSeries:   true,
		},
		common.SubSiteSubtitleCat: {
			siteName:        common.SubSiteSubtitleCat,
			supplierFactory: newFakeSupplierFactory(common.SubSiteSubtitleCat),
			skipPrimary:     true,
			addEnglishFallbacks: func(hub *subSupplier.SubSupplierHub) {
				hub.AddEnglishFallbackSupplier(newFakeSupplierFactory(common.SubSiteSubtitleCat)(), true, true)
			},
			addTranslatedFallbacks: func(hub *subSupplier.SubSupplierHub) {
				hub.AddTranslatedFallbackSupplier(newFakeSupplierFactory(common.SubSiteSubtitleCatTrans)(), true, true)
			},
		},
	}

	hub := buildSubSupplierHub(plans)
	if hub == nil {
		t.Fatal("buildSubSupplierHub() returned nil")
	}
	if len(hub.Suppliers) != 1 {
		t.Fatalf("len(hub.Suppliers) = %d; want 1", len(hub.Suppliers))
	}
	if got := hub.Suppliers[0].GetSupplierName(); got != common.SubSiteXunLei {
		t.Fatalf("hub.Suppliers[0] = %q; want %q", got, common.SubSiteXunLei)
	}
	if hub.HasEnglishFallbackMovieSuppliers() == false || hub.HasEnglishFallbackSeriesSuppliers() == false {
		t.Fatal("expected subtitlecat english fallback to be registered")
	}
	if hub.HasTranslatedFallbackMovieSuppliers() == false || hub.HasTranslatedFallbackSeriesSuppliers() == false {
		t.Fatal("expected subtitlecat translated fallback to be registered")
	}
}

type fakeSupplier struct {
	name string
	log  *logrus.Logger
}

func newFakeSupplierFactory(name string) func() ifaces.ISupplier {
	return func() ifaces.ISupplier {
		return &fakeSupplier{
			name: name,
			log:  logrus.New(),
		}
	}
}

func (f *fakeSupplier) CheckAlive() (bool, int64) { return true, 1 }

func (f *fakeSupplier) IsAlive() bool { return true }

func (f *fakeSupplier) GetSupplierName() string { return f.name }

func (f *fakeSupplier) OverDailyDownloadLimit() bool { return false }

func (f *fakeSupplier) GetLogger() *logrus.Logger { return f.log }

func (f *fakeSupplier) GetSubListFromFile4Movie(string) ([]supplier.SubInfo, error) {
	return nil, nil
}

func (f *fakeSupplier) GetSubListFromFile4Series(*series.SeriesInfo) ([]supplier.SubInfo, error) {
	return nil, nil
}

func (f *fakeSupplier) GetSubListFromFile4Anime(*series.SeriesInfo) ([]supplier.SubInfo, error) {
	return nil, nil
}
