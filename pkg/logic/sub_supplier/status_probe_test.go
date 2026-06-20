package sub_supplier

import (
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/subhd"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	backend2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/backend"
	common2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/sirupsen/logrus"
)

func TestAppendSubHDStatusDisabled(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.SubtitleSources.SubHDSettings.Enabled = false

	oldLiteMode := pkg.LiteMode()
	pkg.SetLiteMode(false)
	defer pkg.SetLiteMode(oldLiteMode)

	reply := backend2.ReplyCheckStatus{SubSiteStatus: make([]backend2.SiteStatus, 0)}
	appendSubHDStatus(&reply, nil, nil)

	if len(reply.SubSiteStatus) != 1 {
		t.Fatalf("status count = %d; want 1", len(reply.SubSiteStatus))
	}
	got := reply.SubSiteStatus[0]
	if got.Name != common2.SubSiteSubHd {
		t.Fatalf("status name = %q; want %q", got.Name, common2.SubSiteSubHd)
	}
	if got.Enabled != false {
		t.Fatalf("status enabled = %v; want false", got.Enabled)
	}
	if got.Reason != subhd.ReasonDisabled {
		t.Fatalf("status reason = %q; want %q", got.Reason, subhd.ReasonDisabled)
	}
	if got.RuntimeMode != RuntimeModeBrowser {
		t.Fatalf("status runtime mode = %q; want %q", got.RuntimeMode, RuntimeModeBrowser)
	}
}

func TestAppendSubHDStatusLiteModeRequiresBrowserRuntime(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.SubtitleSources.SubHDSettings.Enabled = true

	oldLiteMode := pkg.LiteMode()
	pkg.SetLiteMode(true)
	defer pkg.SetLiteMode(oldLiteMode)

	reply := backend2.ReplyCheckStatus{SubSiteStatus: make([]backend2.SiteStatus, 0)}
	appendSubHDStatus(&reply, nil, nil)

	if len(reply.SubSiteStatus) != 1 {
		t.Fatalf("status count = %d; want 1", len(reply.SubSiteStatus))
	}
	got := reply.SubSiteStatus[0]
	if got.Name != common2.SubSiteSubHd {
		t.Fatalf("status name = %q; want %q", got.Name, common2.SubSiteSubHd)
	}
	if got.Enabled != true {
		t.Fatalf("status enabled = %v; want true", got.Enabled)
	}
	if got.Reason != subhd.ReasonBrowserRuntimeRequired {
		t.Fatalf("status reason = %q; want %q", got.Reason, subhd.ReasonBrowserRuntimeRequired)
	}
	if got.RuntimeMode != RuntimeModeBrowser {
		t.Fatalf("status runtime mode = %q; want %q", got.RuntimeMode, RuntimeModeBrowser)
	}
}

func TestProbeSupplierStatusesDisabledSupplierDoesNotMutateTopic(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.Topic = 3
	cfg.SubtitleSources.AssrtSettings.Enabled = false
	cfg.SubtitleSources.AssrtSettings.Token = ""

	oldLiteMode := pkg.LiteMode()
	pkg.SetLiteMode(false)
	defer pkg.SetLiteMode(oldLiteMode)

	fileDownloader := &file_downloader.FileDownloader{Log: logrus.New()}
	reply := ProbeSupplierStatuses(fileDownloader, []string{common2.SubSiteAssrt})

	if len(reply.SubSiteStatus) != 1 {
		t.Fatalf("status count = %d; want 1", len(reply.SubSiteStatus))
	}
	got := reply.SubSiteStatus[0]
	if got.Name != common2.SubSiteAssrt {
		t.Fatalf("status name = %q; want %q", got.Name, common2.SubSiteAssrt)
	}
	if got.Enabled != false {
		t.Fatalf("status enabled = %v; want false", got.Enabled)
	}
	if cfg.AdvancedSettings.Topic != 3 {
		t.Fatalf("topic mutated to %d; want 3", cfg.AdvancedSettings.Topic)
	}
}

func TestProbeSupplierStatusesSubtitleBestReportsCredentialMissingWhenEnabledWithoutKey(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.SubtitleSources.SubtitleBestSettings.Enabled = true
	cfg.SubtitleSources.SubtitleBestSettings.ApiKey = ""

	oldLiteMode := pkg.LiteMode()
	pkg.SetLiteMode(false)
	defer pkg.SetLiteMode(oldLiteMode)

	fileDownloader := &file_downloader.FileDownloader{Log: logrus.New()}
	reply := ProbeSupplierStatuses(fileDownloader, []string{common2.SubSiteSubtitleBest})

	if len(reply.SubSiteStatus) != 1 {
		t.Fatalf("status count = %d; want 1", len(reply.SubSiteStatus))
	}

	got := reply.SubSiteStatus[0]
	if got.Name != common2.SubSiteSubtitleBest {
		t.Fatalf("status name = %q; want %q", got.Name, common2.SubSiteSubtitleBest)
	}
	if got.Enabled != true {
		t.Fatalf("status enabled = %v; want true", got.Enabled)
	}
	if got.Valid != false {
		t.Fatalf("status valid = %v; want false", got.Valid)
	}
	if got.Reason != subhd.ReasonCredentialMissing {
		t.Fatalf("status reason = %q; want %q", got.Reason, subhd.ReasonCredentialMissing)
	}
	if got.RuntimeMode != RuntimeModeLite {
		t.Fatalf("status runtime mode = %q; want %q", got.RuntimeMode, RuntimeModeLite)
	}
}
