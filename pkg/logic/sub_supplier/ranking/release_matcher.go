package ranking

import (
	"path/filepath"
	"strings"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/decode"
	subCommon "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	PTN "github.com/middelink/go-parse-torrent-name"
)

type ReleaseMatchWeights struct {
	TitleMatch      int
	TitleMismatch   int
	YearMatch       int
	YearMismatch    int
	Resolution      int
	Quality         int
	Group           int
	Codec           int
	Audio           int
	SeasonMatch     int
	SeasonMismatch  int
	EpisodeMatch    int
	EpisodeMismatch int
}

var StandardReleaseMatchWeights = ReleaseMatchWeights{
	TitleMatch:      35,
	TitleMismatch:   -25,
	YearMatch:       15,
	YearMismatch:    -10,
	Resolution:      25,
	Quality:         25,
	Group:           15,
	Codec:           10,
	Audio:           5,
	SeasonMatch:     10,
	SeasonMismatch:  -20,
	EpisodeMatch:    10,
	EpisodeMismatch: -30,
}

var SubDLReleaseMatchWeights = ReleaseMatchWeights{
	TitleMatch:      40,
	TitleMismatch:   -30,
	YearMatch:       20,
	YearMismatch:    -10,
	Resolution:      25,
	Quality:         25,
	Group:           15,
	Codec:           10,
	Audio:           5,
	SeasonMatch:     10,
	SeasonMismatch:  -20,
	EpisodeMatch:    10,
	EpisodeMismatch: -20,
}

type TargetMatcher struct {
	targetName string
	targetInfo *PTN.TorrentInfo
	isMovie    bool
}

func (m TargetMatcher) TargetName() string {
	return m.targetName
}

type EpisodeMatchWeights struct {
	ExactMatch     int
	SeasonPack     int
	WrongEpisode   int
	SeasonMatch    int
	WrongSeason    int
	WrongEpisodeSB int
}

type SeasonEpisodeWeights struct {
	SeasonMatch     int
	SeasonMismatch  int
	EpisodeMatch    int
	EpisodeMismatch int
}

type BaseScoreOptions struct {
	IsMovie              bool
	CandidateSeason      int
	CandidateEpisode     int
	TargetSeason         int
	TargetEpisode        int
	EpisodeMatchWeights  *EpisodeMatchWeights
	SeasonEpisodeWeights *SeasonEpisodeWeights
	SubtitleExt          string
	SubTypePriority      int
	Subtype              string
	HasHI                bool
	HIPenalty            int
	AuthorityScore       int
	ReleaseNames         []string
	ReleaseMatchWeights  ReleaseMatchWeights
}

type CandidateMetadata struct {
	Name           string
	ReleaseNames   []string
	Season         int
	Episode        int
	SubtitleExt    string
	Subtype        string
	HasHI          bool
	AuthorityScore int
}

type CandidateScoreSpec struct {
	IsMovie              bool
	TargetSeason         int
	TargetEpisode        int
	EpisodeMatchWeights  *EpisodeMatchWeights
	SeasonEpisodeWeights *SeasonEpisodeWeights
	SubTypePriority      int
	HIPenalty            int
	ReleaseMatchWeights  ReleaseMatchWeights
}

func NewTargetMatcher(videoFPath string, isMovie bool) TargetMatcher {
	targetName := filepath.Base(videoFPath)
	targetInfo, _ := decode.GetVideoInfoFromFileName(targetName)
	return TargetMatcher{
		targetName: targetName,
		targetInfo: targetInfo,
		isMovie:    isMovie,
	}
}

func (m TargetMatcher) BestScore(candidateNames []string, weights ReleaseMatchWeights) int {
	bestScore := 0
	found := false

	for _, candidateName := range candidateNames {
		if candidateName == "" {
			continue
		}
		parsed, err := decode.GetVideoInfoFromFileName(candidateName)
		if err != nil || parsed == nil {
			continue
		}

		score := scoreParsedRelease(parsed, m.targetInfo, m.targetName, m.isMovie, weights)
		if found == false || score > bestScore {
			bestScore = score
			found = true
		}
	}

	if found == false {
		return 0
	}
	return bestScore
}

func scoreParsedRelease(parsed *PTN.TorrentInfo, targetInfo *PTN.TorrentInfo, targetName string, isMovie bool, weights ReleaseMatchWeights) int {
	score := 0
	if parsed == nil {
		return score
	}

	targetTitle := normalizeTitle(targetName)
	candidateTitle := normalizeTitle(parsed.Title)
	if targetTitle != "" && candidateTitle != "" {
		if targetTitle == candidateTitle {
			score += weights.TitleMatch
		} else {
			score += weights.TitleMismatch
		}
	}

	if targetInfo == nil {
		return score
	}

	if targetInfo.Year != 0 && parsed.Year != 0 {
		if targetInfo.Year == parsed.Year {
			score += weights.YearMatch
		} else {
			score += weights.YearMismatch
		}
	}
	score += compareTag(targetInfo.Resolution, parsed.Resolution, weights.Resolution)
	score += compareTag(targetInfo.Quality, parsed.Quality, weights.Quality)
	score += compareTag(targetInfo.Group, parsed.Group, weights.Group)
	score += compareTag(targetInfo.Codec, parsed.Codec, weights.Codec)
	score += compareTag(targetInfo.Audio, parsed.Audio, weights.Audio)
	if isMovie == false && targetInfo.Season != 0 && parsed.Season != 0 {
		if targetInfo.Season == parsed.Season {
			score += weights.SeasonMatch
		} else {
			score += weights.SeasonMismatch
		}
	}
	if isMovie == false && targetInfo.Episode != 0 && parsed.Episode != 0 {
		if targetInfo.Episode == parsed.Episode {
			score += weights.EpisodeMatch
		} else {
			score += weights.EpisodeMismatch
		}
	}

	return score
}

func compareTag(target string, candidate string, weight int) int {
	if target == "" || candidate == "" || weight == 0 {
		return 0
	}
	if strings.EqualFold(target, candidate) {
		return weight
	}
	return -weight / 2
}

func normalizeTitle(input string) string {
	if input == "" {
		return ""
	}
	base := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	base = pkg.ReplaceSpecString(base, " ")
	return strings.ToLower(strings.Join(strings.Fields(base), " "))
}

func ScoreEpisodeMatch(candidateSeason int, candidateEpisode int, targetSeason int, targetEpisode int, weights EpisodeMatchWeights) int {
	score := 0
	if candidateSeason == 0 && candidateEpisode == 0 {
		return score
	}

	if candidateSeason == targetSeason && candidateEpisode == targetEpisode {
		return weights.ExactMatch
	}
	if candidateSeason == targetSeason && candidateEpisode == 0 {
		return weights.SeasonPack
	}
	if candidateEpisode != 0 {
		return weights.WrongEpisode
	}

	return score
}

func ScoreSeasonEpisodePair(candidateSeason int, candidateEpisode int, targetSeason int, targetEpisode int, isMovie bool, seasonMatch int, seasonMismatch int, episodeMatch int, episodeMismatch int) int {
	if isMovie {
		return 0
	}

	score := 0
	if targetSeason != 0 && candidateSeason != 0 {
		if targetSeason == candidateSeason {
			score += seasonMatch
		} else {
			score += seasonMismatch
		}
	}
	if targetEpisode != 0 && candidateEpisode != 0 {
		if targetEpisode == candidateEpisode {
			score += episodeMatch
		} else {
			score += episodeMismatch
		}
	}

	return score
}

func ScoreSubtitleExt(ext string, subTypePriority int) int {
	ext = strings.ToLower(ext)
	switch subTypePriority {
	case 1:
		if ext == subCommon.SubExtSRT {
			return 8
		}
	case 2:
		if ext == subCommon.SubExtASS || ext == subCommon.SubExtSSA {
			return 8
		}
	}
	return 0
}

func ScoreBilingualSubtype(subtype string) int {
	lowerSubtype := strings.ToLower(subtype)
	if strings.Contains(lowerSubtype, "bilingual") || strings.Contains(lowerSubtype, "dual") {
		return 5
	}
	return 0
}

func BaseScore(matcher TargetMatcher, options BaseScoreOptions) int {
	score := 0

	if options.EpisodeMatchWeights != nil && options.IsMovie == false {
		score += ScoreEpisodeMatch(
			options.CandidateSeason,
			options.CandidateEpisode,
			options.TargetSeason,
			options.TargetEpisode,
			*options.EpisodeMatchWeights,
		)
	}

	if options.SeasonEpisodeWeights != nil && options.IsMovie == false {
		score += ScoreSeasonEpisodePair(
			options.CandidateSeason,
			options.CandidateEpisode,
			options.TargetSeason,
			options.TargetEpisode,
			options.IsMovie,
			options.SeasonEpisodeWeights.SeasonMatch,
			options.SeasonEpisodeWeights.SeasonMismatch,
			options.SeasonEpisodeWeights.EpisodeMatch,
			options.SeasonEpisodeWeights.EpisodeMismatch,
		)
	}

	score += ScoreSubtitleExt(options.SubtitleExt, options.SubTypePriority)
	score += ScoreBilingualSubtype(options.Subtype)

	if options.HasHI {
		score += options.HIPenalty
	}

	score += options.AuthorityScore

	return score + matcher.BestScore(options.ReleaseNames, options.ReleaseMatchWeights)
}

func ScoreCandidate(matcher TargetMatcher, metadata CandidateMetadata, spec CandidateScoreSpec) int {
	return BaseScore(matcher, BaseScoreOptions{
		IsMovie:              spec.IsMovie,
		CandidateSeason:      metadata.Season,
		CandidateEpisode:     metadata.Episode,
		TargetSeason:         spec.TargetSeason,
		TargetEpisode:        spec.TargetEpisode,
		EpisodeMatchWeights:  spec.EpisodeMatchWeights,
		SeasonEpisodeWeights: spec.SeasonEpisodeWeights,
		SubtitleExt:          metadata.SubtitleExt,
		SubTypePriority:      spec.SubTypePriority,
		Subtype:              metadata.Subtype,
		HasHI:                metadata.HasHI,
		HIPenalty:            spec.HIPenalty,
		AuthorityScore:       metadata.AuthorityScore,
		ReleaseNames:         metadata.ReleaseNamesWithName(),
		ReleaseMatchWeights:  spec.ReleaseMatchWeights,
	})
}

func (m CandidateMetadata) ReleaseNamesWithName() []string {
	names := make([]string, 0, len(m.ReleaseNames)+1)
	if m.Name != "" {
		names = append(names, m.Name)
	}
	for _, releaseName := range m.ReleaseNames {
		if releaseName == "" {
			continue
		}
		names = append(names, releaseName)
	}
	return names
}
