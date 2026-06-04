package assrt

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"

	common2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/ranking"

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

var assrtSearchKeywordOrder = []string{"cn", "en", "org", "file"}

func NewSupplier(fileDownloader *file_downloader.FileDownloader) *Supplier {

	sup := Supplier{}
	sup.log = fileDownloader.Log
	sup.fileDownloader = fileDownloader
	sup.isAlive = true // 婵帗绋掗…鍫ヮ敇婵犳艾鍙婃い鏍ㄨ壘鐠佹彃霉閻橆喖鍔欏┑鐐叉喘閹粙濡歌閻ｉ亶鏌ㄥ☉妯垮妞も敪鍥у嚑?check 闂佸憡鑹鹃崙鐣屾濠靛绀冪€广儱鐗忓▓鍫曟煛娴ｅ壊鍤熷┑顔芥倐楠炩偓?
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
	searchSubResult, err := s.getSubInfoWithFallback(mediaInfo, videoFPath, isMovie)
	if err != nil {
		s.log.Errorln(s.GetSupplierName(), videoFPath, "getSubInfoWithFallback", err)
		return nil, err
	}
	if searchSubResult == nil || searchSubResult.Sub.Subs == nil || len(searchSubResult.Sub.Subs) == 0 {
		return nil, nil
	}
	sortAssrtSearchSubs(searchSubResult.Sub.Subs, videoFPath, isMovie)

	videoFileName := filepath.Base(videoFPath)
	for index, subInfo := range searchSubResult.Sub.Subs {

		// 闂佸吋鍎抽崲鑼躲亹閸ヮ剙绀傞柧姘€荤粔濂告煟閵娿儱顏х紒妤€鎳忓顏堟寠婢跺瀣€闂佺鈧崑?
		oneSubDetail, err := s.getSubDetail(int(subInfo.Id))
		if err != nil {
			s.log.Errorln("getSubDetail", err)
			continue
		}

		if len(oneSubDetail.Sub.Subs) < 1 {
			continue
		}
		// 闁哄鏅滈悷鈺呭闯閻戣姤顥嗛柍褜鍓涢幉鐗堟媴鐟欏嫭娈橀梺瑙勬緲缁绘帒鈻撻幋锕€鍙?ASSRT 闁荤姴娲ら悺銊ノｉ幋鐐殿洸闁糕槅鍘剧粈澶娾槈閹炬剚鍎撴繛鏉戞喘閹啴宕熼锝呭瑎闂佺鈧崑鎾绘煛閸曢潧鐏℃繝鈧笟鈧顕€鎳滈崹顐ｇ彙闂佽鍎搁崨顔炬殸闂佹寧绋戦惌鍌炲磻閸涱喚鈻曢柛顐ｇ箥濞层倝鏌＄€ｎ偆鐭岀紒鎲嬪閳ь剚绋掕摫闁哄棴绲剧粙澶愵敂閸曨剙瀣€闂佺鈧崑鎾绘煕閹烘挾鎳佺紒妤€顦靛浼搭敍濞戝崬濡х紒缁㈠弾閸犳艾鈻?		// 闂傚倸娲犻崑鎾绘偡閺囨俺鍏岀紒鎲嬪閳ь剚绋掗…鍥р枔閹寸偞鍎熼柡鍐ㄦ祩閸ゅ鏌￠崟闈涚仴缂佺粯鐗楃粙澶愵敂閸曨厽鎲诲Δ鐘靛仦濞叉粌鈻?ID
		nowSubDownloadUrl := oneSubDetail.Sub.Subs[0].Url
		subInfo, err := s.fileDownloader.Get(s.GetSupplierName(), int64(index), videoFileName, nowSubDownloadUrl,
			0, 0,
			// 閻庣數澧楅〃鍛村春鐏炲墽鈻旈柍褜鍓氱粙澶愵敂閸℃妫楀┑鐐茬墕閿曪箑鈻撻幋锕€鍗冲鍓侇焾閺?FileDownloadUrl 闂佹眹鍔岀€氼喗绔熼幒鎴殫濞达絽鎽滈幗鐔虹磼濡ゅ绱伴悷?
			fmt.Sprintf("%s-%s-%d", s.GetSupplierName(), subInfo.NativeName, subInfo.Id),
		)
		if err != nil {
			s.log.Error("FileDownloader.Get", err)
			continue
		}

		outSubInfoList = append(outSubInfoList, *subInfo)
		// 婵犵鈧啿鈧綊鎮樻径瀣窞闁绘柨澧庨崯濠囨⒑椤撱劎瀵肩紒鐘靛仦瀵板嫬顫濋锕€娈濋柣搴㈢⊕椤ㄥ懐绮绘搴濈剨閺夊牜鍋嗙粻鏌ユ煕?
		if len(outSubInfoList) >= settings.Get().AdvancedSettings.Topic {
			return outSubInfoList, nil
		}
	}

	return outSubInfoList, nil
}

func (s *Supplier) getSubInfoWithFallback(mediaInfo *models.MediaInfo, videoFPath string, isMovie bool) (*SearchSubResult, error) {
	videoFileName := filepath.Base(videoFPath)
	for _, keyWordType := range assrtSearchKeywordOrder {
		keyWord, err := mix_media_info.KeyWordSelect(mediaInfo, videoFPath, isMovie, keyWordType)
		if err != nil {
			s.log.Infoln(s.GetSupplierName(), videoFileName, "Skip Search KeyWordType", keyWordType, err)
			continue
		}

		s.log.Infoln(s.GetSupplierName(), videoFileName, "Try Search KeyWordType", keyWordType, "KeyWord:", keyWord)
		searchSubResult, err := s.getSubByKeyWord(keyWord)
		if err != nil {
			s.log.Errorln(s.GetSupplierName(), videoFileName, "Search KeyWordType", keyWordType, err)
			return nil, err
		}
		if searchSubResult.Sub.Subs == nil || len(searchSubResult.Sub.Subs) == 0 {
			s.log.Infoln(s.GetSupplierName(), videoFileName, "No subtitle found", "KeyWordType:", keyWordType, "KeyWord:", keyWord)
			continue
		}

		return searchSubResult, nil
	}

	return nil, nil
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
	// 闁哄鏅滈悷鈺呭闯閻戣棄绠柛顭戝枛閻撳倿鏌?seriesInfo 闂佹寧绋戦惌鍌炲闯閻戣姤顥堥柕蹇曞Т閻﹀爼鏌涘鐓庝簵缂侇煉绻濋弫宥呯暆閸曨兛绮柣鐔告磻濡炴帞绮崨顔藉闁芥ê顦遍幗鐔割殽閻愬瓨绀堟繛?Eps 婵烇絽娲犻崜婵囧?
	for _, episodeInfo := range seriesInfo.NeedDlEpsKeyList {

		index++
		one, err := s.getSubListFromFile(episodeInfo.FileFullPath, false)
		if err != nil {
			s.log.Errorln(s.GetSupplierName(), "getSubListFromFile", episodeInfo.FileFullPath, err)
			continue
		}
		if one == nil {
			// 濠电偛澶囬崜婵嗭耿娓氣偓楠炴牕顭ㄩ崨顓炰憾闂佸憡甯楀姗€鎮鸿閻?
			s.log.Infoln(s.GetSupplierName(), "Not Find Sub can be download",
				episodeInfo.Title, episodeInfo.Season, episodeInfo.Episode)
			continue
		}
		// 闂傚倸娲犻崑鎾绘偡閺囨碍绁扮紒浣规尦瀹曟劙鎳栭埡鍐煑闁诲孩绋掗〃鍛不妞嬪海纾奸柟鎯ь嚟閳?
		for i := range one {
			one[i].Season = episodeInfo.Season
			one[i].Episode = episodeInfo.Episode
		}
		allSupplierSubInfo = append(allSupplierSubInfo, one...)
	}
	// 闁哄鏅滈弻銊ッ洪弽顓炵鐎广儱绻掔粈澶愭⒒閸ワ絽浜鹃柣鐔告磻閻掞附淇婄粙鍟冩帟绠涙惔锝庝紘婵?Eps 闂?Season Episode 婵烇絽娲犻崜婵囧閸涱喚绠欐い鎰╁灩鐢娀鏌涢幒鏂款暭闁伙腹鍓濈粙?SubInfo 婵?
	return allSupplierSubInfo, nil
}

func (s *Supplier) getSubByKeyWord(keyword string) (*SearchSubResult, error) {

	defer func() {
		time.Sleep(s.theSearchInterval)
	}()

	var searchSubResult SearchSubResult
	var errKnow error

	s.log.Infoln("Search KeyWord:", keyword)
	tt := url.QueryEscape(keyword)
	httpClient, err := pkg.NewHttpClient()
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.R().
		Get(settings.Get().AdvancedSettings.SuppliersSettings.Assrt.RootUrl +
			"/sub/search?q=" + tt +
			"&cnt=15&pos=0" +
			"&token=" + settings.Get().SubtitleSources.AssrtSettings.Token)
	if err != nil {
		return nil, err
	}
	/*
		闁哄鏅滈悷鈺呭闯閻戣棄瀚夊璺侯樀閸ゅ鎱ㄦ繝鍐炬缂?Sub 闂佸搫鐗嗛ˇ顖炲焵椤掑啫浜规繛鍫熷灴瀵喚鎹勭悰鈥充壕婵炲棙鍔栫瑧婵炴垶鎸撮崑鎾斥槈閹垮啩绨婚柛顭戜簽閹即鈥﹂幒鏃傤槷婵炶揪绲藉Λ娆徫ｉ崨濠佺箚闁稿本绋撴禍顖氣槈閹捐顏犻柍瑙勭墵閹啴宕熼渚囨Н闂佺锕ラ悷杈╂濠靛鐭楅柛顐ゅ枑绗戞繛鎴炴尨閸嬫挸鈽夐幙鍐х敖闁宠鐗犻幆鍐礋椤撶姵灏濋梺鍝勵儏鐎氼亞绱?		闂佸湱顣介崑鎾趁归悩顔煎姎闁搞値鍙冮幃铏紣娴ｈВ鎷℃繛鎴炴惄娴滅偟鍒掗妸鈺佸嚑闁告洦浜炵粔濂告⒒閸ワ絽浜鹃柣鐔告磻缁€渚€鐛幘鏈电剨婵犻潧锕﹀Σ鎼佹偡濞嗘瑧绋婚柣?		SearchSubResultEmpty
		SearchSubResult
		濠殿噯绲鹃弻銊┿€呰濞艰鈻庢惔銊ユ疂闂佽鍨伴幊搴ㄥ窗瀹€鍕櫖?		jsonString := "{\"sub\":{\"action\":\"search\",\"subs\":{},\"result\":\"succeed\",\"keyword\":\"闁哄鏅炴慨銈咁焽閸愨晛绶為煫鍥ㄦ尨閺?S04E07\"},\"status\":0}"
	*/
	err = json.Unmarshal([]byte(resp.String()), &searchSubResult)
	if err != nil {
		// 闂佸憡鍔曠粔鐢割敆濠靛牅鐒婃繝闈涳功濡叉悂鎮峰▎娆戠ɑ闁诲海鏅划姘跺传閸曨偆浠氶柣?
		errKnow = err
		var searchSubResultEmpty SearchSubResultEmpty
		err = json.Unmarshal([]byte(resp.String()), &searchSubResultEmpty)
		if err != nil {
			// 婵犵鈧啿鈧綊鎮樻径瀣氦婵☆垳绮瑧闁荤喐鐟辩徊楣冩倵娴犲鐓ユ繛鍡樺俯閸ゆ牠鏌ㄥ☉妯肩劯闁稿鎳忕粙濠囧醇濠靛洨娈ら柣鐔告磻閻掞附淇婇幖浣瑰仢闁哄瀵ч煬顒勬煟閵娿儱顏柡浣革功閹风娀顢涘顓烆伅婵炴垶鎸搁敃顏勵焽娴煎瓨鍎嶉柛鏇ㄥ灡閺呪晠鎮归崶璺虹仧闁告閰ｅ畷鎶藉Ω閵娧咁唹闂佹悶鍎抽崑娑㈠吹椤撱垹鍌?			s.log.Errorln(s.GetSupplierName(), "NewHttpClient:", keyword, errKnow.Error())
			s.log.Errorln(s.GetSupplierName(), "json.Unmarshal", err)
			notify_center.Notify.Add(s.GetSupplierName()+" NewHttpClient", fmt.Sprintf("keyword: %s, resp: %s, error: %s", keyword, resp.String(), errKnow.Error()))
			return nil, errKnow
		}
		// 闁荤姍鍐惧剰闁逞屽墲婢瑰牏鎹㈤崘顔煎偍?
		searchSubResult.Sub.Action = searchSubResultEmpty.Sub.Action
		searchSubResult.Sub.Result = searchSubResultEmpty.Sub.Result
		searchSubResult.Sub.Keyword = searchSubResultEmpty.Sub.Keyword
		searchSubResult.Status = searchSubResultEmpty.Status

		return &searchSubResult, nil
	}

	return &searchSubResult, nil
}

func sortAssrtSearchSubs(subs []SearchSubItem, videoFPath string, isMovie bool) {
	if len(subs) < 2 {
		return
	}

	matcher := ranking.NewTargetMatcher(videoFPath, isMovie)
	sort.SliceStable(subs, func(i, j int) bool {
		left := scoreAssrtSearchSub(subs[i], matcher)
		right := scoreAssrtSearchSub(subs[j], matcher)
		if left != right {
			return left > right
		}
		if subs[i].VoteScore != subs[j].VoteScore {
			return subs[i].VoteScore > subs[j].VoteScore
		}
		if subs[i].Revision != subs[j].Revision {
			return subs[i].Revision > subs[j].Revision
		}
		return subs[i].Id < subs[j].Id
	})
}

func scoreAssrtSearchSub(sub SearchSubItem, matcher ranking.TargetMatcher) int {
	return ranking.ScoreCandidate(matcher, assrtCandidateMetadata(sub), ranking.CandidateScoreSpec{
		ReleaseMatchWeights: ranking.StandardReleaseMatchWeights,
	})
}

func assrtCandidateMetadata(sub SearchSubItem) ranking.CandidateMetadata {
	return ranking.CandidateMetadata{
		ReleaseNames:   []string{sub.Videoname, sub.NativeName},
		Subtype:        sub.Subtype,
		AuthorityScore: int(sub.VoteScore)*10 + int(sub.Revision)*2,
	}
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

			// 闁哄鐗婇幐鎼佸吹椤撶姵瀚柛鎰靛幘濡叉悂鏌￠崒姘煑婵?
			cacheCenterFolder, err := pkg.GetRootCacheCenterFolder()
			if err != nil {
				s.log.Errorln(s.GetSupplierName(), "GetRootCacheCenterFolder", err)
			}
			desJsonInfo := filepath.Join(cacheCenterFolder, strconv.Itoa(subID)+"--assrt_search_error_getSubDetail.json")
			// 闂佸憡鍔栭悷銉╂偤瑜忕划顓㈡晜閼愁垼娲梺鍛婂笧婢ф寮搁崘鈺冾浄闁告挷绶″?
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
		Action  string          `json:"action"`
		Subs    []SearchSubItem `json:"subs,omitempty"`
		Result  string          `json:"result,omitempty"`
		Keyword string          `json:"keyword,omitempty"`
	} `json:"sub,omitempty"`
	Status int `json:"status,omitempty"`
}

type SearchSubItem struct {
	Lang struct {
		Desc     string        `json:"desc,omitempty"`
		Langlist assrtLangList `json:"langlist,omitempty"`
	} `json:"lang,omitempty"`
	Id          assrtFlexibleInt `json:"id,omitempty"`
	VoteScore   assrtFlexibleInt `json:"vote_score,omitempty"`
	Videoname   string           `json:"videoname,omitempty"`
	ReleaseSite string           `json:"release_site,omitempty"`
	Revision    assrtFlexibleInt `json:"revision,omitempty"`
	Subtype     string           `json:"subtype,omitempty"`
	NativeName  string           `json:"native_name,omitempty"`
	UploadTime  string           `json:"upload_time,omitempty"`
}

type OneSubDetail struct {
	Sub struct {
		Action string `json:"action"`
		Subs   []struct {
			DownCount assrtFlexibleInt `json:"down_count,omitempty"`
			ViewCount assrtFlexibleInt `json:"view_count,omitempty"`
			Lang      struct {
				Desc     string        `json:"desc,omitempty"`
				Langlist assrtLangList `json:"langlist,omitempty"`
			} `json:"lang,omitempty"`
			Size       assrtFlexibleInt `json:"size,omitempty"`
			Title      string           `json:"title,omitempty"`
			Videoname  string           `json:"videoname,omitempty"`
			Revision   assrtFlexibleInt `json:"revision,omitempty"`
			NativeName string           `json:"native_name,omitempty"`
			UploadTime string           `json:"upload_time,omitempty"`
			Producer   struct {
				Producer string `json:"producer,omitempty"`
				Verifier string `json:"verifier,omitempty"`
				Uploader string `json:"uploader,omitempty"`
				Source   string `json:"source,omitempty"`
			} `json:"producer,omitempty"`
			Subtype     string           `json:"subtype,omitempty"`
			VoteScore   assrtFlexibleInt `json:"vote_score,omitempty"`
			ReleaseSite string           `json:"release_site,omitempty"`
			//Filelist    []struct {
			//	S   string `json:"s,omitempty"`
			//	F   string `json:"f,omitempty"`
			//	Url string `json:"url,omitempty"`
			//} `json:"filelist,omitempty"`
			Id       assrtFlexibleInt `json:"id,omitempty"`
			Filename string           `json:"filename,omitempty"`
			Url      string           `json:"url,omitempty"`
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

type assrtLangList struct {
	Langcht bool `json:"langcht,omitempty"`
	Langdou bool `json:"langdou,omitempty"`
	Langeng bool `json:"langeng,omitempty"`
	Langchs bool `json:"langchs,omitempty"`
}

type assrtFlexibleInt int

func (l *assrtLangList) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || bytes.Equal(trimmed, []byte("[]")) {
		*l = assrtLangList{}
		return nil
	}

	type alias assrtLangList
	var parsed alias
	if err := json.Unmarshal(trimmed, &parsed); err != nil {
		return err
	}

	*l = assrtLangList(parsed)
	return nil
}

func (v *assrtFlexibleInt) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || bytes.Equal(trimmed, []byte(`""`)) {
		*v = 0
		return nil
	}

	if trimmed[0] == '"' {
		var text string
		if err := json.Unmarshal(trimmed, &text); err != nil {
			return err
		}
		if text == "" {
			*v = 0
			return nil
		}
		value, err := strconv.Atoi(text)
		if err != nil {
			return err
		}
		*v = assrtFlexibleInt(value)
		return nil
	}

	var number json.Number
	decoder := json.NewDecoder(bytes.NewReader(trimmed))
	decoder.UseNumber()
	if err := decoder.Decode(&number); err != nil {
		return err
	}

	value, err := number.Int64()
	if err != nil {
		return err
	}
	*v = assrtFlexibleInt(value)
	return nil
}
