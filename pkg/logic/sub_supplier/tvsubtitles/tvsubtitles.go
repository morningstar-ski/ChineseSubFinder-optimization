package tvsubtitles

import (
	"bytes"
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

type finalDownloadTarget struct {
	URL              string
	DirectData       []byte
	DownloadFileName string
}

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

	resp, err := client.R().Get(settings.Get().AdvancedSettings.SuppliersSettings.TVSubtitles.RootUrl)
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

	for _, episodeInfo := range seriesInfo.NeedDlEpsKeyList {
		subInfos, err := s.getEpisodeSubtitle(episodeInfo.FileFullPath, episodeInfo.Season, episodeInfo.Episode)
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

func (s *Supplier) getEpisodeSubtitle(videoFPath string, season, episode int) ([]supplier.SubInfo, error) {
	mediaInfo, err := mix_media_info.GetMixMediaInfo(s.fileDownloader.MediaInfoDealers, videoFPath, false)
	if err != nil {
		return nil, err
	}

	client, err := pkg.NewHttpClient()
	if err != nil {
		return nil, err
	}

	subtitlePageURL, err := s.resolveSubtitlePageURL(client, mediaInfo, videoFPath, season, episode)
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

	finalDownloadTarget, err := s.fetchFinalDownloadTarget(client, downloadPageURL)
	if err != nil {
		return nil, err
	}

	cacheKey := fmt.Sprintf("%s-%s", s.GetSupplierName(), finalDownloadTarget.URL)
	var subInfo *supplier.SubInfo
	if len(finalDownloadTarget.DirectData) > 0 {
		subInfo, err = s.fileDownloader.GetByData(
			s.GetSupplierName(),
			0,
			filepath.Base(videoFPath),
			finalDownloadTarget.URL,
			0,
			0,
			finalDownloadTarget.DirectData,
			finalDownloadTarget.DownloadFileName,
			cacheKey,
		)
	} else {
		subInfo, err = s.fileDownloader.Get(
			s.GetSupplierName(),
			0,
			filepath.Base(videoFPath),
			finalDownloadTarget.URL,
			0,
			0,
			cacheKey,
		)
	}
	if err != nil {
		return nil, err
	}
	subInfo.Season = season
	subInfo.Episode = episode

	return []supplier.SubInfo{*subInfo}, nil
}

func (s *Supplier) resolveSubtitlePageURL(client *resty.Client, mediaInfo *models.MediaInfo, videoFPath string, season, episode int) (string, error) {
	for _, keyword := range buildSearchKeywords(mediaInfo, videoFPath) {
		if keyword == "" {
			continue
		}

		shows, err := s.searchShows(client, keyword)
		if err != nil {
			return "", err
		}
		show := selectBestShow(shows, mediaInfo, keyword)
		if show == nil {
			continue
		}

		plan, err := s.fetchSeasonPlan(client, show.ID, season)
		if err != nil {
			return "", err
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
	resp, err := client.R().
		SetFormData(map[string]string{"qs": keyword}).
		Post(settings.Get().AdvancedSettings.SuppliersSettings.TVSubtitles.RootUrl + settings.Get().AdvancedSettings.SuppliersSettings.TVSubtitles.SearchUrl)
	if err != nil {
		return nil, err
	}

	return parseSearchResults(resp.String())
}

func (s *Supplier) fetchSeasonPlan(client *resty.Client, showID, season int) (*seasonPlan, error) {
	url := fmt.Sprintf("%s/tvshow-%d-%d.html", settings.Get().AdvancedSettings.SuppliersSettings.TVSubtitles.RootUrl, showID, season)
	resp, err := client.R().Get(url)
	if err != nil {
		return nil, err
	}

	return parseSeasonPage(resp.String())
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
	return absoluteURL(settings.Get().AdvancedSettings.SuppliersSettings.TVSubtitles.RootUrl, href), nil
}

func (s *Supplier) fetchFinalDownloadTarget(client *resty.Client, downloadPageURL string) (*finalDownloadTarget, error) {
	resp, err := client.R().Get(downloadPageURL)
	if err != nil {
		return nil, err
	}

	if isDirectDownloadResponse(resp.Body(), resp.Header().Get("Content-Type"), resp.Header().Get("Content-Disposition")) {
		downloadFileName := ""
		if resp.RawResponse != nil {
			downloadFileName = pkg.GetFileName(s.log, resp.RawResponse)
		}
		return &finalDownloadTarget{
			URL:              downloadPageURL,
			DirectData:       append([]byte(nil), resp.Body()...),
			DownloadFileName: downloadFileName,
		}, nil
	}

	path, err := parseDownloadPage(resp.String())
	if err != nil {
		return nil, err
	}
	return &finalDownloadTarget{
		URL: absoluteURL(settings.Get().AdvancedSettings.SuppliersSettings.TVSubtitles.RootUrl, path),
	}, nil
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

func isDirectDownloadResponse(body []byte, contentType string, contentDisposition string) bool {
	if len(body) == 0 {
		return false
	}

	lowerContentType := strings.ToLower(contentType)
	lowerContentDisposition := strings.ToLower(contentDisposition)
	if strings.Contains(lowerContentDisposition, "attachment") {
		return true
	}
	if strings.Contains(lowerContentType, "application/zip") ||
		strings.Contains(lowerContentType, "application/octet-stream") ||
		strings.Contains(lowerContentType, "application/x-rar-compressed") ||
		strings.Contains(lowerContentType, "application/vnd.rar") {
		return true
	}

	return bytes.HasPrefix(body, []byte("PK\x03\x04")) ||
		bytes.HasPrefix(body, []byte("PK\x05\x06")) ||
		bytes.HasPrefix(body, []byte("Rar!\x1a\x07"))
}

func selectBestShow(results []showSearchResult, mediaInfo *models.MediaInfo, keyword string) *showSearchResult {
	if len(results) == 0 {
		return nil
	}

	candidates := compactStrings(mediaInfo.TitleEn, mediaInfo.OriginalTitle, keyword)
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
	switch {
	case showTitle == candidate:
		return 100
	case strings.Contains(showTitle, candidate):
		return 60
	case strings.Contains(candidate, showTitle):
		return 40
	default:
		return 0
	}
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
