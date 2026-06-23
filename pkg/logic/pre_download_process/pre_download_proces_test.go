package pre_download_process

import (
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/ifaces"
	subSupplier "github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
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
			skipPrimary:     true,
			addEnglishFallbacks: func(hub *subSupplier.SubSupplierHub) {
				hub.AddEnglishFallbackSupplier(newFakeSupplierFactory("tvsubtitles-en")(), false, true)
			},
		},
		common.SubSiteMovieSubtitles: {
			siteName:        common.SubSiteMovieSubtitles,
			supplierFactory: newFakeSupplierFactory(common.SubSiteMovieSubtitles),
			skipPrimary:     true,
			addEnglishFallbacks: func(hub *subSupplier.SubSupplierHub) {
				hub.AddEnglishFallbackSupplier(newFakeSupplierFactory("moviesubtitles-en")(), true, false)
			},
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
			skipPrimary:     true,
			addEnglishFallbacks: func(hub *subSupplier.SubSupplierHub) {
				hub.AddEnglishFallbackSupplier(newFakeSupplierFactory("subdl-en")(), true, true)
			},
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
		common.SubSiteShooter,
		common.SubSiteXunLei,
		common.SubSiteOpenSubtitles,
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

func TestBuildSubSupplierHubKeepsSubtitleCatTranslatedFallbackExplicit(t *testing.T) {
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
		},
	}

	hub := buildSubSupplierHub(plans)
	if hub == nil {
		t.Fatal("buildSubSupplierHub() returned nil")
	}
	if hub.HasEnglishFallbackMovieSuppliers() == false || hub.HasEnglishFallbackSeriesSuppliers() == false {
		t.Fatal("expected subtitlecat english fallback to be registered")
	}
	if hub.HasTranslatedFallbackMovieSuppliers() || hub.HasTranslatedFallbackSeriesSuppliers() {
		t.Fatal("translated chinese fallback should require an explicit switch")
	}
}

func TestBuildSubSupplierHubKeepsEnglishOnlySuppliersOutOfPrimarySuppliers(t *testing.T) {
	plans := map[string]supplierPlan{
		common.SubSiteAssrt: {
			siteName:        common.SubSiteAssrt,
			supplierFactory: newFakeSupplierFactory(common.SubSiteAssrt),
			supportMovie:    true,
			supportSeries:   true,
		},
		common.SubSiteSubDL: {
			siteName:        common.SubSiteSubDL,
			supplierFactory: newFakeSupplierFactory(common.SubSiteSubDL),
			supportMovie:    true,
			supportSeries:   true,
			skipPrimary:     true,
			addEnglishFallbacks: func(hub *subSupplier.SubSupplierHub) {
				hub.AddEnglishFallbackSupplier(newFakeSupplierFactory("subdl-en")(), true, true)
			},
		},
		common.SubSiteTVSubtitles: {
			siteName:        common.SubSiteTVSubtitles,
			supplierFactory: newFakeSupplierFactory(common.SubSiteTVSubtitles),
			supportMovie:    false,
			supportSeries:   true,
			skipPrimary:     true,
			addEnglishFallbacks: func(hub *subSupplier.SubSupplierHub) {
				hub.AddEnglishFallbackSupplier(newFakeSupplierFactory("tvsubtitles-en")(), false, true)
			},
		},
		common.SubSiteMovieSubtitles: {
			siteName:        common.SubSiteMovieSubtitles,
			supplierFactory: newFakeSupplierFactory(common.SubSiteMovieSubtitles),
			supportMovie:    true,
			supportSeries:   false,
			skipPrimary:     true,
			addEnglishFallbacks: func(hub *subSupplier.SubSupplierHub) {
				hub.AddEnglishFallbackSupplier(newFakeSupplierFactory("moviesubtitles-en")(), true, false)
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
	if got := hub.Suppliers[0].GetSupplierName(); got != common.SubSiteAssrt {
		t.Fatalf("hub.Suppliers[0] = %q; want %q", got, common.SubSiteAssrt)
	}
	if hub.HasEnglishFallbackMovieSuppliers() == false {
		t.Fatal("expected english fallback movie suppliers to be registered")
	}
	if hub.HasEnglishFallbackSeriesSuppliers() == false {
		t.Fatal("expected english fallback series suppliers to be registered")
	}
}

func TestCollectSupplierPlansKeepsSubDLEnglishFallbackButDropsTVSubtitlesDefaultFallback(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.SubtitleSources.SubDLSettings.Enabled = true
	cfg.SubtitleSources.SubDLSettings.Key = "subdl-key"
	cfg.SubtitleSources.TVsubtitlesSettings.Enabled = true

	plans := collectSupplierPlans(nil)

	subdlPlan, ok := plans[common.SubSiteSubDL]
	if ok == false {
		t.Fatal("expected subdl plan to be present")
	}
	if subdlPlan.skipPrimary == false {
		t.Fatal("expected subdl to stay out of the primary chinese chain")
	}
	if subdlPlan.addEnglishFallbacks == nil {
		t.Fatal("expected subdl to stay in the english fallback chain")
	}

	tvsubtitlesPlan, ok := plans[common.SubSiteTVSubtitles]
	if ok == false {
		t.Fatal("expected tvsubtitles plan to be present when enabled")
	}
	if tvsubtitlesPlan.skipPrimary == false {
		t.Fatal("expected tvsubtitles to stay out of the primary chinese chain")
	}
	if tvsubtitlesPlan.addEnglishFallbacks != nil {
		t.Fatal("expected tvsubtitles to stay out of the default english fallback chain")
	}
}

func TestCollectSupplierPlansKeepsCurrentRuntimeRouteRoles(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	oldLiteMode := pkg.LiteMode()
	pkg.SetLiteMode(false)
	defer pkg.SetLiteMode(oldLiteMode)

	cfg := settings.Get()
	cfg.SubtitleSources.SubDLSettings.Enabled = true
	cfg.SubtitleSources.SubDLSettings.Key = "subdl-key"
	cfg.SubtitleSources.OpenSubtitlesSettings.Enabled = true
	cfg.SubtitleSources.OpenSubtitlesSettings.ApiKey = "opensubtitles-key"
	cfg.SubtitleSources.OpenSubtitlesSettings.Username = "user"
	cfg.SubtitleSources.OpenSubtitlesSettings.Password = "pass"
	cfg.SubtitleSources.TVsubtitlesSettings.Enabled = true
	cfg.SubtitleSources.MoviesubtitlesSettings.Enabled = true
	cfg.SubtitleSources.SubHDSettings.Enabled = true
	cfg.SubtitleSources.SubtitleCatSettings.EnableTranslatedChineseFallback = false

	plans := collectSupplierPlans(nil)

	opensubtitlesPlan, ok := plans[common.SubSiteOpenSubtitles]
	if ok == false {
		t.Fatal("expected opensubtitles plan to be present")
	}
	if opensubtitlesPlan.skipPrimary {
		t.Fatal("expected opensubtitles to stay in the primary chinese chain")
	}
	if opensubtitlesPlan.addEnglishFallbacks == nil {
		t.Fatal("expected opensubtitles to stay in the english fallback chain")
	}

	subhdPlan, ok := plans[common.SubSiteSubHd]
	if ok == false {
		t.Fatal("expected subhd plan to be present in non-lite mode")
	}
	if subhdPlan.skipPrimary {
		t.Fatal("expected subhd to stay in the primary chinese chain")
	}
	if subhdPlan.addEnglishFallbacks != nil {
		t.Fatal("expected subhd to stay out of the default english fallback chain")
	}

	subtitlecatPlan, ok := plans[common.SubSiteSubtitleCat]
	if ok == false {
		t.Fatal("expected subtitlecat plan to be present")
	}
	if subtitlecatPlan.skipPrimary == false {
		t.Fatal("expected subtitlecat to stay out of the primary chinese chain")
	}
	if subtitlecatPlan.addEnglishFallbacks == nil {
		t.Fatal("expected subtitlecat english fallback to stay enabled by default")
	}
	if subtitlecatPlan.addTranslatedFallbacks != nil {
		t.Fatal("expected subtitlecat translated fallback to require an explicit switch")
	}

	moviesubtitlesPlan, ok := plans[common.SubSiteMovieSubtitles]
	if ok == false {
		t.Fatal("expected moviesubtitles plan to be present")
	}
	if moviesubtitlesPlan.skipPrimary == false {
		t.Fatal("expected moviesubtitles to stay out of the primary chinese chain")
	}
	if moviesubtitlesPlan.addEnglishFallbacks == nil {
		t.Fatal("expected moviesubtitles to stay in the english fallback chain")
	}
}

func TestOrderPrimarySupplierPlansUsesDedicatedPrimarySequence(t *testing.T) {
	plans := map[string]supplierPlan{
		common.SubSiteOpenSubtitles: {
			siteName:        common.SubSiteOpenSubtitles,
			supplierFactory: newFakeSupplierFactory(common.SubSiteOpenSubtitles),
		},
		common.SubSiteSubDL: {
			siteName:        common.SubSiteSubDL,
			supplierFactory: newFakeSupplierFactory(common.SubSiteSubDL),
			skipPrimary:     true,
		},
		common.SubSiteAssrt: {
			siteName:        common.SubSiteAssrt,
			supplierFactory: newFakeSupplierFactory(common.SubSiteAssrt),
		},
		common.SubSiteSubHd: {
			siteName:        common.SubSiteSubHd,
			supplierFactory: newFakeSupplierFactory(common.SubSiteSubHd),
		},
	}

	got := orderPrimarySupplierPlans(plans)
	want := []string{
		common.SubSiteAssrt,
		common.SubSiteSubHd,
		common.SubSiteOpenSubtitles,
	}

	if len(got) != len(want) {
		t.Fatalf("len(orderPrimarySupplierPlans) = %d; want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].siteName != want[i] {
			t.Fatalf("orderPrimarySupplierPlans[%d] = %q; want %q", i, got[i].siteName, want[i])
		}
	}
}

func TestOrderEnglishFallbackSupplierPlansUsesDedicatedFallbackSequence(t *testing.T) {
	plans := map[string]supplierPlan{
		common.SubSiteSubtitleCat: {
			siteName:        common.SubSiteSubtitleCat,
			supplierFactory: newFakeSupplierFactory(common.SubSiteSubtitleCat),
			addEnglishFallbacks: func(hub *subSupplier.SubSupplierHub) {
				hub.AddEnglishFallbackSupplier(newFakeSupplierFactory(common.SubSiteSubtitleCat)(), true, true)
			},
		},
		common.SubSiteMovieSubtitles: {
			siteName:        common.SubSiteMovieSubtitles,
			supplierFactory: newFakeSupplierFactory(common.SubSiteMovieSubtitles),
			addEnglishFallbacks: func(hub *subSupplier.SubSupplierHub) {
				hub.AddEnglishFallbackSupplier(newFakeSupplierFactory(common.SubSiteMovieSubtitles)(), true, false)
			},
		},
		common.SubSiteOpenSubtitles: {
			siteName:        common.SubSiteOpenSubtitles,
			supplierFactory: newFakeSupplierFactory(common.SubSiteOpenSubtitles),
			addEnglishFallbacks: func(hub *subSupplier.SubSupplierHub) {
				hub.AddEnglishFallbackSupplier(newFakeSupplierFactory(common.SubSiteOpenSubtitles)(), true, true)
			},
		},
		common.SubSiteSubDL: {
			siteName:        common.SubSiteSubDL,
			supplierFactory: newFakeSupplierFactory(common.SubSiteSubDL),
			addEnglishFallbacks: func(hub *subSupplier.SubSupplierHub) {
				hub.AddEnglishFallbackSupplier(newFakeSupplierFactory(common.SubSiteSubDL)(), true, true)
			},
		},
	}

	got := orderEnglishFallbackSupplierPlans(plans)
	want := []string{
		common.SubSiteOpenSubtitles,
		common.SubSiteSubDL,
		common.SubSiteSubtitleCat,
		common.SubSiteMovieSubtitles,
	}

	if len(got) != len(want) {
		t.Fatalf("len(orderEnglishFallbackSupplierPlans) = %d; want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].siteName != want[i] {
			t.Fatalf("orderEnglishFallbackSupplierPlans[%d] = %q; want %q", i, got[i].siteName, want[i])
		}
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
