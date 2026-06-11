package pre_download_process

import (
	"errors"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/ifaces"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/local_http_proxy_server"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	subSupplier "github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/assrt"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/moviesubtitles"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/opensubtitles"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/shooter"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/subdl"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/subhd"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/subtitle_best"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/tvsubtitles"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/xunlei"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/notify_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/subtitle_best_api"
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
			if errors.Is(err, subtitle_best_api.ErrAuthKeyNotSet) {
				p.log.Warningln("SubtitleBestCodeProvider.GetCode auth key is not set continue without shared code")
			} else {
				notify_center.Notify.Add("GetSubhdCode", "GetCodeFromWeb,"+err.Error())
				p.log.Errorln("SubtitleBestCodeProvider.GetCode", err)
				p.log.Errorln("Skip Subhd download")
			}
			common2.SubhdCode = ""
		} else {
			codeTime, err := time.Parse("2006-01-02", updateTimeString)
			if err != nil {
				p.log.Errorln("SubtitleBestCodeProvider.GetCode.time.Parse", err)
			} else if codeTime.YearDay() != time.Now().YearDay() {
				p.log.Warningln("SubtitleBestCodeProvider.GetCode, GetCodeTime:", updateTimeString, "NowTime:", time.Now().String(), "Skip")
			} else {
				if code == "" {
					p.log.Warningln("SubtitleBestCodeProvider.GetCode returned empty code continue without shared code")
				} else {
					p.log.Infoln("GetCode", updateTimeString, code)
				}
				common2.SubhdCode = code
			}
		}
	}

	if settings.Get().SpeedDevMode {
		p.SubSupplierHub = subSupplier.NewSubSupplierHub(
			assrt.NewSupplier(p.fileDownloader),
		)
	} else {
		enabledSuppliers := p.buildEnabledSuppliers()
		if len(enabledSuppliers) == 0 {
			p.gError = errors.New("no subtitle suppliers enabled")
			return p
		}
		p.SubSupplierHub = subSupplier.NewSubSupplierHub(enabledSuppliers[0], enabledSuppliers[1:]...)
		for _, englishFallbackSupplier := range p.buildEnglishFallbackSuppliers() {
			switch englishFallbackSupplier.GetSupplierName() {
			case common2.SubSiteSubDL:
				p.SubSupplierHub.AddEnglishFallbackSupplier(englishFallbackSupplier, true, true)
			case common2.SubSiteOpenSubtitles:
				p.SubSupplierHub.AddEnglishFallbackSupplier(englishFallbackSupplier, true, true)
			case common2.SubSiteMovieSubtitles:
				p.SubSupplierHub.AddEnglishFallbackSupplier(englishFallbackSupplier, true, false)
			}
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

func (p *PreDownloadProcess) buildEnabledSuppliers() []ifaces.ISupplier {
	out := make([]ifaces.ISupplier, 0, len(common2.DefaultSubSiteSequence()))
	for _, siteName := range common2.DefaultSubSiteSequence() {
		switch siteName {
		case common2.SubSiteSubtitleBest:
			if settings.Get().SubtitleSources.SubtitleBestSettings.Enabled &&
				settings.Get().SubtitleSources.SubtitleBestSettings.ApiKey != "" {
				out = append(out, subtitle_best.NewSupplier(p.fileDownloader))
			}
		case common2.SubSiteOpenSubtitles:
			if settings.Get().SubtitleSources.OpenSubtitlesSettings.Enabled &&
				settings.Get().SubtitleSources.OpenSubtitlesSettings.ApiKey != "" &&
				settings.Get().SubtitleSources.OpenSubtitlesSettings.Username != "" &&
				settings.Get().SubtitleSources.OpenSubtitlesSettings.Password != "" {
				out = append(out, opensubtitles.NewSupplier(p.fileDownloader))
			}
		case common2.SubSiteTVSubtitles:
			if settings.Get().SubtitleSources.TVsubtitlesSettings.Enabled {
				out = append(out, tvsubtitles.NewSupplier(p.fileDownloader))
			}
		case common2.SubSiteMovieSubtitles:
			if settings.Get().SubtitleSources.MoviesubtitlesSettings.Enabled {
				out = append(out, moviesubtitles.NewSupplier(p.fileDownloader))
			}
		case common2.SubSiteAssrt:
			if settings.Get().SubtitleSources.AssrtSettings.Enabled &&
				settings.Get().SubtitleSources.AssrtSettings.Token != "" {
				out = append(out, assrt.NewSupplier(p.fileDownloader))
			}
		case common2.SubSiteSubDL:
			if settings.Get().SubtitleSources.SubDLSettings.Enabled &&
				settings.Get().SubtitleSources.SubDLSettings.Key != "" {
				out = append(out, subdl.NewSupplier(p.fileDownloader))
			}
		case common2.SubSiteSubHd:
			if pkg.LiteMode() == false && settings.Get().SubtitleSources.SubHDSettings.Enabled {
				out = append(out, subhd.NewSupplier(p.fileDownloader))
			}
		case common2.SubSiteShooter:
			out = append(out, shooter.NewSupplier(p.fileDownloader))
		case common2.SubSiteXunLei:
			out = append(out, xunlei.NewSupplier(p.fileDownloader))
		}
	}
	return out
}

func (p *PreDownloadProcess) buildEnglishFallbackSuppliers() []ifaces.ISupplier {
	out := make([]ifaces.ISupplier, 0, 3)
	if settings.Get().SubtitleSources.OpenSubtitlesSettings.Enabled &&
		settings.Get().SubtitleSources.OpenSubtitlesSettings.ApiKey != "" &&
		settings.Get().SubtitleSources.OpenSubtitlesSettings.Username != "" &&
		settings.Get().SubtitleSources.OpenSubtitlesSettings.Password != "" {
		out = append(out, opensubtitles.NewEnglishSupplier(p.fileDownloader))
	}
	if settings.Get().SubtitleSources.MoviesubtitlesSettings.Enabled {
		out = append(out, moviesubtitles.NewEnglishSupplier(p.fileDownloader))
	}
	if settings.Get().SubtitleSources.SubDLSettings.Enabled &&
		settings.Get().SubtitleSources.SubDLSettings.Key != "" {
		out = append(out, subdl.NewEnglishSupplier(p.fileDownloader))
	}
	return out
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
