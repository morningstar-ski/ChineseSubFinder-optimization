package mark_system

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/decode"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/ass"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/srt"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/ranking"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_parser_hub"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/subparser"
	"github.com/sirupsen/logrus"
)

// MarkingSystem 评价系统，解决字幕排序优先级问题
type MarkingSystem struct {
	log             *logrus.Logger
	subSiteSequence []string // 网站的优先级，从高到低
	SubTypePriority int      // 字幕格式的优先级
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
func (m MarkingSystem) SelectOneSubFile(organizeSubFiles []string, targetVideoFullPath ...string) *subparser.FileInfo {
	subInfoDict := m.parseSubFileInfo(organizeSubFiles)
	siteNames := make([]string, 0, len(subInfoDict))
	for siteName := range subInfoDict {
		siteNames = append(siteNames, siteName)
	}
	orderedSiteNames := common.OrderSubSiteNames(siteNames, m.subSiteSequence)
	targetVideo := ""
	if len(targetVideoFullPath) > 0 {
		targetVideo = targetVideoFullPath[0]
	}

	for round := 0; round < 4; round++ {
		candidates := make([]subparser.FileInfo, 0)
		for _, subSite := range orderedSiteNames {
			infos, ok := subInfoDict[subSite]
			if ok == false {
				continue
			}
			for _, info := range infos {
				if m.matchSelectRound(info, round) == false {
					continue
				}
				candidates = append(candidates, info)
			}
		}
		if len(candidates) == 0 {
			if targetVideo != "" {
				m.log.Debugln("MarkingSystem.SelectOneSubFile", filepath.Base(targetVideo), "round", round, "no candidates")
			}
			continue
		}
		if targetVideo != "" {
			return m.pickBestByVideo(candidates, targetVideo)
		}
		return &candidates[0]
	}

	if targetVideo != "" {
		m.log.Warningln("MarkingSystem.SelectOneSubFile", filepath.Base(targetVideo), "no subtitle passed language/type rounds", m.describeSubInfoDict(subInfoDict))
	}

	return nil
}

// SelectEachSiteTop1SubFile 每个网站最优的文件
func (m MarkingSystem) SelectEachSiteTop1SubFile(organizeSubFiles []string) ([]string, []subparser.FileInfo) {
	var finalSubFile *subparser.FileInfo
	var outSiteName = make([]string, 0)
	var outSubParserFileInfos = make([]subparser.FileInfo, 0)
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
			} else {
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

// parseSubFileInfo 从文件解析字幕信息
func (m MarkingSystem) parseSubFileInfo(organizeSubFiles []string) map[string][]subparser.FileInfo {
	var subInfoDict = make(map[string][]subparser.FileInfo)
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
		_, ok := subInfoDict[subFileInfo.FromWhereSite]
		if ok == false {
			subInfoDict[subFileInfo.FromWhereSite] = make([]subparser.FileInfo, 0)
		}
		subInfoDict[subFileInfo.FromWhereSite] = append(subInfoDict[subFileInfo.FromWhereSite], *subFileInfo)
	}
	return subInfoDict
}

func (m MarkingSystem) matchSelectRound(info subparser.FileInfo, round int) bool {
	if round == 0 {
		return sub_helper.SelectChineseBestBilingualSubtitle([]subparser.FileInfo{info}, m.SubTypePriority) != nil
	}
	if round == 1 {
		return sub_helper.SelectChineseBestSubtitle([]subparser.FileInfo{info}, m.SubTypePriority) != nil
	}
	if round == 2 {
		return sub_helper.SelectChineseBestBilingualSubtitle([]subparser.FileInfo{info}, 0) != nil
	}
	return sub_helper.SelectChineseBestSubtitle([]subparser.FileInfo{info}, 0) != nil
}

func (m MarkingSystem) pickBestByVideo(candidates []subparser.FileInfo, targetVideoFullPath string) *subparser.FileInfo {
	bestIndex := 0
	bestScore := m.scoreFileInfo(candidates[0], targetVideoFullPath)
	scoreLogs := []string{fmt.Sprintf("%s score=%d", m.describeFileInfo(candidates[0]), bestScore)}
	for i := 1; i < len(candidates); i++ {
		nowScore := m.scoreFileInfo(candidates[i], targetVideoFullPath)
		scoreLogs = append(scoreLogs, fmt.Sprintf("%s score=%d", m.describeFileInfo(candidates[i]), nowScore))
		if nowScore > bestScore {
			bestIndex = i
			bestScore = nowScore
		}
	}
	m.log.Infoln("MarkingSystem.SelectOneSubFile", filepath.Base(targetVideoFullPath), "candidates", strings.Join(scoreLogs, " | "), "selected", m.describeFileInfo(candidates[bestIndex]))
	return &candidates[bestIndex]
}

func (m MarkingSystem) scoreFileInfo(info subparser.FileInfo, targetVideoFullPath string) int {
	isMovie := true
	if videoInfo, err := decode.GetVideoInfoFromFileName(filepath.Base(targetVideoFullPath)); err == nil && videoInfo != nil {
		if videoInfo.Season > 0 || videoInfo.Episode > 0 {
			isMovie = false
		}
	}

	matcher := ranking.NewTargetMatcher(targetVideoFullPath, isMovie)
	return ranking.BaseScore(matcher, ranking.BaseScoreOptions{
		IsMovie:             isMovie,
		SubtitleExt:         info.Ext,
		ReleaseNames:        m.releaseNamesForScore(info),
		ReleaseMatchWeights: ranking.StandardReleaseMatchWeights,
		AuthorityScore:      m.sitePriorityScore(info.FromWhereSite),
	})
}

func (m MarkingSystem) releaseNamesForScore(info subparser.FileInfo) []string {
	fileName := filepath.Base(info.FileFullPath)
	return []string{
		info.Name,
		fileName,
		stripSitePrefix(info.Name),
		stripSitePrefix(fileName),
	}
}

func (m MarkingSystem) sitePriorityScore(siteName string) int {
	for i, oneSite := range m.subSiteSequence {
		if oneSite == siteName {
			score := len(m.subSiteSequence) - i
			if score < 1 {
				return 1
			}
			return score
		}
	}
	return 0
}

var sitePrefixRegex = regexp.MustCompile(`^\[\w+\]_\d+_`)

func stripSitePrefix(name string) string {
	return sitePrefixRegex.ReplaceAllString(name, "")
}

func (m MarkingSystem) describeSubInfoDict(subInfoDict map[string][]subparser.FileInfo) string {
	parts := make([]string, 0)
	for _, infos := range subInfoDict {
		for _, info := range infos {
			parts = append(parts, m.describeFileInfo(info))
		}
	}
	return strings.Join(parts, " | ")
}

func (m MarkingSystem) describeFileInfo(info subparser.FileInfo) string {
	return fmt.Sprintf("site=%s lang=%s ext=%s name=%s file=%s ch=%d other=%d",
		info.FromWhereSite,
		info.Lang.String(),
		info.Ext,
		info.Name,
		filepath.Base(info.FileFullPath),
		len(info.CHLines),
		len(info.OtherLines),
	)
}
