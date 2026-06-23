package settings

import (
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
)

type SuppliersSettings struct {
	Xunlei         *OneSupplierSettings `json:"xunlei"`
	Shooter        *OneSupplierSettings `json:"shooter"`
	Assrt          *OneSupplierSettings `json:"assrt"`
	SubDL          *OneSupplierSettings `json:"subdl"`
	OpenSubtitles  *OneSupplierSettings `json:"opensubtitles"`
	TVSubtitles    *OneSupplierSettings `json:"tvsubtitles"`
	MovieSubtitles *OneSupplierSettings `json:"moviesubtitles"`
	SubtitleCat    *OneSupplierSettings `json:"subtitlecat"`
	SubHD          *OneSupplierSettings `json:"subhd"`
	Zimuku         *OneSupplierSettings `json:"zimuku"`
}

func NewSuppliersSettings() *SuppliersSettings {
	return &SuppliersSettings{
		Xunlei:         NewOneSupplierSettings(common.SubSiteXunLei, common.SubXunLeiRootUrlDef, "", -1),
		Shooter:        NewOneSupplierSettings(common.SubSiteShooter, common.SubShooterRootUrlDef, "", -1),
		Assrt:          NewOneSupplierSettings(common.SubSiteAssrt, common.SubAssrtRootUrlDef, "", -1),
		SubDL:          NewOneSupplierSettings(common.SubSiteSubDL, common.SubSubDLRootUrlDef, common.SubSubDLSearchUrl, -1),
		OpenSubtitles:  NewOneSupplierSettings(common.SubSiteOpenSubtitles, common.SubOpenSubtitlesRootUrlDef, common.SubOpenSubtitlesSearchUrl, -1),
		TVSubtitles:    NewOneSupplierSettings(common.SubSiteTVSubtitles, common.SubTVSubtitlesRootUrlDef, common.SubTVSubtitlesSearchUrl, -1),
		MovieSubtitles: NewOneSupplierSettings(common.SubSiteMovieSubtitles, common.SubMovieSubtitlesRootUrlDef, common.SubMovieSubtitlesSearchUrl, -1),
		SubtitleCat:    NewOneSupplierSettings(common.SubSiteSubtitleCat, common.SubSubtitleCatRootUrlDef, common.SubSubtitleCatSearchUrl, -1),
		SubHD:          NewOneSupplierSettings(common.SubSiteSubHd, common.SubSubHDRootUrlDef, common.SubSubHDSearchUrl, 20),
		Zimuku:         NewOneSupplierSettings(common.SubSiteZiMuKu, common.SubZiMuKuRootUrlDef, common.SubZiMuKuSearchFormatUrl, 20),
	}
}

// ReSetSearchUrl keeps built-in provider search paths aligned with code defaults.
func (s *SuppliersSettings) ReSetSearchUrl() {
	s.ensureDefaults()
	s.SubDL.SearchUrl = common.SubSubDLSearchUrl
	s.OpenSubtitles.SearchUrl = common.SubOpenSubtitlesSearchUrl
	s.TVSubtitles.SearchUrl = common.SubTVSubtitlesSearchUrl
	s.MovieSubtitles.SearchUrl = common.SubMovieSubtitlesSearchUrl
	s.SubtitleCat.SearchUrl = common.SubSubtitleCatSearchUrl
	s.SubHD.SearchUrl = common.SubSubHDSearchUrl
	s.Zimuku.SearchUrl = common.SubZiMuKuSearchFormatUrl
}

type OneSupplierSettings struct {
	Name               string `json:"name"`
	RootUrl            string `json:"root_url"`
	SearchUrl          string `json:"search_url"`
	DailyDownloadLimit int    `json:"daily_download_limit" default:"-1"`
}

func NewOneSupplierSettings(name string, rootUrl, searchUrl string, dailyDownloadLimit int) *OneSupplierSettings {
	return &OneSupplierSettings{Name: name, RootUrl: rootUrl, SearchUrl: searchUrl, DailyDownloadLimit: dailyDownloadLimit}
}

func (s *OneSupplierSettings) GetSearchUrl() string {
	return s.RootUrl + s.SearchUrl
}

func (s *SuppliersSettings) ensureDefaults() {
	defaults := NewSuppliersSettings()

	if s.Xunlei == nil {
		s.Xunlei = defaults.Xunlei
	}
	if s.Shooter == nil {
		s.Shooter = defaults.Shooter
	}
	if s.Assrt == nil {
		s.Assrt = defaults.Assrt
	}
	if s.SubDL == nil {
		s.SubDL = defaults.SubDL
	}
	if s.OpenSubtitles == nil {
		s.OpenSubtitles = defaults.OpenSubtitles
	}
	if s.TVSubtitles == nil {
		s.TVSubtitles = defaults.TVSubtitles
	}
	if s.MovieSubtitles == nil {
		s.MovieSubtitles = defaults.MovieSubtitles
	}
	if s.SubtitleCat == nil {
		s.SubtitleCat = defaults.SubtitleCat
	}
	if s.SubHD == nil {
		s.SubHD = defaults.SubHD
	}
	if s.Zimuku == nil {
		s.Zimuku = defaults.Zimuku
	}
}
