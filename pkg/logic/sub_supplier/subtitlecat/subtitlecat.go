package subtitlecat

import (
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/decode"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/ranking"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/mix_media_info"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	common2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	language2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
)

type subtitleCatMode string

const (
	subtitleCatModeEnglish           subtitleCatMode = "english"
	subtitleCatModeTranslatedChinese subtitleCatMode = "translated_chinese"
	subtitleCatSearchRetryCount                      = 2
)

type Supplier struct {
	log            *logrus.Logger
	fileDownloader *file_downloader.FileDownloader
	topic          int
	isAlive        bool
	mode           subtitleCatMode
}

type searchResult struct {
	title          string
	detailURL      string
	translatedFrom string
	downloads      int
	languages      int
}

type subtitleCandidate struct {
	name           string
	detailURL      string
	downloadURL    string
	translatedFrom string
	downloads      int
	languages      int
}

func NewEnglishSupplier(fileDownloader *file_downloader.FileDownloader) *Supplier {
	return newSupplier(fileDownloader, subtitleCatModeEnglish)
}

func NewTranslatedChineseSupplier(fileDownloader *file_downloader.FileDownloader) *Supplier {
	return newSupplier(fileDownloader, subtitleCatModeTranslatedChinese)
}

func newSupplier(fileDownloader *file_downloader.FileDownloader, mode subtitleCatMode) *Supplier {
	sup := Supplier{
		log:            fileDownloader.Log,
		fileDownloader: fileDownloader,
		topic:          common2.DownloadSubsPerSite,
		isAlive:        true,
		mode:           mode,
	}
	if settings.Get().AdvancedSettings.Topic > 0 && settings.Get().AdvancedSettings.Topic != sup.topic {
		sup.topic = settings.Get().AdvancedSettings.Topic
	}
	return &sup
}

func (s *Supplier) CheckAlive() (bool, int64) {
	startT := time.Now()
	client, err := pkg.NewHttpClient()
	if err != nil {
		s.log.Errorln(s.GetSupplierName(), "CheckAlive.NewHttpClient", err)
		s.isAlive = false
		return false, 0
	}

	resp, err := client.R().Get(settings.Get().AdvancedSettings.SuppliersSettings.SubtitleCat.RootUrl)
	if err != nil {
		s.log.Errorln(s.GetSupplierName(), "CheckAlive.Get", err)
		s.isAlive = false
		return false, 0
	}
	if resp.StatusCode() != http.StatusOK {
		s.log.Errorln(s.GetSupplierName(), "CheckAlive.StatusCode", resp.StatusCode())
		s.isAlive = false
		return false, 0
	}

	s.isAlive = true
	return true, time.Since(startT).Milliseconds()
}

func (s *Supplier) IsAlive() bool {
	return s.isAlive
}

func (s *Supplier) GetSupplierName() string {
	if s.mode == subtitleCatModeTranslatedChinese {
		return common2.SubSiteSubtitleCatTrans
	}
	return common2.SubSiteSubtitleCat
}

func (s *Supplier) OverDailyDownloadLimit() bool {
	if settings.Get().AdvancedSettings.SuppliersSettings.SubtitleCat.DailyDownloadLimit == 0 {
		s.log.Warningln(s.GetSupplierName(), "DailyDownloadLimit is 0, will Skip Download")
		return true
	}
	return false
}

func (s *Supplier) GetLogger() *logrus.Logger {
	return s.log
}

func (s *Supplier) GetSubListFromFile4Movie(filePath string) ([]supplier.SubInfo, error) {
	if s.mode == subtitleCatModeTranslatedChinese {
		cfg := settings.Get().SubtitleSources.SubtitleCatSettings
		if cfg == nil || cfg.EnableTranslatedChineseFallback == false {
			return nil, nil
		}
	}
	return s.getSubListFromFile(filePath, true, 0, 0)
}

func (s *Supplier) GetSubListFromFile4Series(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	if s.mode == subtitleCatModeTranslatedChinese {
		cfg := settings.Get().SubtitleSources.SubtitleCatSettings
		if cfg == nil || cfg.EnableTranslatedChineseFallback == false {
			return nil, nil
		}
	}
	return s.downloadSub4Series(seriesInfo)
}

func (s *Supplier) GetSubListFromFile4Anime(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	return s.GetSubListFromFile4Series(seriesInfo)
}

func (s *Supplier) downloadSub4Series(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	allSupplierSubInfo := make([]supplier.SubInfo, 0)

	for _, episodeInfo := range seriesInfo.NeedDlEpsKeyList {
		one, err := s.getSubListFromFile(episodeInfo.FileFullPath, false, episodeInfo.Season, episodeInfo.Episode)
		if err != nil {
			s.log.Errorln(s.GetSupplierName(), "getSubListFromFile", episodeInfo.FileFullPath, err)
			continue
		}
		if one == nil {
			continue
		}
		for i := range one {
			one[i].Season = episodeInfo.Season
			one[i].Episode = episodeInfo.Episode
		}
		allSupplierSubInfo = append(allSupplierSubInfo, one...)
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
		s.log.Warningln(s.GetSupplierName(), videoFPath, "GetMixMediaInfo", err, "fallback to title-based search")
		mediaInfo = nil
	}

	client, err := pkg.NewHttpClient()
	if err != nil {
		return nil, err
	}

	candidates, err := s.resolveCandidatesWithFallback(client, mediaInfo, videoFPath, isMovie, season, episode)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	videoFileName := filepath.Base(videoFPath)
	outSubInfos := make([]supplier.SubInfo, 0, len(candidates))
	for index, candidate := range candidates {
		cacheKey := file_downloader.BuildCacheKey(s.GetSupplierName(), candidate.downloadURL)
		subInfo, err := s.fileDownloader.Get(s.GetSupplierName(), int64(index), videoFileName, candidate.downloadURL, int64(candidate.downloads), 0, cacheKey)
		if err != nil {
			s.log.Errorln(s.GetSupplierName(), "FileDownloader.Get", candidate.downloadURL, err)
			continue
		}

		if s.mode == subtitleCatModeTranslatedChinese {
			subInfo.SetFileUrlSha256(file_downloader.BuildCacheKey(s.GetSupplierName(), candidate.detailURL, candidate.downloadURL))
			subInfo.FromWhere = common2.SubSiteSubtitleCatTrans
			subInfo.Language = language2.ChineseSimple
		} else {
			subInfo.FromWhere = common2.SubSiteSubtitleCat
			subInfo.Language = language2.English
		}

		outSubInfos = append(outSubInfos, *subInfo)
		if len(outSubInfos) >= s.topic {
			break
		}
	}

	return outSubInfos, nil
}

func (s *Supplier) resolveCandidatesWithFallback(client *resty.Client, mediaInfo *models.MediaInfo, videoFPath string, isMovie bool, season, episode int) ([]subtitleCandidate, error) {
	for _, keyword := range buildSearchKeywords(mediaInfo, videoFPath, isMovie, season, episode) {
		if keyword == "" {
			continue
		}

		results, err := s.search(client, keyword)
		if err != nil {
			s.log.Warningln(s.GetSupplierName(), "search", keyword, err)
			continue
		}
		candidates, err := s.filterCandidates(client, results)
		if err != nil {
			s.log.Warningln(s.GetSupplierName(), "filterCandidates", keyword, err)
			continue
		}
		candidates = filterLowConfidenceCandidates(candidates, mediaInfo, videoFPath, isMovie, season, episode)
		rankCandidates(candidates, videoFPath, isMovie, season, episode)
		if len(candidates) == 0 {
			continue
		}
		if len(candidates) > s.topic {
			return candidates[:s.topic], nil
		}
		return candidates, nil
	}

	return nil, nil
}

func filterLowConfidenceCandidates(candidates []subtitleCandidate, mediaInfo *models.MediaInfo, videoFPath string, isMovie bool, season, episode int) []subtitleCandidate {
	if len(candidates) == 0 {
		return candidates
	}

	matcher := ranking.NewTargetMatcher(videoFPath, isMovie)
	targetTitles := candidateTargetTitles(mediaInfo, videoFPath)
	filtered := make([]subtitleCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if len(targetTitles) > 0 && candidateTitleMatchesTargets(candidate, targetTitles) == false {
			continue
		}
		if isMovie == false && season > 0 && episode > 0 && seriesEpisodeMatchesTarget(candidate, season, episode) == false {
			continue
		}
		releaseScore := matcher.BestScore(candidateMetadata(candidate).ReleaseNamesWithName(), ranking.StandardReleaseMatchWeights)
		if isMovie && releaseScore < 0 {
			continue
		}
		filtered = append(filtered, candidate)
	}

	return filtered
}

func seriesEpisodeMatchesTarget(candidate subtitleCandidate, targetSeason, targetEpisode int) bool {
	meta := candidateMetadata(candidate)
	score := ranking.ScoreEpisodeMatch(meta.Season, meta.Episode, targetSeason, targetEpisode, ranking.EpisodeMatchWeights{
		ExactMatch:   120,
		SeasonPack:   15,
		WrongEpisode: -120,
	})
	return score > 0
}

func candidateTargetTitles(mediaInfo *models.MediaInfo, videoFPath string) []string {
	items := make([]string, 0, 5)
	if mediaInfo != nil {
		items = append(items, mediaInfo.TitleEn, mediaInfo.OriginalTitle, mediaInfo.TitleCn)
	}
	items = append(items,
		getLocalNfoOriginalTitle(videoFPath),
		getLocalNfoTitle(videoFPath),
		normalizeVideoTitle(videoFPath),
	)
	return compactNonEmptyTitles(items...)
}

func candidateTitleMatchesTargets(candidate subtitleCandidate, targetTitles []string) bool {
	if len(targetTitles) == 0 {
		return true
	}

	candidateTitles := compactNonEmptyTitles(extractCandidateComparableTitle(candidate.name), candidate.name)
	for _, candidateTitle := range candidateTitles {
		for _, targetTitle := range targetTitles {
			if scoreComparableTitle(candidateTitle, targetTitle) > 0 {
				return true
			}
		}
	}
	return false
}

func extractCandidateComparableTitle(input string) string {
	parsed, err := decode.GetVideoInfoFromFileName(input)
	if err == nil && parsed != nil && strings.TrimSpace(parsed.Title) != "" {
		return parsed.Title
	}
	return input
}

func scoreComparableTitle(left string, right string) int {
	left = normalizeComparableTitle(left)
	right = normalizeComparableTitle(right)
	if left == "" || right == "" {
		return 0
	}
	switch {
	case left == right:
		return 100
	case strings.Contains(left, right):
		return 60
	case strings.Contains(right, left):
		return 40
	default:
		return 0
	}
}

func normalizeComparableTitle(title string) string {
	parsed, err := decode.GetVideoInfoFromFileName(title)
	if err == nil && parsed != nil && strings.TrimSpace(parsed.Title) != "" {
		title = parsed.Title
	}
	title = pkg.ReplaceSpecString(title, " ")
	return strings.ToLower(strings.Join(strings.Fields(title), " "))
}

func compactNonEmptyTitles(items ...string) []string {
	out := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
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

func (s *Supplier) filterCandidates(client *resty.Client, results []searchResult) ([]subtitleCandidate, error) {
	if s.mode == subtitleCatModeEnglish {
		out := make([]subtitleCandidate, 0, len(results))
		for _, result := range results {
			out = append(out, subtitleCandidate{
				name:           result.title,
				detailURL:      result.detailURL,
				downloadURL:    detailToOriginalDownloadURL(result.detailURL),
				translatedFrom: result.translatedFrom,
				downloads:      result.downloads,
				languages:      result.languages,
			})
		}
		return out, nil
	}

	out := make([]subtitleCandidate, 0, len(results))
	for _, result := range results {
		resp, err := client.R().Get(result.detailURL)
		if err != nil {
			s.log.Warningln(s.GetSupplierName(), "detail.Get", result.detailURL, err)
			continue
		}
		downloadURL, found, err := parseTranslatedDownloadURL(resp.String(), result.detailURL)
		if err != nil {
			s.log.Warningln(s.GetSupplierName(), "parseTranslatedDownloadURL", result.detailURL, err)
			continue
		}
		if found == false {
			continue
		}
		out = append(out, subtitleCandidate{
			name:           result.title,
			detailURL:      result.detailURL,
			downloadURL:    downloadURL,
			translatedFrom: result.translatedFrom,
			downloads:      result.downloads,
			languages:      result.languages,
		})
	}
	return out, nil
}

func (s *Supplier) search(client *resty.Client, keyword string) ([]searchResult, error) {
	restoreRetryCount := client.RetryCount
	client.SetRetryCount(subtitleCatSearchRetryCount)
	defer client.SetRetryCount(restoreRetryCount)

	resp, err := client.R().
		SetQueryParam("search", keyword).
		Get(settings.Get().AdvancedSettings.SuppliersSettings.SubtitleCat.GetSearchUrl())
	if err != nil {
		return nil, err
	}

	return parseSearchResults(resp.String())
}

func parseSearchResults(html string) ([]searchResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	out := make([]searchResult, 0)
	doc.Find("table.sub-table tbody tr").Each(func(_ int, row *goquery.Selection) {
		link := row.Find("td").First().Find("a").First()
		href, ok := link.Attr("href")
		if ok == false || strings.Contains(href, "subs/") == false {
			return
		}
		title := strings.TrimSpace(link.Text())
		if title == "" {
			return
		}

		firstCellText := normalizeWhitespace(row.Find("td").First().Text())
		translatedFrom := ""
		if matches := subtitleCatTranslatedFromRegex.FindStringSubmatch(firstCellText); len(matches) == 2 {
			translatedFrom = strings.TrimSpace(matches[1])
		}

		downloads := parseCountCell(row.Find("td").Eq(3).Text())
		languages := parseCountCell(row.Find("td").Eq(4).Text())

		out = append(out, searchResult{
			title:          title,
			detailURL:      absoluteURL(settings.Get().AdvancedSettings.SuppliersSettings.SubtitleCat.RootUrl, href),
			translatedFrom: translatedFrom,
			downloads:      downloads,
			languages:      languages,
		})
	})

	return dedupeSearchResults(out), nil
}

func rankCandidates(candidates []subtitleCandidate, videoFPath string, isMovie bool, season, episode int) {
	if len(candidates) < 2 {
		return
	}

	matcher := ranking.NewTargetMatcher(videoFPath, isMovie)
	sort.SliceStable(candidates, func(i, j int) bool {
		left := ranking.ScoreCandidate(matcher, candidateMetadata(candidates[i]), ranking.CandidateScoreSpec{
			IsMovie:       isMovie,
			TargetSeason:  season,
			TargetEpisode: episode,
			EpisodeMatchWeights: &ranking.EpisodeMatchWeights{
				ExactMatch:   120,
				SeasonPack:   15,
				WrongEpisode: -120,
			},
			SubTypePriority:     settings.Get().AdvancedSettings.SubTypePriority,
			HIPenalty:           -5,
			ReleaseMatchWeights: ranking.StandardReleaseMatchWeights,
		})
		right := ranking.ScoreCandidate(matcher, candidateMetadata(candidates[j]), ranking.CandidateScoreSpec{
			IsMovie:       isMovie,
			TargetSeason:  season,
			TargetEpisode: episode,
			EpisodeMatchWeights: &ranking.EpisodeMatchWeights{
				ExactMatch:   120,
				SeasonPack:   15,
				WrongEpisode: -120,
			},
			SubTypePriority:     settings.Get().AdvancedSettings.SubTypePriority,
			HIPenalty:           -5,
			ReleaseMatchWeights: ranking.StandardReleaseMatchWeights,
		})
		if left != right {
			return left > right
		}
		if candidates[i].downloads != candidates[j].downloads {
			return candidates[i].downloads > candidates[j].downloads
		}
		return candidates[i].detailURL < candidates[j].detailURL
	})
}

func candidateMetadata(candidate subtitleCandidate) ranking.CandidateMetadata {
	meta := ranking.CandidateMetadata{
		Name:           candidate.name,
		SubtitleExt:    strings.ToLower(filepath.Ext(candidate.downloadURL)),
		AuthorityScore: minInt(candidate.downloads/5, 12),
	}
	if parsed, err := decode.GetVideoInfoFromFileName(candidate.name); err == nil && parsed != nil {
		meta.Season = parsed.Season
		meta.Episode = parsed.Episode
	}
	return meta
}

func buildSearchKeywords(mediaInfo *models.MediaInfo, videoFPath string, isMovie bool, season, episode int) []string {
	baseTitles := make([]string, 0, 5)
	if mediaInfo == nil {
		baseTitles = []string{
			getLocalNfoTitle(videoFPath),
			getLocalNfoOriginalTitle(videoFPath),
			normalizeVideoTitle(videoFPath),
		}
	} else {
		baseTitles = []string{
			mediaInfo.TitleEn,
			mediaInfo.OriginalTitle,
			getLocalNfoTitle(videoFPath),
			getLocalNfoOriginalTitle(videoFPath),
			normalizeVideoTitle(videoFPath),
		}
	}

	baseTitles = compactNonEmptyTitles(baseTitles...)
	if isMovie {
		return expandSearchKeywordVariants(baseTitles)
	}

	return expandSeriesSearchKeywordVariants(baseTitles, season, episode)
}

func getLocalNfoTitle(videoFPath string) string {
	info, err := decode.GetVideoNfoInfoFromEpisode(videoFPath)
	if err == nil && strings.TrimSpace(info.Title) != "" {
		return info.Title
	}

	info, err = decode.GetVideoNfoInfo4Movie(videoFPath)
	if err == nil {
		return strings.TrimSpace(info.Title)
	}
	return ""
}

func getLocalNfoOriginalTitle(videoFPath string) string {
	info, err := decode.GetVideoNfoInfoFromEpisode(videoFPath)
	if err == nil && strings.TrimSpace(info.OriginalTitle) != "" {
		return info.OriginalTitle
	}

	info, err = decode.GetVideoNfoInfo4Movie(videoFPath)
	if err == nil {
		return strings.TrimSpace(info.OriginalTitle)
	}
	return ""
}

func normalizeVideoTitle(videoFPath string) string {
	fileName := strings.TrimSuffix(filepath.Base(videoFPath), filepath.Ext(videoFPath))
	if fileInfo, err := decode.GetVideoInfoFromFileName(fileName); err == nil && fileInfo != nil && fileInfo.Title != "" {
		fileName = fileInfo.Title
	}
	fileName = pkg.ReplaceSpecString(fileName, " ")
	return strings.Join(strings.Fields(fileName), " ")
}

func expandSearchKeywordVariants(items []string) []string {
	out := make([]string, 0, len(items)*2)
	seen := make(map[string]struct{})
	for _, item := range items {
		for _, variant := range keywordVariants(item) {
			key := strings.ToLower(strings.TrimSpace(variant))
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, variant)
		}
	}
	return out
}

func expandSeriesSearchKeywordVariants(items []string, season, episode int) []string {
	out := expandSearchKeywordVariants(items)
	if season <= 0 || episode <= 0 {
		return out
	}

	seen := make(map[string]struct{}, len(out))
	for _, item := range out {
		seen[strings.ToLower(strings.TrimSpace(item))] = struct{}{}
	}

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		for _, episodeVariant := range seriesEpisodeKeywordVariants(item, season, episode) {
			key := strings.ToLower(strings.TrimSpace(episodeVariant))
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, episodeVariant)
		}
	}

	return out
}

func seriesEpisodeKeywordVariants(title string, season, episode int) []string {
	title = strings.TrimSpace(title)
	if title == "" || season <= 0 || episode <= 0 {
		return nil
	}

	return []string{
		title + " " + formatSeasonEpisodeTag(season, episode),
		title + " " + formatSeasonXEpisodeTag(season, episode),
	}
}

func formatSeasonEpisodeTag(season, episode int) string {
	return "S" + strconv.Itoa(season/10) + strconv.Itoa(season%10) + "E" + strconv.Itoa(episode/10) + strconv.Itoa(episode%10)
}

func formatSeasonXEpisodeTag(season, episode int) string {
	return strconv.Itoa(season) + "x" + strconv.Itoa(episode)
}

func keywordVariants(input string) []string {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	variants := []string{input}
	normalized := normalizeWhitespace(stripKeywordPunctuation(input))
	if normalized != "" && strings.EqualFold(normalized, input) == false {
		variants = append(variants, normalized)
	}
	return variants
}

func stripKeywordPunctuation(input string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r), unicode.IsNumber(r), unicode.IsSpace(r):
			return r
		default:
			return ' '
		}
	}, input)
}

func absoluteURL(rootURL string, href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	baseURL, err := url.Parse(strings.TrimSpace(rootURL))
	if err == nil {
		refURL, refErr := url.Parse(strings.TrimSpace(href))
		if refErr == nil {
			return baseURL.ResolveReference(refURL).String()
		}
	}
	rootURL = strings.TrimRight(rootURL, "/")
	href = strings.TrimSpace(href)
	if strings.HasPrefix(href, "/") {
		return rootURL + href
	}
	return rootURL + "/" + href
}

func detailToOriginalDownloadURL(detailURL string) string {
	if detailURL == "" {
		return ""
	}
	orig := strings.TrimSuffix(detailURL, ".html")
	if strings.HasSuffix(orig, "/") == false {
		orig += "-orig.srt"
	}
	return orig
}

func parseCountCell(input string) int {
	match := subtitleCatNumberRegex.FindStringSubmatch(input)
	if len(match) < 2 {
		return 0
	}
	value, err := strconv.Atoi(match[1])
	if err != nil {
		return 0
	}
	return value
}

func normalizeWhitespace(input string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(input)), " ")
}

func dedupeSearchResults(items []searchResult) []searchResult {
	out := make([]searchResult, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		if item.detailURL == "" {
			continue
		}
		if _, ok := seen[item.detailURL]; ok {
			continue
		}
		seen[item.detailURL] = struct{}{}
		out = append(out, item)
	}
	return out
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func parseTranslatedDownloadURL(html string, pageURL string) (string, bool, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", false, err
	}

	for _, langCode := range []string{"zh-CN", "zh-TW"} {
		selector := "#download_" + escapeSelectionID(langCode)
		if href, ok := doc.Find(selector).First().Attr("href"); ok && href != "" {
			return absoluteURL(pageURL, href), true, nil
		}
	}

	return "", false, nil
}

func escapeSelectionID(raw string) string {
	return strings.NewReplacer(":", "\\:", ".", "\\.").Replace(raw)
}

var (
	subtitleCatTranslatedFromRegex = regexp.MustCompile(`(?i)\(translated from ([^)]+)\)`)
	subtitleCatNumberRegex         = regexp.MustCompile(`(\d+)`)
)
