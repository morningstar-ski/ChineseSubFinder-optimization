package mark_system

import (
	"path/filepath"
	"strings"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/decode"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/ass"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/srt"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/ranking"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_parser_hub"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	language2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/subparser"
	"github.com/sirupsen/logrus"
)

// MarkingSystem 评估系统，解决字幕排序优先级问题
type MarkingSystem struct {
	log             *logrus.Logger
	subSiteSequence []string // 网站优先级，从高到低
	SubTypePriority int      // 字幕格式优先级
	subParserHub    *sub_parser_hub.SubParserHub
}

func NewMarkingSystem(log *logrus.Logger, subSiteSequence []string, subTypePriority int) *MarkingSystem {
	mk := MarkingSystem{
		subSiteSequence: subSiteSequence,
		log:             log,
		SubTypePriority: subTypePriority,
		subParserHub:    sub_parser_hub.NewSubParserHub(log, ass.NewParser(log), srt.NewParser(log)),
	}
	return &mk
}

// SelectOneSubFile 选择最优的一个字幕文件
func (m MarkingSystem) SelectOneSubFile(organizeSubFiles []string) *subparser.FileInfo {
	return m.SelectOneSubFileWithVideo(organizeSubFiles, "")
}

// SelectOneSubFileWithVideo 在已知目标视频时，过滤明显不匹配的中文字幕候选。
func (m MarkingSystem) SelectOneSubFileWithVideo(organizeSubFiles []string, targetVideoFullPath string) *subparser.FileInfo {
	subInfoDict := m.parseSubFileInfo(organizeSubFiles)
	siteNames := make([]string, 0, len(subInfoDict))
	for siteName := range subInfoDict {
		siteNames = append(siteNames, siteName)
	}
	orderedSiteNames := common.OrderSubSiteNames(siteNames, m.subSiteSequence)

	if strings.TrimSpace(targetVideoFullPath) == "" {
		return m.selectOneSubFileLegacy(subInfoDict, orderedSiteNames)
	}

	selectionPhases := []struct {
		bilingualOnly   bool
		subTypePriority int
	}{
		{bilingualOnly: true, subTypePriority: m.SubTypePriority},
		{bilingualOnly: false, subTypePriority: m.SubTypePriority},
		{bilingualOnly: true, subTypePriority: 0},
		{bilingualOnly: false, subTypePriority: 0},
	}

	for _, phase := range selectionPhases {
		finalSubFile := m.selectBestChineseCandidateForVideo(
			subInfoDict,
			orderedSiteNames,
			targetVideoFullPath,
			phase.bilingualOnly,
			phase.subTypePriority,
		)
		if finalSubFile != nil {
			return finalSubFile
		}
	}

	return nil
}

func (m MarkingSystem) selectOneSubFileLegacy(subInfoDict map[string][]subparser.FileInfo, orderedSiteNames []string) *subparser.FileInfo {
	var finalSubFile *subparser.FileInfo
	for i := 0; i < 4; i++ {
		for _, subSite := range orderedSiteNames {
			infos, ok := subInfoDict[subSite]
			if ok == false {
				continue
			}
			if i == 0 {
				finalSubFile = sub_helper.SelectChineseBestBilingualSubtitle(infos, m.SubTypePriority)
			} else if i == 1 {
				finalSubFile = sub_helper.SelectChineseBestSubtitle(infos, m.SubTypePriority)
			} else if i == 2 {
				finalSubFile = sub_helper.SelectChineseBestBilingualSubtitle(infos, 0)
			} else if i == 3 {
				finalSubFile = sub_helper.SelectChineseBestSubtitle(infos, 0)
			}
			if finalSubFile != nil {
				return finalSubFile
			}
		}
	}
	return nil
}

func (m MarkingSystem) selectBestChineseCandidateForVideo(subInfoDict map[string][]subparser.FileInfo, orderedSiteNames []string, targetVideoFullPath string, bilingualOnly bool, subTypePriority int) *subparser.FileInfo {
	targetName := filepath.Base(targetVideoFullPath)
	targetInfo, _ := decode.GetVideoInfoFromFileName(targetName)
	isMovie := targetInfo == nil || (targetInfo.Season == 0 && targetInfo.Episode == 0)
	targetSeason := 0
	targetEpisode := 0
	targetParsedTitle := ""
	if targetInfo != nil {
		targetSeason = targetInfo.Season
		targetEpisode = targetInfo.Episode
		targetParsedTitle = targetInfo.Title
	}
	matcher := ranking.NewTargetMatcher(targetVideoFullPath, isMovie)

	var best *subparser.FileInfo
	bestScore := 0
	hasBest := false

	for siteIndex, siteName := range orderedSiteNames {
		infos := subInfoDict[siteName]
		for idx := range infos {
			info := infos[idx]
			if isChineseCandidateForPhase(info, bilingualOnly, subTypePriority) == false {
				continue
			}
			if hasExplicitTitleMismatch(targetName, targetParsedTitle, candidateReleaseNames(info)) {
				continue
			}

			score := ranking.ScoreCandidate(matcher, subtitleCandidateMetadata(info, len(orderedSiteNames)-siteIndex), ranking.CandidateScoreSpec{
				IsMovie:       isMovie,
				TargetSeason:  targetSeason,
				TargetEpisode: targetEpisode,
				EpisodeMatchWeights: &ranking.EpisodeMatchWeights{
					ExactMatch:     40,
					SeasonPack:     10,
					WrongEpisode:   -35,
					SeasonMatch:    0,
					WrongSeason:    0,
					WrongEpisodeSB: 0,
				},
				SubTypePriority:     subTypePriority,
				HIPenalty:           -3,
				ReleaseMatchWeights: ranking.StandardReleaseMatchWeights,
			})

			if hasBest == false || score > bestScore {
				infoCopy := info
				best = &infoCopy
				bestScore = score
				hasBest = true
			}
		}
	}

	return best
}

// SelectEachSiteTop1SubFile 每个网站最优的文件
func (m MarkingSystem) SelectEachSiteTop1SubFile(organizeSubFiles []string) ([]string, []subparser.FileInfo) {
	var finalSubFile *subparser.FileInfo
	outSiteName := make([]string, 0)
	outSubParserFileInfos := make([]subparser.FileInfo, 0)
	subInfoDict := m.parseSubFileInfo(organizeSubFiles)
	siteNames := make([]string, 0, len(subInfoDict))
	for siteName := range subInfoDict {
		siteNames = append(siteNames, siteName)
	}
	orderedSiteNames := common.OrderSubSiteNames(siteNames, m.subSiteSequence)
	for _, siteName := range orderedSiteNames {
		infos := subInfoDict[siteName]
		for i := 0; i < 4; i++ {
			if i == 0 {
				finalSubFile = sub_helper.SelectChineseBestBilingualSubtitle(infos, m.SubTypePriority)
			} else if i == 1 {
				finalSubFile = sub_helper.SelectChineseBestSubtitle(infos, m.SubTypePriority)
			} else if i == 2 {
				finalSubFile = sub_helper.SelectChineseBestBilingualSubtitle(infos, 0)
			} else if i == 3 {
				finalSubFile = sub_helper.SelectChineseBestSubtitle(infos, 0)
			}
			if finalSubFile != nil {
				outSiteName = append(outSiteName, siteName)
				outSubParserFileInfos = append(outSubParserFileInfos, *finalSubFile)
				break
			}
		}
	}

	return outSiteName, outSubParserFileInfos
}

// SelectBestEnglishSubFile 在中文字幕候选失败后，选择最匹配视频版本的英文候选字幕。
func (m MarkingSystem) SelectBestEnglishSubFile(organizeSubFiles []string, targetVideoFullPath string) *subparser.FileInfo {
	subInfoDict := m.parseSubFileInfo(organizeSubFiles)
	siteNames := make([]string, 0, len(subInfoDict))
	for siteName := range subInfoDict {
		siteNames = append(siteNames, siteName)
	}
	orderedSiteNames := common.OrderSubSiteNames(siteNames, m.subSiteSequence)

	targetName := filepath.Base(targetVideoFullPath)
	targetInfo, _ := decode.GetVideoInfoFromFileName(targetName)
	isMovie := targetInfo == nil || (targetInfo.Season == 0 && targetInfo.Episode == 0)
	targetSeason := 0
	targetEpisode := 0
	if targetInfo != nil {
		targetSeason = targetInfo.Season
		targetEpisode = targetInfo.Episode
	}
	matcher := ranking.NewTargetMatcher(targetVideoFullPath, isMovie)

	var best *subparser.FileInfo
	bestScore := 0
	hasBest := false

	for siteIndex, siteName := range orderedSiteNames {
		infos := subInfoDict[siteName]
		for idx := range infos {
			info := infos[idx]
			if isEnglishFallbackCandidate(info) == false {
				continue
			}

			score := ranking.ScoreCandidate(matcher, subtitleCandidateMetadata(info, len(orderedSiteNames)-siteIndex), ranking.CandidateScoreSpec{
				IsMovie:       isMovie,
				TargetSeason:  targetSeason,
				TargetEpisode: targetEpisode,
				EpisodeMatchWeights: &ranking.EpisodeMatchWeights{
					ExactMatch:     40,
					SeasonPack:     10,
					WrongEpisode:   -35,
					SeasonMatch:    0,
					WrongSeason:    0,
					WrongEpisodeSB: 0,
				},
				SubTypePriority:     1,
				HIPenalty:           -3,
				ReleaseMatchWeights: ranking.StandardReleaseMatchWeights,
			})
			if hasBest == false || score > bestScore {
				infoCopy := info
				best = &infoCopy
				bestScore = score
				hasBest = true
			}
		}
	}

	return best
}

// parseSubFileInfo 从文件解析字幕信息
func (m MarkingSystem) parseSubFileInfo(organizeSubFiles []string) map[string][]subparser.FileInfo {
	subInfoDict := make(map[string][]subparser.FileInfo)
	for _, oneSubFileFullPath := range organizeSubFiles {
		bFind, subFileInfo, err := m.subParserHub.DetermineFileTypeFromFile(oneSubFileFullPath)
		if err != nil {
			m.log.Errorln("DetermineFileTypeFromFile", oneSubFileFullPath, err)
			continue
		}
		if bFind == false {
			m.log.Warnln("DetermineFileTypeFromFile", oneSubFileFullPath, "not support SubType")
			continue
		}
		if _, ok := subInfoDict[subFileInfo.FromWhereSite]; ok == false {
			subInfoDict[subFileInfo.FromWhereSite] = make([]subparser.FileInfo, 0)
		}
		subInfoDict[subFileInfo.FromWhereSite] = append(subInfoDict[subFileInfo.FromWhereSite], *subFileInfo)
	}
	return subInfoDict
}

func isEnglishFallbackCandidate(info subparser.FileInfo) bool {
	if language.HasChineseLang(info.Lang) {
		return false
	}
	if info.Lang == language2.English {
		return true
	}
	return len(info.OtherLines) > 0 && len(info.CHLines) == 0
}

func isChineseCandidateForPhase(info subparser.FileInfo, bilingualOnly bool, subTypePriority int) bool {
	if language.HasChineseLang(info.Lang) == false {
		return false
	}
	if bilingualOnly && language.IsBilingualSubtitle(info.Lang) == false {
		return false
	}
	if subTypePriority == 1 {
		return strings.EqualFold(info.Ext, common.SubExtSRT)
	}
	if subTypePriority == 2 {
		return strings.EqualFold(info.Ext, common.SubExtASS) || strings.EqualFold(info.Ext, common.SubExtSSA)
	}
	return true
}

func subtitleCandidateMetadata(info subparser.FileInfo, siteAuthority int) ranking.CandidateMetadata {
	candidate := ranking.CandidateMetadata{
		Name:           info.Name,
		ReleaseNames:   candidateReleaseNames(info),
		SubtitleExt:    strings.ToLower(info.Ext),
		AuthorityScore: siteAuthority * 10,
	}

	if parsed, err := decode.GetVideoInfoFromFileName(info.Name); err == nil && parsed != nil {
		candidate.Season = parsed.Season
		candidate.Episode = parsed.Episode
	}

	lowerName := strings.ToLower(info.Name)
	if strings.Contains(lowerName, ".hi.") || strings.Contains(lowerName, " sdh") || strings.Contains(lowerName, ".sdh.") || strings.Contains(lowerName, " hearing") {
		candidate.HasHI = true
	}

	return candidate
}

func candidateReleaseNames(info subparser.FileInfo) []string {
	names := make([]string, 0, 2)
	rawName := strings.TrimSpace(info.Name)
	if rawName == "" {
		return names
	}

	trimmed := rawName
	if strings.HasPrefix(trimmed, "[") {
		if siteEnd := strings.Index(trimmed, "]_"); siteEnd >= 0 {
			rest := trimmed[siteEnd+2:]
			if rankEnd := strings.Index(rest, "_"); rankEnd >= 0 && rankEnd+1 < len(rest) {
				trimmed = rest[rankEnd+1:]
			}
		}
	}

	if trimmed != "" && trimmed != rawName {
		names = append(names, trimmed)
	}
	return names
}

func hasExplicitTitleMismatch(targetName string, targetParsedTitle string, releaseNames []string) bool {
	targetTitle := normalizeSubtitleTitle(targetName)
	if parsedTitle := normalizeSubtitleTitle(targetParsedTitle); parsedTitle != "" {
		targetTitle = parsedTitle
	}
	for _, releaseName := range releaseNames {
		parsed, err := decode.GetVideoInfoFromFileName(releaseName)
		if err != nil || parsed == nil || parsed.Title == "" {
			continue
		}
		candidateTitle := normalizeSubtitleTitle(parsed.Title)
		if targetTitle != "" && candidateTitle != "" && targetTitle != candidateTitle {
			return true
		}
		return false
	}
	return false
}

func normalizeSubtitleTitle(input string) string {
	base := strings.TrimSuffix(filepath.Base(strings.TrimSpace(input)), filepath.Ext(strings.TrimSpace(input)))
	base = strings.ReplaceAll(base, "_", " ")
	base = strings.ReplaceAll(base, ".", " ")
	return strings.ToLower(strings.Join(strings.Fields(base), " "))
}
