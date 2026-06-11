package assrt

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/decode"

	common2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/ranking"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/mix_media_info"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/notify_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_helper"
	"github.com/sirupsen/logrus"
)

type Supplier struct {
	log               *logrus.Logger
	fileDownloader    *file_downloader.FileDownloader
	isAlive           bool
	theSearchInterval time.Duration
	badDownloadURLs   map[string]struct{}
}

var assrtSearchKeywordOrder = []string{"cn", "en", "org", "file"}
var assrtEpisodeTokenPattern = regexp.MustCompile(`(?i)^s\d{1,2}e\d{1,3}$`)

type assrtDownloadCandidate struct {
	url     string
	subName string
}

func NewSupplier(fileDownloader *file_downloader.FileDownloader) *Supplier {

	sup := Supplier{}
	sup.log = fileDownloader.Log
	sup.fileDownloader = fileDownloader
	sup.isAlive = true // 濠殿喗甯楃粙鎺椻€﹂崼銉晣濠电姵鑹鹃崣濠冦亜閺嶃劏澹橀悹浣瑰絻闇夐柣姗嗗枛閸旀瑥鈹戦悙鍙夊枠闁诡喕绮欐俊姝岊槼闁伙綁浜堕弻銊モ槈濡灝顏銈傛暘閸パ冨殤?check 闂備礁鎲￠懝楣冨礄閻ｅ本顫曟繝闈涱儏缁€鍐偓骞垮劚閻楀繐鈻撻崼鏇熺厸濞达絽澹婇崵鐔封攽椤旇姤鍊愭鐐╁亾?
	if settings.Get().AdvancedSettings.Topic != common2.DownloadSubsPerSite {
		settings.Get().AdvancedSettings.Topic = common2.DownloadSubsPerSite
	}

	sup.theSearchInterval = 500 * time.Millisecond
	sup.badDownloadURLs = make(map[string]struct{})

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
	videoFileName := filepath.Base(videoFPath)
	var lastDownloadErr error
	for _, keyWordType := range assrtSearchKeywordOrder {
		found, searchSubResult, searchErr := s.getSubInfoEx(mediaInfo, videoFPath, isMovie, keyWordType)
		if searchErr != nil {
			s.log.Errorln(s.GetSupplierName(), videoFPath, "getSubInfoEx", keyWordType, searchErr)
			return nil, searchErr
		}
		if found == false || searchSubResult == nil || searchSubResult.Sub.Subs == nil || len(searchSubResult.Sub.Subs) == 0 {
			continue
		}
		sortAssrtSearchSubs(searchSubResult.Sub.Subs, videoFPath, isMovie)

		keywordSubInfoList := make([]supplier.SubInfo, 0)
		for index, subInfo := range searchSubResult.Sub.Subs {
			oneSubDetail, err := s.getSubDetail(int(subInfo.Id))
			if err != nil {
				s.log.Errorln("getSubDetail", err)
				continue
			}

			if len(oneSubDetail.Sub.Subs) < 1 {
				continue
			}
			detailSubInfo, downloadedFromDetail, detailErr := firstUsableAssrtDownload(
				s.log,
				videoFPath,
				isMovie,
				s.filterBadDownloadCandidates(buildAssrtDownloadCandidates(videoFileName, subInfo, oneSubDetail.Sub.Subs)),
				func(candidate assrtDownloadCandidate, err error) {
					if shouldRememberBadAssrtDownload(err) {
						s.rememberBadDownloadURL(candidate.url)
					}
				},
				func(candidateIndex int, candidate assrtDownloadCandidate) (*supplier.SubInfo, error) {
					return s.fileDownloader.Get(
						s.GetSupplierName(),
						int64(index),
						candidate.subName,
						candidate.url,
						0,
						0,
						fmt.Sprintf("%s-%s-%d-%d", s.GetSupplierName(), subInfo.NativeName, subInfo.Id, candidateIndex),
					)
				},
			)
			if detailErr != nil {
				lastDownloadErr = detailErr
			}
			if downloadedFromDetail == false {
				continue
			}
			keywordSubInfoList = append(keywordSubInfoList, *detailSubInfo)
			if len(keywordSubInfoList) >= settings.Get().AdvancedSettings.Topic {
				return keywordSubInfoList, nil
			}
		}

		if len(keywordSubInfoList) > 0 {
			return keywordSubInfoList, nil
		}
	}

	if lastDownloadErr != nil {
		return nil, lastDownloadErr
	}
	return outSubInfoList, nil
}
func (s *Supplier) getSubInfoWithFallback(mediaInfo *models.MediaInfo, videoFPath string, isMovie bool) (*SearchSubResult, error) {
	videoFileName := filepath.Base(videoFPath)
	merged := &SearchSubResult{}
	seenIDs := make(map[int]struct{})
	for _, keyWordType := range assrtSearchKeywordOrder {
		keyWord, err := mix_media_info.KeyWordSelect(mediaInfo, videoFPath, isMovie, keyWordType)
		if err != nil {
			s.log.Infoln(s.GetSupplierName(), videoFileName, "Skip Search KeyWordType", keyWordType, err)
			continue
		}

		for _, candidateKeyword := range buildAssrtSearchKeywords(keyWord) {
			s.log.Infoln(s.GetSupplierName(), videoFileName, "Try Search KeyWordType", keyWordType, "KeyWord:", candidateKeyword)
			searchSubResult, err := s.getSubByKeyWord(candidateKeyword)
			if err != nil {
				s.log.Errorln(s.GetSupplierName(), videoFileName, "Search KeyWordType", keyWordType, err)
				return nil, err
			}
			if searchSubResult.Sub.Subs == nil || len(searchSubResult.Sub.Subs) == 0 {
				s.log.Infoln(s.GetSupplierName(), videoFileName, "No subtitle found", "KeyWordType:", keyWordType, "KeyWord:", candidateKeyword)
				continue
			}

			merged.Sub.Action = searchSubResult.Sub.Action
			merged.Sub.Result = searchSubResult.Sub.Result
			merged.Sub.Keyword = searchSubResult.Sub.Keyword
			merged.Status = searchSubResult.Status
			for _, sub := range searchSubResult.Sub.Subs {
				if _, found := seenIDs[int(sub.Id)]; found {
					continue
				}
				seenIDs[int(sub.Id)] = struct{}{}
				merged.Sub.Subs = append(merged.Sub.Subs, sub)
			}
		}
	}

	if len(merged.Sub.Subs) == 0 {
		return nil, nil
	}
	return merged, nil
}

func (s *Supplier) getSubInfoEx(mediaInfo *models.MediaInfo, videoFPath string, isMovie bool, keyWordType string) (bool, *SearchSubResult, error) {

	var searchSubResult *SearchSubResult
	var err error
	keyWord, err := mix_media_info.KeyWordSelect(mediaInfo, videoFPath, isMovie, keyWordType)
	if err != nil {
		s.log.Errorln(s.GetSupplierName(), videoFPath, "keyWordSelect", err)
		return false, searchSubResult, err
	}
	videoFileName := filepath.Base(videoFPath)
	for _, candidateKeyword := range buildAssrtSearchKeywords(keyWord) {
		searchSubResult, err = s.getSubByKeyWord(candidateKeyword)
		if err != nil {
			s.log.Errorln("getSubByKeyWord", err)
			return false, searchSubResult, err
		}

		if searchSubResult.Sub.Subs == nil || len(searchSubResult.Sub.Subs) == 0 {
			s.log.Infoln(s.GetSupplierName(), videoFileName, "No subtitle found", "KeyWord:", candidateKeyword)
			continue
		}

		return true, searchSubResult, nil
	}
	return false, searchSubResult, nil
}

func (s *Supplier) downloadSub4Series(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	var allSupplierSubInfo = make([]supplier.SubInfo, 0)
	var lastErr error

	index := 0
	// 闂佸搫顦弲婊堟偡閳哄懎闂柣鎴ｆ缁狀垶鏌涢…鎴濇灈闁绘挸鍊块弻?seriesInfo 闂備焦瀵х粙鎴︽儗閸岀偛闂柣鎴ｅГ椤ュ牓鏌曡箛鏇炐㈤柣锕€鐖奸弻娑橆潩閻撳簼绨电紓渚囩厜缁绘繈寮鍛殕闁告洦鍏涚划顖炴煟閻斿憡纾绘俊鐐村笧缁參宕ㄩ钘夘潯闂佽姤锚椤﹂亶骞楅悢鍓叉闁绘劕鐡ㄧ粈鍫熺箾?Eps 濠电儑绲藉ú鐘诲礈濠靛洤顕?
	for _, episodeInfo := range seriesInfo.NeedDlEpsKeyList {

		index++
		one, err := s.getSubListFromFile(episodeInfo.FileFullPath, false)
		if err != nil {
			s.log.Errorln(s.GetSupplierName(), "getSubListFromFile", episodeInfo.FileFullPath, err)
			lastErr = err
			continue
		}
		if one == nil {
			// 婵犵數鍋涙径鍥礈濠靛棴鑰垮〒姘ｅ亾妤犵偞鐗曢…銊╁川椤撶偘鎲鹃梻浣告啞鐢顭垮鈧幃楦款樄闁?
			s.log.Infoln(s.GetSupplierName(), "Not Find Sub can be download",
				episodeInfo.Title, episodeInfo.Season, episodeInfo.Episode)
			continue
		}
		// 闂傚倸鍊稿ú鐘诲磻閹剧粯鍋￠柡鍥ㄧ缁佹壆绱掓担瑙勫唉鐎规洘鍔欓幊鏍煛閸愵厼鐓戦梺璇插缁嬫帡銆冮崨顖滀笉濡炲娴风壕濂告煙閹屽殶闁?
		for i := range one {
			one[i].Season = episodeInfo.Season
			one[i].Episode = episodeInfo.Episode
		}
		allSupplierSubInfo = append(allSupplierSubInfo, one...)
	}
	// 闂佸搫顦弲婊堝蓟閵娿儍娲冀椤撶偟顦悗骞垮劚缁绘帞绮堟径鎰拻闁搞儻绲芥禍楣冩煟閻斿憡纾婚柣鎺為檮娣囧﹦绮欓崯鍐╁笩缁犳稒鎯旈敐搴濈礃濠?Eps 闂?Season Episode 濠电儑绲藉ú鐘诲礈濠靛洤顕遍柛娑卞枤缁犳瑦銇勯幇鈺佺仼閻㈩垳濞€閺屾盯骞掗弬娆炬毉闂佷紮鑵归崜婵堢矙?SubInfo 濠?
	if len(allSupplierSubInfo) == 0 && lastErr != nil {
		return nil, lastErr
	}
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
		闂佸搫顦弲婊堟偡閳哄懎闂柣鎴ｆ鐎氬顭跨捄渚█闁搞倕顑嗛幈銊︾節閸愮偓顓虹紓?Sub 闂備礁鎼悧鍡浰囬鐐茬劦妞ゆ帒鍟禍瑙勭箾閸喎鐏寸€殿噮鍠氶幑鍕偘閳ュ厖澹曞┑鐐叉閸旀牜鐟у┑鐐村灦閹告挳宕戦幘鏂ユ闁瑰灝鍟╃花濠氭煕椤垳绨介柟顔诲嵆閳ワ箓骞掗弮鍌ゆХ濠电偠鎻徊钘壩涘▎寰綁宕ㄦ繝浣虹畾闂佺鏈粙鎾寸椤栨埃妲堥柟鎹愵潐椤忕娀鏌嶇憴鍕⒌闁诡垰鍟村畷鐔碱敆娓氬洦袧闂備胶顭堥敃銉╂偡鏉堚晜顫曟繝闈涱儏閻鏌涢銈呮瀾缁楁垶绻涢幋鐐村皑闁稿鎸搁埥澶愬箼閸愌呮晼闂佸疇顫夐悧鐘诲箚閸愵喖绀嬫い鎾跺У鐏忔繈姊洪崫鍕靛剰閻庢凹浜炵槐?		闂備礁婀遍。浠嬪磻閹捐秮褰掓偐椤旂厧濮庨梺鎼炲€ら崣鍐箖閾忣偓绱ｅù锝埿掗幏鈩冪箾閹寸偞鎯勫ù婊呭仧閸掓帡濡搁埡浣稿殤闂佸憡娲︽禍鐐电矓婵傚憡鈷掗柛銉到娴滈箖鏌ｉ悢鍛婄；缂佲偓娓氣偓閻涱噣骞橀張鐢靛墾濠电娀娼ч敃锕€危閹间焦鍋℃繛鍡樼懅缁嬪鏌?		SearchSubResultEmpty
		SearchSubResult
		婵犳鍣徊楣冨蓟閵娾斂鈧懓顦虫繛鑹邦嚙閳诲孩鎯旈妸銉︾杺闂備浇顕栭崹浼村箠鎼淬劌绐楃€光偓閸曨剚娅?		jsonString := "{\"sub\":{\"action\":\"search\",\"subs\":{},\"result\":\"succeed\",\"keyword\":\"闂佸搫顦弲鐐存叏閵堝拋鐒介柛鎰ㄦ櫅缁剁偤鐓崶銊﹀皑闁?S04E07\"},\"status\":0}"
	*/
	err = json.Unmarshal([]byte(resp.String()), &searchSubResult)
	if err != nil {
		// 闂備礁鎲￠崝鏇犵矓閻㈠壊鏁嗘繝闈涚墔閻掑﹥绻濋棃娑冲姛婵″弶鎮傞幃宄扳枎濞嗘垹蓱闂佽娴烽弲顐ゅ垝濮樿泛浼犻柛鏇ㄥ亞娴犳岸鏌?
		errKnow = err
		var searchSubResultEmpty SearchSubResultEmpty
		err = json.Unmarshal([]byte(resp.String()), &searchSubResultEmpty)
		if err != nil {
			// 濠电姷顣介埀顒€鍟块埀顒€缍婇幃妯诲緞鐎ｎ偂姘﹀┑鈽嗗灣缁垳鐟ч梺鑽ゅ枑閻熻京寰婃ィ鍐╁€靛ù鐘差儐閻撱儲绻涢崱妯轰刊闁搞倖鐗犻弻銊モ槈濡偐鍔梺绋款儏閹冲繒绮欐繝鍥ч唶婵犻潧娲ㄥ▓銈夋煟閻斿憡纾婚柣鎺為檮娣囧﹪骞栨担鐟颁虎闂佸搫顦扮€笛囩叕椤掑嫭鐓熼柕濞垮劚椤忣參鏌℃担闈╁姛闁归濞€椤㈡稑顫濋鐑嗕紖濠电偞鍨堕幐鎼佹晝椤忓嫷鐒藉ù鐓庣摠閸庡秹鏌涢弴銊ョ仭闁哄應鏅犻幃褰掑炊鐠鸿櫣浠ч梺鍛婎殔闁帮絽鐣烽幎钘壩╅柕濞у拋鍞归梻浣规偠閸庢娊宕戝☉銏犲惞妞ゆ挶鍨归崒?			s.log.Errorln(s.GetSupplierName(), "NewHttpClient:", keyword, errKnow.Error())
			s.log.Errorln(s.GetSupplierName(), "json.Unmarshal", err)
			notify_center.Notify.Add(s.GetSupplierName()+" NewHttpClient", fmt.Sprintf("keyword: %s, resp: %s, error: %s", keyword, resp.String(), errKnow.Error()))
			return nil, errKnow
		}
		// 闂佽崵濮嶉崘鎯у壈闂侀€炲苯澧插鐟扮墢閹广垽宕橀鐓庡亶?
		searchSubResult.Sub.Action = searchSubResultEmpty.Sub.Action
		searchSubResult.Sub.Result = searchSubResultEmpty.Sub.Result
		searchSubResult.Sub.Keyword = searchSubResultEmpty.Sub.Keyword
		searchSubResult.Status = searchSubResultEmpty.Status

		return &searchSubResult, nil
	}

	return &searchSubResult, nil
}

func buildAssrtSearchKeywords(keyword string) []string {
	out := make([]string, 0, 2)
	seen := make(map[string]struct{}, 2)
	appendKeyword := func(item string) {
		item = strings.TrimSpace(item)
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

	for _, item := range mix_media_info.ExpandSearchKeywords(
		keyword,
		stripTrailingYearFromAssrtKeyword(keyword),
	) {
		appendKeyword(item)
	}
	return out
}

func buildAssrtDownloadCandidates(videoFileName string, searchSub SearchSubItem, detailSubs []AssrtDetailSubItem) []assrtDownloadCandidate {
	out := make([]assrtDownloadCandidate, 0, len(detailSubs))
	seen := make(map[string]struct{}, len(detailSubs))

	appendCandidate := func(urlValue string, names ...string) {
		urlValue = strings.TrimSpace(urlValue)
		if urlValue == "" {
			return
		}
		if _, ok := seen[urlValue]; ok {
			return
		}
		seen[urlValue] = struct{}{}

		subName := strings.TrimSpace(videoFileName)
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name != "" {
				subName = name
				break
			}
		}
		out = append(out, assrtDownloadCandidate{
			url:     urlValue,
			subName: subName,
		})
	}

	for _, detail := range detailSubs {
		appendCandidate(detail.Url, detail.NativeName, detail.Videoname, detail.Filename, searchSub.NativeName, searchSub.Videoname)
	}

	return out
}

func firstUsableAssrtDownload(log *logrus.Logger, videoFPath string, isMovie bool, candidates []assrtDownloadCandidate, rememberBad func(candidate assrtDownloadCandidate, err error), downloader func(candidateIndex int, candidate assrtDownloadCandidate) (*supplier.SubInfo, error)) (*supplier.SubInfo, bool, error) {
	var lastErr error
	for candidateIndex, candidate := range candidates {
		detailSubInfo, detailErr := downloader(candidateIndex, candidate)
		if detailErr != nil {
			lastErr = detailErr
			if log != nil {
				log.Errorln("firstUsableAssrtDownload", detailErr)
			}
			if rememberBad != nil {
				rememberBad(candidate, detailErr)
			}
			continue
		}
		if assrtDownloadedSubtitleUsable(log, videoFPath, isMovie, *detailSubInfo) == false {
			lastErr = fmt.Errorf("assrt unusable downloaded candidate for %s", candidate.url)
			if log != nil {
				log.Warningln("firstUsableAssrtDownload", filepath.Base(videoFPath), "discard unusable candidate", candidate.url, candidate.subName)
			}
			if rememberBad != nil {
				rememberBad(candidate, lastErr)
			}
			continue
		}
		return detailSubInfo, true, nil
	}

	return nil, false, lastErr
}

func assrtDownloadedSubtitleUsable(log *logrus.Logger, videoFPath string, isMovie bool, subInfo supplier.SubInfo) bool {
	if isMovie == false && (subInfo.Season == 0 || subInfo.Episode == 0) {
		if _, season, episode, err := decode.GetSeasonAndEpisodeFromSubFileName(filepath.Base(videoFPath)); err == nil {
			subInfo.Season = season
			subInfo.Episode = episode
		}
	}

	tmpFolderName := sanitizeAssrtProbeFolderName(videoFPath, subInfo)
	_ = pkg.ClearTmpFolderByName(tmpFolderName)
	defer func() {
		_ = pkg.ClearTmpFolderByName(tmpFolderName)
	}()

	organized, err := sub_helper.OrganizeDlSubFiles(log, tmpFolderName, []supplier.SubInfo{subInfo}, isMovie)
	if err != nil {
		if log != nil {
			log.Warningln("assrtDownloadedSubtitleUsable", filepath.Base(videoFPath), err)
		}
		return false
	}

	if isMovie {
		for _, files := range organized {
			if len(files) > 0 {
				return true
			}
		}
		return false
	}

	if subInfo.Season == 0 || subInfo.Episode == 0 {
		return false
	}

	return len(organized[pkg.GetEpisodeKeyName(subInfo.Season, subInfo.Episode)]) > 0
}

func sanitizeAssrtProbeFolderName(videoFPath string, subInfo supplier.SubInfo) string {
	replacer := strings.NewReplacer(
		"\\", "_",
		"/", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	return replacer.Replace(fmt.Sprintf("assrt_probe_%s_%s_%d_%d", filepath.Base(videoFPath), subInfo.FromWhere, subInfo.TopN, len(subInfo.Data)))
}

func shouldRememberBadAssrtDownload(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())
	return strings.Contains(lower, "invalid archive payload") ||
		strings.Contains(lower, "unexpected content-type") ||
		strings.Contains(lower, "download payload is not a subtitle file") ||
		strings.Contains(lower, "empty download body") ||
		strings.Contains(lower, "assrt unusable downloaded candidate")
}

func (s *Supplier) filterBadDownloadCandidates(candidates []assrtDownloadCandidate) []assrtDownloadCandidate {
	if len(candidates) == 0 {
		return nil
	}
	out := make([]assrtDownloadCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if s.isBadDownloadURL(candidate.url) {
			continue
		}
		out = append(out, candidate)
	}
	return out
}

func (s *Supplier) rememberBadDownloadURL(urlValue string) {
	urlValue = strings.TrimSpace(urlValue)
	if urlValue == "" {
		return
	}
	if s.badDownloadURLs == nil {
		s.badDownloadURLs = make(map[string]struct{})
	}
	s.badDownloadURLs[urlValue] = struct{}{}
}

func (s *Supplier) isBadDownloadURL(urlValue string) bool {
	urlValue = strings.TrimSpace(urlValue)
	if urlValue == "" || len(s.badDownloadURLs) == 0 {
		return false
	}
	_, found := s.badDownloadURLs[urlValue]
	return found
}

func stripTrailingYearFromAssrtKeyword(keyword string) string {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return ""
	}

	replacer := strings.NewReplacer(
		"(", " ", ")", " ",
		"[", " ", "]", " ",
	)
	parts := strings.Fields(replacer.Replace(keyword))
	if len(parts) == 0 {
		return ""
	}

	removeIndex := -1
	lastIndex := len(parts) - 1
	if len(parts[lastIndex]) == 4 {
		if _, err := strconv.Atoi(parts[lastIndex]); err == nil {
			removeIndex = lastIndex
		}
	} else if assrtEpisodeTokenPattern.MatchString(parts[lastIndex]) && len(parts) >= 2 && len(parts[lastIndex-1]) == 4 {
		if _, err := strconv.Atoi(parts[lastIndex-1]); err == nil {
			removeIndex = lastIndex - 1
		}
	}
	if removeIndex < 0 {
		return keyword
	}

	outParts := append([]string{}, parts[:removeIndex]...)
	outParts = append(outParts, parts[removeIndex+1:]...)
	return strings.TrimSpace(strings.Join(outParts, " "))
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
		IsMovie:       false,
		TargetSeason:  parseAssrtTargetSeason(matcher),
		TargetEpisode: parseAssrtTargetEpisode(matcher),
		EpisodeMatchWeights: &ranking.EpisodeMatchWeights{
			ExactMatch:   120,
			SeasonPack:   15,
			WrongEpisode: -120,
		},
		ReleaseMatchWeights: ranking.StandardReleaseMatchWeights,
	})
}

func assrtCandidateMetadata(sub SearchSubItem) ranking.CandidateMetadata {
	season, episode := parseAssrtSeasonEpisode(sub)
	return ranking.CandidateMetadata{
		ReleaseNames:   []string{sub.Videoname, sub.NativeName},
		Season:         season,
		Episode:        episode,
		Subtype:        sub.Subtype,
		AuthorityScore: int(sub.VoteScore)*10 + int(sub.Revision)*2,
	}
}

func parseAssrtSeasonEpisode(sub SearchSubItem) (int, int) {
	for _, name := range []string{sub.Videoname, sub.NativeName} {
		if _, season, episode, err := decode.GetSeasonAndEpisodeFromSubFileName(name); err == nil {
			if season != 0 || episode != 0 {
				return season, episode
			}
		}
	}
	return 0, 0
}

func parseAssrtTargetSeason(matcher ranking.TargetMatcher) int {
	_, season, _, err := decode.GetSeasonAndEpisodeFromSubFileName(matcher.TargetName())
	if err != nil {
		return 0
	}
	return season
}

func parseAssrtTargetEpisode(matcher ranking.TargetMatcher) int {
	_, _, episode, err := decode.GetSeasonAndEpisodeFromSubFileName(matcher.TargetName())
	if err != nil {
		return 0
	}
	return episode
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

			// 闂佸搫顦悧濠囧箰閹间礁鍚规い鎾跺У鐎氼剟鏌涢幇闈涘箻婵″弶鎮傞弻锟犲磼濮橆厾鐓戝┑?
			cacheCenterFolder, err := pkg.GetRootCacheCenterFolder()
			if err != nil {
				s.log.Errorln(s.GetSupplierName(), "GetRootCacheCenterFolder", err)
			}
			desJsonInfo := filepath.Join(cacheCenterFolder, strconv.Itoa(subID)+"--assrt_search_error_getSubDetail.json")
			// 闂備礁鎲￠崝鏍偡閵夆晜鍋ょ憸蹇曞垝椤撱垺鏅滈柤鎰佸灱濞差剟姊洪崨濠傜濠⒀勵殜瀵悂宕橀埡鍐炬祫闂佸憡鎸风欢鈥愁焽?
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
		Subs    assrtSearchSubs `json:"subs,omitempty"`
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
		Action string          `json:"action"`
		Subs   assrtDetailSubs `json:"subs,omitempty"`
		Result string          `json:"result,omitempty"`
	} `json:"sub,omitempty"`
	Status int `json:"status,omitempty"`
}

type AssrtDetailSubItem struct {
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
	Id          assrtFlexibleInt `json:"id,omitempty"`
	Filename    string           `json:"filename,omitempty"`
	Url         string           `json:"url,omitempty"`
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
type assrtSearchSubs []SearchSubItem
type assrtDetailSubs []AssrtDetailSubItem

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

func (s *assrtSearchSubs) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || bytes.Equal(trimmed, []byte("{}")) {
		*s = nil
		return nil
	}
	if bytes.Equal(trimmed, []byte("[]")) {
		*s = assrtSearchSubs{}
		return nil
	}
	if trimmed[0] == '[' {
		var items []SearchSubItem
		if err := json.Unmarshal(trimmed, &items); err != nil {
			return err
		}
		*s = assrtSearchSubs(items)
		return nil
	}
	if trimmed[0] == '{' {
		var item SearchSubItem
		if err := json.Unmarshal(trimmed, &item); err != nil {
			return err
		}
		*s = assrtSearchSubs{item}
		return nil
	}
	return fmt.Errorf("unexpected assrt search subs payload: %s", string(trimmed))
}

func (s *assrtDetailSubs) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) || bytes.Equal(trimmed, []byte("{}")) {
		*s = nil
		return nil
	}
	if bytes.Equal(trimmed, []byte("[]")) {
		*s = assrtDetailSubs{}
		return nil
	}
	if trimmed[0] == '[' {
		var items []AssrtDetailSubItem
		if err := json.Unmarshal(trimmed, &items); err != nil {
			return err
		}
		*s = assrtDetailSubs(items)
		return nil
	}
	if trimmed[0] == '{' {
		var item AssrtDetailSubItem
		if err := json.Unmarshal(trimmed, &item); err != nil {
			return err
		}
		*s = assrtDetailSubs{item}
		return nil
	}
	return fmt.Errorf("unexpected assrt detail subs payload: %s", string(trimmed))
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
