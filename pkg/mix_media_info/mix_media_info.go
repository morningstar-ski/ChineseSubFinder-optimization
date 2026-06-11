package mix_media_info

import (
	"errors"
	"path/filepath"
	"strings"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/media_info_dealers"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/decode"
	"gorm.io/gorm"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/dao"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/imdb_helper"
)

func GetMixMediaInfo(
	dealers *media_info_dealers.Dealers,
	videoFPath string, isMovie bool) (*models.MediaInfo, error) {
	// 从本地读取 IMDB ID 信息，找到基本 ID 信息后，也会去 IMDB web 找到对应的额外信息填充
	imdbInfo, err := imdb_helper.GetIMDBInfoFromVideoFile(dealers, videoFPath, isMovie)
	if err != nil {
		return buildFallbackMediaInfo(videoFPath, isMovie, nil), nil
	}

	source := "imdb"
	videoType := "movie"
	if isMovie == false {
		videoType = "series"
	}

	// TMDB ID 是否存在
	if imdbInfo.TmdbId == "" {
		// 需要去 web 查询
		source = "imdb"
		mediaInfo, err := GetMediaInfoAndSave(dealers, imdbInfo, imdbInfo.IMDBID, source, videoType)
		if err == nil {
			return mediaInfo, nil
		}
		return buildFallbackMediaInfo(videoFPath, isMovie, imdbInfo), nil
	} else {
		// 已经存在，从本地拿去信息
		// 首先从数据库中查找是否存在这个 IMDB 信息，如果不存在再使用 Web 查找，且写入数据库
		var mediaInfos []models.MediaInfo
		// 把嵌套关联的 has many 的信息都查询出来
		dao.GetDb().Limit(1).Where(&models.MediaInfo{TmdbId: imdbInfo.TmdbId}).Find(&mediaInfos)

		if len(mediaInfos) > 0 {
			// 找到
			return &mediaInfos[0], nil
		} else {
			// 没有找到本地缓存的 TMDB ID 信息，需要去 web 查询
			source = "imdb"
			mediaInfo, err := GetMediaInfoAndSave(dealers, imdbInfo, imdbInfo.IMDBID, source, videoType)
			if err == nil {
				return mediaInfo, nil
			}
			return buildFallbackMediaInfo(videoFPath, isMovie, imdbInfo), nil
		}
	}
}

// GetMediaInfoAndSave 通过 IMDB ID 查询媒体信息，并保存到数据库，IMDB 和 MediaInfo 都会进行保存 // source，options=imdb|tmdb  videoType，options=movie|series
func GetMediaInfoAndSave(dealers *media_info_dealers.Dealers, imdbInfo *models.IMDBInfo, id, source, videoType string) (*models.MediaInfo, error) {

	mediaInfo, err := dealers.GetMediaInfo(id, source, videoType)
	if err != nil {
		return nil, err
	}
	if mediaInfo == nil {
		// 超过 9次 30s 等待都没有查询到，返回错误
		return nil, errors.New("can't get media info from subtitle.best api")
	}
	// 更新 ID
	imdbInfo.TmdbId = mediaInfo.TmdbId
	err = dao.GetDb().Transaction(func(tx *gorm.DB) error {

		// 在事务中执行一些 db 操作（从这里开始，您应该使用 'tx' 而不是 'db'）
		if err := tx.Save(imdbInfo).Error; err != nil {
			// 返回任何错误都会回滚事务
			return err
		}
		if err := tx.Save(mediaInfo).Error; err != nil {
			// 返回任何错误都会回滚事务
			return err
		}
		// 返回 nil 提交事务
		return nil
	})
	if err != nil {
		return nil, err
	}

	return mediaInfo, nil
}

// KeyWordSelect keyWordType cn, 中文， en，英文，org，原始名称，file，归一化文件名
func KeyWordSelect(mediaInfo *models.MediaInfo, videoFPath string, isMovie bool, keyWordType string) (string, error) {

	keyWord := ""

	if keyWordType == "cn" {
		keyWord = mediaInfo.TitleCn
		if keyWord == "" {
			return "", errors.New("TitleCn is empty")
		}
	} else if keyWordType == "en" {
		keyWord = mediaInfo.TitleEn
		if keyWord == "" {
			return "", errors.New("TitleEn is empty")
		}
	} else if keyWordType == "org" {
		keyWord = mediaInfo.OriginalTitle
		if keyWord == "" {
			return "", errors.New("OriginalTitle is empty")
		}
	} else if keyWordType == "file" {
		keyWord = normalizeFileKeyword(videoFPath)
		if keyWord == "" {
			return "", errors.New("file keyword is empty")
		}
	} else {
		return "", errors.New("keyWordType is not cn, en, org, file")
	}

	if isMovie == false {
		// 连续剧需要额外补充 S01E01 这样的信息
		epsVideoNfoInfo, err := decode.GetVideoNfoInfo4OneSeriesEpisode(videoFPath)
		if err != nil {
			return "", err
		}
		keyWord += " " + pkg.GetEpisodeKeyName(epsVideoNfoInfo.Season, epsVideoNfoInfo.Episode, true)
	}

	return keyWord, nil
}

func normalizeFileKeyword(videoFPath string) string {
	fileName := strings.TrimSuffix(filepath.Base(videoFPath), filepath.Ext(videoFPath))
	if fileInfo, err := decode.GetVideoInfoFromFileName(fileName); err == nil && fileInfo != nil && fileInfo.Title != "" {
		fileName = fileInfo.Title
	}
	fileName = pkg.ReplaceSpecString(fileName, " ")
	fileName = strings.Join(strings.Fields(fileName), " ")
	return fileName
}

func ExpandSearchKeywords(items ...string) []string {
	out := make([]string, 0, len(items)*2)
	seen := make(map[string]struct{}, len(items)*2)
	appendKeyword := func(item string) {
		item = strings.TrimSpace(item)
		if item == "" {
			return
		}
		item = strings.Join(strings.Fields(item), " ")
		if item == "" {
			return
		}
		key := strings.ToLower(item)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}

	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		appendKeyword(item)
		appendKeyword(replaceKeywordAmpersand(item))
		appendKeyword(replaceKeywordAnd(item))
	}

	return out
}

func NormalizeComparableTitle(title string) string {
	title = normalizeKeywordConnectorTokens(title)
	title = pkg.ReplaceSpecString(title, " ")
	title = strings.ToLower(strings.Join(strings.Fields(title), " "))
	return title
}

func replaceKeywordAmpersand(item string) string {
	fields := strings.Fields(strings.ReplaceAll(strings.ReplaceAll(item, "＆", " & "), "&", " & "))
	if len(fields) == 0 {
		return ""
	}
	for index, field := range fields {
		if field == "&" {
			fields[index] = "and"
		}
	}
	return strings.Join(fields, " ")
}

func replaceKeywordAnd(item string) string {
	fields := strings.Fields(item)
	if len(fields) == 0 {
		return ""
	}
	for index, field := range fields {
		if strings.EqualFold(field, "and") {
			fields[index] = "&"
		}
	}
	return strings.Join(fields, " ")
}

func normalizeKeywordConnectorTokens(item string) string {
	fields := strings.Fields(strings.ReplaceAll(strings.ReplaceAll(item, "＆", " & "), "&", " & "))
	if len(fields) == 0 {
		return ""
	}
	for index, field := range fields {
		if field == "&" || strings.EqualFold(field, "and") {
			fields[index] = "and"
		}
	}
	return strings.Join(fields, " ")
}

func buildFallbackMediaInfo(videoFPath string, isMovie bool, imdbInfo *models.IMDBInfo) *models.MediaInfo {
	videoNfoInfo, _, _ := decode.GetVideoInfoFromFileFullPath(videoFPath, isMovie)

	title := strings.TrimSpace(videoNfoInfo.Title)
	if title == "" {
		title = normalizeFileKeyword(videoFPath)
	}
	year := strings.TrimSpace(videoNfoInfo.Year)
	if year == "" {
		releaseDate := strings.TrimSpace(videoNfoInfo.ReleaseDate)
		if releaseDate != "" {
			if parsed, err := time.Parse("2006-01-02", releaseDate); err == nil {
				year = parsed.Format("2006-01-02")
			} else if len(releaseDate) >= 4 {
				year = releaseDate[:4]
			}
		}
	}

	out := &models.MediaInfo{
		ImdbId:        strings.TrimSpace(videoNfoInfo.ImdbId),
		TmdbId:        strings.TrimSpace(videoNfoInfo.TmdbId),
		OriginalTitle: title,
		TitleEn:       title,
		TitleCn:       title,
		Year:          year,
	}
	if imdbInfo != nil {
		if out.ImdbId == "" {
			out.ImdbId = imdbInfo.IMDBID
		}
		if out.TmdbId == "" {
			out.TmdbId = imdbInfo.TmdbId
		}
	}
	return out
}
