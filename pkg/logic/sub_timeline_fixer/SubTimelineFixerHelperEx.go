package sub_timeline_fixer

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/subparser"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/ffmpeg_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/ass"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/srt"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_parser_hub"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_timeline_fixer"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/vad"
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/emirpasic/gods/utils"
	"github.com/sirupsen/logrus"
)

type SubTimelineFixerHelperEx struct {
	log                 *logrus.Logger
	ffmpegHelper        *ffmpeg_helper.FFMPEGHelper
	subParserHub        *sub_parser_hub.SubParserHub
	timelineFixPipeLine *sub_timeline_fixer.Pipeline
	fixerConfig         settings.TimelineFixerSettings
	needDownloadFFMPeg  bool
}

var ffmpegVersionProbe = func(helper *ffmpeg_helper.FFMPEGHelper) (string, error) {
	return helper.Version()
}

var ffsubsyncVersionProbe = func() (string, error) {
	return getFFSubSyncVersion()
}

func NewSubTimelineFixerHelperEx(log *logrus.Logger, fixerConfig settings.TimelineFixerSettings) *SubTimelineFixerHelperEx {

	fixerConfig.Check()

	return &SubTimelineFixerHelperEx{
		log:                 log,
		ffmpegHelper:        ffmpeg_helper.NewFFMPEGHelper(log),
		subParserHub:        sub_parser_hub.NewSubParserHub(log, ass.NewParser(log), srt.NewParser(log)),
		timelineFixPipeLine: sub_timeline_fixer.NewPipeline(fixerConfig.MaxOffsetTime),
		fixerConfig:         fixerConfig,
		needDownloadFFMPeg:  false,
	}
}

func (s *SubTimelineFixerHelperEx) ensureReady() error {
	version, err := ffmpegVersionProbe(s.ffmpegHelper)
	if err != nil {
		s.needDownloadFFMPeg = false
		return errors.New("ffmpeg/ffprobe not ready: " + err.Error())
	}
	ffsubsyncVersion, err := ffsubsyncVersionProbe()
	if err != nil {
		s.needDownloadFFMPeg = false
		return errors.New("ffsubsync not ready: " + err.Error())
	}
	s.needDownloadFFMPeg = true
	s.log.Infoln(version)
	s.log.Infoln(ffsubsyncVersion)
	return nil
}

// Check 是否安装了 ffmpeg 和 ffprobe
func (s *SubTimelineFixerHelperEx) Check() bool {
	if err := s.ensureReady(); err != nil {
		s.log.Errorln(err)
		return false
	}
	return true
}

func (s *SubTimelineFixerHelperEx) Process(videoFileFullPath, srcSubFPath string) error {

	if s.needDownloadFFMPeg == false {
		if err := s.ensureReady(); err != nil {
			s.log.Errorln("Need Install ffmpeg and ffprobe, Can't Do TimeLine Fix")
			return err
		}
	}

	return s.processWithFFSubSync(videoFileFullPath, srcSubFPath)
}
func invalidTimelineFixedSubtitleReason(fileInfo *subparser.FileInfo, videoDuration float64) string {
	if fileInfo == nil {
		return "nil subtitle info"
	}
	if len(fileInfo.Dialogues) == 0 {
		return "no parsed dialogues"
	}
	for i := 1; i < len(fileInfo.Dialogues); i++ {
		prevStart := pkg.Time2SecondNumber(fileInfo.Dialogues[i-1].GetStartTime())
		currStart := pkg.Time2SecondNumber(fileInfo.Dialogues[i].GetStartTime())
		if currStart < prevStart {
			return "subtitle timeline is not monotonic"
		}
	}
	subEndSeconds := pkg.Time2SecondNumber(fileInfo.GetEndTime())
	if subEndSeconds > 6*60*60 {
		return "subtitle end time exceeds absolute threshold"
	}
	if videoDuration > 0 {
		maxAllowedEnd := math.Max(videoDuration*1.5, videoDuration+15*60)
		if subEndSeconds > maxAllowedEnd {
			return "subtitle end time exceeds video duration tolerance"
		}
	}
	vectorGarbageLines := 0
	for _, dialogue := range fileInfo.Dialogues {
		for _, line := range dialogue.Lines {
			trimmed := strings.TrimSpace(line)
			if len(trimmed) > 2 {
				first := strings.ToLower(trimmed[:1])
				if strings.Contains("mnlbspc", first) && strings.ContainsAny(trimmed[1:], "-0123456789") {
					vectorGarbageLines++
					if vectorGarbageLines >= 3 {
						return "subtitle contains vector drawing garbage"
					}
				}
			}
		}
	}
	return ""
}

func (s *SubTimelineFixerHelperEx) ProcessBySubFileInfo(infoBase *subparser.FileInfo, infoSrc *subparser.FileInfo) (bool, *subparser.FileInfo, sub_timeline_fixer.PipeResult, error) {

	// ---------------------------------------------------------------------------------------
	pipeResult, err := s.timelineFixPipeLine.CalcOffsetTime(infoBase, infoSrc, nil, false)
	if err != nil {
		return false, nil, sub_timeline_fixer.PipeResult{}, err
	}

	return true, infoSrc, pipeResult, nil
}

func (s *SubTimelineFixerHelperEx) ProcessBySubFile(baseSubFileFPath, srcSubFileFPath string) (bool, *subparser.FileInfo, sub_timeline_fixer.PipeResult, error) {

	bFind, infoBase, err := s.subParserHub.DetermineFileTypeFromFile(baseSubFileFPath)
	if err != nil {
		return false, nil, sub_timeline_fixer.PipeResult{}, err
	}
	if bFind == false {
		s.log.Warnln("ProcessBySubFile.DetermineFileTypeFromFile sub not match --", baseSubFileFPath)
		return false, nil, sub_timeline_fixer.PipeResult{}, nil
	}

	bFind, infoSrc, err := s.subParserHub.DetermineFileTypeFromFile(srcSubFileFPath)
	if err != nil {
		return false, nil, sub_timeline_fixer.PipeResult{}, err
	}
	if bFind == false {
		s.log.Warnln("ProcessBySubFile.DetermineFileTypeFromFile sub not match --", srcSubFileFPath)
		return false, nil, sub_timeline_fixer.PipeResult{}, nil
	}

	return s.ProcessBySubFileInfo(infoBase, infoSrc)
}

func (s *SubTimelineFixerHelperEx) ProcessByAudioVAD(audioVADInfos []vad.VADInfo, infoSrc *subparser.FileInfo) (bool, *subparser.FileInfo, sub_timeline_fixer.PipeResult, error) {

	// ---------------------------------------------------------------------------------------
	pipeResult, err := s.timelineFixPipeLine.CalcOffsetTime(nil, infoSrc, audioVADInfos, false)
	if err != nil {
		return false, nil, sub_timeline_fixer.PipeResult{}, err
	}

	return true, infoSrc, pipeResult, nil
}

func (s *SubTimelineFixerHelperEx) ProcessByAudioFile(baseAudioFileFPath, srcSubFileFPath string) (bool, *subparser.FileInfo, sub_timeline_fixer.PipeResult, error) {

	audioVADInfos, err := vad.GetVADInfoFromAudio(vad.AudioInfo{
		FileFullPath: baseAudioFileFPath,
		SampleRate:   16000,
		BitDepth:     16,
	}, true)
	if err != nil {
		return false, nil, sub_timeline_fixer.PipeResult{}, err
	}

	bFind, infoSrc, err := s.subParserHub.DetermineFileTypeFromFile(srcSubFileFPath)
	if err != nil {
		return false, nil, sub_timeline_fixer.PipeResult{}, err
	}
	if bFind == false {
		s.log.Warnln("ProcessByAudioFile.DetermineFileTypeFromFile sub not match --", srcSubFileFPath)
		return false, nil, sub_timeline_fixer.PipeResult{}, nil
	}

	return s.ProcessByAudioVAD(audioVADInfos, infoSrc)
}

func (s *SubTimelineFixerHelperEx) IsVideoCanExportSubtitleAndAudio(videoFileFullPath string) (bool, *ffmpeg_helper.FFMPEGInfo, []vad.VADInfo, *subparser.FileInfo, error) {

	// 先尝试获取内置字幕的信息
	bok, ffmpegInfo, err := s.ffmpegHelper.ExportFFMPEGInfo(videoFileFullPath, ffmpeg_helper.SubtitleAndAudio)
	if err != nil {
		return false, nil, nil, nil, err
	}
	if bok == false {
		return false, nil, nil, nil, nil
	}
	// ---------------------------------------------------------------------------------------
	// 音频
	if len(ffmpegInfo.AudioInfoList) <= 0 {
		return false, nil, nil, nil, nil
	}
	audioVADInfos, err := vad.GetVADInfoFromAudio(vad.AudioInfo{
		FileFullPath: ffmpegInfo.AudioInfoList[0].FullPath,
		SampleRate:   16000,
		BitDepth:     16,
	}, true)
	if err != nil {
		return false, nil, nil, nil, err
	}
	// ---------------------------------------------------------------------------------------
	// 字幕
	if len(ffmpegInfo.SubtitleInfoList) <= 0 {
		return false, nil, nil, nil, nil
	}
	// 使用内置的字幕进行时间轴的校正，这里需要考虑一个问题，内置的字幕可能是有问题的（先考虑一种，就是字幕的长度不对，是一小段的）
	// 那么就可以比较多个内置字幕的大小选择大的去使用
	// 如果有多个内置的字幕，还是要判断下的，选体积最大的那个吧
	fileSizes := treemap.NewWith(utils.Int64Comparator)
	for index, info := range ffmpegInfo.SubtitleInfoList {
		fi, err := os.Stat(info.FullPath)
		if err != nil {
			fileSizes.Put(0, index)
		} else {
			fileSizes.Put(fi.Size(), index)
		}
	}
	_, index := fileSizes.Max()
	baseSubFPath := ffmpegInfo.SubtitleInfoList[index.(int)].FullPath
	bFind, infoBase, err := s.subParserHub.DetermineFileTypeFromFile(baseSubFPath)
	if err != nil {
		return false, nil, nil, nil, err
	}
	if bFind == false {
		return false, nil, nil, nil, nil
	}
	// ---------------------------------------------------------------------------------------

	return true, ffmpegInfo, audioVADInfos, infoBase, nil
}

func (s *SubTimelineFixerHelperEx) IsMatchBySubFile(ffmpegInfo *ffmpeg_helper.FFMPEGInfo, audioVADInfos []vad.VADInfo, infoBase *subparser.FileInfo, srcSubFileFPath string, config CompareConfig) (bool, *MatchResult, error) {

	bFind, srcBase, err := s.subParserHub.DetermineFileTypeFromFile(srcSubFileFPath)
	if err != nil {
		return false, nil, err
	}
	if bFind == false {
		return false, nil, nil
	}
	// ---------------------------------------------------------------------------------------
	// 音频
	s.log.Infoln("IsMatchBySubFile:", srcSubFileFPath)
	bProcess, _, pipeResultMaxAudio, err := s.ProcessByAudioVAD(audioVADInfos, srcBase)
	if err != nil {
		return false, nil, err
	}
	if bProcess == false {
		return false, nil, nil
	}
	// ---------------------------------------------------------------------------------------
	// 字幕
	bProcess, _, pipeResultMaxSub, err := s.ProcessBySubFileInfo(infoBase, srcBase)
	if err != nil {
		return false, nil, err
	}
	if bProcess == false {
		return false, nil, nil
	}

	targetSubEndTime := pkg.Time2SecondNumber(srcBase.GetEndTime())

	matchResult := &MatchResult{
		VideoDuration:          ffmpegInfo.Duration,
		TargetSubEndTime:       targetSubEndTime,
		AudioCompareScore:      pipeResultMaxAudio.Score,
		AudioCompareOffsetTime: pipeResultMaxAudio.GetOffsetTime(),
		SubCompareScore:        pipeResultMaxSub.Score,
		SubCompareOffsetTime:   pipeResultMaxSub.GetOffsetTime(),
	}
	// ---------------------------------------------------------------------------------------
	// 分数需要大于某个值
	if pipeResultMaxAudio.Score < config.MinScore || pipeResultMaxSub.Score < config.MinScore {
		return false, matchResult, nil
	}
	// 两种方式获取到的时间轴的偏移量，差值需要在一定范围内
	if math.Abs(pipeResultMaxAudio.GetOffsetTime()-pipeResultMaxSub.GetOffsetTime()) > config.OffsetRange {
		return false, matchResult, nil
	}
	// ---------------------------------------------------------------------------------------
	// 待判断的字幕的时间长度要小于等于视频的总长度
	if targetSubEndTime > ffmpegInfo.Duration {
		return false, matchResult, nil
	}
	// ---------------------------------------------------------------------------------------
	// 两个对比字幕的对白数量不能超过 10%
	minRage := float64(len(infoBase.Dialogues)) * config.DialoguesDifferencePercentage
	if math.Abs(float64(len(srcBase.Dialogues)-len(infoBase.Dialogues))) > minRage {
		return false, matchResult, nil
	}
	return true, matchResult, nil
}

func (s *SubTimelineFixerHelperEx) changeTimeLineAndSave(infoSrc *subparser.FileInfo, pipeResult sub_timeline_fixer.PipeResult, desSubSaveFPath string) error {
	/*
		修复的字幕先存放到缓存目录，然后需要把原有的字幕进行“备份”，改名，然后再替换过来
	*/
	subFileName := desSubSaveFPath + sub_timeline_fixer.TmpExt
	if pkg.IsFile(subFileName) == true {
		err := os.Remove(subFileName)
		if err != nil {
			return err
		}
	}
	_, err := s.timelineFixPipeLine.FixSubFileTimeline(infoSrc, pipeResult.ScaledFileInfo, pipeResult.GetOffsetTime(), subFileName)
	if err != nil {
		return err
	}

	if pkg.IsFile(desSubSaveFPath+sub_timeline_fixer.BackUpExt) == true {
		err = os.Remove(desSubSaveFPath + sub_timeline_fixer.BackUpExt)
		if err != nil {
			return err
		}
	}

	err = os.Rename(desSubSaveFPath, desSubSaveFPath+sub_timeline_fixer.BackUpExt)
	if err != nil {
		return err
	}

	err = os.Rename(subFileName, desSubSaveFPath)
	if err != nil {
		return err
	}

	return nil
}

func (s *SubTimelineFixerHelperEx) processWithFFSubSync(videoFileFullPath, srcSubFPath string) error {
	if s.fixerConfig.Engine != settings.TimelineFixerEngineFFSubSync {
		return fmt.Errorf("unsupported timeline fixer engine: %s", s.fixerConfig.Engine)
	}

	videoDuration := s.ffmpegHelper.GetVideoDuration(videoFileFullPath)
	logDir, err := os.MkdirTemp("", "csf-ffsubsync-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(logDir)

	tmpOutputPath := timelineFixTempOutputPath(srcSubFPath)
	if pkg.IsFile(tmpOutputPath) == true {
		if err := os.Remove(tmpOutputPath); err != nil {
			return err
		}
	}

	args := buildFFSubSyncArgs(videoFileFullPath, srcSubFPath, tmpOutputPath, logDir, s.fixerConfig.MaxOffsetTime)

	ffsubsyncBin, err := findFFSubSyncBin()
	if err != nil {
		return err
	}
	cmd := exec.Command(ffsubsyncBin, args...)
	output := bytes.NewBuffer(nil)
	cmd.Stdout = output
	cmd.Stderr = output
	if err := cmd.Run(); err != nil && pkg.IsFile(tmpOutputPath) == false {
		return fmt.Errorf("ffsubsync failed: %w; output=%s", err, strings.TrimSpace(output.String()))
	}
	if pkg.IsFile(tmpOutputPath) == false {
		return fmt.Errorf("ffsubsync produced no output: %s; output=%s", srcSubFPath, strings.TrimSpace(output.String()))
	}

	bFind, fixedInfo, parseErr := s.subParserHub.DetermineFileTypeFromFile(tmpOutputPath)
	if parseErr != nil {
		_ = os.Remove(tmpOutputPath)
		return parseErr
	}
	if bFind == false || fixedInfo == nil {
		_ = os.Remove(tmpOutputPath)
		return errors.New("timeline fixed subtitle could not be parsed")
	}
	if reason := invalidTimelineFixedSubtitleReason(fixedInfo, videoDuration); reason != "" {
		_ = os.Remove(tmpOutputPath)
		return errors.New("timeline fix produced invalid subtitle: " + reason)
	}

	offsetSeconds, err := s.estimateOffsetSeconds(srcSubFPath, tmpOutputPath)
	if err != nil {
		_ = os.Remove(tmpOutputPath)
		return err
	}
	if math.Abs(offsetSeconds) < s.fixerConfig.MinOffset {
		_ = os.Remove(tmpOutputPath)
		s.log.Infoln("Skip TimeLine Fix -- OffsetTime:", offsetSeconds, srcSubFPath)
		return nil
	}

	if pkg.IsFile(srcSubFPath+sub_timeline_fixer.BackUpExt) == true {
		if err := os.Remove(srcSubFPath + sub_timeline_fixer.BackUpExt); err != nil {
			_ = os.Remove(tmpOutputPath)
			return err
		}
	}
	if err := os.Rename(srcSubFPath, srcSubFPath+sub_timeline_fixer.BackUpExt); err != nil {
		_ = os.Remove(tmpOutputPath)
		return err
	}
	if err := os.Rename(tmpOutputPath, srcSubFPath); err != nil {
		_ = os.Rename(srcSubFPath+sub_timeline_fixer.BackUpExt, srcSubFPath)
		_ = os.Remove(tmpOutputPath)
		return err
	}

	s.log.Infoln("TimeLine Fix -- Engine:", s.fixerConfig.Engine, srcSubFPath)
	s.log.Infoln("Fix Offset:", offsetSeconds, srcSubFPath)
	s.log.Infoln("BackUp Org SubFile:", srcSubFPath+sub_timeline_fixer.BackUpExt)
	if trimmed := strings.TrimSpace(output.String()); trimmed != "" {
		s.log.Debugln("ffsubsync:", trimmed)
	}
	return nil
}

func buildFFSubSyncArgs(videoFileFullPath, srcSubFPath, tmpOutputPath, logDir string, maxOffsetSeconds int) []string {
	return []string{
		videoFileFullPath,
		"-i", srcSubFPath,
		"-o", tmpOutputPath,
		"--max-offset-seconds", fmt.Sprintf("%d", maxOffsetSeconds),
		"--log-dir-path", logDir,
	}
}

func (s *SubTimelineFixerHelperEx) estimateOffsetSeconds(srcSubFPath, fixedSubFPath string) (float64, error) {
	bFindSrc, srcInfo, err := s.subParserHub.DetermineFileTypeFromFile(srcSubFPath)
	if err != nil {
		return 0, err
	}
	bFindFixed, fixedInfo, err := s.subParserHub.DetermineFileTypeFromFile(fixedSubFPath)
	if err != nil {
		return 0, err
	}
	if bFindSrc == false || bFindFixed == false || srcInfo == nil || fixedInfo == nil {
		return 0, errors.New("unable to parse subtitles for offset estimation")
	}
	if len(srcInfo.Dialogues) == 0 || len(fixedInfo.Dialogues) == 0 {
		return 0, errors.New("subtitle has no dialogues for offset estimation")
	}
	srcStart := pkg.Time2SecondNumber(srcInfo.Dialogues[0].GetStartTime())
	fixedStart := pkg.Time2SecondNumber(fixedInfo.Dialogues[0].GetStartTime())
	return fixedStart - srcStart, nil
}

func getFFSubSyncVersion() (string, error) {
	ffsubsyncBin, err := findFFSubSyncBin()
	if err != nil {
		return "", err
	}
	cmd := exec.Command(ffsubsyncBin, "--version")
	buf := bytes.NewBuffer(nil)
	cmd.Stdout = buf
	cmd.Stderr = buf
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

func findFFSubSyncBin() (string, error) {
	candidates := []string{
		os.Getenv("CSF_FFSUBSYNC_BIN"),
		"ffsubsync",
		"/opt/csf-ocr/bin/ffsubsync",
		"/usr/local/bin/ffsubsync",
	}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		if filepath.IsAbs(candidate) {
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
			continue
		}
		if resolved, err := exec.LookPath(candidate); err == nil {
			return resolved, nil
		}
	}
	return "", errors.New("ffsubsync binary not found")
}

func timelineFixTempOutputPath(srcSubFPath string) string {
	ext := filepath.Ext(srcSubFPath)
	if ext == "" {
		return srcSubFPath + sub_timeline_fixer.TmpExt
	}
	return strings.TrimSuffix(srcSubFPath, ext) + sub_timeline_fixer.TmpExt + ext
}

type CompareConfig struct {
	MinScore                      float64 // 最低的分数
	OffsetRange                   float64 // 偏移量的范围
	DialoguesDifferencePercentage float64 // 两个字幕的对白字幕差异百分比
}

type MatchResult struct {
	VideoDuration          float64 // 视频的时长
	TargetSubEndTime       float64 // 目标字幕的结束时间
	AudioCompareScore      float64 // 音频的对比分数
	AudioCompareOffsetTime float64 // 音频的对比偏移量
	SubCompareScore        float64 // 字幕的对比分数
	SubCompareOffsetTime   float64 // 字幕的对比偏移量

}
