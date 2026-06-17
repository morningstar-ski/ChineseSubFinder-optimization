package tvsubtitles

import (
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/decode"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/mix_media_info"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	subCommon "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
	"github.com/sirupsen/logrus"
)

type Supplier struct {
	log            *logrus.Logger
	fileDownloader *file_downloader.FileDownloader
	isAlive        bool
}

type showSearchResult struct {
	ID    int
	Title string
}

type seasonPlan struct {
	EpisodeSubtitlePages map[int]string
	AllEpisodesPage      string
}

const tvSubtitlesSearchRetryCount = 1
const tvSubtitlesTimeout = 8 * time.Second
const tvSubtitlesHTTPRetryAttempts = 3
const tvSubtitlesHTTPRetryDelay = 700 * time.Millisecond

func NewSupplier(fileDownloader *file_downloader.FileDownloader) *Supplier {
	return &Supplier{
		log:            fileDownloader.Log,
		fileDownloader: fileDownloader,
		isAlive:        true,
	}
}

func (s *Supplier) CheckAlive() (bool, int64) {
	startT := time.Now()
	client, err := pkg.NewHttpClient()
	if err != nil {
		s.log.Errorln(s.GetSupplierName(), "CheckAlive.NewHttpClient", err)
		s.isAlive = false
		return false, 0
	}

	resp, err := tvSubtitlesDoRequest(client, httpRequestSpec{
		method: "GET",
		url:    settings.Get().AdvancedSettings.SuppliersSettings.TVSubtitles.RootUrl,
	})
	if err != nil {
		s.log.Errorln(s.GetSupplierName(), "CheckAlive.Get", err)
		s.isAlive = false
		return false, 0
	}
	if resp.StatusCode() != 200 {
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

func (s *Supplier) OverDailyDownloadLimit() bool {
	if settings.Get().AdvancedSettings.SuppliersSettings.TVSubtitles.DailyDownloadLimit == 0 {
		s.log.Warningln(s.GetSupplierName(), "DailyDownloadLimit is 0, will Skip Download")
		return true
	}
	return false
}

func (s *Supplier) GetLogger() *logrus.Logger {
	return s.log
}

func (s *Supplier) GetSupplierName() string {
	return subCommon.SubSiteTVSubtitles
}

func (s *Supplier) GetSubListFromFile4Movie(filePath string) ([]supplier.SubInfo, error) {
	return make([]supplier.SubInfo, 0), nil
}

func (s *Supplier) GetSubListFromFile4Series(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	outSubInfos := make([]supplier.SubInfo, 0)
	if settings.Get().SubtitleSources.TVsubtitlesSettings.Enabled == false {
		return outSubInfos, nil
	}

	return s.downloadSub4Series(seriesInfo)
}

func (s *Supplier) GetSubListFromFile4Anime(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	return s.GetSubListFromFile4Series(seriesInfo)
}

func (s *Supplier) downloadSub4Series(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	allSupplierSubInfo := make([]supplier.SubInfo, 0)
	seriesKeywords := buildSeriesSearchKeywords(seriesInfo)
	if len(seriesKeywords) > 0 {
		s.log.Infoln(s.GetSupplierName(), "series search keywords", strings.Join(seriesKeywords, " | "))
	}

	for _, episodeInfo := range seriesInfo.NeedDlEpsKeyList {
		subInfos, err := s.getEpisodeSubtitle(episodeInfo.FileFullPath, episodeInfo.Season, episodeInfo.Episode, seriesKeywords)
		if err != nil {
			s.log.Errorln(s.GetSupplierName(), "getEpisodeSubtitle", episodeInfo.FileFullPath, err)
			continue
		}
		if len(subInfos) == 0 {
			s.log.Infoln(s.GetSupplierName(), "Not Find Sub can be download", episodeInfo.Title, episodeInfo.Season, episodeInfo.Episode)
			continue
		}
		for i := range subInfos {
			subInfos[i].Season = episodeInfo.Season
			subInfos[i].Episode = episodeInfo.Episode
		}
		allSupplierSubInfo = append(allSupplierSubInfo, subInfos...)
	}

	return allSupplierSubInfo, nil
}

func (s *Supplier) getEpisodeSubtitle(videoFPath string, season, episode int, seriesKeywords []string) ([]supplier.SubInfo, error) {
	mediaInfo, err := mix_media_info.GetMixMediaInfo(s.fileDownloader.MediaInfoDealers, videoFPath, false)
	if err != nil {
		s.log.Warningln(s.GetSupplierName(), videoFPath, "GetMixMediaInfo", err, "fallback to series title search")
		mediaInfo = nil
	}

	client, err := pkg.NewHttpClient()
	if err != nil {
		return nil, err
	}
	client.SetTimeout(tvSubtitlesTimeout)

	subtitlePageURL, err := s.resolveSubtitlePageURL(client, mediaInfo, videoFPath, season, episode, seriesKeywords)
	if err != nil {
		return nil, err
	}
	if subtitlePageURL == "" {
		return nil, nil
	}

	downloadPageURL, err := s.fetchDownloadPageURL(client, subtitlePageURL)
	if err != nil {
		return nil, err
	}

	finalDownloadURL, err := s.fetchFinalDownloadURL(client, downloadPageURL)
	if err != nil {
		return nil, err
	}

	cacheKey := file_downloader.BuildCacheKey(s.GetSupplierName(), finalDownloadURL)
	subInfo, err := s.fileDownloader.Get(
		s.GetSupplierName(),
		0,
		filepath.Base(videoFPath),
		finalDownloadURL,
		0,
		0,
		cacheKey,
	)
	if err != nil {
		return nil, err
	}
	subInfo.Season = season
	subInfo.Episode = episode

	return []supplier.SubInfo{*subInfo}, nil
}

func (s *Supplier) resolveSubtitlePageURL(client *resty.Client, mediaInfo *models.MediaInfo, videoFPath string, season, episode int, seriesKeywords []string) (string, error) {
	for _, keyword := range buildSearchKeywords(mediaInfo, videoFPath, seriesKeywords...) {
		if keyword == "" {
			continue
		}

		shows, err := s.searchShows(client, keyword)
		if err != nil {
			s.log.Warningln(s.GetSupplierName(), "searchShows", keyword, err)
			continue
		}
		show := selectBestShow(shows, mediaInfo, keyword)
		if show == nil {
			continue
		}
		s.log.Infoln(s.GetSupplierName(), "matched show", "keyword", keyword, "title", show.Title, "id", show.ID)

		plan, err := s.fetchSeasonPlan(client, show.ID, season)
		if err != nil {
			s.log.Warningln(s.GetSupplierName(), "fetchSeasonPlan", keyword, show.ID, season, err)
			continue
		}
		if plan.EpisodeSubtitlePages[episode] != "" {
			return absoluteURL(settings.Get().AdvancedSettings.SuppliersSettings.TVSubtitles.RootUrl, plan.EpisodeSubtitlePages[episode]), nil
		}
		if plan.AllEpisodesPage != "" {
			return absoluteURL(settings.Get().AdvancedSettings.SuppliersSettings.TVSubtitles.RootUrl, plan.AllEpisodesPage), nil
		}
	}

	return "", nil
}

func (s *Supplier) searchShows(client *resty.Client, keyword string) ([]showSearchResult, error) {
	resp, err := tvSubtitlesDoRequest(client, httpRequestSpec{
		method:   "POST",
		url:      settings.Get().AdvancedSettings.SuppliersSettings.TVSubtitles.RootUrl + settings.Get().AdvancedSettings.SuppliersSettings.TVSubtitles.SearchUrl,
		formData: map[string]string{"qs": keyword},
	})
	if err != nil {
		return nil, err
	}

	return parseSearchResults(resp.String())
}

func (s *Supplier) fetchSeasonPlan(client *resty.Client, showID, season int) (*seasonPlan, error) {
	url := fmt.Sprintf("%s/tvshow-%d-%d.html", settings.Get().AdvancedSettings.SuppliersSettings.TVSubtitles.RootUrl, showID, season)
	resp, err := tvSubtitlesDoRequest(client, httpRequestSpec{
		method: "GET",
		url:    url,
	})
	if err != nil {
		return nil, err
	}

	return parseSeasonPage(resp.String())
}

func (s *Supplier) fetchDownloadPageURL(client *resty.Client, subtitlePageURL string) (string, error) {
	resp, err := tvSubtitlesDoRequest(client, httpRequestSpec{
		method: "GET",
		url:    subtitlePageURL,
	})
	if err != nil {
		return "", err
	}

	href, err := parseSubtitleDetailPage(resp.String())
	if err != nil {
		return "", err
	}
	return absoluteURL(settings.Get().AdvancedSettings.SuppliersSettings.TVSubtitles.RootUrl, href), nil
}

func (s *Supplier) fetchFinalDownloadURL(client *resty.Client, downloadPageURL string) (string, error) {
	resp, err := tvSubtitlesDoRequest(client, httpRequestSpec{
		method: "GET",
		url:    downloadPageURL,
	})
	if err != nil {
		return "", err
	}

	path, err := parseDownloadPage(resp.String())
	if err != nil {
		return "", err
	}
	return absoluteURL(settings.Get().AdvancedSettings.SuppliersSettings.TVSubtitles.RootUrl, path), nil
}

type httpRequestSpec struct {
	method   string
	url      string
	formData map[string]string
}

func tvSubtitlesDoRequest(client *resty.Client, spec httpRequestSpec) (*resty.Response, error) {
	restoreRetryCount := client.RetryCount
	client.SetRetryCount(0)
	defer client.SetRetryCount(restoreRetryCount)

	var lastResp *resty.Response
	var lastErr error
	for attempt := 1; attempt <= tvSubtitlesHTTPRetryAttempts; attempt++ {
		req := client.R()
		if len(spec.formData) > 0 {
			req.SetFormData(spec.formData)
		}
		switch strings.ToUpper(spec.method) {
		case "POST":
			lastResp, lastErr = req.Post(spec.url)
		default:
			lastResp, lastErr = req.Get(spec.url)
		}
		if lastErr == nil && lastResp != nil && lastResp.StatusCode() >= 200 && lastResp.StatusCode() < 300 {
			return lastResp, nil
		}
		if lastErr == nil && lastResp != nil {
			lastErr = fmt.Errorf("unexpected http status %d for %s", lastResp.StatusCode(), spec.url)
		}
		if shouldRetryTVSubtitlesRequest(lastErr) == false || attempt == tvSubtitlesHTTPRetryAttempts {
			return lastResp, lastErr
		}
		time.Sleep(tvSubtitlesHTTPRetryDelay)
	}
	return lastResp, lastErr
}

func shouldRetryTVSubtitlesRequest(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "eof") ||
		strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "forcibly closed by the remote host") ||
		strings.Contains(msg, "unexpected http status 502") ||
		strings.Contains(msg, "unexpected http status 503") ||
		strings.Contains(msg, "unexpected http status 504")
}

func parseSearchResults(html string) ([]showSearchResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	out := make([]showSearchResult, 0)
	doc.Find("a[href]").Each(func(_ int, selection *goquery.Selection) {
		href, ok := selection.Attr("href")
		if ok == false || strings.Contains(href, "tvshow-") == false {
			return
		}
		id, ok := extractNumericID(href, `tvshow-(\d+)`)
		if ok == false {
			return
		}
		title := strings.TrimSpace(selection.Text())
		if title == "" {
			return
		}
		out = append(out, showSearchResult{
			ID:    id,
			Title: stripShowYearSuffix(title),
		})
	})

	out = dedupeShows(out)
	return out, nil
}

func parseSeasonPage(html string) (*seasonPlan, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	plan := &seasonPlan{EpisodeSubtitlePages: make(map[int]string)}
	doc.Find("tr").Each(func(_ int, row *goquery.Selection) {
		cells := row.Find("td")
		if cells.Length() < 4 {
			return
		}

		token := strings.TrimSpace(cells.Eq(0).Text())
		title := strings.TrimSpace(cells.Eq(1).Text())
		subtitlePage := findChineseSubtitlePage(cells.Eq(3))
		if subtitlePage == "" {
			return
		}

		if strings.EqualFold(title, "All episodes") {
			plan.AllEpisodesPage = subtitlePage
			return
		}

		seasonNumber, episodeNumber, ok := parseEpisodeToken(token)
		if ok == false {
			return
		}
		if seasonNumber == 0 || episodeNumber == 0 {
			return
		}
		plan.EpisodeSubtitlePages[episodeNumber] = subtitlePage
	})

	return plan, nil
}

func parseSubtitleDetailPage(html string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	if href, ok := doc.Find("a[href]").FilterFunction(func(_ int, selection *goquery.Selection) bool {
		href, ok := selection.Attr("href")
		return ok && strings.Contains(href, "download-")
	}).First().Attr("href"); ok {
		return href, nil
	}

	return "", errors.New("tvsubtitles download page link not found")
}

func parseDownloadPage(html string) (string, error) {
	partRegex := regexp.MustCompile(`var\s+s\d+\s*=\s*'([^']+)'`)
	matches := partRegex.FindAllStringSubmatch(html, -1)
	if len(matches) > 0 {
		builder := strings.Builder{}
		for _, match := range matches {
			if len(match) >= 2 {
				builder.WriteString(match[1])
			}
		}
		if builder.Len() > 0 {
			return builder.String(), nil
		}
	}

	directRegex := regexp.MustCompile(`document\.location\s*=\s*'([^']+)'`)
	match := directRegex.FindStringSubmatch(html)
	if len(match) >= 2 {
		return match[1], nil
	}

	return "", errors.New("tvsubtitles final download path not found")
}

func selectBestShow(results []showSearchResult, mediaInfo *models.MediaInfo, keyword string) *showSearchResult {
	if len(results) == 0 {
		return nil
	}

	candidates := []string{keyword}
	if mediaInfo != nil {
		candidates = compactStrings(mediaInfo.TitleEn, mediaInfo.OriginalTitle, keyword)
	}
	type scoredShow struct {
		show  showSearchResult
		score int
	}
	scored := make([]scoredShow, 0, len(results))
	for _, result := range results {
		score := 0
		for _, candidate := range candidates {
			score = maxInt(score, scoreShowTitle(result.Title, candidate))
		}
		scored = append(scored, scoredShow{show: result, score: score})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].show.ID < scored[j].show.ID
	})
	if scored[0].score <= 0 {
		return nil
	}
	return &scored[0].show
}

func findChineseSubtitlePage(cell *goquery.Selection) string {
	var href string
	cell.Find("a[href]").EachWithBreak(func(_ int, selection *goquery.Selection) bool {
		img := selection.Find("img")
		if img.Length() == 0 {
			return true
		}
		src, _ := img.Attr("src")
		alt, _ := img.Attr("alt")
		lowerSrc := strings.ToLower(src)
		lowerAlt := strings.ToLower(alt)
		if strings.Contains(lowerSrc, "flags/cn") || lowerAlt == "cn" || strings.Contains(lowerAlt, "chinese") {
			href, _ = selection.Attr("href")
			return false
		}
		return true
	})
	return href
}

func buildSeriesSearchKeywords(seriesInfo *series.SeriesInfo) []string {
	if seriesInfo == nil {
		return nil
	}

	keywords := make([]string, 0, 3)
	if seriesInfo.DirPath != "" {
		if videoInfo, err := decode.GetVideoNfoInfo4SeriesDir(seriesInfo.DirPath); err == nil {
			keywords = append(keywords, compactStrings(videoInfo.OriginalTitle, videoInfo.Title)...)
		}
	}
	keywords = append(keywords, compactStrings(seriesInfo.Name)...)

	return compactStrings(keywords...)
}

func buildSearchKeywords(mediaInfo *models.MediaInfo, videoFPath string, extraKeywords ...string) []string {
	keywords := make([]string, 0, 4+len(extraKeywords))
	if mediaInfo != nil {
		keywords = append(keywords, mediaInfo.TitleEn, mediaInfo.OriginalTitle)
	}
	keywords = append(keywords, extraKeywords...)
	keywords = append(keywords, normalizeSeriesTitle(videoFPath), normalizeVideoTitle(videoFPath))

	return compactStrings(keywords...)
}

func normalizeSeriesTitle(videoFPath string) string {
	return normalizePathTitleSegment(inferSeriesRootTitle(videoFPath))
}

func normalizeVideoTitle(videoFPath string) string {
	fileName := crossPlatformBase(videoFPath)
	fileName = strings.TrimSuffix(fileName, filepath.Ext(fileName))
	if fileInfo, err := decode.GetVideoInfoFromFileName(fileName); err == nil && fileInfo != nil && fileInfo.Title != "" {
		fileName = fileInfo.Title
	}
	fileName = pkg.ReplaceSpecString(fileName, " ")
	return strings.Join(strings.Fields(fileName), " ")
}

var windowsDriveLetterPattern = regexp.MustCompile(`^[a-zA-Z]:$`)
var genericSeasonFolderPattern = regexp.MustCompile(`(?i)^season[ ._-]*\d+$`)

func inferSeriesRootTitle(videoFPath string) string {
	segments := splitPathSegments(videoFPath)
	if len(segments) < 2 {
		return ""
	}

	parent := segments[len(segments)-2]
	if genericSeasonFolderPattern.MatchString(parent) && len(segments) >= 3 {
		return segments[len(segments)-3]
	}
	return parent
}

func crossPlatformBase(videoFPath string) string {
	segments := splitPathSegments(videoFPath)
	if len(segments) == 0 {
		return ""
	}
	return segments[len(segments)-1]
}

func splitPathSegments(videoFPath string) []string {
	raw := strings.TrimSpace(videoFPath)
	if raw == "" {
		return nil
	}

	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == '/' || r == '\\'
	})
	out := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		out = append(out, field)
	}
	return out
}

func normalizePathTitleSegment(segment string) string {
	segment = strings.TrimSpace(segment)
	if isJunkPathTitleSegment(segment) {
		return ""
	}
	segment = stripShowYearSuffix(segment)
	segment = pkg.ReplaceSpecString(segment, " ")
	segment = strings.Join(strings.Fields(segment), " ")
	if isJunkPathTitleSegment(segment) {
		return ""
	}
	return segment
}

func isJunkPathTitleSegment(segment string) bool {
	segment = strings.TrimSpace(segment)
	if segment == "" {
		return true
	}
	if windowsDriveLetterPattern.MatchString(segment) {
		return true
	}

	switch strings.ToLower(segment) {
	case ".", "..", "media", "movie", "movies", "tv", "show", "shows", "series", "video", "videos":
		return true
	}

	return genericSeasonFolderPattern.MatchString(segment)
}

func absoluteURL(rootURL string, href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}
	rootURL = strings.TrimRight(rootURL, "/")
	href = strings.TrimSpace(href)
	if strings.HasPrefix(href, "/") {
		return rootURL + href
	}
	return rootURL + "/" + href
}

func stripShowYearSuffix(title string) string {
	re := regexp.MustCompile(`\s+\(\d{4}.*\)$`)
	return strings.TrimSpace(re.ReplaceAllString(title, ""))
}

func parseEpisodeToken(token string) (int, int, bool) {
	match := regexp.MustCompile(`^(\d+)x(\d+)$`).FindStringSubmatch(strings.ToLower(strings.TrimSpace(token)))
	if len(match) != 3 {
		return 0, 0, false
	}

	season, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, 0, false
	}
	episode, err := strconv.Atoi(match[2])
	if err != nil {
		return 0, 0, false
	}
	return season, episode, true
}

func extractNumericID(input string, pattern string) (int, bool) {
	match := regexp.MustCompile(pattern).FindStringSubmatch(input)
	if len(match) < 2 {
		return 0, false
	}
	id, err := strconv.Atoi(match[1])
	if err != nil {
		return 0, false
	}
	return id, true
}

func scoreShowTitle(showTitle string, candidate string) int {
	showTitle = normalizeComparableTitle(showTitle)
	candidate = normalizeComparableTitle(candidate)
	if showTitle == "" || candidate == "" {
		return 0
	}
	if showTitle == candidate {
		return 100
	}

	showTokens := meaningfulTitleTokens(showTitle)
	candidateTokens := meaningfulTitleTokens(candidate)
	if len(showTokens) == 0 || len(candidateTokens) == 0 {
		return 0
	}

	if len(showTokens) == 1 || len(candidateTokens) == 1 {
		if len(showTokens) == 1 && len(candidateTokens) == 1 && showTokens[0] == candidateTokens[0] {
			return 90
		}
		return 0
	}

	overlap := sharedTokenCount(showTokens, candidateTokens)
	if overlap == 0 {
		return 0
	}

	showCoverage := overlap * 100 / len(showTokens)
	candidateCoverage := overlap * 100 / len(candidateTokens)
	switch {
	case showCoverage == 100 && candidateCoverage == 100:
		return 95
	case showCoverage == 100 && candidateCoverage >= 80:
		return 80
	case candidateCoverage == 100 && showCoverage >= 80:
		return 70
	case strings.Contains(showTitle, candidate) && candidateCoverage >= 80:
		return 60
	default:
		return 0
	}
}

func meaningfulTitleTokens(title string) []string {
	rawTokens := strings.Fields(title)
	if len(rawTokens) == 0 {
		return nil
	}

	stopWords := map[string]struct{}{
		"a":   {},
		"an":  {},
		"the": {},
		"of":  {},
		"and": {},
	}

	out := make([]string, 0, len(rawTokens))
	for _, token := range rawTokens {
		if _, ok := stopWords[token]; ok {
			continue
		}
		out = append(out, token)
	}
	if len(out) == 0 {
		return rawTokens
	}
	return out
}

func sharedTokenCount(left []string, right []string) int {
	seen := make(map[string]struct{}, len(left))
	for _, token := range left {
		seen[token] = struct{}{}
	}

	count := 0
	counted := make(map[string]struct{}, len(right))
	for _, token := range right {
		if _, ok := counted[token]; ok {
			continue
		}
		if _, ok := seen[token]; ok {
			count++
			counted[token] = struct{}{}
		}
	}
	return count
}

func normalizeComparableTitle(title string) string {
	title = stripShowYearSuffix(title)
	title = pkg.ReplaceSpecString(title, " ")
	title = strings.ToLower(strings.Join(strings.Fields(title), " "))
	return title
}

func dedupeShows(items []showSearchResult) []showSearchResult {
	out := make([]showSearchResult, 0, len(items))
	seen := make(map[int]struct{})
	for _, item := range items {
		if _, ok := seen[item.ID]; ok {
			continue
		}
		seen[item.ID] = struct{}{}
		out = append(out, item)
	}
	return out
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
