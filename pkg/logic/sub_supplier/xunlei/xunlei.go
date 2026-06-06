package xunlei

import (
	"crypto/sha1"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/decode"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/notify_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_parser_hub"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/sirupsen/logrus"
)

type Supplier struct {
	log            *logrus.Logger
	fileDownloader *file_downloader.FileDownloader
	topic          int
	isAlive        bool
}

func NewSupplier(fileDownloader *file_downloader.FileDownloader) *Supplier {

	sup := Supplier{}
	sup.log = fileDownloader.Log
	sup.fileDownloader = fileDownloader
	sup.topic = common.DownloadSubsPerSite
	sup.isAlive = true

	if settings.Get().AdvancedSettings.Topic > 0 && settings.Get().AdvancedSettings.Topic != sup.topic {
		sup.topic = settings.Get().AdvancedSettings.Topic
	}

	return &sup
}

func (s *Supplier) CheckAlive() (bool, int64) {

	startT := time.Now()
	jsonList, err := s.getSubInfos(checkFileName, checkCID)
	if err != nil {
		s.log.Errorln(s.GetSupplierName(), "CheckAlive", "Error", err)
		s.isAlive = false
		return false, 0
	}

	if len(jsonList.Sublist) < 1 {
		s.log.Errorln(s.GetSupplierName(), "CheckAlive", "Sublist < 1")
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

	if settings.Get().AdvancedSettings.SuppliersSettings.Xunlei.DailyDownloadLimit == 0 {
		s.log.Warningln(s.GetSupplierName(), "DailyDownloadLimit is 0, will Skip Download")
		return true
	}

	return false
}

func (s *Supplier) GetLogger() *logrus.Logger {
	return s.log
}

func (s *Supplier) GetSupplierName() string {
	return common.SubSiteXunLei
}

func (s *Supplier) GetSubListFromFile4Movie(filePath string) ([]supplier.SubInfo, error) {
	return s.getSubListFromFile(filePath)
}

func (s *Supplier) GetSubListFromFile4Series(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	return s.downloadSub4Series(seriesInfo)
}

func (s *Supplier) GetSubListFromFile4Anime(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	return s.downloadSub4Series(seriesInfo)
}

func (s *Supplier) getSubListFromFile(filePath string) ([]supplier.SubInfo, error) {

	defer func() {
		s.log.Debugln(s.GetSupplierName(), filePath, "End...")
	}()

	s.log.Debugln(s.GetSupplierName(), filePath, "Start...")

	if pkg.IsFile(filePath) == false {
		bok, _, _ := decode.IsFakeBDMVWorked(filePath)
		if bok == false {
			nowError := errors.New(fmt.Sprintf("%s %s %s",
				s.GetSupplierName(),
				filePath,
				"not exist, and it`s not a Blue ray Video FakeFileName"))
			s.log.Errorln(nowError)
			return nil, nowError
		}
	}

	cid, err := s.getCid(filePath)
	if len(cid) == 0 {
		return nil, common.XunLeiCIdIsEmpty
	}

	jsonList, err := s.getSubInfos(filePath, cid)
	if err != nil {
		return nil, err
	}

	selectedSubtitles := filterAndSelectSubtitles(jsonList.Sublist, s.topic)
	videoFileName := filepath.Base(filePath)
	outSubList := make([]supplier.SubInfo, 0, len(selectedSubtitles))
	for i, v := range selectedSubtitles {
		subInfo, err := s.fileDownloader.Get(s.GetSupplierName(), int64(i), videoFileName, v.Surl, v.Svote, v.Roffset)
		if err != nil {
			s.log.Error("FileDownloader.Get", err)
			continue
		}

		outSubList = append(outSubList, *subInfo)
	}

	return outSubList, nil
}

func filterAndSelectSubtitles(subtitles []SublistXunLei, topic int) []SublistXunLei {
	selected := make([]SublistXunLei, 0)
	for _, v := range subtitles {
		if strings.TrimSpace(v.Scid) == "" {
			continue
		}
		tmpLang := language.LangConverter4Sub_Supplier(v.Language)
		if language.HasChineseLang(tmpLang) == true && sub_parser_hub.IsSubTypeWanted(v.Sname) == true {
			selected = append(selected, v)
		}
	}

	if len(selected) >= topic {
		return selected
	}

	for _, v := range subtitles {
		if len(selected) >= topic {
			break
		}
		if strings.TrimSpace(v.Scid) == "" {
			continue
		}
		tmpLang := language.LangConverter4Sub_Supplier(v.Language)
		if language.HasChineseLang(tmpLang) == false {
			selected = append(selected, v)
		}
	}

	return selected
}

func (s *Supplier) getSubInfos(filePath, cid string) (SublistSliceXunLei, error) {
	var jsonList SublistSliceXunLei

	httpClient, err := pkg.NewHttpClient()
	if err != nil {
		return jsonList, err
	}
	resp, err := httpClient.R().
		SetResult(&jsonList).
		Get(fmt.Sprintf(settings.Get().AdvancedSettings.SuppliersSettings.Xunlei.RootUrl, cid))
	if err != nil {
		if resp != nil {
			s.log.Errorln(s.GetSupplierName(), "NewHttpClient:", filePath, err.Error())
			notify_center.Notify.Add(s.GetSupplierName()+" NewHttpClient", fmt.Sprintf("filePath: %s, resp: %s, error: %s", filePath, resp.String(), err.Error()))
		}
		return jsonList, err
	}

	return jsonList, nil
}

func (s *Supplier) getCid(filePath string) (string, error) {
	hash := ""
	sha1Ctx := sha1.New()

	fp, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = fp.Close()
	}()
	stat, err := fp.Stat()
	if err != nil {
		return "", err
	}
	fileLength := stat.Size()
	if fileLength < 0xF000 {
		return "", err
	}
	bufferSize := int64(0x5000)
	positions := []int64{0, int64(math.Floor(float64(fileLength) / 3)), fileLength - bufferSize}
	for _, position := range positions {
		buffer := make([]byte, bufferSize)
		_, err = fp.Seek(position, 0)
		if err != nil {
			return "", err
		}
		_, err = fp.Read(buffer)
		if err != nil {
			return "", err
		}
		sha1Ctx.Write(buffer)
	}

	hash = fmt.Sprintf("%X", sha1Ctx.Sum(nil))
	return hash, nil
}

func (s *Supplier) downloadSub4Series(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	allSupplierSubInfo := make([]supplier.SubInfo, 0)
	for _, episodeInfo := range seriesInfo.NeedDlEpsKeyList {
		one, err := s.getSubListFromFile(episodeInfo.FileFullPath)
		if err != nil {
			s.log.Errorln(s.GetSupplierName(), "getSubListFromFile", episodeInfo.Season, episodeInfo.Episode,
				episodeInfo.FileFullPath)
			continue
		}
		if one == nil {
			s.log.Infoln(s.GetSupplierName(), "Not Find Sub can be download",
				episodeInfo.Title, episodeInfo.Season, episodeInfo.Episode)
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

type SublistXunLei struct {
	Scid     string `json:"scid"`
	Sname    string `json:"sname"`
	Language string `json:"language"`
	Rate     string `json:"rate"`
	Surl     string `json:"surl"`
	Svote    int64  `json:"svote"`
	Roffset  int64  `json:"roffset"`
}

type SublistSliceXunLei struct {
	Sublist []SublistXunLei
}

const (
	checkFileName = "CheckFileName"
	checkCID      = "FB4E2AFF106112136DFC5ACC7339EB29D1EC0CF8"
)
