package assrt

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"

	common2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/mix_media_info"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/notify_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/sirupsen/logrus"
)

type Supplier struct {
	log               *logrus.Logger
	fileDownloader    *file_downloader.FileDownloader
	isAlive           bool
	theSearchInterval time.Duration
}

func NewSupplier(fileDownloader *file_downloader.FileDownloader) *Supplier {

	sup := Supplier{}
	sup.log = fileDownloader.Log
	sup.fileDownloader = fileDownloader
	sup.isAlive = true // 姒涙顓婚弰顖氬讲娴犮儰濞囬悽銊ф畱閿涘苯顩ч弸?check 閸氬函绱濋崘宥堢殶閺佸濮搁幀?
	if settings.Get().AdvancedSettings.Topic != common2.DownloadSubsPerSite {
		settings.Get().AdvancedSettings.Topic = common2.DownloadSubsPerSite
	}

	sup.theSearchInterval = 20 * time.Second

	return &sup
}

func (s *Supplier) CheckAlive() (bool, int64) {
	if settings.Get().SubtitleSources.AssrtSettings.Token == "" {
		s.isAlive = false
		return false, 0
	}

	startT := time.Now()
	userInfo, err := s.getUserInfo()
	if err != nil {
		s.log.Errorln(s.GetSupplierName(), "CheckAlive", "Error", err)
		s.isAlive = false
		return false, 0
	}
	s.log.Infoln(s.GetSupplierName(), "CheckAlive", "UserInfo.Status:", userInfo.Status, "UserInfo.Quota:", userInfo.User.Quota)
	s.isAlive = true
	return true, time.Since(startT).Milliseconds()
}

func (s *Supplier) IsAlive() bool {
	return s.isAlive
}

func (s *Supplier) OverDailyDownloadLimit() bool {
	if settings.Get().AdvancedSettings.SuppliersSettings.Assrt.DailyDownloadLimit == 0 {
		s.log.Warningln(s.GetSupplierName(), "DailyDownloadLimit is 0, will Skip Download")
		return true
	}

	return false
}

func (s *Supplier) GetLogger() *logrus.Logger {
	return s.log
}

func (s *Supplier) GetSupplierName() string {
	return common2.SubSiteAssrt
}

func (s *Supplier) GetSubListFromFile4Movie(filePath string) ([]supplier.SubInfo, error) {

	outSubInfos := make([]supplier.SubInfo, 0)
	if settings.Get().SubtitleSources.AssrtSettings.Enabled == false {
		return outSubInfos, nil
	}

	if settings.Get().SubtitleSources.AssrtSettings.Token == "" {
		return nil, errors.New("Token is empty")
	}

	return s.getSubListFromFile(filePath, true)
}

func (s *Supplier) GetSubListFromFile4Series(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {

	outSubInfos := make([]supplier.SubInfo, 0)
	if settings.Get().SubtitleSources.AssrtSettings.Enabled == false {
		return outSubInfos, nil
	}

	if settings.Get().SubtitleSources.AssrtSettings.Token == "" {
		return nil, errors.New("Token is empty")
	}

	return s.downloadSub4Series(seriesInfo)
}

func (s *Supplier) GetSubListFromFile4Anime(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {

	outSubInfos := make([]supplier.SubInfo, 0)
	if settings.Get().SubtitleSources.AssrtSettings.Enabled == false {
		return outSubInfos, nil
	}

	if settings.Get().SubtitleSources.AssrtSettings.Token == "" {
		return nil, errors.New("Token is empty")
	}

	return s.downloadSub4Series(seriesInfo)
}

func (s *Supplier) getSubListFromFile(videoFPath string, isMovie bool) ([]supplier.SubInfo, error) {

	defer func() {
		s.log.Debugln(s.GetSupplierName(), videoFPath, "End...")
	}()

	s.log.Debugln(s.GetSupplierName(), videoFPath, "Start...")

	outSubInfoList := make([]supplier.SubInfo, 0)
	mediaInfo, err := mix_media_info.GetMixMediaInfo(s.fileDownloader.MediaInfoDealers, videoFPath, isMovie)
	if err != nil {
		s.log.Errorln(s.GetSupplierName(), videoFPath, "GetMixMediaInfo", err)
		return nil, err
	}
	// 闂団偓鐟曚焦澹橀崚棰佽厬閺傚洤鎮曠粔鏉垮箵閹兼粎鍌ㄩ敍灞惧娑撳秴鍩岀亸杈ㄦЦ閻劏瀚抽弬鍥ф倳缁夊府绱濇潻妯诲娑撳秴鍩岀亸杈ㄦЦ OriginalTitle
	searchOrder := []string{"cn", "en", "org", "file"}
	var found bool
	var searchSubResult *SearchSubResult
	for _, keyWordType := range searchOrder {
		s.log.Infoln(s.GetSupplierName(), videoFPath, "Try Search KeyWord Type", keyWordType)
		found, searchSubResult, err = s.getSubInfoEx(mediaInfo, videoFPath, isMovie, keyWordType)
		if err != nil {
			s.log.Errorln(s.GetSupplierName(), videoFPath, "GetSubInfoEx", keyWordType, err)
			return nil, err
		}
		if found {
			break
		}
	}
	if found == false {
		return nil, nil
	}

	videoFileName := filepath.Base(videoFPath)
	for index, subInfo := range searchSubResult.Sub.Subs {

		// 閼惧嘲褰囬崗铚傜秼閻ㄥ嫪绗呮潪钘夋勾閸р偓
		oneSubDetail, err := s.getSubDetail(subInfo.Id)
		if err != nil {
			s.log.Errorln("getSubDetail", err)
			continue
		}

		if len(oneSubDetail.Sub.Subs) < 1 {
			continue
		}
		// 鏉╂瑩鍣烽棁鈧憰浣规暈閹板繒娈戦弰?ASSRT 鐠囧瓨妲戞禍鍡礉娑撳娴囬惃鍕勾閸р偓閺勵垱婀侀弮鑸垫櫏閹呮畱閿涘矂鍋呮稊鍫濐洤閺嬫粎绱︾€涙ɑ鏆ｆ稉顏勬勾閸р偓閸掓瑤绗夐弰顖涱劀绾喚娈?		// 闂団偓鐟曚胶绱︾€涙娈戞惔鏃囶嚉閺勵垵绻栨稉顏勭摟楠炴洜娈?ID
		nowSubDownloadUrl := oneSubDetail.Sub.Subs[0].Url
		subInfo, err := s.fileDownloader.Get(s.GetSupplierName(), int64(index), videoFileName, nowSubDownloadUrl,
			0, 0,
			// 瀵版鍩屾稉鈧稉顏嗗濞堝﹦娈戦弴澶稿敩 FileDownloadUrl 閻ㄥ嫮澹掑浣哥摟缁楋缚瑕?			fmt.Sprintf("%s-%s-%d", s.GetSupplierName(), subInfo.NativeName, subInfo.Id),
		)
		if err != nil {
			s.log.Error("FileDownloader.Get", err)
			continue
		}

		outSubInfoList = append(outSubInfoList, *subInfo)
		// 婵″倹鐏夋径鐔剁啊闁絼绠炴径姘嚋鐎涙绠风亸杈箲閸?		if len(outSubInfoList) >= settings.Get().AdvancedSettings.Topic {
			return outSubInfoList, nil
		}
	}

	return outSubInfoList, nil
}

func (s *Supplier) getSubInfoEx(mediaInfo *models.MediaInfo, videoFPath string, isMovie bool, keyWordType string) (bool, *SearchSubResult, error) {

	var searchSubResult *SearchSubResult
	var err error
	keyWord, err := mix_media_info.KeyWordSelect(mediaInfo, videoFPath, isMovie, keyWordType)
	if err != nil {
		s.log.Errorln(s.GetSupplierName(), videoFPath, "keyWordSelect", err)
		return false, searchSubResult, err
	}
	searchSubResult, err = s.getSubByKeyWord(keyWord)
	if err != nil {
		s.log.Errorln("getSubByKeyWord", err)
		return false, searchSubResult, err
	}

	videoFileName := filepath.Base(videoFPath)
	if searchSubResult.Sub.Subs == nil || len(searchSubResult.Sub.Subs) == 0 {
		s.log.Infoln(s.GetSupplierName(), videoFileName, "No subtitle found", "KeyWord:", keyWord)
		return false, searchSubResult, nil
	} else {
		return true, searchSubResult, nil
	}
}

func (s *Supplier) downloadSub4Series(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	var allSupplierSubInfo = make([]supplier.SubInfo, 0)

	index := 0
	// 鏉╂瑩鍣烽幏鍨煂閻?seriesInfo 閿涘矂鍣烽棃銏犲瘶閸氼偂绨￠敍宀勬付鐟曚椒绗呮潪钘夌摟楠炴洜娈?Eps 娣団剝浼?	for _, episodeInfo := range seriesInfo.NeedDlEpsKeyList {

		index++
		one, err := s.getSubListFromFile(episodeInfo.FileFullPath, false)
		if err != nil {
			s.log.Errorln(s.GetSupplierName(), "getSubListFromFile", episodeInfo.FileFullPath, err)
			continue
		}
		if one == nil {
			// 濞屸剝婀侀幖婊呭偍閸掓澘鐡ч獮?			s.log.Infoln(s.GetSupplierName(), "Not Find Sub can be download",
				episodeInfo.Title, episodeInfo.Season, episodeInfo.Episode)
			continue
		}
		// 闂団偓鐟曚浇绁撮崐鑲╃舶鐎涙绠风紒鎾寸€?		for i := range one {
			one[i].Season = episodeInfo.Season
			one[i].Episode = episodeInfo.Episode
		}
		allSupplierSubInfo = append(allSupplierSubInfo, one...)
	}
	// 鏉╂柨娲栭崜宥忕礉闂団偓鐟曚焦濡稿В蹇庣娑?Eps 閻?Season Episode 娣団剝浼呮繅顐㈠帠閸掔増鐦℃稉?SubInfo 娑?	return allSupplierSubInfo, nil
}

func (s *Supplier) getSubByKeyWord(keyword string) (*SearchSubResult, error) {

	defer func() {
		time.Sleep(s.theSearchInterval)
	}()

	var searchSubResult SearchSubResult

	s.log.Infoln("Search KeyWord:", keyword)
	tt := url.QueryEscape(keyword)
	httpClient, err := pkg.NewHttpClient()
	if err != nil {
		return nil, err
	}
	var errKnow error
	resp, err := httpClient.R().
		Get(settings.Get().AdvancedSettings.SuppliersSettings.Assrt.RootUrl +
			"/sub/search?q=" + tt +
			"&cnt=15&pos=0" +
			"&token=" + settings.Get().SubtitleSources.AssrtSettings.Token)
	if err != nil {
		return nil, err
	}
	/*
		鏉╂瑩鍣烽張澶夐嚋濮婃绱?Sub 閺堝鈧偐娈戦弮璺衡偓娆愭Ц娑撯偓娑擃亜鍨悰顭掔礉娴ｅ棙妲告俊鍌涚亯娑撹櫣鈹栭惃鍕閸婃瑱绱濋崣鍫熸Ц娑撯偓娑擃亞鈹栭惃鍕波閺嬪嫪缍?		閹碘偓娴犮儱鍤悳棰佽⒈娑擃亞绮ㄩ弸鍕秼闂団偓鐟曚礁骞撶亸婵婄槸鐟欙絾鐎?		SearchSubResultEmpty
		SearchSubResult
		濮ｆ柨顩ф潻娆庨嚋閹懎鍠岄敍?		jsonString := "{\"sub\":{\"action\":\"search\",\"subs\":{},\"result\":\"succeed\",\"keyword\":\"鏉╄姤娼冩径蹇撯枊 S04E07\"},\"status\":0}"
	*/
	err = json.Unmarshal([]byte(resp.String()), &searchSubResult)
	if err != nil {
		// 閸愬秵顒濈亸婵婄槸鐟欙絾鐎界粚鍝勫灙鐞?		var searchSubResultEmpty SearchSubResultEmpty
		err = json.Unmarshal([]byte(resp.String()), &searchSubResultEmpty)
		if err != nil {
			// 婵″倹鐏夋潻妯绘Ц鐟欙絾鐎介柨娆掝嚖閿涘矂鍋呮稊鍫濇皑鐟曚焦濡搁悳鏉挎躬閻ㄥ嫰鏁婄拠顖氭嫲娑撳﹪娼伴惃鍕晩鐠囶垯鍗庨崳銊ㄧ箲閸ョ偛鍤崢?			s.log.Errorln(s.GetSupplierName(), "NewHttpClient:", keyword, errKnow.Error())
			s.log.Errorln(s.GetSupplierName(), "json.Unmarshal", err)
			notify_center.Notify.Add(s.GetSupplierName()+" NewHttpClient", fmt.Sprintf("keyword: %s, resp: %s, error: %s", keyword, resp.String(), errKnow.Error()))
			return nil, err
		}
		// 鐠у鈧壈绻冮崢?		searchSubResult.Sub.Action = searchSubResultEmpty.Sub.Action
		searchSubResult.Sub.Result = searchSubResultEmpty.Sub.Result
		searchSubResult.Sub.Keyword = searchSubResultEmpty.Sub.Keyword
		searchSubResult.Status = searchSubResultEmpty.Status

		return &searchSubResult, nil
	}

	return &searchSubResult, nil
}

func (s *Supplier) getSubDetail(subID int) (OneSubDetail, error) {

	defer func() {
		time.Sleep(s.theSearchInterval)
	}()

	var subDetail OneSubDetail

	httpClient, err := pkg.NewHttpClient()
	if err != nil {
		return subDetail, err
	}
	resp, err := httpClient.R().
		SetQueryParams(map[string]string{
			"token": settings.Get().SubtitleSources.AssrtSettings.Token,
			"id":    strconv.Itoa(subID),
		}).
		SetResult(&subDetail).
		Get(settings.Get().AdvancedSettings.SuppliersSettings.Assrt.RootUrl + "/sub/detail")
	if err != nil {
		if resp != nil {
			s.log.Errorln(s.GetSupplierName(), "NewHttpClient:", subID, err.Error())
			notify_center.Notify.Add(s.GetSupplierName()+" NewHttpClient", fmt.Sprintf("subID: %d, resp: %s, error: %s", subID, resp.String(), err.Error()))

			// 鏉堟挸鍤拫鍐槸閺傚洣娆?			cacheCenterFolder, err := pkg.GetRootCacheCenterFolder()
			if err != nil {
				s.log.Errorln(s.GetSupplierName(), "GetRootCacheCenterFolder", err)
			}
			desJsonInfo := filepath.Join(cacheCenterFolder, strconv.Itoa(subID)+"--assrt_search_error_getSubDetail.json")
			// 閸愭瑥鐡х粭锔胯閸掔増鏋冩禒鍓侇潚
			file, _ := os.Create(desJsonInfo)
			defer func() {
				_ = file.Close()
			}()
			file.WriteString(resp.String())
		}
		return subDetail, err
	}

	return subDetail, nil
}

func (s *Supplier) getUserInfo() (UserInfo, error) {

	var userInfo UserInfo

	httpClient, err := pkg.NewHttpClient()
	if err != nil {
		return userInfo, err
	}
	resp, err := httpClient.R().
		SetQueryParams(map[string]string{
			"token": settings.Get().SubtitleSources.AssrtSettings.Token,
		}).
		SetResult(&userInfo).
		Get(settings.Get().AdvancedSettings.SuppliersSettings.Assrt.RootUrl + "/user/quota")
	if err != nil {
		if resp != nil {
			s.log.Errorln(s.GetSupplierName(), "NewHttpClient:", err.Error())
			notify_center.Notify.Add(s.GetSupplierName()+" NewHttpClient", fmt.Sprintf("resp: %s, error: %s", resp.String(), err.Error()))
		}
		return userInfo, err
	}

	return userInfo, nil
}

type SearchSubResultEmpty struct {
	Sub struct {
		Action string `json:"action"`
		Subs   struct {
		} `json:"subs"`
		Result  string `json:"result"`
		Keyword string `json:"keyword"`
	} `json:"sub"`
	Status int `json:"status"`
}

type SearchSubResult struct {
	Sub struct {
		Action string `json:"action"`
		Subs   []struct {
			Lang struct {
				Desc     string `json:"desc,omitempty"`
				Langlist struct {
					Langcht bool `json:"langcht,omitempty"`
					Langdou bool `json:"langdou,omitempty"`
					Langeng bool `json:"langeng,omitempty"`
					Langchs bool `json:"langchs,omitempty"`
				} `json:"langlist,omitempty"`
			} `json:"lang,omitempty"`
			Id          int    `json:"id,omitempty"`
			VoteScore   int    `json:"vote_score,omitempty"`
			Videoname   string `json:"videoname,omitempty"`
			ReleaseSite string `json:"release_site,omitempty"`
			Revision    int    `json:"revision,omitempty"`
			Subtype     string `json:"subtype,omitempty"`
			NativeName  string `json:"native_name,omitempty"`
			UploadTime  string `json:"upload_time,omitempty"`
		} `json:"subs,omitempty"`
		Result  string `json:"result,omitempty"`
		Keyword string `json:"keyword,omitempty"`
	} `json:"sub,omitempty"`
	Status int `json:"status,omitempty"`
}

type OneSubDetail struct {
	Sub struct {
		Action string `json:"action"`
		Subs   []struct {
			DownCount int `json:"down_count,omitempty"`
			ViewCount int `json:"view_count,omitempty"`
			Lang      struct {
				Desc     string `json:"desc,omitempty"`
				Langlist struct {
					Langcht bool `json:"langcht,omitempty"`
					Langdou bool `json:"langdou,omitempty"`
					Langeng bool `json:"langeng,omitempty"`
					Langchs bool `json:"langchs,omitempty"`
				} `json:"langlist,omitempty"`
			} `json:"lang,omitempty"`
			Size       int    `json:"size,omitempty"`
			Title      string `json:"title,omitempty"`
			Videoname  string `json:"videoname,omitempty"`
			Revision   int    `json:"revision,omitempty"`
			NativeName string `json:"native_name,omitempty"`
			UploadTime string `json:"upload_time,omitempty"`
			Producer   struct {
				Producer string `json:"producer,omitempty"`
				Verifier string `json:"verifier,omitempty"`
				Uploader string `json:"uploader,omitempty"`
				Source   string `json:"source,omitempty"`
			} `json:"producer,omitempty"`
			Subtype     string `json:"subtype,omitempty"`
			VoteScore   int    `json:"vote_score,omitempty"`
			ReleaseSite string `json:"release_site,omitempty"`
			//Filelist    []struct {
			//	S   string `json:"s,omitempty"`
			//	F   string `json:"f,omitempty"`
			//	Url string `json:"url,omitempty"`
			//} `json:"filelist,omitempty"`
			Id       int    `json:"id,omitempty"`
			Filename string `json:"filename,omitempty"`
			Url      string `json:"url,omitempty"`
		} `json:"subs,omitempty"`
		Result string `json:"result,omitempty"`
	} `json:"sub,omitempty"`
	Status int `json:"status,omitempty"`
}

type UserInfo struct {
	User struct {
		Action string `json:"action,omitempty"`
		Result string `json:"result,omitempty"`
		Quota  int    `json:"quota,omitempty"`
	} `json:"user,omitempty"`
	Status int `json:"status,omitempty"`
}