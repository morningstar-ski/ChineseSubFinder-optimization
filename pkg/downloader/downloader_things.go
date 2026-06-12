package downloader

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/decode"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	subcommon "github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_formatter/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/subparser"
)

var errNoUsableChineseSubtitle = errors.New("no usable chinese subtitle candidate")

// oneVideoSelectBestSub 只负责从当前阶段候选里选择并保存中文字幕。
func (d *Downloader) oneVideoSelectBestSub(oneVideoFullPath string, organizeSubFiles []string) error {
	organizeSubFiles = d.prepareSubtitleCandidates(oneVideoFullPath, organizeSubFiles)
	if len(organizeSubFiles) < 1 {
		return common.AllSiteDownloadSubNotFound
	}

	if settings.Get().AdvancedSettings.SaveMultiSub == false {
		finalSubFile := d.mk.SelectOneSubFileWithVideo(organizeSubFiles, oneVideoFullPath)
		if finalSubFile == nil {
			outString := fmt.Sprintln("Found", len(organizeSubFiles), "subtitles but not one chinese fit:", oneVideoFullPath)
			d.log.Warnln(outString)
			return errNoUsableChineseSubtitle
		}
		return d.writeSingleSubtitle(oneVideoFullPath, *finalSubFile)
	}

	siteNames, finalSubFiles := d.mk.SelectEachSiteTop1SubFile(organizeSubFiles)
	if len(siteNames) == 0 {
		outString := fmt.Sprintln("SelectEachSiteTop1SubFile found none sub file")
		d.log.Warnln(outString)
		return errors.New(outString)
	}

	return d.writeMultiSubtitles(oneVideoFullPath, siteNames, finalSubFiles)
}

func (d *Downloader) prepareSubtitleCandidates(oneVideoFullPath string, organizeSubFiles []string) []string {
	if len(organizeSubFiles) < 1 {
		return nil
	}

	organizeSubFiles = d.filterInvalidSubtitleCandidates(oneVideoFullPath, organizeSubFiles)
	if len(organizeSubFiles) < 1 {
		return nil
	}

	if settings.Get().AdvancedSettings.DebugMode {
		videoFileName := filepath.Base(oneVideoFullPath)
		if err := pkg.CopyFiles2DebugFolder([]string{videoFileName}, organizeSubFiles); err != nil {
			d.log.Errorln("copySubFile2DesFolder", err)
		}
	}

	return organizeSubFiles
}

func (d *Downloader) writeSingleSubtitle(oneVideoFullPath string, finalSubFile subparser.FileInfo) error {
	d.clearExistingSubtitleMarks(oneVideoFullPath)

	setDefault := true
	if d.subNameFormatter == subcommon.Normal {
		setDefault = false
	}

	if err := d.SaveSubHelper.WriteSubFile2VideoPath(oneVideoFullPath, finalSubFile, "", setDefault, false); err != nil {
		return errors.New(fmt.Sprintf("SaveMultiSub: %v, writeSubFile2VideoPath, Error: %v ", settings.Get().AdvancedSettings.SaveMultiSub, err))
	}

	return nil
}

func (d *Downloader) writeMultiSubtitles(oneVideoFullPath string, siteNames []string, finalSubFiles []subparser.FileInfo) error {
	d.clearExistingSubtitleMarks(oneVideoFullPath)

	if d.subNameFormatter == subcommon.Emby {
		for i, file := range finalSubFiles {
			setDefault := i == 0
			if err := d.SaveSubHelper.WriteSubFile2VideoPath(oneVideoFullPath, file, siteNames[i], setDefault, false); err != nil {
				return errors.New(fmt.Sprintf("SaveMultiSub: %v, writeSubFile2VideoPath, Error: %v ", settings.Get().AdvancedSettings.SaveMultiSub, err))
			}
		}
		return nil
	}

	for i := len(finalSubFiles) - 1; i > -1; i-- {
		if err := d.SaveSubHelper.WriteSubFile2VideoPath(oneVideoFullPath, finalSubFiles[i], siteNames[i], false, false); err != nil {
			return errors.New(fmt.Sprintf("SaveMultiSub: %v, writeSubFile2VideoPath, Error: %v ", settings.Get().AdvancedSettings.SaveMultiSub, err))
		}
	}
	return nil
}

func (d *Downloader) clearExistingSubtitleMarks(oneVideoFullPath string) {
	if err := sub_helper.SearchVideoMatchSubFileAndRemoveExtMark(d.log, oneVideoFullPath); err != nil {
		d.log.Errorln("SearchVideoMatchSubFileAndRemoveExtMark,", oneVideoFullPath, err)
	}
}

func (d *Downloader) canTryLLMStageFallback() bool {
	if settings.Get().AdvancedSettings.SaveMultiSub {
		return false
	}
	return d.llmSubtitleFallback != nil && d.llmSubtitleFallback.Enabled()
}

func (d *Downloader) tryWriteLLMSubtitleFallback(videoPath string, organizeSubFiles []string) error {
	if d.canTryLLMStageFallback() == false {
		return common.AllSiteDownloadSubNotFound
	}

	organizeSubFiles = d.prepareSubtitleCandidates(videoPath, organizeSubFiles)
	if len(organizeSubFiles) < 1 {
		return common.AllSiteDownloadSubNotFound
	}

	fallbackSub := d.tryLLMSubtitleFallback(videoPath, organizeSubFiles)
	if fallbackSub == nil {
		return common.AllSiteDownloadSubNotFound
	}

	return d.writeSingleSubtitle(videoPath, *fallbackSub)
}

// saveFullSeasonSub 需要单独存储到连续剧每一季的特殊缓存目录中。
func (d *Downloader) saveFullSeasonSub(seriesInfo *series.SeriesInfo, organizeSubFiles map[string][]string) map[string][]string {
	fullSeasonSubDict := make(map[string][]string)

	for _, season := range seriesInfo.SeasonDict {
		seasonKey := pkg.GetEpisodeKeyName(season, 0)
		subs, ok := organizeSubFiles[seasonKey]
		if ok == false {
			continue
		}
		for _, sub := range subs {
			subFileName := filepath.Base(sub)

			newSeasonSubRootPath, err := pkg.GetDebugFolderByName([]string{
				filepath.Base(seriesInfo.DirPath),
				"Sub_" + seasonKey})
			if err != nil {
				d.log.Errorln("saveFullSeasonSub.GetDebugFolderByName", subFileName, err)
				continue
			}

			newSubFullPath := filepath.Join(newSeasonSubRootPath, subFileName)
			err = pkg.CopyFile(sub, newSubFullPath)
			if err != nil {
				d.log.Errorln("saveFullSeasonSub.CopyFile", subFileName, err)
				continue
			}

			isFullSeasonSub, gusSeason, gusEpisode, err := decode.GetSeasonAndEpisodeFromSubFileName(subFileName)
			if err != nil {
				d.log.Debugln("saveFullSeasonSub.GetSeasonAndEpisodeFromSubFileName", subFileName, err)
				continue
			}
			if gusSeason <= 0 || isFullSeasonSub || gusEpisode <= 0 {
				d.log.Debugln("saveFullSeasonSub.SkipUnmatchedSeriesSub", subFileName, "Season:", gusSeason, "Episode:", gusEpisode, "IsFullSeason:", isFullSeasonSub)
				continue
			}

			seasonEpsKey := pkg.GetEpisodeKeyName(gusSeason, gusEpisode)
			if _, ok := fullSeasonSubDict[seasonEpsKey]; ok == false {
				fullSeasonSubDict[seasonEpsKey] = make([]string, 0)
			}
			fullSeasonSubDict[seasonEpsKey] = append(fullSeasonSubDict[seasonEpsKey], sub)
		}
	}

	return fullSeasonSubDict
}

func (d *Downloader) tryLLMSubtitleFallback(videoPath string, organizeSubFiles []string) *subparser.FileInfo {
	if d.llmSubtitleFallback == nil || d.llmSubtitleFallback.Enabled() == false {
		return nil
	}

	englishCandidate := d.mk.SelectBestEnglishSubFile(organizeSubFiles, videoPath)
	if englishCandidate == nil {
		return nil
	}

	fallbackSub, err := d.llmSubtitleFallback.BuildChineseSubtitleFromEnglish(videoPath, englishCandidate)
	if err != nil {
		d.log.Warningln("tryLLMSubtitleFallback.BuildChineseSubtitleFromEnglish", filepath.Base(videoPath), err)
		return nil
	}

	d.log.Infoln("tryLLMSubtitleFallback.Success", "video", filepath.Base(videoPath), "source", englishCandidate.Name)
	return fallbackSub
}
