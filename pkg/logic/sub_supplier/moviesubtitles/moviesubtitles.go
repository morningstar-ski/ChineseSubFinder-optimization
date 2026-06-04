package moviesubtitles

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
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
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/ranking"
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
	topic          int
	isAlive        bool
}

type movieSearchResult struct {
	ID    int
	Title string
	Year  string
	URL   string
}

type subtitleCandidate struct {
	Name            string
	ReleaseNames    []string
	SubtitlePageURL string
	SubtitleExt     string
	AuthorityScore  int
}

func NewSupplier(fileDownloader *file_downloader.FileDownloader) *Supplier {
	sup := Supplier{}
	sup.log = fileDownloader.Log
	sup.fileDownloader = fileDownloader
	sup.topic = subCommon.DownloadSubsPerSite
	sup.isAlive = true

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

	resp, err := client.R().Get(settings.Get().AdvancedSettings.SuppliersSettings.MovieSubtitles.RootUrl)
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

func (s *Supplier) OverDailyDownloadLimit() bool {
	if settings.Get().AdvancedSettings.SuppliersSettings.MovieSubtitles.DailyDownloadLimit == 0 {
		s.log.Warningln(s.GetSupplierName(), "DailyDownloadLimit is 0, will Skip Download")
		return true
	}
	return false
}

func (s *Supplier) GetLogger() *logrus.Logger {
	return s.log
}

func (s *Supplier) GetSupplierName() string {
	return subCommon.SubSiteMovieSubtitles
}

func (s *Supplier) GetSubListFromFile4Movie(filePath string) ([]supplier.SubInfo, error) {
	outSubInfos := make([]supplier.SubInfo, 0)
	if settings.Get().SubtitleSources.MoviesubtitlesSettings.Enabled == false {
		return outSubInfos, nil
	}

	return s.getSubListFromFile(filePath)
}

func (s *Supplier) GetSubListFromFile4Series(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	return make([]supplier.SubInfo, 0), nil
}

func (s *Supplier) GetSubListFromFile4Anime(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	return make([]supplier.SubInfo, 0), nil
}

func (s *Supplier) getSubListFromFile(videoFPath string) ([]supplier.SubInfo, error) {
	defer func() {
		s.log.Debugln(s.GetSupplierName(), videoFPath, "End...")
	}()
	s.log.Debugln(s.GetSupplierName(), videoFPath, "Start...")

	mediaInfo, err := mix_media_info.GetMixMediaInfo(s.fileDownloader.MediaInfoDealers, videoFPath, true)
	if err != nil {
		s.log.Errorln(s.GetSupplierName(), videoFPath, "GetMixMediaInfo", err)
		return nil, err
	}

	client, err := pkg.NewHttpClient()
	if err != nil {
		s.log.Errorln(s.GetSupplierName(), "NewHttpClient", err)
		return nil, err
	}

	candidates, err := s.resolveCandidatesWithFallback(client, mediaInfo, videoFPath)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	videoFileName := filepath.Base(videoFPath)
	outSubInfoList := make([]supplier.SubInfo, 0)
	for index, candidate := range candidates {
		downloadPageURL, err := s.fetchDownloadPageURL(client, candidate.SubtitlePageURL)
		if err != nil {
			s.log.Errorln(s.GetSupplierName(), "fetchDownloadPageURL", candidate.SubtitlePageURL, err)
			continue
		}

		finalDownloadURL, err := s.fetchFinalDownloadURL(client, downloadPageURL)
		if err != nil {
			s.log.Errorln(s.GetSupplierName(), "fetchFinalDownloadURL", downloadPageURL, err)
			continue
		}

		cacheKey := fmt.Sprintf("%s-%s", s.GetSupplierName(), finalDownloadURL)
		subInfo, err := s.fileDownloader.Get(
			s.GetSupplierName(),
			int64(index),
			videoFileName,
			finalDownloadURL,
			0,
			0,
			cacheKey,
		)
		if err != nil {
			s.log.Errorln(s.GetSupplierName(), "FileDownloader.Get", err)
			continue
		}
		outSubInfoList = append(outSubInfoList, *subInfo)
		if len(outSubInfoList) >= s.topic {
			return outSubInfoList, nil
		}
	}

	return outSubInfoList, nil
}

func (s *Supplier) resolveCandidatesWithFallback(client *resty.Client, mediaInfo *models.MediaInfo, videoFPath string) ([]subtitleCandidate, error) {
	for _, keyword := range buildSearchKeywords(mediaInfo, videoFPath) {
		if keyword == "" {
			continue
		}

		movies, err := s.searchMovies(client, keyword)
		if err != nil {
			return nil, err
		}

		movie := selectBestMovie(movies, mediaInfo, keyword)
		if movie == nil {
			continue
		}

		candidates, err := s.fetchMovieCandidates(client, absoluteURL(settings.Get().AdvancedSettings.SuppliersSettings.MovieSubtitles.RootUrl, movie.URL), videoFPath)
		if err != nil {
			return nil, err
		}
		if len(candidates) == 0 {
			continue
		}

		return candidates, nil
	}

	return nil, nil
}

func (s *Supplier) searchMovies(client *resty.Client, keyword string) ([]movieSearchResult, error) {
	resp, err := client.R().
		SetFormData(map[string]string{"q": keyword}).
		Post(settings.Get().AdvancedSettings.SuppliersSettings.MovieSubtitles.RootUrl + settings.Get().AdvancedSettings.SuppliersSettings.MovieSubtitles.SearchUrl)
	if err != nil {
		return nil, err
	}

	return parseSearchResults(resp.String())
}

func (s *Supplier) fetchMovieCandidates(client *resty.Client, movieURL string, videoFPath string) ([]subtitleCandidate, error) {
	resp, err := client.R().Get(movieURL)
	if err != nil {
		return nil, err
	}

	candidates, err := parseMoviePage(resp.String())
	if err != nil {
		return nil, err
	}

	rankCandidates(candidates, videoFPath, settings.Get().AdvancedSettings.SubTypePriority)
	if len(candidates) > s.topic {
		return candidates[:s.topic], nil
	}
	return candidates, nil
}

func (s *Supplier) fetchDownloadPageURL(client *resty.Client, subtitlePageURL string) (string, error) {
	resp, err := client.R().Get(subtitlePageURL)
	if err != nil {
		return "", err
	}

	href, err := parseSubtitleDetailPage(resp.String())
	if err != nil {
		return "", err
	}
	return absoluteURL(settings.Get().AdvancedSettings.SuppliersSettings.MovieSubtitles.RootUrl, href), nil
}

func (s *Supplier) fetchFinalDownloadURL(client *resty.Client, downloadPageURL string) (string, error) {
	if client == nil || client.GetClient() == nil {
		return "", errors.New("http client is nil")
	}

	baseClient := client.GetClient()
	redirectClient := *baseClient
	redirectClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}

	req, err := http.NewRequest(http.MethodGet, downloadPageURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	resp, err := redirectClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode >= http.StatusMultipleChoices && resp.StatusCode < http.StatusBadRequest {
		location := strings.TrimSpace(resp.Header.Get("Location"))
		if location == "" {
			return "", errors.New("moviesubtitles redirect location is empty")
		}
		return absoluteURL(downloadPageURL, location), nil
	}

	if resp.Request != nil && resp.Request.URL != nil {
		return resp.Request.URL.String(), nil
	}

	return "", errors.New("moviesubtitles final download url not found")
}

func parseSearchResults(html string) ([]movieSearchResult, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	out := make([]movieSearchResult, 0)
	doc.Find("a[href]").Each(func(_ int, selection *goquery.Selection) {
		href, ok := selection.Attr("href")
		if ok == false || strings.Contains(href, "movie-") == false {
			return
		}
		id, ok := extractNumericID(href, `movie-(\d+)`)
		if ok == false {
			return
		}

		title := strings.TrimSpace(selection.Text())
		if title == "" {
			return
		}
		baseTitle, year := splitMovieTitleAndYear(title)
		out = append(out, movieSearchResult{
			ID:    id,
			Title: baseTitle,
			Year:  year,
			URL:   href,
		})
	})

	return dedupeMovies(out), nil
}

func parseMoviePage(html string) ([]subtitleCandidate, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	out := make([]subtitleCandidate, 0)
	seen := make(map[string]struct{})
	currentLanguage := ""

	doc.Find("tr").Each(func(_ int, row *goquery.Selection) {
		if heading := strings.TrimSpace(row.Find("th").Text()); heading != "" {
			currentLanguage = heading
			return
		}

		if isChineseLanguage(currentLanguage) == false {
			return
		}

		block := row.Find("div.subtitle").First()
		if block.Length() == 0 {
			return
		}

		href, ok := block.Find(`a[href]`).FilterFunction(func(_ int, selection *goquery.Selection) bool {
			href, ok := selection.Attr("href")
			return ok && strings.Contains(href, "subtitle-")
		}).First().Attr("href")
		if ok == false || href == "" {
			return
		}
		if _, found := seen[href]; found {
			return
		}

		fields := parseCandidateFields(block)
		name := normalizeWhitespace(block.Find("b").First().Text())
		if name == "" {
			return
		}

		seen[href] = struct{}{}
		out = append(out, subtitleCandidate{
			Name:            name,
			ReleaseNames:    compactStrings(fields["release"], fields["rip"], name),
			SubtitlePageURL: href,
			AuthorityScore:  scoreAuthority(fields["downloaded"]),
		})
	})

	return out, nil
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

	return "", errors.New("moviesubtitles download page link not found")
}

func selectBestMovie(results []movieSearchResult, mediaInfo *models.MediaInfo, keyword string) *movieSearchResult {
	if len(results) == 0 {
		return nil
	}

	targetYear := normalizeYear(mediaInfo.Year)
	candidates := compactStrings(mediaInfo.TitleEn, mediaInfo.OriginalTitle, keyword)
	type scoredMovie struct {
		movie movieSearchResult
		score int
	}
	scored := make([]scoredMovie, 0, len(results))
	for _, result := range results {
		score := 0
		for _, candidate := range candidates {
			score = maxInt(score, scoreMovieTitle(result.Title, candidate))
		}
		if targetYear != "" && result.Year != "" {
			if targetYear == result.Year {
				score += 15
			} else {
				score -= 5
			}
		}
		scored = append(scored, scoredMovie{movie: result, score: score})
	}

	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].movie.ID < scored[j].movie.ID
	})
	if scored[0].score <= 0 {
		return nil
	}
	return &scored[0].movie
}

func rankCandidates(candidates []subtitleCandidate, videoFPath string, subTypePriority int) {
	if len(candidates) < 2 {
		return
	}

	matcher := ranking.NewTargetMatcher(videoFPath, true)
	sort.SliceStable(candidates, func(i, j int) bool {
		left := scoreCandidate(candidates[i], matcher, subTypePriority)
		right := scoreCandidate(candidates[j], matcher, subTypePriority)
		if left != right {
			return left > right
		}
		return candidates[i].SubtitlePageURL < candidates[j].SubtitlePageURL
	})
}

func scoreCandidate(candidate subtitleCandidate, matcher ranking.TargetMatcher, subTypePriority int) int {
	return ranking.ScoreCandidate(matcher, ranking.CandidateMetadata{
		Name:           candidate.Name,
		ReleaseNames:   append([]string(nil), candidate.ReleaseNames...),
		SubtitleExt:    candidate.SubtitleExt,
		AuthorityScore: candidate.AuthorityScore,
	}, ranking.CandidateScoreSpec{
		IsMovie:             true,
		SubTypePriority:     subTypePriority,
		ReleaseMatchWeights: ranking.StandardReleaseMatchWeights,
	})
}

func parseCandidateFields(block *goquery.Selection) map[string]string {
	fields := make(map[string]string)
	block.Find("td[title]").Each(func(_ int, selection *goquery.Selection) {
		title, ok := selection.Attr("title")
		if ok == false {
			return
		}
		value := normalizeWhitespace(selection.Text())
		if value == "" {
			return
		}
		fields[strings.ToLower(strings.TrimSpace(title))] = value
	})
	return fields
}

func buildSearchKeywords(mediaInfo *models.MediaInfo, videoFPath string) []string {
	return compactStrings(
		mediaInfo.TitleEn,
		mediaInfo.OriginalTitle,
		normalizeVideoTitle(videoFPath),
	)
}

func normalizeVideoTitle(videoFPath string) string {
	fileName := strings.TrimSuffix(filepath.Base(videoFPath), filepath.Ext(videoFPath))
	if fileInfo, err := decode.GetVideoInfoFromFileName(fileName); err == nil && fileInfo != nil && fileInfo.Title != "" {
		fileName = fileInfo.Title
	}
	fileName = pkg.ReplaceSpecString(fileName, " ")
	return strings.Join(strings.Fields(fileName), " ")
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

func splitMovieTitleAndYear(title string) (string, string) {
	match := regexp.MustCompile(`^(.*?)\s+\((\d{4})\)$`).FindStringSubmatch(strings.TrimSpace(title))
	if len(match) == 3 {
		return strings.TrimSpace(match[1]), match[2]
	}
	return strings.TrimSpace(title), ""
}

func isChineseLanguage(language string) bool {
	language = strings.ToLower(normalizeWhitespace(language))
	return strings.Contains(language, "chinese") || strings.Contains(language, "china")
}

func normalizeYear(year string) string {
	if len(year) >= 4 {
		return year[:4]
	}
	return ""
}

func normalizeWhitespace(input string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(input)), " ")
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

func scoreMovieTitle(movieTitle string, candidate string) int {
	movieTitle = normalizeComparableTitle(movieTitle)
	candidate = normalizeComparableTitle(candidate)
	if movieTitle == "" || candidate == "" {
		return 0
	}
	switch {
	case movieTitle == candidate:
		return 100
	case strings.Contains(movieTitle, candidate):
		return 60
	case strings.Contains(candidate, movieTitle):
		return 40
	default:
		return 0
	}
}

func normalizeComparableTitle(title string) string {
	baseTitle, _ := splitMovieTitleAndYear(title)
	baseTitle = pkg.ReplaceSpecString(baseTitle, " ")
	return strings.ToLower(strings.Join(strings.Fields(baseTitle), " "))
}

func scoreAuthority(downloads string) int {
	downloads = strings.ReplaceAll(downloads, ",", "")
	value, err := strconv.Atoi(downloads)
	if err != nil || value <= 0 {
		return 0
	}
	score := value / 2000
	if score > 12 {
		return 12
	}
	return score
}

func dedupeMovies(items []movieSearchResult) []movieSearchResult {
	out := make([]movieSearchResult, 0, len(items))
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
