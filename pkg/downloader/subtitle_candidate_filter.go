package downloader

import (
	"math"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/ffmpeg_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/ass"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/srt"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_parser_hub"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/subparser"
)

const (
	maxAbsoluteSubtitleEndSeconds = 6 * 60 * 60
	maxSubtitleLeadOverVideo      = 15 * 60
	maxSubtitleVideoRatio         = 1.5
	minVectorGarbageLines         = 3
)

var subtitleVectorGarbagePattern = regexp.MustCompile(`(?i)^[mnlbspc]\s[-0-9]`)

func (d *Downloader) filterInvalidSubtitleCandidates(videoPath string, organizeSubFiles []string) []string {
	if len(organizeSubFiles) == 0 {
		return nil
	}

	videoDuration := ffmpeg_helper.NewFFMPEGHelper(d.log).GetVideoDuration(videoPath)
	parserHub := sub_parser_hub.NewSubParserHub(d.log, ass.NewParser(d.log), srt.NewParser(d.log))

	filtered := make([]string, 0, len(organizeSubFiles))
	for _, subPath := range organizeSubFiles {
		found, fileInfo, err := parserHub.DetermineFileTypeFromFile(subPath)
		if err != nil {
			d.log.Warningln("filterInvalidSubtitleCandidates.DetermineFileTypeFromFile", filepath.Base(subPath), err)
			continue
		}
		if found == false || fileInfo == nil {
			d.log.Warningln("filterInvalidSubtitleCandidates.SkipUnparsed", filepath.Base(subPath))
			continue
		}

		if reason := invalidSubtitleReason(fileInfo, videoDuration); reason != "" {
			d.log.Warningln("filterInvalidSubtitleCandidates.SkipInvalid",
				"video", filepath.Base(videoPath),
				"subtitle", filepath.Base(subPath),
				"site", fileInfo.FromWhereSite,
				"reason", reason)
			continue
		}

		filtered = append(filtered, subPath)
	}

	return filtered
}

func invalidSubtitleReason(fileInfo *subparser.FileInfo, videoDuration float64) string {
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
	if subEndSeconds > maxAbsoluteSubtitleEndSeconds {
		return "subtitle end time exceeds absolute threshold"
	}

	if videoDuration > 0 {
		maxAllowedEnd := math.Max(videoDuration*maxSubtitleVideoRatio, videoDuration+maxSubtitleLeadOverVideo)
		if subEndSeconds > maxAllowedEnd {
			return "subtitle end time exceeds video duration tolerance"
		}
	}

	vectorGarbageLines := 0
	for _, dialogue := range fileInfo.Dialogues {
		for _, line := range dialogue.Lines {
			if subtitleVectorGarbagePattern.MatchString(strings.TrimSpace(line)) {
				vectorGarbageLines++
				if vectorGarbageLines >= minVectorGarbageLines {
					return "subtitle contains vector drawing garbage"
				}
			}
		}
	}

	return ""
}
