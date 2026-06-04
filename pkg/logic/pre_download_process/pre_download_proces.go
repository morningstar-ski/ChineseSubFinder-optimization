package pre_download_process

import (
	"errors"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/local_http_proxy_server"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/moviesubtitles"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/opensubtitles"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/subdl"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/subhd"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/subtitle_best"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/tvsubtitles"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	subSupplier "github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/assrt"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/shooter"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/xunlei"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/notify_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	common2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/url_connectedness_helper"
	"github.com/sirupsen/logrus"
)

type PreDownloadProcess struct {
	stageName      string
	gError         error
	log            *logrus.Logger
	fileDownloader *file_downloader.FileDownloader
	SubSupplierHub *subSupplier.SubSupplierHub
}

func NewPreDownloadProcess(fileDownloader *file_downloader.FileDownloader) *PreDownloadProcess {
	preDownloadProcess := PreDownloadProcess{
		fileDownloader: fileDownloader,
		log:            fileDownloader.Log,
	}
	return &preDownloadProcess
}

func (p *PreDownloadProcess) Init() *PreDownloadProcess {
	if p.gError != nil {
		p.log.Infoln("Skip PreDownloadProcess.Init()")
		return p
	}

	p.stageName = stageNameInit
	defer func() {
		p.log.Infoln("PreDownloadProcess.Init() End")
	}()
	p.log.Infoln("PreDownloadProcess.Init() Start...")

	notify_center.Notify = notify_center.NewNotifyCenter(p.log, settings.Get().DeveloperSettings.BarkServerAddress)
	notify_center.Notify.Clear()

	common2.SubhdCode = ""
	if pkg.LiteMode() == false && settings.Get().SubtitleSources.SubHDSettings.Enabled {
		codeProvider := subhd.NewSubtitleBestCodeProvider(p.fileDownloader)
		updateTimeString, code, err := codeProvider.GetCode()
		if err != nil {
			p.log.Warningln("SubtitleBestCodeProvider.GetCode", err, "continue without shared code")
		} else {
			codeTime, err := time.Parse("2006-01-02", updateTimeString)
			if err != nil {
				p.log.Errorln("SubtitleBestCodeProvider.GetCode.time.Parse", err)
			} else if codeTime.YearDay() != time.Now().YearDay() {
				p.log.Warningln("SubtitleBestCodeProvider.GetCode, GetCodeTime:", updateTimeString, "NowTime:", time.Now().String(), "Skip")
			} else {
				p.log.Infoln("GetCode", updateTimeString, code)
				common2.SubhdCode = code
			}
		}
	}

	if settings.Get().SpeedDevMode {
		p.SubSupplierHub = subSupplier.NewSubSupplierHub(
			assrt.NewSupplier(p.fileDownloader),
		)
	} else {
		p.SubSupplierHub = subSupplier.NewSubSupplierHub(
			xunlei.NewSupplier(p.fileDownloader),
			shooter.NewSupplier(p.fileDownloader),
		)

		if settings.Get().SubtitleSources.AssrtSettings.Enabled &&
			settings.Get().SubtitleSources.AssrtSettings.Token != "" {
			p.SubSupplierHub.AddSubSupplier(assrt.NewSupplier(p.fileDownloader))
		}

		if settings.Get().SubtitleSources.SubDLSettings.Enabled &&
			settings.Get().SubtitleSources.SubDLSettings.Key != "" {
			p.SubSupplierHub.AddSubSupplier(subdl.NewSupplier(p.fileDownloader))
		}

		if settings.Get().SubtitleSources.SubtitleBestSettings.Enabled &&
			settings.Get().SubtitleSources.SubtitleBestSettings.ApiKey != "" {
			p.SubSupplierHub.AddSubSupplier(subtitle_best.NewSupplier(p.fileDownloader))
		}

		if settings.Get().SubtitleSources.OpenSubtitlesSettings.Enabled &&
			settings.Get().SubtitleSources.OpenSubtitlesSettings.ApiKey != "" &&
			settings.Get().SubtitleSources.OpenSubtitlesSettings.Username != "" &&
			settings.Get().SubtitleSources.OpenSubtitlesSettings.Password != "" {
			p.SubSupplierHub.AddSubSupplier(opensubtitles.NewSupplier(p.fileDownloader))
		}

		if settings.Get().SubtitleSources.TVsubtitlesSettings.Enabled {
			p.SubSupplierHub.AddSubSupplier(tvsubtitles.NewSupplier(p.fileDownloader))
		}

		if settings.Get().SubtitleSources.MoviesubtitlesSettings.Enabled {
			p.SubSupplierHub.AddSubSupplier(moviesubtitles.NewSupplier(p.fileDownloader))
		}

		if pkg.LiteMode() == false && settings.Get().SubtitleSources.SubHDSettings.Enabled {
			p.SubSupplierHub.AddSubSupplier(subhd.NewSupplier(p.fileDownloader))
		}
	}

	err := pkg.ClearRodTmpRootFolder()
	if err != nil {
		p.gError = errors.New("ClearRodTmpRootFolder " + err.Error())
		return p
	}

	p.log.Infoln("ClearRodTmpRootFolder Done")
	return p
}

func (p *PreDownloadProcess) Check() *PreDownloadProcess {
	if p.gError != nil {
		p.log.Infoln("Skip PreDownloadProcess.Check()")
		return p
	}

	p.stageName = stageNameCheck
	defer func() {
		p.log.Infoln("PreDownloadProcess.Check() End")
	}()
	p.log.Infoln("PreDownloadProcess.Check() Start...")

	if settings.Get().AdvancedSettings.ProxySettings.UseProxy == false {
		p.log.Infoln("UseHttpProxy = false")
		proxyStatus, proxySpeed, err := url_connectedness_helper.UrlConnectednessTest(url_connectedness_helper.BaiduUrl, "")
		if err != nil {
			p.log.Errorln(errors.New("UrlConnectednessTest Target Site " + url_connectedness_helper.BaiduUrl + ", " + err.Error()))
		} else {
			p.log.Infoln("UrlConnectednessTest Target Site", url_connectedness_helper.BaiduUrl, "Speed:", proxySpeed, "ms,", "Status:", proxyStatus)
		}
	} else {
		p.log.Infoln("UseHttpProxy By:", settings.Get().AdvancedSettings.ProxySettings.UseWhichProxyProtocol)
		proxyStatus, proxySpeed, err := url_connectedness_helper.UrlConnectednessTest(url_connectedness_helper.GoogleUrl, local_http_proxy_server.GetProxyUrl())
		if err != nil {
			p.log.Errorln(errors.New("UrlConnectednessTest Target Site " + url_connectedness_helper.GoogleUrl + ", " + err.Error()))
		} else {
			p.log.Infoln("UrlConnectednessTest Target Site", url_connectedness_helper.GoogleUrl, "Speed:", proxySpeed, "ms,", "Status:", proxyStatus)
		}
	}

	p.SubSupplierHub.CheckSubSiteStatus()

	if len(settings.Get().CommonSettings.MoviePaths) < 1 {
		p.log.Warningln("MoviePaths not set, len == 0")
	}
	if len(settings.Get().CommonSettings.SeriesPaths) < 1 {
		p.log.Warningln("SeriesPaths not set, len == 0")
	}
	for i, path := range settings.Get().CommonSettings.MoviePaths {
		if pkg.IsDir(path) == false {
			p.log.Errorln("MovieFolder not found Index", i, "--", path)
		} else {
			p.log.Infoln("MovieFolder Index", i, "--", path)
		}
	}
	for i, path := range settings.Get().CommonSettings.SeriesPaths {
		if pkg.IsDir(path) == false {
			p.log.Errorln("SeriesPaths not found Index", i, "--", path)
		} else {
			p.log.Infoln("SeriesPaths Index", i, "--", path)
		}
	}

	return p
}

func (p *PreDownloadProcess) Wait() error {
	defer func() {
		p.log.Infoln("PreDownloadProcess.Wait() Done.")
	}()
	if p.gError != nil {
		outErrString := "PreDownloadProcess.Wait() Get Error, " + "stageName:" + p.stageName + " -- " + p.gError.Error()
		p.log.Errorln(outErrString)
		return errors.New(outErrString)
	}
	return nil
}

const (
	stageNameInit  = "Init"
	stageNameCheck = "Check"
)
