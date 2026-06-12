package subdl

import (
	"errors"
	"path/filepath"
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
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/sirupsen/logrus"
)

const (
	subdlDefaultLanguage = "ZH"
	subdlEnglishLanguage = "EN"
	subdlCheckIMDbID     = "tt1375666"
)

type Supplier struct {
	log            *logrus.Logger
	fileDownloader *file_downloader.FileDownloader
	topic          int
	isAlive        bool
	api            *Api
	queryLanguage  string
}

func NewSupplier(fileDownloader *file_downloader.FileDownloader) *Supplier {
	return newSupplier(fileDownloader, subdlDefaultLanguage)
}

func NewEnglishSupplier(fileDownloader *file_downloader.FileDownloader) *Supplier {
	return newSupplier(fileDownloader, subdlEnglishLanguage)
}

func newSupplier(fileDownloader *file_downloader.FileDownloader, queryLanguage string) *Supplier {
	sup := Supplier{}
	sup.log = fileDownloader.Log
	sup.fileDownloader = fileDownloader
	sup.topic = common2.DownloadSubsPerSite
	sup.isAlive = true
	sup.api = NewApi(settings.Get().SubtitleSources.SubDLSettings.Key)
	sup.queryLanguage = queryLanguage

	if settings.Get().AdvancedSettings.Topic > 0 && settings.Get().AdvancedSettings.Topic != sup.topic {
		sup.topic = settings.Get().AdvancedSettings.Topic
	}

	return &sup
}

func (s *Supplier) CheckAlive() (bool, int64) {
	if settings.Get().SubtitleSources.SubDLSettings.Key == "" {
		s.isAlive = false
		return false, 0
	}

	startT := time.Now()
	httpClient, err := pkg.NewHttpClient()
	if err != nil {
		s.log.Errorln(s.GetSupplierName(), "CheckAlive.NewHttpClient", err)
		s.isAlive = false
		return false, 0
	}

	_, err = s.api.SearchSubtitles(httpClient, map[string]string{
		"api_key":       settings.Get().SubtitleSources.SubDLSettings.Key,
		"type":          "movie",
		"imdb_id":       subdlCheckIMDbID,
		"languages":     s.queryLanguage,
		"subs_per_page": "1",
	})
	if shouldTreatCheckAliveProbeAsHealthy(err) {
		s.isAlive = true
		return true, time.Since(startT).Milliseconds()
	}
	if err != nil {
		s.log.Errorln(s.GetSupplierName(), "CheckAlive.SearchSubtitles", err)
		s.isAlive = false
		return false, 0
	}

	s.isAlive = true
	return true, time.Since(startT).Milliseconds()
}

func shouldTreatCheckAliveProbeAsHealthy(err error) bool {
	return err == nil || errors.Is(err, errSubdlStatusFalse)
}

func (s *Supplier) IsAlive() bool {
	return s.isAlive
}

func (s *Supplier) OverDailyDownloadLimit() bool {
	if settings.Get().AdvancedSettings.SuppliersSettings.SubDL.DailyDownloadLimit == 0 {
		s.log.Warningln(s.GetSupplierName(), "DailyDownloadLimit is 0, will Skip Download")
		return true
	}
	return false
}

func (s *Supplier) GetLogger() *logrus.Logger {
	return s.log
}

func (s *Supplier) GetSupplierName() string {
	return common2.SubSiteSubDL
}

func (s *Supplier) GetSubListFromFile4Movie(filePath string) ([]supplier.SubInfo, error) {
	outSubInfos := make([]supplier.SubInfo, 0)
	if settings.Get().SubtitleSources.SubDLSettings.Enabled == false {
		return outSubInfos, nil
	}
	if settings.Get().SubtitleSources.SubDLSettings.Key == "" {
		return nil, errors.New("api key is empty")
	}

	return s.getSubListFromFile(filePath, true, 0, 0)
}

func (s *Supplier) GetSubListFromFile4Series(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	outSubInfos := make([]supplier.SubInfo, 0)
	if settings.Get().SubtitleSources.SubDLSettings.Enabled == false {
		return outSubInfos, nil
	}
	if settings.Get().SubtitleSources.SubDLSettings.Key == "" {
		return nil, errors.New("api key is empty")
	}

	return s.downloadSub4Series(seriesInfo)
}

func (s *Supplier) GetSubListFromFile4Anime(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	return s.GetSubListFromFile4Series(seriesInfo)
}

func (s *Supplier) downloadSub4Series(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	var allSupplierSubInfo = make([]supplier.SubInfo, 0)

	for _, episodeInfo := range seriesInfo.NeedDlEpsKeyList {
		one, err := s.getSubListFromFile(episodeInfo.FileFullPath, false, episodeInfo.Season, episodeInfo.Episode)
		if err != nil {
			s.log.Errorln(s.GetSupplierName(), "getSubListFromFile", episodeInfo.FileFullPath, err)
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

	candidates, err := s.searchCandidatesWithFallback(mediaInfo, videoFPath, isMovie, season, episode)
	if err != nil {
		return nil, err
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	videoFileName := filepath.Base(videoFPath)
	outSubInfoList := make([]supplier.SubInfo, 0)
	for index, candidate := range candidates {
		imdbID := ""
		if mediaInfo != nil {
			imdbID = mediaInfo.ImdbId
		}
		cacheKey := file_downloader.BuildCacheKey(
			s.GetSupplierName(),
			imdbID,
			strconv.Itoa(index),
			candidate.DownloadURL,
		)
		subInfo, err := s.fileDownloader.Get(s.GetSupplierName(), int64(index), videoFileName, candidate.DownloadURL, 0, 0, cacheKey)
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

func (s *Supplier) searchCandidatesWithFallback(mediaInfo *models.MediaInfo, videoFPath string, isMovie bool, season, episode int) ([]subtitleCandidate, error) {
	httpClient, err := pkg.NewHttpClient()
	if err != nil {
		return nil, err
	}

	queryList := s.buildSearchQueries(mediaInfo, videoFPath, isMovie, season, episode)
	videoFileName := filepath.Base(videoFPath)
	for _, query := range queryList {
		s.log.Infoln(s.GetSupplierName(), videoFileName, "Try Search Query", query)
		searchResponse, err := s.api.SearchSubtitles(httpClient, query)
		if err != nil {
			if errors.Is(err, errSubdlStatusFalse) {
				s.log.Infoln(s.GetSupplierName(), videoFileName, "No subtitle found for query", query)
				continue
			}
			s.log.Errorln(s.GetSupplierName(), videoFileName, "SearchSubtitles", err)
			return nil, err
		}

		candidates := selectCandidates(searchResponse.SubtitleHits(), videoFPath, isMovie, season, episode, s.topic)
		if len(candidates) == 0 {
			continue
		}

		return candidates, nil
	}

	return nil, nil
}

func (s *Supplier) buildSearchQueries(mediaInfo *models.MediaInfo, videoFPath string, isMovie bool, season, episode int) []map[string]string {
	base := map[string]string{
		"api_key":       s.api.apiKey,
		"languages":     s.queryLanguage,
		"subs_per_page": strconv.Itoa(maxInt(3, s.topic)),
		"type":          "movie",
	}
	if isMovie == false {
		base["type"] = "tv"
		base["season_number"] = strconv.Itoa(season)
		base["episode_number"] = strconv.Itoa(episode)
		base["unpack"] = "1"
	}

	if mediaInfo != nil {
		if year := normalizeYear(mediaInfo.Year); year != "" {
			base["year"] = year
		}
	}

	out := make([]map[string]string, 0)
	if mediaInfo != nil {
		if mediaInfo.ImdbId != "" {
			out = append(out, cloneQueryMap(base, map[string]string{"imdb_id": mediaInfo.ImdbId}))
		}
		if mediaInfo.TmdbId != "" {
			out = append(out, cloneQueryMap(base, map[string]string{"tmdb_id": mediaInfo.TmdbId}))
		}
	}

	for _, title := range orderedSearchTitles(mediaInfo, videoFPath) {
		if title == "" {
			continue
		}
		out = append(out, cloneQueryMap(base, map[string]string{"film_name": title}))
	}

	return dedupeQueryMaps(out)
}

func orderedSearchTitles(mediaInfo *models.MediaInfo, videoFPath string) []string {
	if mediaInfo == nil {
		return expandSearchTitleVariants([]string{
			normalizeVideoTitle(videoFPath),
		})
	}
	return expandSearchTitleVariants([]string{
		mediaInfo.TitleEn,
		mediaInfo.OriginalTitle,
		mediaInfo.TitleCn,
		normalizeVideoTitle(videoFPath),
	})
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

func expandSearchTitleVariants(items []string) []string {
	out := make([]string, 0, len(items)*2)
	seen := make(map[string]struct{})
	for _, item := range items {
		for _, variant := range titleVariants(item) {
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

func titleVariants(input string) []string {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil
	}

	variants := []string{input}
	normalized := strings.Join(strings.Fields(stripSearchPunctuation(input)), " ")
	if normalized != "" && strings.EqualFold(normalized, input) == false {
		variants = append(variants, normalized)
	}
	return variants
}

func stripSearchPunctuation(input string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r), unicode.IsNumber(r), unicode.IsSpace(r):
			return r
		default:
			return ' '
		}
	}, input)
}

func selectCandidates(results []SubtitleHit, videoFPath string, isMovie bool, season, episode, limit int) []subtitleCandidate {
	out := make([]subtitleCandidate, 0)
	seen := make(map[string]struct{})

	for _, result := range results {
		if isMovie == false && len(result.UnpackFiles) > 0 {
			for _, unpack := range result.UnpackFiles {
				if unpack.Episode != episode {
					continue
				}
				url := normalizeDownloadURL(unpack.URL)
				if url == "" {
					continue
				}
				if _, ok := seen[url]; ok {
					continue
				}
				seen[url] = struct{}{}
				out = append(out, subtitleCandidate{
					Name:        unpack.Name,
					DownloadURL: url,
					Season:      season,
					Episode:     episode,
					Hi:          unpack.Hi,
					Releases:    result.ReleaseNames(),
				})
			}
		}

		url := normalizeDownloadURL(result.URL)
		if url == "" {
			continue
		}
		if _, ok := seen[url]; ok {
			continue
		}
		seen[url] = struct{}{}
		out = append(out, subtitleCandidate{
			Name:        firstNonEmpty(result.ReleaseName, result.Name),
			DownloadURL: url,
			Season:      result.Season,
			Episode:     result.Episode,
			Hi:          result.Hi,
			Releases:    result.ReleaseNames(),
		})
	}

	rankCandidates(out, videoFPath, isMovie, season, episode)
	if len(out) > limit {
		return out[:limit]
	}
	return out
}

func rankCandidates(candidates []subtitleCandidate, videoFPath string, isMovie bool, season, episode int) {
	if len(candidates) < 2 {
		return
	}

	matcher := ranking.NewTargetMatcher(videoFPath, isMovie)

	sort.SliceStable(candidates, func(i, j int) bool {
		left := scoreCandidate(candidates[i], matcher, isMovie, season, episode)
		right := scoreCandidate(candidates[j], matcher, isMovie, season, episode)
		if left != right {
			return left > right
		}
		return candidates[i].Name < candidates[j].Name
	})
}

func scoreCandidate(candidate subtitleCandidate, matcher ranking.TargetMatcher, isMovie bool, season, episode int) int {
	return ranking.ScoreCandidate(matcher, subdlCandidateMetadata(candidate), ranking.CandidateScoreSpec{
		IsMovie:       isMovie,
		TargetSeason:  season,
		TargetEpisode: episode,
		EpisodeMatchWeights: &ranking.EpisodeMatchWeights{
			ExactMatch:   120,
			SeasonPack:   20,
			WrongEpisode: -120,
		},
		HIPenalty:           -5,
		ReleaseMatchWeights: ranking.SubDLReleaseMatchWeights,
	})
}

func subdlCandidateMetadata(candidate subtitleCandidate) ranking.CandidateMetadata {
	return ranking.CandidateMetadata{
		Name:         candidate.Name,
		ReleaseNames: append([]string(nil), candidate.Releases...),
		Season:       candidate.Season,
		Episode:      candidate.Episode,
		HasHI:        candidate.Hi,
	}
}

func normalizeDownloadURL(rawURL string) string {
	if rawURL == "" {
		return ""
	}
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return rawURL
	}
	if strings.HasPrefix(rawURL, "/") {
		return common2.SubSubDLDownloadRoot + rawURL
	}
	return common2.SubSubDLDownloadRoot + "/" + rawURL
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
		key := item["type"] + "|" + item["imdb_id"] + "|" + item["tmdb_id"] + "|" + item["film_name"] + "|" + item["season_number"] + "|" + item["episode_number"]
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

func firstNonEmpty(items ...string) string {
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			return item
		}
	}
	return ""
}
