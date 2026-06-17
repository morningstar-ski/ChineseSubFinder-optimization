package sub_supplier

import (
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/ifaces"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/assrt"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/moviesubtitles"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/opensubtitles"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/shooter"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/subdl"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/subhd"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/subtitle_best"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/subtitlecat"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/tvsubtitles"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/xunlei"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/backend"
	common2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
)

const (
	RuntimeModeLite    = "lite"
	RuntimeModeBrowser = "browser"
)

func ProbeSupplierStatuses(fileDownloader *file_downloader.FileDownloader, supplierNames []string) backend.ReplyCheckStatus {
	reply := backend.ReplyCheckStatus{
		SubSiteStatus: make([]backend.SiteStatus, 0),
	}
	wanted := makeSupplierFilter(supplierNames)

	appendDynamicStatus(&reply, wanted, xunlei.NewSupplier(fileDownloader), true, RuntimeModeLite)
	appendDynamicStatus(&reply, wanted, shooter.NewSupplier(fileDownloader), true, RuntimeModeLite)

	appendOptionalSupplierStatus(
		&reply,
		wanted,
		common2.SubSiteAssrt,
		RuntimeModeLite,
		settings.Get().SubtitleSources.AssrtSettings.Enabled,
		settings.Get().SubtitleSources.AssrtSettings.Token != "",
		func() ifaces.ISupplier { return assrt.NewSupplier(fileDownloader) },
	)

	appendOptionalSupplierStatus(
		&reply,
		wanted,
		common2.SubSiteSubDL,
		RuntimeModeLite,
		settings.Get().SubtitleSources.SubDLSettings.Enabled,
		settings.Get().SubtitleSources.SubDLSettings.Key != "",
		func() ifaces.ISupplier { return subdl.NewSupplier(fileDownloader) },
	)

	appendOptionalSupplierStatus(
		&reply,
		wanted,
		common2.SubSiteSubtitleBest,
		RuntimeModeLite,
		settings.Get().SubtitleSources.SubtitleBestSettings.Enabled,
		settings.Get().SubtitleSources.SubtitleBestSettings.ApiKey != "",
		func() ifaces.ISupplier { return subtitle_best.NewSupplier(fileDownloader) },
	)

	appendOptionalSupplierStatus(
		&reply,
		wanted,
		common2.SubSiteOpenSubtitles,
		RuntimeModeLite,
		settings.Get().SubtitleSources.OpenSubtitlesSettings.Enabled,
		settings.Get().SubtitleSources.OpenSubtitlesSettings.ApiKey != "" &&
			settings.Get().SubtitleSources.OpenSubtitlesSettings.Username != "" &&
			settings.Get().SubtitleSources.OpenSubtitlesSettings.Password != "",
		func() ifaces.ISupplier { return opensubtitles.NewSupplier(fileDownloader) },
	)

	appendOptionalSupplierStatus(
		&reply,
		wanted,
		common2.SubSiteTVSubtitles,
		RuntimeModeLite,
		settings.Get().SubtitleSources.TVsubtitlesSettings.Enabled,
		true,
		func() ifaces.ISupplier { return tvsubtitles.NewSupplier(fileDownloader) },
	)

	appendOptionalSupplierStatus(
		&reply,
		wanted,
		common2.SubSiteMovieSubtitles,
		RuntimeModeLite,
		settings.Get().SubtitleSources.MoviesubtitlesSettings.Enabled,
		true,
		func() ifaces.ISupplier { return moviesubtitles.NewSupplier(fileDownloader) },
	)

	appendOptionalSupplierStatus(
		&reply,
		wanted,
		common2.SubSiteSubtitleCat,
		RuntimeModeLite,
		true,
		true,
		func() ifaces.ISupplier { return subtitlecat.NewEnglishSupplier(fileDownloader) },
	)

	appendSubHDStatus(&reply, wanted, fileDownloader)

	return reply
}

func appendSubHDStatus(reply *backend.ReplyCheckStatus, wanted map[string]struct{}, fileDownloader *file_downloader.FileDownloader) {
	if wantSupplier(wanted, common2.SubSiteSubHd) == false {
		return
	}

	enabled := settings.Get().SubtitleSources.SubHDSettings.Enabled
	if enabled == false {
		reply.SubSiteStatus = append(reply.SubSiteStatus, newStatus(common2.SubSiteSubHd, false, 0, false, RuntimeModeBrowser, subhd.ReasonDisabled))
		return
	}
	if pkg.LiteMode() {
		reply.SubSiteStatus = append(reply.SubSiteStatus, newStatus(common2.SubSiteSubHd, false, 0, true, RuntimeModeBrowser, subhd.ReasonBrowserRuntimeRequired))
		return
	}

	appendDynamicStatus(reply, wanted, subhd.NewSupplier(fileDownloader), true, RuntimeModeBrowser)
}

func appendOptionalSupplierStatus(reply *backend.ReplyCheckStatus, wanted map[string]struct{}, name string, runtimeMode string, enabled bool, credentialReady bool, supplierFactory func() ifaces.ISupplier) {
	if wantSupplier(wanted, name) == false {
		return
	}
	if enabled == false {
		reply.SubSiteStatus = append(reply.SubSiteStatus, newStatus(name, false, 0, false, runtimeMode, subhd.ReasonDisabled))
		return
	}
	if credentialReady == false {
		reply.SubSiteStatus = append(reply.SubSiteStatus, newStatus(name, false, 0, true, runtimeMode, subhd.ReasonCredentialMissing))
		return
	}

	appendDynamicStatus(reply, wanted, supplierFactory(), true, runtimeMode)
}

func appendDynamicStatus(reply *backend.ReplyCheckStatus, wanted map[string]struct{}, supplier ifaces.ISupplier, enabled bool, runtimeMode string) {
	if wantSupplier(wanted, supplier.GetSupplierName()) == false {
		return
	}
	valid, speed := supplier.CheckAlive()
	reason := ""
	if valid == false {
		reason = subhd.ReasonProbeFailed
	}
	reply.SubSiteStatus = append(reply.SubSiteStatus, newStatus(supplier.GetSupplierName(), valid, speed, enabled, runtimeMode, reason))
}

func newStatus(name string, valid bool, speed int64, enabled bool, runtimeMode string, reason string) backend.SiteStatus {
	return backend.SiteStatus{
		Name:          name,
		Valid:         valid,
		Speed:         speed,
		Enabled:       enabled,
		RuntimeMode:   runtimeMode,
		Reason:        reason,
		LastCheckedAt: time.Now(),
	}
}

func makeSupplierFilter(supplierNames []string) map[string]struct{} {
	if len(supplierNames) == 0 {
		return nil
	}

	wanted := make(map[string]struct{}, len(supplierNames))
	for _, supplierName := range supplierNames {
		wanted[supplierName] = struct{}{}
	}
	return wanted
}

func wantSupplier(wanted map[string]struct{}, supplierName string) bool {
	if wanted == nil {
		return true
	}
	_, ok := wanted[supplierName]
	return ok
}
