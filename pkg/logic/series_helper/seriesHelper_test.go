package series_helper

import (
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/media_info_dealers"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/unit_test_helper"
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
