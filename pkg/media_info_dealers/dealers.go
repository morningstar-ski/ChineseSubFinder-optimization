package media_info_dealers

import (
	"fmt"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/tmdb_api"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type Dealers struct {
	Logger     *logrus.Logger
	tmdbHelper *tmdb_api.TmdbApi
}

func NewDealers(log *logrus.Logger) *Dealers {
	return &Dealers{Logger: log}
}

func (d *Dealers) SetTmdbHelperInstance(tmdbHelper *tmdb_api.TmdbApi) {
	d.tmdbHelper = tmdbHelper
}

// ConvertId currently only supports converting a TMDB ID into an IMDb ID.
func (d *Dealers) ConvertId(iD string, idType string, isMovieOrSeries bool) (*tmdb_api.ConvertIdResult, error) {
	if d.tmdbHelper != nil && settings.Get().AdvancedSettings.TmdbApiSettings.Enable == true && settings.Get().AdvancedSettings.TmdbApiSettings.ApiKey != "" {
		return d.tmdbHelper.ConvertId(iD, idType, isMovieOrSeries)
	}

	return nil, errors.New("tmdb api is not configured")
}

func (d *Dealers) GetMediaInfo(id, source, videoType string) (*models.MediaInfo, error) {
	if d.tmdbHelper != nil && settings.Get().AdvancedSettings.TmdbApiSettings.Enable == true && settings.Get().AdvancedSettings.TmdbApiSettings.ApiKey != "" {
		return d.getMediaInfoFromSelfApi(id, source, videoType)
	}

	return nil, errors.New("tmdb api is not configured")
}

// getMediaInfoFromSelfApi queries media info through the user's configured TMDB API.
func (d *Dealers) getMediaInfoFromSelfApi(id, source, videoType string) (*models.MediaInfo, error) {
	imdbId := ""
	var tmdbID int64
	idType := ""
	isMovieOrSeries := false
	if source == "imdb" {
		idType = tmdb_api.ImdbID
		imdbId = id
		if videoType == "movie" {
			isMovieOrSeries = true
		} else if videoType == "series" {
			isMovieOrSeries = false
		} else {
			return nil, errors.New("videoType is not movie or series")
		}
	} else if source == "tmdb" {
		if videoType == "movie" {
			idType = tmdb_api.TmdbID
			isMovieOrSeries = true
		} else if videoType == "series" {
			idType = tmdb_api.TmdbID
			isMovieOrSeries = false
		} else {
			return nil, errors.New("videoType is not movie or series")
		}
	} else {
		return nil, errors.New("source is not support")
	}

	findByIDEn, err := d.tmdbHelper.GetInfo(id, idType, isMovieOrSeries, true)
	if err != nil {
		return nil, fmt.Errorf("error while getting info from TMDB: %v", err)
	}
	findByIDCn, err := d.tmdbHelper.GetInfo(id, idType, isMovieOrSeries, false)
	if err != nil {
		return nil, fmt.Errorf("error while getting info from TMDB: %v", err)
	}

	originalTitle := ""
	originalLanguage := ""
	titleEn := ""
	titleCn := ""
	year := ""
	if isMovieOrSeries == true {
		if len(findByIDEn.MovieResults) < 1 {
			return nil, errors.New("not found movie info from tmdb")
		}
		tmdbID = findByIDEn.MovieResults[0].ID
		originalTitle = findByIDEn.MovieResults[0].OriginalTitle
		originalLanguage = findByIDEn.MovieResults[0].OriginalLanguage
		titleEn = findByIDEn.MovieResults[0].Title
		titleCn = findByIDCn.MovieResults[0].Title
		year = findByIDEn.MovieResults[0].ReleaseDate
	} else {
		if len(findByIDEn.TvResults) < 1 {
			return nil, errors.New("not found series info from tmdb")
		}
		tmdbID = findByIDEn.TvResults[0].ID
		originalTitle = findByIDEn.TvResults[0].OriginalName
		originalLanguage = findByIDEn.TvResults[0].OriginalLanguage
		titleEn = findByIDEn.TvResults[0].Name
		titleCn = findByIDCn.TvResults[0].Name
		year = findByIDEn.TvResults[0].FirstAirDate
	}

	return &models.MediaInfo{
		TmdbId:           fmt.Sprintf("%d", tmdbID),
		ImdbId:           imdbId,
		OriginalTitle:    originalTitle,
		OriginalLanguage: originalLanguage,
		TitleEn:          titleEn,
		TitleCn:          titleCn,
		Year:             year,
	}, nil
}
