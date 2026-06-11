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

	primaryErr := common.AllSiteDownloadSubNotFound
	if len(organizeSubFiles) > 0 {
		primaryErr = d.oneVideoSelectBestSub(job.VideoFPath, organizeSubFiles)
		if primaryErr == nil {
			d.downloadQueue.AutoDetectUpdateJobStatus(job, nil)
			return d.refreshEmbyMovieSubtitle(job)
		}
	}

	if d.shouldTryMovieLLMFallback(primaryErr) && nowSubSupplierHub.HasEnglishFallbackMovieSuppliers() {
		fallbackSubFiles, fallbackErr := nowSubSupplierHub.DownloadEnglishFallbackSub4Movie(job.VideoFPath, downloadIndex)
		if fallbackErr != nil {
			err = errors.New(fmt.Sprintf("subSupplierHub.DownloadEnglishFallbackSub4Movie: %v, %v", job.VideoFPath, fallbackErr))
			d.downloadQueue.AutoDetectUpdateJobStatus(job, err)
			return err
		}
		if err = d.tryWriteLLMSubtitleFallback(job.VideoFPath, fallbackSubFiles); err == nil {
			d.downloadQueue.AutoDetectUpdateJobStatus(job, nil)
			return d.refreshEmbyMovieSubtitle(job)
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

func (d *Downloader) shouldTryMovieLLMFallback(primaryErr error) bool {
	if d.canTryLLMStageFallback() == false || primaryErr == nil {
		return false
	}
	onlyWhenNoChineseCandidate := true
	if settings.Get().ExperimentalFunction != nil {
		onlyWhenNoChineseCandidate = settings.Get().ExperimentalFunction.LLMSubtitleFallback.OnlyWhenNoChineseCandidate
	}
	if onlyWhenNoChineseCandidate {
		return errors.Is(primaryErr, errNoUsableChineseSubtitle) || errors.Is(primaryErr, common.AllSiteDownloadSubNotFound)
	}
	return true
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
	pendingEnglishFallback := make(map[string]series.EpisodeInfo)

	for epsKey, episodeInfo := range seriesInfo.NeedDlEpsKeyList {
		err = d.selectSeriesEpisodeSubtitle(ctx, episodeInfo.FileFullPath, organizeSubFiles[epsKey])
		if err == nil {
			save2LocalSubCount++
			continue
		}
		if d.canTryLLMStageFallback() && nowSubSupplierHub.HasEnglishFallbackSeriesSuppliers() && errors.Is(err, errNoUsableChineseSubtitle) {
			pendingEnglishFallback[epsKey] = episodeInfo
			continue
		}
		if d.canTryLLMStageFallback() && nowSubSupplierHub.HasEnglishFallbackSeriesSuppliers() && errors.Is(err, common.AllSiteDownloadSubNotFound) {
			pendingEnglishFallback[epsKey] = episodeInfo
			continue
		}
		errSave2Local = err
		d.log.Errorln(err)
	}

	fullSeasonSubDict := d.saveFullSeasonSub(seriesInfo, organizeSubFiles)
	for _, episodeInfo := range seriesInfo.EpList {
		if _, ok := seriesInfo.NeedDlSeasonDict[episodeInfo.Season]; ok == false {
			continue
		}

		seasonEpsKey := pkg.GetEpisodeKeyName(episodeInfo.Season, episodeInfo.Episode)
		subs := fullSeasonSubDict[seasonEpsKey]
		if len(subs) < 1 {
			d.log.Infoln("seriesDlFunc.saveFullSeasonSub, no sub found, Skip", seasonEpsKey)
			continue
		}

		err = d.selectSeriesEpisodeSubtitle(ctx, episodeInfo.FileFullPath, subs)
		if err != nil {
			errSave2Local = err
			d.log.Errorln(err)
			continue
		}
		save2LocalSubCount++
		delete(pendingEnglishFallback, seasonEpsKey)
	}

	if len(pendingEnglishFallback) > 0 && d.canTryLLMStageFallback() && nowSubSupplierHub.HasEnglishFallbackSeriesSuppliers() {
		fallbackSeriesInfo := buildSeriesFallbackInfo(seriesInfo, pendingEnglishFallback)
		englishSubFiles, fallbackErr := nowSubSupplierHub.DownloadEnglishFallbackSub4Series(job.SeriesRootDirPath, fallbackSeriesInfo, downloadIndex)
		if fallbackErr != nil {
			err = errors.New(fmt.Sprintf("seriesDlFunc.DownloadEnglishFallbackSub4Series %v S%vE%v %v", filepath.Base(job.SeriesRootDirPath), job.Season, job.Episode, fallbackErr))
			d.downloadQueue.AutoDetectUpdateJobStatus(job, err)
			return err
		}
		for epsKey, episodeInfo := range pendingEnglishFallback {
			err = d.tryLLMSeriesFallback(ctx, episodeInfo.FileFullPath, englishSubFiles[epsKey])
			if err != nil {
				errSave2Local = err
				d.log.Errorln(err)
				continue
			}
			save2LocalSubCount++
		}
	}

	if settings.Get().AdvancedSettings.SaveFullSeasonTmpSubtitles == false {
		if err = sub_helper.DeleteOneSeasonSubCacheFolder(seriesInfo.DirPath); err != nil {
			d.log.Errorln("seriesDlFunc.DeleteOneSeasonSubCacheFolder", err)
		}
	}

	if save2LocalSubCount < 1 {
		if errSave2Local == nil {
			errSave2Local = task_queue.ErrNoSubFound
		}
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

func (d *Downloader) selectSeriesEpisodeSubtitle(ctx context.Context, videoPath string, organizeSubFiles []string) error {
	done := make(chan interface{}, 1)
	panicChan := make(chan interface{}, 1)

	go func() {
		defer func() {
			if p := recover(); p != nil {
				panicChan <- p
			}
			close(done)
			close(panicChan)
		}()
		done <- d.oneVideoSelectBestSub(videoPath, organizeSubFiles)
	}()

	select {
	case errInterface := <-done:
		if errInterface == nil {
			return nil
		}
		return errInterface.(error)
	case p := <-panicChan:
		d.log.Errorln("seriesDlFunc.oneVideoSelectBestSub panicChan", p)
		return errors.New("seriesDlFunc.oneVideoSelectBestSub panic")
	case <-ctx.Done():
		return errors.New(fmt.Sprintf("cancel at NeedDlEpsKeyList.oneVideoSelectBestSub, %s", filepath.Base(videoPath)))
	}
}

func (d *Downloader) tryLLMSeriesFallback(ctx context.Context, videoPath string, organizeSubFiles []string) error {
	done := make(chan interface{}, 1)
	panicChan := make(chan interface{}, 1)

	go func() {
		defer func() {
			if p := recover(); p != nil {
				panicChan <- p
			}
			close(done)
			close(panicChan)
		}()
		done <- d.tryWriteLLMSubtitleFallback(videoPath, organizeSubFiles)
	}()

	select {
	case errInterface := <-done:
		if errInterface == nil {
			return nil
		}
		return errInterface.(error)
	case p := <-panicChan:
		d.log.Errorln("seriesDlFunc.tryWriteLLMSubtitleFallback panicChan", p)
		return errors.New("seriesDlFunc.tryWriteLLMSubtitleFallback panic")
	case <-ctx.Done():
		return errors.New(fmt.Sprintf("cancel at NeedDlEpsKeyList.tryWriteLLMSubtitleFallback, %s", filepath.Base(videoPath)))
	}
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
