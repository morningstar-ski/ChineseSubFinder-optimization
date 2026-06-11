package sub_supplier

import (
	"path/filepath"
	"sync"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/ifaces"
	movieHelper "github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/movie_helper"
	seriesHelper "github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/series_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/media_info_dealers"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/backend"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/emby"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/sirupsen/logrus"
	"gopkg.in/errgo.v2/fmt/errors"
)

type SubSupplierHub struct {
	log                            *logrus.Logger
	Suppliers                      []ifaces.ISupplier
	englishFallbackMovieSuppliers  []ifaces.ISupplier
	englishFallbackSeriesSuppliers []ifaces.ISupplier
	locker                         sync.Mutex
}

func NewSubSupplierHub(one ifaces.ISupplier, inSuppliers ...ifaces.ISupplier) *SubSupplierHub {
	s := &SubSupplierHub{
		log:                            one.GetLogger(),
		Suppliers:                      make([]ifaces.ISupplier, 0, 1+len(inSuppliers)),
		englishFallbackMovieSuppliers:  make([]ifaces.ISupplier, 0),
		englishFallbackSeriesSuppliers: make([]ifaces.ISupplier, 0),
	}
	s.Suppliers = append(s.Suppliers, one)
	s.Suppliers = append(s.Suppliers, inSuppliers...)
	return s
}

// AddSubSupplier 添加中文字幕阶段供应源。
func (d *SubSupplierHub) AddSubSupplier(one ifaces.ISupplier) {
	d.Suppliers = append(d.Suppliers, one)
}

// AddEnglishFallbackSupplier 添加英文兜底阶段供应源。
func (d *SubSupplierHub) AddEnglishFallbackSupplier(one ifaces.ISupplier, supportMovie bool, supportSeries bool) {
	if supportMovie {
		d.englishFallbackMovieSuppliers = append(d.englishFallbackMovieSuppliers, one)
	}
	if supportSeries {
		d.englishFallbackSeriesSuppliers = append(d.englishFallbackSeriesSuppliers, one)
	}
}

// DelSubSupplier 移除供应源。
func (d *SubSupplierHub) DelSubSupplier(one ifaces.ISupplier) {
	removeByName := func(items []ifaces.ISupplier) []ifaces.ISupplier {
		out := items[:0]
		for _, supplier := range items {
			if supplier.GetSupplierName() == one.GetSupplierName() {
				continue
			}
			out = append(out, supplier)
		}
		return out
	}

	d.Suppliers = removeByName(d.Suppliers)
	d.englishFallbackMovieSuppliers = removeByName(d.englishFallbackMovieSuppliers)
	d.englishFallbackSeriesSuppliers = removeByName(d.englishFallbackSeriesSuppliers)
}

// MovieNeedDlSub 电影是否需要下载字幕。
func (d *SubSupplierHub) MovieNeedDlSub(dealers *media_info_dealers.Dealers, videoFullPath string, forcedScanAndDownloadSub bool) bool {
	if forcedScanAndDownloadSub {
		return true
	}

	var err error
	if settings.Get().AdvancedSettings.ScanLogic.SkipChineseMovie {
		var skip bool
		skip, err = movieHelper.SkipChineseMovie(dealers, videoFullPath)
		if err != nil {
			d.log.Warnln("SkipChineseMovie", videoFullPath, err)
		}
		if skip {
			return false
		}
	}

	needDlSub := false
	if forcedScanAndDownloadSub {
		needDlSub = true
	} else {
		needDlSub, err = movieHelper.MovieNeedDlSub(d.log, videoFullPath, settings.Get().AdvancedSettings.TaskQueue.ExpirationTime)
		if err != nil {
			d.log.Errorln(errors.Newf("MovieNeedDlSub %v %v", videoFullPath, err))
			return false
		}
	}

	return needDlSub
}

// SeriesNeedDlSub 连续剧是否需要下载字幕。
func (d *SubSupplierHub) SeriesNeedDlSub(dealers *media_info_dealers.Dealers, seriesRootPath string, forcedScanAndDownloadSub bool, need2AnalyzeSub bool) (bool, *series.SeriesInfo, error) {
	if forcedScanAndDownloadSub == false && settings.Get().AdvancedSettings.ScanLogic.SkipChineseSeries {
		skip, _, err := seriesHelper.SkipChineseSeries(dealers, seriesRootPath)
		if err != nil {
			d.log.Warnln("SkipChineseMovie", seriesRootPath, err)
		}
		if skip {
			return false, nil, nil
		}
	}

	seriesInfo, err := seriesHelper.ReadSeriesInfoFromDir(
		dealers,
		seriesRootPath,
		settings.Get().AdvancedSettings.TaskQueue.ExpirationTime,
		forcedScanAndDownloadSub,
		need2AnalyzeSub,
	)
	if err != nil {
		return false, nil, errors.Newf("ReadSeriesInfoFromDir %v %v", seriesRootPath, err)
	}

	return true, seriesInfo, nil
}

// SeriesNeedDlSubFromEmby 连续剧是否需要下载字幕。
func (d *SubSupplierHub) SeriesNeedDlSubFromEmby(dealers *media_info_dealers.Dealers, seriesRootPath string, seriesVideoList []emby.EmbyMixInfo, expirationTime int, skipChineseMovie, forcedScanAndDownloadSub bool) (bool, *series.SeriesInfo, error) {
	if skipChineseMovie {
		skip, _, err := seriesHelper.SkipChineseSeries(dealers, seriesRootPath)
		if err != nil {
			d.log.Warnln("SkipChineseMovie", seriesRootPath, err)
		}
		if skip {
			return false, nil, nil
		}
	}

	seriesInfo, err := seriesHelper.ReadSeriesInfoFromEmby(dealers, seriesRootPath, seriesVideoList, expirationTime, forcedScanAndDownloadSub, false)
	if err != nil {
		return false, nil, errors.Newf("ReadSeriesInfoFromDir %v %v", seriesRootPath, err)
	}

	return true, seriesInfo, nil
}

// DownloadSub4Movie 下载中文字幕阶段的电影字幕。
func (d *SubSupplierHub) DownloadSub4Movie(videoFullPath string, index int64) ([]string, error) {
	return d.downloadMovieSubFromSuppliers(videoFullPath, index, d.Suppliers, true)
}

// DownloadEnglishFallbackSub4Movie 下载英文兜底阶段的电影字幕。
func (d *SubSupplierHub) DownloadEnglishFallbackSub4Movie(videoFullPath string, index int64) ([]string, error) {
	return d.downloadMovieSubFromSuppliers(videoFullPath, index, d.englishFallbackMovieSuppliers, false)
}

// DownloadSub4Series 下载中文字幕阶段的剧集字幕。
func (d *SubSupplierHub) DownloadSub4Series(seriesDirPath string, seriesInfo *series.SeriesInfo, index int64) (map[string][]string, error) {
	return d.dlSubFromSeriesInfo(seriesDirPath, index, seriesInfo, d.Suppliers, true)
}

// DownloadEnglishFallbackSub4Series 下载英文兜底阶段的剧集字幕。
func (d *SubSupplierHub) DownloadEnglishFallbackSub4Series(seriesDirPath string, seriesInfo *series.SeriesInfo, index int64) (map[string][]string, error) {
	return d.dlSubFromSeriesInfo(seriesDirPath, index, seriesInfo, d.englishFallbackSeriesSuppliers, false)
}

func (d *SubSupplierHub) HasEnglishFallbackMovieSuppliers() bool {
	return len(d.englishFallbackMovieSuppliers) > 0
}

func (d *SubSupplierHub) HasEnglishFallbackSeriesSuppliers() bool {
	return len(d.englishFallbackSeriesSuppliers) > 0
}

// CheckSubSiteStatus 检查中文字幕阶段供应源状态。
func (d *SubSupplierHub) CheckSubSiteStatus() backend.ReplyCheckStatus {
	outStatus := backend.ReplyCheckStatus{
		SubSiteStatus: make([]backend.SiteStatus, 0),
	}

	var wg sync.WaitGroup
	d.log.Infoln("Check Sub Supplier Start...")
	for _, supplier := range d.Suppliers {
		wg.Add(1)
		go func(supplier ifaces.ISupplier) {
			defer wg.Done()

			bAlive, speed := supplier.CheckAlive()
			if bAlive == false {
				d.log.Warningln(supplier.GetSupplierName(), "Check Alive = false")
			} else {
				d.log.Infoln(supplier.GetSupplierName(), "Check Alive = true, Speed =", speed, "ms")
			}

			d.locker.Lock()
			outStatus.SubSiteStatus = append(outStatus.SubSiteStatus, backend.SiteStatus{
				Name:  supplier.GetSupplierName(),
				Valid: bAlive,
				Speed: speed,
			})
			d.locker.Unlock()
		}(supplier)
	}
	wg.Wait()

	suppliersLen := len(d.Suppliers)
	for i := 0; i < suppliersLen; {
		if d.Suppliers[i].IsAlive() == false || d.Suppliers[i].OverDailyDownloadLimit() {
			d.DelSubSupplier(d.Suppliers[i])
			suppliersLen = len(d.Suppliers)
			i = 0
			continue
		}
		i++
	}

	for _, supplier := range d.Suppliers {
		if supplier.IsAlive() {
			d.log.Infoln("Alive Supplier:", supplier.GetSupplierName())
		}
	}

	d.log.Infoln("Check Sub Supplier End")
	return outStatus
}

func (d *SubSupplierHub) downloadMovieSubFromSuppliers(videoFullPath string, index int64, suppliers []ifaces.ISupplier, requireChinese bool) ([]string, error) {
	if len(suppliers) < 1 {
		return nil, nil
	}

	subInfos := movieHelper.OneMovieDlSubInAllSite(d.log, suppliers, videoFullPath, index, requireChinese)
	if len(subInfos) < 1 {
		d.log.Warningln("OneMovieDlSubInAllSite.subInfos == 0, No Sub Downloaded.")
		return nil, nil
	}

	organizeSubFiles, err := sub_helper.OrganizeDlSubFiles(d.log, filepath.Base(videoFullPath), subInfos, true)
	if err != nil {
		return nil, errors.Newf("OrganizeDlSubFiles %v %v", videoFullPath, err)
	}

	outSubFileFullPathList := make([]string, 0)
	for _, subFiles := range organizeSubFiles {
		outSubFileFullPathList = append(outSubFileFullPathList, subFiles...)
	}

	for i, subFile := range outSubFileFullPathList {
		d.log.Debugln("OneMovieDlSubInAllSite", videoFullPath, i, "SubFileFPath:", subFile)
	}

	return outSubFileFullPathList, nil
}

func (d *SubSupplierHub) dlSubFromSeriesInfo(seriesDirPath string, index int64, seriesInfo *series.SeriesInfo, suppliers []ifaces.ISupplier, requireChinese bool) (map[string][]string, error) {
	if len(suppliers) < 1 {
		return nil, nil
	}

	subInfos := seriesHelper.DownloadSubtitleInAllSiteByOneSeries(d.log, suppliers, seriesInfo, index, requireChinese)
	if len(subInfos) < 1 {
		d.log.Warningln("DownloadSubtitleInAllSiteByOneSeries.subInfos == 0, No Sub Downloaded.")
	}

	organizeSubFiles, err := sub_helper.OrganizeDlSubFiles(d.log, filepath.Base(seriesDirPath), subInfos, false)
	if err != nil {
		return nil, errors.Newf("OrganizeDlSubFiles %v %v", seriesDirPath, err)
	}
	return organizeSubFiles, nil
}
