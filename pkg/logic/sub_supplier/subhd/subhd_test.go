package subhd

import (
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
)

func TestSupplier_OverDailyDownloadLimitUnlimited(t *testing.T) {
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())

	oldLimit := settings.Get().AdvancedSettings.SuppliersSettings.SubHD.DailyDownloadLimit
	settings.Get().AdvancedSettings.SuppliersSettings.SubHD.DailyDownloadLimit = -1
	defer func() {
		settings.Get().AdvancedSettings.SuppliersSettings.SubHD.DailyDownloadLimit = oldLimit
	}()

	supplier := &Supplier{log: log_helper.GetLogger4Tester()}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic for unlimited limit: %v", r)
		}
	}()

	if supplier.OverDailyDownloadLimit() {
		t.Fatal("expected -1 to mean unlimited")
	}
}

func TestSupplier_OverDailyDownloadLimitZeroBlocks(t *testing.T) {
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())

	oldLimit := settings.Get().AdvancedSettings.SuppliersSettings.SubHD.DailyDownloadLimit
	settings.Get().AdvancedSettings.SuppliersSettings.SubHD.DailyDownloadLimit = 0
	defer func() {
		settings.Get().AdvancedSettings.SuppliersSettings.SubHD.DailyDownloadLimit = oldLimit
	}()

	supplier := &Supplier{log: log_helper.GetLogger4Tester()}
	if supplier.OverDailyDownloadLimit() == false {
		t.Fatal("expected 0 to block downloads")
	}
}
