package downloader

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/series_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/task_queue"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	taskQueue2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/task_queue"
	"golang.org/x/net/context"
)

type subtitleFallbackStage int

const (
	subtitleFallbackStageTranslatedChinese subtitleFallbackStage = iota
	subtitleFallbackStageLLM
	subtitleFallbackStageEnglish
)

func (d *Downloader) orderedSubtitleFallbackStages() []subtitleFallbackStage {
	stages := make([]subtitleFallbackStage, 0, 3)
	if d.canTryTranslatedChineseFallback() {
		stages = append(stages, subtitleFallbackStageTranslatedChinese)
	}
	if d.canTryLLMStageFallback() {
		stages = append(stages, subtitleFallbackStageLLM)
	}
	if d.canTryEnglishFallback() {
		stages = append(stages, subtitleFallbackStageEnglish)
	}
	return stages
}

func (d *Downloader) movieDlFunc(ctx context.Context, job taskQueue2.OneJob, downloadIndex int64) error {
	nowSubSupplierHub := d.subSupplierHub
	if nowSubSupplierHub.Suppliers == nil || len(nowSubSupplierHub.Suppliers) < 1 {
		d.log.Infoln("Wait SupplierCheck Update *subSupplierHub, movieDlFunc Skip this time")
		return nil
	}

	organizeSubFiles, err := nowSubSupplierHub.DownloadSub4Movie(job.VideoFPath, downloadIndex)
	if err != nil {
		err = errors.New(fmt.Sprintf("subSupplierHub.DownloadSub4Movie: %v, %v", job.VideoFPath, err))
		d.downloadQueue.AutoDetectUpdateJobStatus(job, err)
		return err
	}

	var primaryErr error
	if len(organizeSubFiles) > 0 {
		primaryErr = d.oneVideoSelectBestSub(job.VideoFPath, organizeSubFiles)
		if primaryErr == nil {
			d.downloadQueue.AutoDetectUpdateJobStatus(job, nil)
			return d.refreshEmbyMovieSubtitle(job)
		}
	}

	var englishFallbackSubFiles []string
	englishFallbackLoaded := false
	loadEnglishFallbackSubFiles := func() ([]string, error) {
		if englishFallbackLoaded {
			return englishFallbackSubFiles, nil
		}
		englishFallbackLoaded = true
		var fallbackErr error
		englishFallbackSubFiles, fallbackErr = nowSubSupplierHub.DownloadEnglishFallbackSub4Movie(job.VideoFPath, downloadIndex)
		if fallbackErr != nil {
			return nil, fallbackErr
		}
		return englishFallbackSubFiles, nil
	}

	for _, stage := range d.orderedSubtitleFallbackStages() {
		switch stage {
		case subtitleFallbackStageTranslatedChinese:
			if nowSubSupplierHub.HasTranslatedFallbackMovieSuppliers() == false {
				continue
			}
			translatedSubFiles, translatedErr := nowSubSupplierHub.DownloadTranslatedFallbackSub4Movie(job.VideoFPath, downloadIndex)
			if translatedErr != nil {
				err = errors.New(fmt.Sprintf("subSupplierHub.DownloadTranslatedFallbackSub4Movie: %v, %v", job.VideoFPath, translatedErr))
				d.downloadQueue.AutoDetectUpdateJobStatus(job, err)
				return err
			}
			if err = d.oneVideoSelectBestSub(job.VideoFPath, translatedSubFiles); err == nil {
				d.downloadQueue.AutoDetectUpdateJobStatus(job, nil)
				return d.refreshEmbyMovieSubtitle(job)
			}
		case subtitleFallbackStageLLM:
			if nowSubSupplierHub.HasEnglishFallbackMovieSuppliers() == false {
				continue
			}
			fallbackSubFiles, fallbackErr := loadEnglishFallbackSubFiles()
			if fallbackErr != nil {
				err = errors.New(fmt.Sprintf("subSupplierHub.DownloadEnglishFallbackSub4Movie: %v, %v", job.VideoFPath, fallbackErr))
				d.downloadQueue.AutoDetectUpdateJobStatus(job, err)
				return err
			}
			if err = d.tryWriteLLMSubtitleFallback(job.VideoFPath, fallbackSubFiles); err == nil {
				d.downloadQueue.AutoDetectUpdateJobStatus(job, nil)
				return d.refreshEmbyMovieSubtitle(job)
			}
		case subtitleFallbackStageEnglish:
			if nowSubSupplierHub.HasEnglishFallbackMovieSuppliers() == false {
				continue
			}
			fallbackSubFiles, fallbackErr := loadEnglishFallbackSubFiles()
			if fallbackErr != nil {
				err = errors.New(fmt.Sprintf("subSupplierHub.DownloadEnglishFallbackSub4Movie: %v, %v", job.VideoFPath, fallbackErr))
				d.downloadQueue.AutoDetectUpdateJobStatus(job, err)
				return err
			}
			if err = d.tryWriteEnglishSubtitleFallback(job.VideoFPath, fallbackSubFiles); err == nil {
				d.downloadQueue.AutoDetectUpdateJobStatus(job, nil)
				return d.refreshEmbyMovieSubtitle(job)
			}
		}
	}

	if primaryErr != nil {
		d.downloadQueue.AutoDetectUpdateJobStatus(job, primaryErr)
		return primaryErr
	}

	d.log.Infoln(task_queue.ErrNoSubFound.Error(), filepath.Base(job.VideoFPath))
	d.downloadQueue.AutoDetectUpdateJobStatus(job, task_queue.ErrNoSubFound)
	return nil
}

func (d *Downloader) refreshEmbyMovieSubtitle(job taskQueue2.OneJob) error {
	if settings.Get().EmbySettings.Enable == false || d.embyHelper == nil || job.MediaServerInsideVideoID == "" {
		if settings.Get().EmbySettings.Enable == false {
			d.log.Infoln("字幕下载完毕，尝试刷新 Emby 中对应字幕", job.VideoFPath, "Skip, because Emby enable is false")
		} else if d.embyHelper == nil {
			d.log.Infoln("字幕下载完毕，尝试刷新 Emby 中对应字幕", job.VideoFPath, "Skip, because EmbyHelper is nil")
		} else {
			d.log.Infoln("字幕下载完毕，尝试刷新 Emby 中对应字幕", job.VideoFPath, "Skip, because MediaServerInsideVideoID is empty")
		}
		return nil
	}

	d.log.Infoln("字幕下载完毕，尝试刷新 Emby 中对应字幕", job.VideoFPath, job.MediaServerInsideVideoID)
	if err := d.embyHelper.EmbyApi.UpdateVideoSubList(settings.Get().EmbySettings, job.MediaServerInsideVideoID); err != nil {
		d.log.Errorln("UpdateVideoSubList", job.VideoFPath, job.MediaServerInsideVideoID, "Error:", err)
		return err
	}
	return nil
}

func (d *Downloader) seriesDlFunc(ctx context.Context, job taskQueue2.OneJob, downloadIndex int64) error {
	nowSubSupplierHub := d.subSupplierHub
	if nowSubSupplierHub == nil || nowSubSupplierHub.Suppliers == nil || len(nowSubSupplierHub.Suppliers) < 1 {
		d.log.Infoln("Wait SupplierCheck Update *subSupplierHub, movieDlFunc Skip this time")
		return nil
	}

	epsMap := make(map[int][]int, 0)
	epsMap[job.Season] = []int{job.Episode}

	seriesInfo, err := series_helper.ReadSeriesInfoFromDir(
		d.fileDownloader.MediaInfoDealers,
		job.SeriesRootDirPath,
		settings.Get().AdvancedSettings.TaskQueue.ExpirationTime,
		false,
		false,
		epsMap,
	)
	if err != nil {
		err = errors.New(fmt.Sprintf("seriesDlFunc.ReadSeriesInfoFromDir, Error: %v", err))
		d.downloadQueue.AutoDetectUpdateJobStatus(job, err)
		return err
	}

	organizeSubFiles, err := nowSubSupplierHub.DownloadSub4Series(job.SeriesRootDirPath, seriesInfo, downloadIndex)
	if err != nil {
		err = errors.New(fmt.Sprintf("seriesDlFunc.DownloadSub4Series %v S%vE%v %v", filepath.Base(job.SeriesRootDirPath), job.Season, job.Episode, err))
		d.downloadQueue.AutoDetectUpdateJobStatus(job, err)
		return err
	}
	if organizeSubFiles == nil {
		organizeSubFiles = make(map[string][]string)
	}

	var errSave2Local error
	save2LocalSubCount := 0
	savedEpisodeKeys := make(map[string]struct{}, len(seriesInfo.NeedDlEpsKeyList))
	pendingFallbackEpisodes := make(map[string]series.EpisodeInfo)

	for epsKey, episodeInfo := range seriesInfo.NeedDlEpsKeyList {
		err = d.selectSeriesEpisodeSubtitle(ctx, episodeInfo.FileFullPath, organizeSubFiles[epsKey])
		if err == nil {
			save2LocalSubCount++
			savedEpisodeKeys[epsKey] = struct{}{}
			continue
		}
		if d.canTryEnglishFallback() && nowSubSupplierHub.HasEnglishFallbackSeriesSuppliers() && errors.Is(err, errNoUsableChineseSubtitle) {
			pendingFallbackEpisodes[epsKey] = episodeInfo
			continue
		}
		if d.canTryEnglishFallback() && nowSubSupplierHub.HasEnglishFallbackSeriesSuppliers() && errors.Is(err, common.AllSiteDownloadSubNotFound) {
			pendingFallbackEpisodes[epsKey] = episodeInfo
			continue
		}
		errSave2Local = err
		d.log.Errorln(err)
	}

	fullSeasonSubDict := d.saveFullSeasonSub(seriesInfo, organizeSubFiles)
	for _, episodeInfo := range pendingSeasonPackEpisodes(seriesInfo, savedEpisodeKeys) {
		seasonEpsKey := pkg.GetEpisodeKeyName(episodeInfo.Season, episodeInfo.Episode)
		subs := fullSeasonSubDict[seasonEpsKey]
		if len(subs) < 1 {
			d.log.Debugln("seriesDlFunc.saveFullSeasonSub, no sub found, Skip", seasonEpsKey)
			continue
		}

		err = d.selectSeriesEpisodeSubtitle(ctx, episodeInfo.FileFullPath, subs)
		if err != nil {
			errSave2Local = err
			d.log.Errorln(err)
			continue
		}
		save2LocalSubCount++
		savedEpisodeKeys[seasonEpsKey] = struct{}{}
		delete(pendingFallbackEpisodes, seasonEpsKey)
	}

	var englishFallbackSeriesSubFiles map[string][]string
	englishFallbackSeriesLoaded := false
	loadEnglishFallbackSeriesSubFiles := func() (map[string][]string, error) {
		if englishFallbackSeriesLoaded {
			return englishFallbackSeriesSubFiles, nil
		}
		englishFallbackSeriesLoaded = true
		fallbackSeriesInfo := buildSeriesFallbackInfo(seriesInfo, pendingFallbackEpisodes)
		var fallbackErr error
		englishFallbackSeriesSubFiles, fallbackErr = nowSubSupplierHub.DownloadEnglishFallbackSub4Series(job.SeriesRootDirPath, fallbackSeriesInfo, downloadIndex)
		if fallbackErr != nil {
			return nil, fallbackErr
		}
		return englishFallbackSeriesSubFiles, nil
	}

	for _, stage := range d.orderedSubtitleFallbackStages() {
		if len(pendingFallbackEpisodes) == 0 {
			break
		}

		switch stage {
		case subtitleFallbackStageTranslatedChinese:
			if nowSubSupplierHub.HasTranslatedFallbackSeriesSuppliers() == false {
				continue
			}
			fallbackSeriesInfo := buildSeriesFallbackInfo(seriesInfo, pendingFallbackEpisodes)
			translatedSubFiles, fallbackErr := nowSubSupplierHub.DownloadTranslatedFallbackSub4Series(job.SeriesRootDirPath, fallbackSeriesInfo, downloadIndex)
			if fallbackErr != nil {
				err = errors.New(fmt.Sprintf("seriesDlFunc.DownloadTranslatedFallbackSub4Series %v S%vE%v %v", filepath.Base(job.SeriesRootDirPath), job.Season, job.Episode, fallbackErr))
				d.downloadQueue.AutoDetectUpdateJobStatus(job, err)
				return err
			}
			for epsKey, episodeInfo := range pendingFallbackEpisodes {
				err = d.selectSeriesEpisodeSubtitle(ctx, episodeInfo.FileFullPath, translatedSubFiles[epsKey])
				if err != nil {
					errSave2Local = err
					d.log.Errorln(err)
					continue
				}
				save2LocalSubCount++
				savedEpisodeKeys[epsKey] = struct{}{}
				delete(pendingFallbackEpisodes, epsKey)
			}
		case subtitleFallbackStageLLM:
			if nowSubSupplierHub.HasEnglishFallbackSeriesSuppliers() == false {
				continue
			}
			englishSubFiles, fallbackErr := loadEnglishFallbackSeriesSubFiles()
			if fallbackErr != nil {
				err = errors.New(fmt.Sprintf("seriesDlFunc.DownloadEnglishFallbackSub4Series %v S%vE%v %v", filepath.Base(job.SeriesRootDirPath), job.Season, job.Episode, fallbackErr))
				d.downloadQueue.AutoDetectUpdateJobStatus(job, err)
				return err
			}
			for epsKey, episodeInfo := range pendingFallbackEpisodes {
				err = d.tryLLMSeriesFallback(ctx, episodeInfo.FileFullPath, englishSubFiles[epsKey])
				if err != nil {
					errSave2Local = err
					d.log.Errorln(err)
					continue
				}
				save2LocalSubCount++
				savedEpisodeKeys[epsKey] = struct{}{}
				delete(pendingFallbackEpisodes, epsKey)
			}
		case subtitleFallbackStageEnglish:
			if nowSubSupplierHub.HasEnglishFallbackSeriesSuppliers() == false {
				continue
			}
			englishSubFiles, fallbackErr := loadEnglishFallbackSeriesSubFiles()
			if fallbackErr != nil {
				err = errors.New(fmt.Sprintf("seriesDlFunc.DownloadEnglishFallbackSub4Series %v S%vE%v %v", filepath.Base(job.SeriesRootDirPath), job.Season, job.Episode, fallbackErr))
				d.downloadQueue.AutoDetectUpdateJobStatus(job, err)
				return err
			}
			for epsKey, episodeInfo := range pendingFallbackEpisodes {
				err = d.tryEnglishSeriesFallback(ctx, episodeInfo.FileFullPath, englishSubFiles[epsKey])
				if err != nil {
					errSave2Local = err
					d.log.Errorln(err)
					continue
				}
				save2LocalSubCount++
				savedEpisodeKeys[epsKey] = struct{}{}
				delete(pendingFallbackEpisodes, epsKey)
			}
		}
	}

	if settings.Get().AdvancedSettings.SaveFullSeasonTmpSubtitles == false {
		if err = sub_helper.DeleteOneSeasonSubCacheFolder(seriesInfo.DirPath); err != nil {
			d.log.Errorln("seriesDlFunc.DeleteOneSeasonSubCacheFolder", err)
		}
	}

	if save2LocalSubCount < 1 {
		errSave2Local = normalizeSeriesTerminalError(errSave2Local)
		d.downloadQueue.AutoDetectUpdateJobStatus(job, errSave2Local)
		return errSave2Local
	}

	d.downloadQueue.AutoDetectUpdateJobStatus(job, nil)
	if settings.Get().EmbySettings.Enable == true && d.embyHelper != nil {
		if job.MediaServerInsideVideoID != "" {
			d.log.Infoln("字幕下载完毕，尝试刷新 Emby 中对应字幕", job.SeriesRootDirPath, job.MediaServerInsideVideoID, job.Season, job.Episode)
			err = d.embyHelper.EmbyApi.UpdateVideoSubList(settings.Get().EmbySettings, job.MediaServerInsideVideoID)
			if err != nil {
				d.log.Errorln("UpdateVideoSubList", job.SeriesRootDirPath, job.MediaServerInsideVideoID, job.Season, job.Episode, "Error:", err)
				return err
			}
		} else {
			d.log.Warningln("字幕下载完毕，尝试刷新 Emby 中对应字幕，跳过，因为 MediaServerInsideVideoID 为空", job.SeriesRootDirPath, job.Season, job.Episode)
		}
	}

	return nil
}

func pendingSeasonPackEpisodes(seriesInfo *series.SeriesInfo, savedEpisodeKeys map[string]struct{}) []series.EpisodeInfo {
	if seriesInfo == nil || len(seriesInfo.EpList) == 0 || len(seriesInfo.NeedDlSeasonDict) == 0 {
		return nil
	}

	pending := make([]series.EpisodeInfo, 0, len(seriesInfo.EpList))
	for _, episodeInfo := range seriesInfo.EpList {
		if _, ok := seriesInfo.NeedDlSeasonDict[episodeInfo.Season]; ok == false {
			continue
		}

		seasonEpsKey := pkg.GetEpisodeKeyName(episodeInfo.Season, episodeInfo.Episode)
		if _, ok := savedEpisodeKeys[seasonEpsKey]; ok {
			continue
		}

		pending = append(pending, episodeInfo)
	}

	return pending
}

func normalizeSeriesTerminalError(err error) error {
	switch {
	case err == nil:
		return task_queue.ErrNoSubFound
	case errors.Is(err, task_queue.ErrNoSubFound):
		return task_queue.ErrNoSubFound
	case errors.Is(err, common.AllSiteDownloadSubNotFound):
		return task_queue.ErrNoSubFound
	case errors.Is(err, errNoUsableChineseSubtitle):
		return task_queue.ErrNoSubFound
	default:
		return err
	}
}

func (d *Downloader) selectSeriesEpisodeSubtitle(ctx context.Context, videoPath string, organizeSubFiles []string) error {
	err, p, canceled := runDownloaderErrorStep(ctx, func() error {
		return d.oneVideoSelectBestSub(videoPath, organizeSubFiles)
	})
	if p != nil {
		d.log.Errorln("seriesDlFunc.oneVideoSelectBestSub panicChan", p)
		return errors.New("seriesDlFunc.oneVideoSelectBestSub panic")
	}
	if canceled {
		return errors.New(fmt.Sprintf("cancel at NeedDlEpsKeyList.oneVideoSelectBestSub, %s", filepath.Base(videoPath)))
	}
	return err
}

func (d *Downloader) tryLLMSeriesFallback(ctx context.Context, videoPath string, organizeSubFiles []string) error {
	err, p, canceled := runDownloaderErrorStep(ctx, func() error {
		return d.tryWriteLLMSubtitleFallback(videoPath, organizeSubFiles)
	})
	if p != nil {
		d.log.Errorln("seriesDlFunc.tryWriteLLMSubtitleFallback panicChan", p)
		return errors.New("seriesDlFunc.tryWriteLLMSubtitleFallback panic")
	}
	if canceled {
		return errors.New(fmt.Sprintf("cancel at NeedDlEpsKeyList.tryWriteLLMSubtitleFallback, %s", filepath.Base(videoPath)))
	}
	return err
}

func buildSeriesFallbackInfo(seriesInfo *series.SeriesInfo, pending map[string]series.EpisodeInfo) *series.SeriesInfo {
	seasonDict := make(map[int]int)
	needDlSeasonDict := make(map[int]int)
	epList := make([]series.EpisodeInfo, 0, len(pending))
	needDlEpsKeyList := make(map[string]series.EpisodeInfo, len(pending))

	for epsKey, episodeInfo := range pending {
		epList = append(epList, episodeInfo)
		needDlEpsKeyList[epsKey] = episodeInfo
		seasonDict[episodeInfo.Season] = episodeInfo.Season
		needDlSeasonDict[episodeInfo.Season] = episodeInfo.Season
	}

	return &series.SeriesInfo{
		ImdbId:           seriesInfo.ImdbId,
		Name:             seriesInfo.Name,
		Year:             seriesInfo.Year,
		ReleaseDate:      seriesInfo.ReleaseDate,
		EpList:           epList,
		DirPath:          seriesInfo.DirPath,
		SeasonDict:       seasonDict,
		NeedDlSeasonDict: needDlSeasonDict,
		NeedDlEpsKeyList: needDlEpsKeyList,
	}
}
