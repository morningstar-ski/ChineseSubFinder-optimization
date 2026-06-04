package search

import (
	"path/filepath"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/unit_test_helper"
)

func TestSearchSeriesAllEpsAndSubtitles(t *testing.T) {

	seriesDir := unit_test_helper.SkipIfTestDataResourceAbsent(t, []string{"series", "Pantheon"}, 4, false)
	seasonInfo, err := SeriesAllEpsAndSubtitles(log_helper.GetLogger4Tester(), filepath.Clean(seriesDir))
	if err != nil {
		t.Fatal(err)
	}
	println(seasonInfo.Name)
}
