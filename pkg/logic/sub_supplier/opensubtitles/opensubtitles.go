package opensubtitles

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/decode"
	langutil "github.com/ChineseSubFinder/ChineseSubFinder/pkg/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/ranking"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/mix_media_info"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	subCommon "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
)

type Supplier struct {
	log            *logrus.Logger
	fileDownloader *file_downloader.FileDownloader
	topic          int
	isAlive        bool
	api            *Api
	quotaExceeded  bool
}

func NewSupplier(fileDownloader *file_downloader.FileDownloader) *Supplier {
	sup := Supplier{}
	sup.log = fileDownloader.Log
	sup.fileDownloader = fileDownloader
	sup.topic = subCommon.DownloadSubsPerSite
	sup.isAlive = true
	sup.api = NewApi(
		settings.Get().AdvancedSettings.SuppliersSettings.OpenSubtitles.RootUrl,
		settings.Get().SubtitleSources.OpenSubtitlesSettings.ApiKey,
		settings.Get().SubtitleSources.OpenSubtitlesSettings.Username,
		settings.Get().SubtitleSources.OpenSubtitlesSettings.Password,
	)

	if settings.Get().AdvancedSettings.Topic > 0 && settings.Get().AdvancedSettings.Topic != sup.topic {
		sup.topic = settings.Get().AdvancedSettings.Topic
	}

	return &sup
}

func (s *Supplier) CheckAlive() (bool, int64) {
	if s.canUse() == false {
		s.isAlive = false
		return false, 0
	}

	startT := time.Now()
	client, err := pkg.NewHttpClient()
	if err != nil {
		s.log.Errorln(s.GetSupplierName(), "CheckAlive.NewHttpClient", err)
		s.isAlive = false
		return false, 0
	}

	if err := s.api.CheckAlive(client); err != nil {
		s.log.Errorln(s.GetSupplierName(), "CheckAlive.Login", err)
		s.isAlive = false
		return false, 0
	}

	s.isAlive = true
	return true, time.Since(startT).Milliseconds()
}

func (s *Supplier) IsAlive() bool {
	return s.isAlive
}

func (s *Supplier) OverDailyDownloadLimit() bool {
	if settings.Get().AdvancedSettings.SuppliersSettings.OpenSubtitles.DailyDownloadLimit == 0 {
		s.log.Warningln(s.GetSupplierName(), "DailyDownloadLimit is 0, will Skip Download")
		return true
	}
	return false
}

func (s *Supplier) GetLogger() *logrus.Logger {
	return s.log
}

func (s *Supplier) GetSupplierName() string {
	return subCommon.SubSiteOpenSubtitles
}

func (s *Supplier) GetSubListFromFile4Movie(filePath string) ([]supplier.SubInfo, error) {
	outSubInfos := make([]supplier.SubInfo, 0)
	if settings.Get().SubtitleSources.OpenSubtitlesSettings.Enabled == false {
		return outSubInfos, nil
	}
	if s.canUse() == false {
		return nil, errors.New("opensubtitles credentials are incomplete")
	}
	if s.quotaExceeded {
		return nil, errors.New("opensubtitles quota_exceeded: daily download quota exhausted")
	}

	return s.getSubListFromFile(filePath, true, 0, 0)
}

func (s *Supplier) GetSubListFromFile4Series(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	outSubInfos := make([]supplier.SubInfo, 0)
	if settings.Get().SubtitleSources.OpenSubtitlesSettings.Enabled == false {
		return outSubInfos, nil
	}
	if s.canUse() == false {
		return nil, errors.New("opensubtitles credentials are incomplete")
	}
	if s.quotaExceeded {
		return nil, errors.New("opensubtitles quota_exceeded: daily download quota exhausted")
	}

	return s.downloadSub4Series(seriesInfo)
}

func (s *Supplier) GetSubListFromFile4Anime(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	return s.GetSubListFromFile4Series(seriesInfo)
}

func (s *Supplier) downloadSub4Series(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	allSupplierSubInfo := make([]supplier.SubInfo, 0)
	var lastErr error

	for _, episodeInfo := range seriesInfo.NeedDlEpsKeyList {
		one, err := s.getSubListFromFile(episodeInfo.FileFullPath, false, episodeInfo.Season, episodeInfo.Episode)
		if err != nil {
			s.log.Errorln(s.GetSupplierName(), "getSubListFromFile", episodeInfo.FileFullPath, err)
			lastErr = err
			if isQuotaExceededOpenSubtitlesError(err) {
				break
			}
			continue
		}
		if one == nil {
			s.log.Infoln(s.GetSupplierName(), "Not Find Sub can be download", episodeInfo.Title, episodeInfo.Season, episodeInfo.Episode)
			continue
		}
		for i := range one {
			one[i].Season = episodeInfo.Season
			one[i].Episode = episodeInfo.Episode
		}
		allSupplierSubInfo = append(allSupplierSubInfo, one...)
	}

	if len(allSupplierSubInfo) == 0 && lastErr != nil {
		return nil, lastErr
	}
	return allSupplierSubInfo, nil
}

func (s *Supplier) getSubListFromFile(videoFPath string, isMovie bool, season, episode int) ([]supplier.SubInfo, error) {
	defer func() {
		s.log.Debugln(s.GetSupplierName(), videoFPath, "End...")
	}()
	s.log.Debugln(s.GetSupplierName(), videoFPath, "Start...")

	mediaInfo, err := mix_media_info.GetMixMediaInfo(s.fileDownloader.MediaInfoDealers, videoFPath, isMovie)
	if err != nil {
		s.log.Errorln(s.GetSupplierName(), videoFPath, "GetMixMediaInfo", err)
		return nil, err
	}

	client, err := pkg.NewHttpClient()
	if err != nil {
		s.log.Errorln(s.GetSupplierName(), "NewHttpClient", err)
		return nil, err
	}

	candidates, err := s.searchCandidatesWithFallback(client, mediaInfo, videoFPath, isMovie, season, episode)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	videoFileName := filepath.Base(videoFPath)
	outSubInfoList := make([]supplier.SubInfo, 0)
	var lastDownloadErr error
	for index, candidate := range candidates {
		downloadInfo, err := s.api.DownloadByFileID(client, candidate.FileID)
		if err != nil {
			s.log.Errorln(s.GetSupplierName(), "DownloadByFileID", candidate.FileID, err)
			lastDownloadErr = err
			if isQuotaExceededOpenSubtitlesError(err) {
				s.quotaExceeded = true
				lastDownloadErr = fmt.Errorf("opensubtitles quota_exceeded: %w", err)
				break
			}
			continue
		}

		cacheKey := fmt.Sprintf("%s-%d", s.GetSupplierName(), candidate.FileID)
		subName := firstNonEmpty(candidate.FileName, candidate.Name, videoFileName)
		subInfo, err := s.fileDownloader.Get(s.GetSupplierName(), int64(index), subName, downloadInfo.Link, 0, 0, cacheKey)
		if err != nil {
			s.log.Errorln(s.GetSupplierName(), "FileDownloader.Get", err)
			lastDownloadErr = err
			continue
		}
		subInfo.Season = season
		subInfo.Episode = episode
		outSubInfoList = append(outSubInfoList, *subInfo)
		if len(outSubInfoList) >= s.topic {
			return outSubInfoList, nil
		}
	}

	return finalizeDownloadAttempts(outSubInfoList, lastDownloadErr)
}

func (s *Supplier) searchCandidatesWithFallback(client *resty.Client, mediaInfo *models.MediaInfo, videoFPath string, isMovie bool, season, episode int) ([]subtitleCandidate, error) {
	queryList := buildSearchQueries(mediaInfo, videoFPath, isMovie, season, episode)
	videoFileName := filepath.Base(videoFPath)
	for _, query := range queryList {
		s.log.Infoln(s.GetSupplierName(), videoFileName, "Try Search Query", query)
		searchResponse, err := s.api.SearchSubtitles(client, query)
		if err != nil {
			if isIgnorableOpenSubtitlesSearchError(err) {
				s.log.Warningln(s.GetSupplierName(), videoFileName, "SearchSubtitles ignore and continue", err)
				continue
			}
			s.log.Errorln(s.GetSupplierName(), videoFileName, "SearchSubtitles", err)
			return nil, err
		}

		candidates := selectCandidates(
			searchResponse.Data,
			mediaInfo,
			videoFPath,
			isMovie,
			season,
			episode,
			s.topic,
			settings.Get().AdvancedSettings.SubTypePriority,
		)
		if len(candidates) == 0 {
			continue
		}

		return candidates, nil
	}

	return nil, nil
}

func (s *Supplier) canUse() bool {
	cfg := settings.Get().SubtitleSources.OpenSubtitlesSettings
	return cfg.ApiKey != "" && cfg.Username != "" && cfg.Password != ""
}

func buildSearchQueries(mediaInfo *models.MediaInfo, videoFPath string, isMovie bool, season, episode int) []map[string]string {
	base := map[string]string{
		"order_by":        "download_count",
		"order_direction": "desc",
	}
	if isMovie == false {
		base["season_number"] = strconv.Itoa(season)
		base["episode_number"] = strconv.Itoa(episode)
	}

	out := make([]map[string]string, 0)
	if imdbID := normalizeOpenSubtitlesIMDbID(mediaInfo.ImdbId); imdbID != "" {
		out = append(out, cloneQueryMap(base, map[string]string{"imdb_id": imdbID}))
	}
	if mediaInfo.TmdbId != "" {
		out = append(out, cloneQueryMap(base, map[string]string{"tmdb_id": mediaInfo.TmdbId}))
	}

	year := ""
	if isMovie {
		year = normalizeYear(mediaInfo.Year)
	}
	for _, title := range openSubtitlesOrderedTitles(mediaInfo, videoFPath) {
		if title == "" {
			continue
		}
		if isLikelyTooShortOpenSubtitlesTextQuery(title) {
			continue
		}
		query := map[string]string{"query": title}
		if isMovie && year != "" && titleLooksYearSpecific(title) {
			query["year"] = year
		}
		out = append(out, cloneQueryMap(base, query))
	}

	return dedupeQueryMaps(out)
}

func selectCandidates(items []SearchItem, mediaInfo *models.MediaInfo, videoFPath string, isMovie bool, season, episode, limit, subTypePriority int) []subtitleCandidate {
	out := make([]subtitleCandidate, 0)
	seen := make(map[int64]struct{})

	for _, item := range items {
		candidate, ok := searchItemToCandidate(item, isMovie)
		if ok == false {
			continue
		}
		if _, found := seen[candidate.FileID]; found {
			continue
		}
		if candidateLikelyMatchesTarget(candidate, mediaInfo, videoFPath, isMovie) == false {
			continue
		}
		seen[candidate.FileID] = struct{}{}
		out = append(out, candidate)
	}

	rankCandidates(out, videoFPath, isMovie, season, episode, subTypePriority)
	if len(out) > limit {
		return out[:limit]
	}
	return out
}

func searchItemToCandidate(item SearchItem, isMovie bool) (subtitleCandidate, bool) {
	attrs := item.Attributes
	if isChineseAttribute(attrs) == false {
		return subtitleCandidate{}, false
	}

	file, ok := firstValidFile(attrs.Files)
	if ok == false {
		return subtitleCandidate{}, false
	}

	candidate := subtitleCandidate{
		FileID:       file.FileID,
		Name:         firstNonEmpty(file.FileName, attrs.Release, attrs.MovieName, attrs.FeatureDetails.Title),
		FileName:     file.FileName,
		ReleaseNames: compactStrings(attrs.Release, file.FileName, attrs.MovieName),
		FeatureTitle: attrs.FeatureDetails.Title,
		Year:         attrs.FeatureDetails.Year,
		Season:       attrs.FeatureDetails.SeasonNumber,
		Episode:      attrs.FeatureDetails.EpisodeNumber,
		Ext:          normalizeSubtitleExt(attrs.SubFormat, file.FileName),
		HasHI:        attrs.HearingImpaired,
	}

	if isMovie == false && candidate.Season == 0 && candidate.Episode == 0 {
		if _, parsedSeason, parsedEpisode, err := decode.GetSeasonAndEpisodeFromSubFileName(file.FileName); err == nil {
			candidate.Season = parsedSeason
			candidate.Episode = parsedEpisode
		}
	}

	return candidate, true
}

func candidateLikelyMatchesTarget(candidate subtitleCandidate, mediaInfo *models.MediaInfo, videoFPath string, isMovie bool) bool {
	expectedTitles := openSubtitlesExpectedTitles(mediaInfo, videoFPath)
	if len(expectedTitles) == 0 {
		return true
	}

	candidateTitles := compactStrings(candidate.FeatureTitle, candidate.Name, candidate.FileName)
	candidateTitles = append(candidateTitles, candidate.ReleaseNames...)
	titleMatched := false
	for _, expectedTitle := range expectedTitles {
		for _, candidateTitle := range candidateTitles {
			if titlesRoughlyMatch(expectedTitle, candidateTitle) {
				titleMatched = true
				break
			}
		}
		if titleMatched {
			break
		}
	}
	if titleMatched == false {
		return false
	}

	if isMovie == false {
		return true
	}

	targetYear := normalizeYear("")
	if mediaInfo != nil {
		targetYear = normalizeYear(mediaInfo.Year)
	}
	if targetYear == "" || candidate.Year == 0 {
		return true
	}

	return targetYear == strconv.Itoa(candidate.Year)
}

func rankCandidates(candidates []subtitleCandidate, videoFPath string, isMovie bool, season, episode, subTypePriority int) {
	if len(candidates) < 2 {
		return
	}

	matcher := ranking.NewTargetMatcher(videoFPath, isMovie)
	sort.SliceStable(candidates, func(i, j int) bool {
		left := scoreCandidate(candidates[i], matcher, isMovie, season, episode, subTypePriority)
		right := scoreCandidate(candidates[j], matcher, isMovie, season, episode, subTypePriority)
		if left != right {
			return left > right
		}
		return candidates[i].FileID < candidates[j].FileID
	})
}

func scoreCandidate(candidate subtitleCandidate, matcher ranking.TargetMatcher, isMovie bool, season, episode, subTypePriority int) int {
	return ranking.ScoreCandidate(matcher, openSubtitlesCandidateMetadata(candidate), ranking.CandidateScoreSpec{
		IsMovie:       isMovie,
		TargetSeason:  season,
		TargetEpisode: episode,
		EpisodeMatchWeights: &ranking.EpisodeMatchWeights{
			ExactMatch:   120,
			SeasonPack:   25,
			WrongEpisode: -120,
		},
		SubTypePriority:     subTypePriority,
		HIPenalty:           -5,
		ReleaseMatchWeights: ranking.StandardReleaseMatchWeights,
	})
}

func openSubtitlesCandidateMetadata(candidate subtitleCandidate) ranking.CandidateMetadata {
	return ranking.CandidateMetadata{
		Name:         candidate.Name,
		ReleaseNames: append([]string(nil), candidate.ReleaseNames...),
		Season:       candidate.Season,
		Episode:      candidate.Episode,
		SubtitleExt:  candidate.Ext,
		HasHI:        candidate.HasHI,
	}
}

func isChineseAttribute(attrs SearchItemAttribute) bool {
	for _, item := range []string{attrs.Language, attrs.ISO639} {
		lower := strings.ToLower(strings.TrimSpace(item))
		if lower == "" {
			continue
		}
		if strings.HasPrefix(lower, "zh") || strings.Contains(lower, "chinese") || strings.Contains(lower, "中文") {
			return true
		}
		if langutil.IsSupportISOChineseString(lower) {
			return true
		}
	}
	return false
}

func firstValidFile(files []SearchFile) (SearchFile, bool) {
	for _, file := range files {
		if file.FileID > 0 {
			return file, true
		}
	}
	return SearchFile{}, false
}

func normalizeOpenSubtitlesIMDbID(imdbID string) string {
	imdbID = strings.TrimSpace(strings.ToLower(imdbID))
	imdbID = strings.TrimPrefix(imdbID, "tt")
	return imdbID
}

func openSubtitlesOrderedTitles(mediaInfo *models.MediaInfo, videoFPath string) []string {
	videoTitle := normalizeVideoTitle(videoFPath)
	return mix_media_info.ExpandSearchKeywords(
		mediaInfo.TitleEn,
		mediaInfo.TitleCn,
		mediaInfo.OriginalTitle,
		videoTitle,
		stripTrailingYear(mediaInfo.TitleEn),
		stripTrailingYear(mediaInfo.TitleCn),
		stripTrailingYear(mediaInfo.OriginalTitle),
		stripTrailingYear(videoTitle),
	)
}

func openSubtitlesExpectedTitles(mediaInfo *models.MediaInfo, videoFPath string) []string {
	if mediaInfo == nil {
		return compactStrings(normalizeVideoTitle(videoFPath))
	}
	return compactStrings(
		mediaInfo.TitleEn,
		mediaInfo.TitleCn,
		mediaInfo.OriginalTitle,
		normalizeVideoTitle(videoFPath),
	)
}

func finalizeDownloadAttempts(subInfos []supplier.SubInfo, lastDownloadErr error) ([]supplier.SubInfo, error) {
	if len(subInfos) > 0 {
		return subInfos, nil
	}
	if lastDownloadErr != nil {
		return nil, lastDownloadErr
	}
	return nil, nil
}

func isQuotaExceededOpenSubtitlesError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "http 406") ||
		strings.Contains(lower, "downloaded your allowed") ||
		strings.Contains(lower, "quota will be renewed")
}

func normalizeVideoTitle(videoFPath string) string {
	fileName := strings.TrimSuffix(filepath.Base(videoFPath), filepath.Ext(videoFPath))
	if fileInfo, err := decode.GetVideoInfoFromFileName(fileName); err == nil && fileInfo != nil && fileInfo.Title != "" {
		fileName = fileInfo.Title
	}
	fileName = pkg.ReplaceSpecString(fileName, " ")
	return strings.Join(strings.Fields(fileName), " ")
}

func normalizeYear(year string) string {
	if len(year) >= 4 {
		return year[:4]
	}
	return ""
}

func titleLooksYearSpecific(title string) bool {
	title = strings.TrimSpace(title)
	if title == "" {
		return false
	}
	return title != stripTrailingYear(title)
}

func isIgnorableOpenSubtitlesSearchError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "query is too short")
}

func isLikelyTooShortOpenSubtitlesTextQuery(title string) bool {
	title = normalizeLooseTitle(title)
	if title == "" {
		return true
	}

	runes := []rune(strings.ReplaceAll(title, " ", ""))
	if len(runes) >= 3 {
		return false
	}

	for _, r := range runes {
		if unicode.IsLetter(r) && r <= unicode.MaxASCII {
			return false
		}
		if unicode.IsDigit(r) && r <= unicode.MaxASCII {
			return false
		}
	}

	return true
}

func stripTrailingYear(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"(", " ", ")", " ",
		"[", " ", "]", " ",
	)
	parts := strings.Fields(replacer.Replace(title))
	if len(parts) == 0 {
		return ""
	}
	last := parts[len(parts)-1]
	if len(last) == 4 {
		if _, err := strconv.Atoi(last); err == nil {
			return strings.TrimSpace(strings.Join(parts[:len(parts)-1], " "))
		}
	}
	return title
}

func normalizeSubtitleExt(subFormat string, fileName string) string {
	if ext := strings.ToLower(strings.TrimSpace(filepath.Ext(fileName))); ext != "" {
		return ext
	}
	subFormat = strings.TrimSpace(strings.ToLower(subFormat))
	if subFormat == "" {
		return ""
	}
	if strings.HasPrefix(subFormat, ".") {
		return subFormat
	}
	return "." + subFormat
}

func cloneQueryMap(base map[string]string, extra map[string]string) map[string]string {
	out := make(map[string]string, len(base)+len(extra))
	for key, value := range base {
		out[key] = value
	}
	for key, value := range extra {
		out[key] = value
	}
	return out
}

func dedupeQueryMaps(items []map[string]string) []map[string]string {
	out := make([]map[string]string, 0, len(items))
	seen := make(map[string]struct{})
	for _, item := range items {
		key := item["imdb_id"] + "|" + item["tmdb_id"] + "|" + item["query"] + "|" + item["year"] + "|" + item["season_number"] + "|" + item["episode_number"]
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func firstNonEmpty(items ...string) string {
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			return item
		}
	}
	return ""
}

func compactStrings(items ...string) []string {
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{})
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key := strings.ToLower(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func titlesRoughlyMatch(expected string, candidate string) bool {
	expected = normalizeLooseTitle(expected)
	candidate = normalizeLooseTitle(candidate)
	if expected == "" || candidate == "" {
		return false
	}
	if expected == candidate {
		return true
	}
	if len(expected) >= 6 && strings.Contains(candidate, expected) {
		return true
	}
	if len(candidate) >= 6 && strings.Contains(expected, candidate) {
		return true
	}
	return false
}

func normalizeLooseTitle(input string) string {
	input = mix_media_info.NormalizeComparableTitle(input)
	if input == "" {
		return ""
	}

	var builder strings.Builder
	lastSpace := false
	for _, r := range input {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			builder.WriteRune(r)
			lastSpace = false
		case unicode.IsSpace(r):
			if lastSpace == false {
				builder.WriteByte(' ')
				lastSpace = true
			}
		}
	}

	return strings.TrimSpace(builder.String())
}
