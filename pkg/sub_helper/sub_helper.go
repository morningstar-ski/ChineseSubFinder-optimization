package sub_helper

import (
	"errors"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/subparser"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/archive_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/decode"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/filter"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/regex_things"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_parser_hub"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/vad"
	"github.com/sirupsen/logrus"
)

// OrganizeDlSubFiles 闇€瑕佷粠姹囨€绘潵鏄綉绔欏瓧骞曚腑锛岃В鍘嬪搴旂殑鍘嬬缉鍖呬腑鐨勫瓧骞曞嚭鏉?
func OrganizeDlSubFiles(log *logrus.Logger, tmpFolderName string, subInfos []supplier.SubInfo, isMovie bool) (map[string][]string, error) {

	// 缂撳瓨鍒楄〃锛屾暣鐞嗗悗鐨勫瓧骞曞垪琛?
	// SxEx - []string 瀛楀箷鐨勮矾寰?
	var siteSubInfoDict = make(map[string][]string)
	tmpFolderFullPath, err := pkg.GetTmpFolderByName(tmpFolderName)
	if err != nil {
		return nil, err
	}

	// 鎶婂悗缂€鍚嶇粰鏀瑰ソ
	ChangeVideoExt2SubExt(subInfos)

	// 绗笁鏂圭殑瑙ｅ帇搴擄紝棣栧厛涓嶆敮鎸?io.Reader 鐨勬搷浣滐紝涔熷氨鏄緱缂撳瓨鍒版湰鍦扮‖鐩樺啀璇诲彇瑙ｅ帇
	// 涓斾娇鐢?walk 浼氭棤娉曡В鍘?rar锛屽緱鎸囧畾鍏蜂綋鐨勫疄渚嬶紝澶夯鐑︿簡锛岀洿鎺ョ敤閫氱敤鐨勬帴鍙ｅ緱浜嗭紝灏辨槸寰楅兘缂撳瓨涓嬫潵鍐嶅垽鏂?
	// 鍩轰簬浠ヤ笂涓ょ偣锛屽啓浜嗕竴鍫嗗暟鍡︾殑閫昏緫路路路
	for i := range subInfos {
		if err := pkg.ValidateDownloadedPayload(subInfos[i].Name, subInfos[i].Data); err != nil {
			log.Errorln("ValidateDownloadedPayload", subInfos[i].FromWhere, subInfos[i].Name, subInfos[i].TopN, err)
			continue
		}
		// 鍏堝瓨涓嬫潵锛屼繚瀛樻槸鏃跺€欓渶瑕佸墠缂€锛屽墠缂€灏辨槸浠庨偅涓綉绔欎笅杞芥潵鐨?
		nowFileSaveFullPath := filepath.Join(tmpFolderFullPath, GetFrontNameAndOrgName(log, &subInfos[i]))
		err = pkg.WriteFile(nowFileSaveFullPath, subInfos[i].Data)
		if err != nil {
			log.Errorln("getFrontNameAndOrgName - WriteFile", nowFileSaveFullPath, "FromWhere Name TopN", subInfos[i].FromWhere, subInfos[i].Name, subInfos[i].TopN, err)
			continue
		}
		nowExt := strings.ToLower(subInfos[i].Ext)
		epsKey := pkg.GetEpisodeKeyName(subInfos[i].Season, subInfos[i].Episode)
		_, ok := siteSubInfoDict[epsKey]
		if ok == false {
			// 涓嶅瓨鍦ㄥ垯瀹炰緥鍖?
			siteSubInfoDict[epsKey] = make([]string, 0)
		}
		if nowExt != ".zip" && nowExt != ".tar" && nowExt != ".rar" && nowExt != ".7z" {
			// 鏄惁鏄彈鏀寔鐨勫瓧骞曠被鍨?
			if sub_parser_hub.IsSubExtWanted(nowExt) == false {
				log.Debugln("OrganizeDlSubFiles -> IsSubExtWanted == false", "Name:", subInfos[i].Name, "FileUrl:", subInfos[i].FileUrl)
				continue
			}
			// 鍔犲叆缂撳瓨鍒楄〃
			siteSubInfoDict[epsKey] = append(siteSubInfoDict[epsKey], nowFileSaveFullPath)
		} else {
			// 閭ｄ箞灏辨槸闇€瑕佽В鍘嬬殑鏂囦欢浜?
			// 瑙ｅ帇锛岀粰涓€涓崟鐙殑鏂囦欢澶?
			unzipTmpFolder := filepath.Join(tmpFolderFullPath, subInfos[i].FromWhere)
			err = os.MkdirAll(unzipTmpFolder, os.ModePerm)
			if err != nil {
				return nil, err
			}
			err = archive_helper.UnArchiveFileEx(nowFileSaveFullPath, unzipTmpFolder)
			// 瑙ｅ帇瀹屾垚鍚庯紝閬嶅巻鍙楁敮鎸佺殑瀛楀箷鍒楄〃锛屽姞鍏ョ紦瀛樺垪琛?
			if err != nil {
				log.Errorln("archiver.UnArchive", subInfos[i].FromWhere, subInfos[i].Name, subInfos[i].TopN, err)
				continue
			}
			// 鎼滅储杩欎釜鐩綍涓嬬殑鎵€鏈夌鍚堝瓧骞曟牸寮忕殑鏂囦欢
			subFileFullPaths, err := SearchMatchedSubFileByDir(log, unzipTmpFolder)
			if err != nil {
				log.Errorln("searchMatchedSubFile", subInfos[i].FromWhere, subInfos[i].Name, subInfos[i].TopN, err)
				continue
			}
			// 杩欓噷闇€瑕佺粰杩欎簺涓嬭浇鍒扮殑鏂囦欢杩涜鏀瑰悕锛屽姞鏄粠閭ｄ釜缃戠珯鏉ョ殑鍓嶇紑锛屽悗缁ソ鏌ユ壘
			for _, fileFullPath := range subFileFullPaths {
				if isMovie == false {
					// 杩炵画鍓х殑鎯呭喌
					// 浠庤В鍘嬬殑鏂囦欢鍚嶇О鎺ㄦ柇 Season 鍜?Episode 淇℃伅
					_, nowSeason, nowEps, err := decode.GetSeasonAndEpisodeFromSubFileName(filepath.Base(fileFullPath))
					if err != nil {
						continue
					}
					newSubName := AddFrontName(subInfos[i], filepath.Base(fileFullPath))
					newSubNameFullPath := filepath.Join(tmpFolderFullPath, newSubName)
					// 鏀瑰悕
					err = os.Rename(fileFullPath, newSubNameFullPath)
					if err != nil {
						log.Errorln("os.Rename", subInfos[i].FromWhere, subInfos[i].Name, subInfos[i].TopN, err)
						continue
					}
					// 鍔犲叆缂撳瓨鍒楄〃
					// 鏍规嵁褰撳墠瀛楀箷鐨勪俊鎭潵鏋勫缓 key
					SEPKey := pkg.GetEpisodeKeyName(nowSeason, nowEps)
					_, ok = siteSubInfoDict[SEPKey]
					if ok == false {
						siteSubInfoDict[SEPKey] = make([]string, 0)
					}
					siteSubInfoDict[SEPKey] = append(siteSubInfoDict[SEPKey], newSubNameFullPath)
				} else {
					// 鐢靛奖鐨勬儏鍐?
					newSubName := AddFrontName(subInfos[i], filepath.Base(fileFullPath))
					newSubNameFullPath := filepath.Join(tmpFolderFullPath, newSubName)
					// 鏀瑰悕
					err = os.Rename(fileFullPath, newSubNameFullPath)
					if err != nil {
						log.Errorln("os.Rename", subInfos[i].FromWhere, subInfos[i].Name, subInfos[i].TopN, err)
						continue
					}
					// 鍔犲叆缂撳瓨鍒楄〃
					siteSubInfoDict[epsKey] = append(siteSubInfoDict[epsKey], newSubNameFullPath)
				}

			}
		}
	}

	return siteSubInfoDict, nil
}

// ChangeVideoExt2SubExt 妫€娴?Name锛屽鏋滄槸瑙嗛鐨勫悗缂€鍚嶅氨鏀逛负瀛楀箷鐨勫悗缂€鍚?
func ChangeVideoExt2SubExt(subInfos []supplier.SubInfo) {
	for x, info := range subInfos {
		tmpSubFileName := info.Name
		// 濡傛灉鍚庣紑鍚嶆槸涓嬭浇瀛楀箷鐩爣鐨勫悗缂€鍚? 鎴栬€?鏄帇缂╁寘鏍煎紡鐨勶紝鍒欒烦杩?
		if strings.Contains(tmpSubFileName, info.Ext) == true || archive_helper.IsWantedArchiveExtName(tmpSubFileName) == true {

		} else {
			subInfos[x].Name = tmpSubFileName + info.Ext
		}
	}
}

// SelectChineseBestBilingualSubtitle 鎵惧埌鍚堥€傜殑鍙岃涓枃瀛楀箷锛岀畝浣?>绻佷綋锛屼互鍙?瀛楀箷绫诲瀷鐨勪紭鍏堢骇閫夋嫨
func SelectChineseBestBilingualSubtitle(subs []subparser.FileInfo, subTypePriority int) *subparser.FileInfo {

	// 鍏堝偦涓€鐐瑰疄鐜颁紭鍏堝弻璇殑锛屼箣鍓嶇殑鍐欐硶鏈?bug
	for _, info := range subs {
		// 鎵惧埌浜嗕腑鏂囧瓧骞?
		if language.HasChineseLang(info.Lang) == true {
			// 瀛楀箷鐨勪紭鍏堢骇 0 - 鍘熸牱, 1 - srt , 2 - ass/ssa
			if subTypePriority == 1 {
				// 1 - srt
				if strings.ToLower(info.Ext) == common.SubExtSRT {
					// 浼樺厛鍙岃
					if language.IsBilingualSubtitle(info.Lang) == true {
						return &info
					}
				}
			} else if subTypePriority == 2 {
				//  2 - ass/ssa
				if strings.ToLower(info.Ext) == common.SubExtASS || strings.ToLower(info.Ext) == common.SubExtSSA {
					// 浼樺厛鍙岃
					if language.IsBilingualSubtitle(info.Lang) == true {
						return &info
					}
				}
			} else {
				// 浼樺厛鍙岃
				if language.IsBilingualSubtitle(info.Lang) == true {
					return &info
				}
			}
		}
	}

	return nil
}

// SelectChineseBestSubtitle 鎵惧埌鍚堥€傜殑涓枃瀛楀箷锛岀畝浣?>绻佷綋锛屼互鍙?瀛楀箷绫诲瀷鐨勪紭鍏堢骇閫夋嫨
func SelectChineseBestSubtitle(subs []subparser.FileInfo, subTypePriority int) *subparser.FileInfo {

	// 鍏堝偦涓€鐐瑰疄鐜颁紭鍏堝弻璇殑锛屼箣鍓嶇殑鍐欐硶鏈?bug
	for _, info := range subs {
		// 鎵惧埌浜嗕腑鏂囧瓧骞?
		if language.HasChineseLang(info.Lang) == true {
			// 瀛楀箷鐨勪紭鍏堢骇 0 - 鍘熸牱, 1 - srt , 2 - ass/ssa
			if subTypePriority == 1 {
				// 1 - srt
				if strings.ToLower(info.Ext) == common.SubExtSRT {
					return &info
				}
			} else if subTypePriority == 2 {
				//  2 - ass/ssa
				if strings.ToLower(info.Ext) == common.SubExtASS || strings.ToLower(info.Ext) == common.SubExtSSA {
					return &info
				}
			} else {
				return &info
			}
		}
	}

	return nil
}

// GetFrontNameAndOrgName 杩斿洖鐨勫悕绉板寘鍚紝閭ｄ釜缃戠珯涓嬭浇鐨勶紝杩欎釜缃戠珯涓帓鍚嶇鍑狅紝鏂囦欢鍚?
func GetFrontNameAndOrgName(log *logrus.Logger, info *supplier.SubInfo) string {

	infoName := ""
	fileName, err := decode.GetVideoInfoFromFileName(info.Name)
	if err != nil {
		log.Warnln("", err)
		// 鏇挎崲鐗规畩瀛楃
		infoName = pkg.ReplaceSpecString(info.Name, "x")
	} else {
		infoName = fileName.Title + "_S" + strconv.Itoa(fileName.Season) + "E" + strconv.Itoa(fileName.Episode) + filepath.Ext(info.Name)
	}
	if len(infoName) < 1 {
		infoName = pkg.RandStringBytesMaskImprSrcSB(10) + filepath.Ext(info.Name)
	}
	info.Name = infoName

	return "[" + info.FromWhere + "]_" + strconv.FormatInt(info.TopN, 10) + "_" + infoName
}

// AddFrontName 娣诲姞鏂囦欢鐨勫墠缂€
func AddFrontName(info supplier.SubInfo, orgName string) string {
	return "[" + info.FromWhere + "]_" + strconv.FormatInt(info.TopN, 10) + "_" + orgName
}

// SearchMatchedSubFileByDir 鎼滅储绗﹀悎鍚庣紑鍚嶇殑瑙嗛鏂囦欢锛屾帓闄?Sub_SxE0 杩欐牱鐨勬枃浠跺す涓殑鏂囦欢
func SearchMatchedSubFileByDir(log *logrus.Logger, dir string) ([]string, error) {
	// 杩欓噷鏈変釜姊楋紝浼氬嚭鐜?__MACOSX 杩欑被鏂囦欢澶癸紝閭ｄ箞閲岄潰浼氭湁涓€鏍风殑鏂囦欢锛岄渶瑕佺敤鏂囦欢澶у皬鎺掗櫎涓€涓嬶紝鑷冲皯澶т簬 1 kb 鍚?
	var fileFullPathList = make([]string, 0)
	pathSep := string(os.PathSeparator)
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, curFile := range files {
		fullPath := dir + pathSep + curFile.Name()
		if pkg.IsDir(fullPath) == true {
			// 闇€瑕佹帓闄?Sub_S1E0銆丼ub_S2E0 杩欐牱鐨勬暣瀛ｇ殑瀛楀箷鏂囦欢澶癸紝杩欓噷浠呬粎鏄紦瀛橈紝涓嶄細琚姞杞界殑
			matched := regex_things.RegOneSeasonSubFolderNameMatch.FindAllStringSubmatch(curFile.Name(), -1)
			if matched != nil && len(matched) > 0 {
				continue
			}
			// 鍐呭眰鐨勯敊璇氨鏃犺浜?
			oneList, _ := SearchMatchedSubFileByDir(log, fullPath)
			if oneList != nil {
				fileFullPathList = append(fileFullPathList, oneList...)
			}
		} else {
			// 杩欓噷灏辨槸鏂囦欢浜?
			if filter.SkipFileInfo(log, curFile, fullPath) == true {
				continue
			}

			if sub_parser_hub.IsSubExtWanted(filepath.Ext(curFile.Name())) == true {
				fileFullPathList = append(fileFullPathList, fullPath)
			}
		}
	}
	return fileFullPathList, nil
}

// SearchMatchedSubFileByOneVideo 鎼滅储杩欎釜瑙嗛褰撳墠鐩綍涓嬪尮閰嶇殑瀛楀箷
func SearchMatchedSubFileByOneVideo(l *logrus.Logger, oneVideoFullPath string) ([]string, error) {
	dir := filepath.Dir(oneVideoFullPath)
	fileName := filepath.Base(oneVideoFullPath)
	fileName = strings.ToLower(fileName)
	fileName = strings.ReplaceAll(fileName, filepath.Ext(fileName), "")
	pathSep := string(os.PathSeparator)
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var matchedSubs = make([]string, 0)

	for _, curFile := range files {
		if curFile.IsDir() {
			continue
		}
		// 杩欓噷灏辨槸鏂囦欢浜?
		oldPath := dir + pathSep + curFile.Name()
		if filter.SkipFileInfo(l, curFile, oldPath) == true {
			continue
		}

		// 鍒ゆ柇鐨勬椂鍊欑敤灏忓啓鐨勶紝鍚庣画閲嶅懡鍚嶇殑鏃跺€欑敤鍘熸湁鐨勫悕绉?
		nowFileName := strings.ToLower(curFile.Name())
		// 鍚庣紑鍚嶅緱瀵?
		if sub_parser_hub.IsSubExtWanted(filepath.Ext(nowFileName)) == false {
			continue
		}
		// 瀛楀箷鏂囦欢鍚嶅簲璇ュ寘鍚?瑙嗛鏂囦欢鍚嶏紙鏃犲悗缂€锛?
		if strings.HasPrefix(nowFileName, fileName) == false {
			continue
		}

		matchedSubs = append(matchedSubs, oldPath)
	}

	return matchedSubs, nil
}

// SearchVideoMatchSubFileAndRemoveExtMark 鎵惧埌鎵句釜瑙嗛鐩綍涓嬬浉鍖归厤鐨勫瓧骞曪紝鍚屾椂鍘婚櫎杩欎簺瀛楀箷涓?.default 鎴栬€?.forced 鐨勬爣璁般€傛敞鎰忚繖涓や釜鏍囪涓嶅簲璇ュ悓鏃跺嚭鐜帮紝鍚﹀垯鏃犳硶姝ｇ‘鍘婚櫎
func SearchVideoMatchSubFileAndRemoveExtMark(l *logrus.Logger, oneVideoFullPath string) error {

	dir := filepath.Dir(oneVideoFullPath)
	fileName := filepath.Base(oneVideoFullPath)
	fileName = strings.ToLower(fileName)
	fileName = strings.ReplaceAll(fileName, filepath.Ext(fileName), "")
	pathSep := string(os.PathSeparator)
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, curFile := range files {
		if curFile.IsDir() {
			continue
		} else {
			// 杩欓噷灏辨槸鏂囦欢浜?
			oldPath := dir + pathSep + curFile.Name()
			if filter.SkipFileInfo(l, curFile, oldPath) == true {
				continue
			}
			// 鍒ゆ柇鐨勬椂鍊欑敤灏忓啓鐨勶紝鍚庣画閲嶅懡鍚嶇殑鏃跺€欑敤鍘熸湁鐨勫悕绉?
			nowFileName := strings.ToLower(curFile.Name())
			// 鍚庣紑鍚嶅緱瀵?
			if sub_parser_hub.IsSubExtWanted(filepath.Ext(nowFileName)) == false {
				continue
			}
			// 瀛楀箷鏂囦欢鍚嶅簲璇ュ寘鍚?瑙嗛鏂囦欢鍚嶏紙鏃犲悗缂€锛?
			if strings.HasPrefix(nowFileName, fileName) == false {
				continue
			}

			if strings.Contains(nowFileName, subparser.Sub_Ext_Mark_Default+".") == true {
				// 寰楀寘鍚?.default. 鎵句釜鍏抽敭璇?
				// 鍘婚櫎 .default.
				newPath := dir + pathSep + strings.ReplaceAll(curFile.Name(), subparser.Sub_Ext_Mark_Default+".", ".")
				err = os.Rename(oldPath, newPath)
				if err != nil {
					return err
				}
			} else if strings.Contains(nowFileName, subparser.Sub_Ext_Mark_Forced+".") == true {
				// 寰楀寘鍚?.forced. 鎵句釜鍏抽敭璇?
				oldPath := dir + pathSep + curFile.Name()
				newPath := dir + pathSep + strings.ReplaceAll(curFile.Name(), subparser.Sub_Ext_Mark_Forced+".", ".")
				err = os.Rename(oldPath, newPath)
				if err != nil {
					return err
				}
			} else {
				continue
			}
		}
	}

	return nil
}

// DeleteOneSeasonSubCacheFolder 鍒犻櫎涓€涓繛缁墽涓殑鎵€鏈変竴瀛ｅ瓧骞曠殑缂撳瓨鏂囦欢澶?
func DeleteOneSeasonSubCacheFolder(seriesDir string) error {

	debugFolderByName, err := pkg.GetDebugFolderByName([]string{filepath.Base(seriesDir)})
	if err != nil {
		return err
	}
	files, err := os.ReadDir(debugFolderByName)
	if err != nil {
		return err
	}
	pathSep := string(os.PathSeparator)
	for _, curFile := range files {
		if curFile.IsDir() == true {
			matched := regex_things.RegOneSeasonSubFolderNameMatch.FindAllStringSubmatch(curFile.Name(), -1)
			if matched == nil || len(matched) < 1 {
				continue
			}

			fullPath := debugFolderByName + pathSep + curFile.Name()
			err = os.RemoveAll(fullPath)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

/*
	鍙拡瀵硅嫳鏂囧瓧骞曡繘琛屽悎骞跺垎鏁ｇ殑 DialoguesFilter
	浼氶亣鍒拌繖鏍风殑瀛楀箷锛屽涓?
	2line-The Card Counter (2021) WEBDL-1080p.chinese(inside).ass
	瀹冪殑瀵圭櫧涓€鍙ヨ瘽鍒嗕簡涓や釜 dialogue 鍘诲仛銆傝繖鏍峰仛鍚庣画瀛楀箷鏃堕棿杞存牎姝ｅ氨浼氶亣鍒伴棶棰橈紝鍥犱负鍙湁涓€鍗婏紝鍖归厤鍗犳瘮浼氬緢浣?
	(姣忎竴涓?Dialogue 鐨勯瀛楁瘝闇€瑕佸垎鏋愶紝澶у啓鍜屽皬鍐欑殑鍗犳瘮鏄灏戯紝缁熻涓€涓嬶紝姝ｅ父鐨勶紝鍜屼笂杩扮壒娈婄殑)
	閭ｄ箞锛屽氨闇€瑕侀澶栫殑閫昏緫鍘诲 DialoguesFilterEx 杩涜棰濆鐨勬帹鏂?
	鏆傛椂鑰冭檻鐨勬柟妗堟槸锛岃嫳鏂囧鐧芥瘡涓€鍙ョ殑寮€澶村簲璇ユ槸鑻辨枃澶у啓瀛楀箷锛屽鏋滄槸灏忓啓瀛楀箷锛屽氨搴旇涓庝笂璇彞鍚堝苟锛屼笖姣忎竴鍙ョ殑瀛楃闀垮害鏈夊ぇ浜庝竴瀹氭墠瑙﹀彂
*/
func MergeMultiDialogue4EngSubtitle(inSubParser *subparser.FileInfo) {
	merger := NewDialogueMerger()
	for _, dialogueEx := range inSubParser.DialoguesFilterEx {
		merger.Add(dialogueEx)
	}
	inSubParser.DialoguesFilterEx = merger.Get()
}

// GetVADInfoFeatureFromSub 璺熶笅闈㈢殑 GetVADInfoFeatureFromSubNeedOffsetTimeWillInsert 鍑芥暟鍔熻兘涓€鑷?
func GetVADInfoFeatureFromSub(fileInfo *subparser.FileInfo, frontAndEndPer float64, subUnitMaxCount int, insert bool) ([]SubUnit, error) {

	return GetVADInfoFeatureFromSubNeedOffsetTimeWillInsert(fileInfo, frontAndEndPer, subUnitMaxCount, 0, insert)
}

/*
	GetVADInfoFeatureFromSubNeedOffsetTimeWillInsert 鍙笉杩囪繖閲屽彲浠ュ姞涓€涓瘡涓€鍙ヨ瘽鍥哄畾鐨勫亸绉绘椂闂?
	杩欓噷鐨勫瓧骞曡姹傛槸瀹屾暣鐨勪竴涓瓧骞?
	1. 鎶藉彇瀛楀箷鐨勬椂闂寸墖娈电殑鏃跺€欙紝鏆傚畾锛屽墠 15% 鍜屽悗 15% 瑕侀伩寮€锛屽墠濂忋€佷富棰樻洸銆佺粨灏炬洸
	2. 灏嗘暣涓瓧骞曪紝鎶藉彇杩炵画 5 鍙ュ璇濅负涓€涓崟鍏冿紝鎻愬彇鏃堕棿鐗囨淇℃伅
	3. 杩欓噷鎶藉彇鐨勬槸鐗瑰緛锛屼篃灏辨湁棰濆鐨勯€昏緫鍘绘壘杩欎釜鐗瑰緛锛堟湰绋嬪簭鍐呬細鎻忚堪涓衡€滈挜鍖欌€濓級
*/
func GetVADInfoFeatureFromSubNeedOffsetTimeWillInsert(fileInfo *subparser.FileInfo, SkipFrontAndEndPer float64, subUnitMaxCount int, offsetTime float64, insert bool) ([]SubUnit, error) {
	if subUnitMaxCount < 0 {
		subUnitMaxCount = 0
	}

	nowDialogue := fileInfo.Dialogues

	srcSubUnitList := make([]SubUnit, 0)
	srcSubDialogueList := make([]subparser.OneDialogue, 0)
	srcOneSubUnit := NewSubUnit()

	// 鏈€鍚庝竴涓璇濈殑缁撴潫鏃堕棿
	lastDialogueExTimeEnd, err := pkg.ParseTime(nowDialogue[len(nowDialogue)-1].EndTime)
	if err != nil {
		return nil, err
	}
	// 鐩稿綋浜庢€绘椂闀?
	fullDuration := pkg.Time2SecondNumber(lastDialogueExTimeEnd)
	// 鏈€浣庣殑璧峰鏃堕棿锛屽洜涓哄彲鑳介渶瑕佽鍓寖鍥?
	startRangeTimeMin := fullDuration * SkipFrontAndEndPer
	endRangeTimeMax := fullDuration * (1.0 - SkipFrontAndEndPer)

	println(startRangeTimeMin)
	println(endRangeTimeMax)

	for i := 0; i < len(nowDialogue); i++ {

		oneDialogueExTimeStart, err := pkg.ParseTime(nowDialogue[i].StartTime)
		if err != nil {
			return nil, err
		}
		oneDialogueExTimeEnd, err := pkg.ParseTime(nowDialogue[i].EndTime)
		if err != nil {
			return nil, err
		}

		oneStart := pkg.Time2SecondNumber(oneDialogueExTimeStart)
		if SkipFrontAndEndPer > 0 {
			if fullDuration*SkipFrontAndEndPer > oneStart || fullDuration*(1.0-SkipFrontAndEndPer) < oneStart {
				continue
			}
		}

		if nowDialogue[i].Lines == nil || len(nowDialogue[i].Lines) == 0 {
			continue
		}
		// 濡傛灉褰撳墠鐨勮繖涓€鍙ヨ瘽锛屼负绌猴紝鎴栬€呰繘杩囨鍒欒〃杈惧紡鍓旈櫎鐗规畩瀛楃鍚庝负绌猴紝鍒欒烦杩?
		if pkg.ReplaceSpecString(nowDialogue[i].Lines[0], "") == "" {
			continue
		}
		// 濡傛灉褰撳墠鐨勮繖涓€鍙ヨ瘽锛屼负绌猴紝鎴栬€呰繘杩囨鍒欒〃杈惧紡鍓旈櫎鐗规畩瀛楃鍚庝负绌猴紝鍒欒烦杩?
		if pkg.ReplaceSpecString(fileInfo.GetDialogueExContent(i), "") == "" {
			continue
		}
		// 浣庝簬 5鍙ュ鐧斤紝鍒欐坊鍔?
		if srcOneSubUnit.GetDialogueCount() < subUnitMaxCount {
			// 绠椾笂鍋忕Щ
			offsetTimeDuration := time.Duration(offsetTime * math.Pow10(9))
			oneDialogueExTimeStart = oneDialogueExTimeStart.Add(offsetTimeDuration)
			oneDialogueExTimeEnd = oneDialogueExTimeEnd.Add(offsetTimeDuration)
			// 濡傛灉娌℃湁鍋忕Щ灏辨槸 0
			if insert == true {
				srcOneSubUnit.AddAndInsert(oneDialogueExTimeStart, oneDialogueExTimeEnd)
			} else {
				srcOneSubUnit.Add(oneDialogueExTimeStart, oneDialogueExTimeEnd)
			}
			// 杩欎竴涓崟鍏冪殑 Dialogue 闇€瑕佸悎骞惰捣鏉ワ紝鎵嶈兘鍒ゆ柇鏄惁绗﹀悎鈥滈挜鍖欌€濈殑瑕佹眰
			srcSubDialogueList = append(srcSubDialogueList, nowDialogue[i])

		} else {
			// 鐢ㄥ畬娓呯┖
			srcSubDialogueList = make([]subparser.OneDialogue, 0)
			// 灏嗘嫾鍑戣捣鏉ョ殑瀵硅瘽缁勬垚涓€涓崟鍏冭繘琛屽瓨鍌ㄨ捣鏉?
			srcSubUnitList = append(srcSubUnitList, *srcOneSubUnit)
			// 鐒跺悗閲嶇疆
			srcOneSubUnit = NewSubUnit()
		}
	}
	if srcOneSubUnit.GetDialogueCount() > 0 {
		srcSubUnitList = append(srcSubUnitList, *srcOneSubUnit)
	}

	return srcSubUnitList, nil
}

/*
	GetVADInfoFeatureFromSubNew 灏?Sub 鏂囦欢杞崲涓?VAD List 淇℃伅
*/
func GetVADInfoFeatureFromSubNew(fileInfo *subparser.FileInfo, SkipFrontAndEndPer float64) (*SubUnit, error) {

	outSubUnits := NewSubUnit()
	if len(fileInfo.Dialogues) <= 0 {
		return nil, errors.New("GetVADInfoFeatureFromSubNew fileInfo Dialogue Length is 0")
	}
	/*
		鍏堟嫾鍑戝嚭瀹屾暣鐨勪竴涓?VAD List
		鍥犱负 VAD 鐨勭獥鍙ｆ槸 10ms锛岄偅涔堥渶瑕佸姣忎竴鍙ヨ瘽鎸?10 ms 鐨勫崟浣嶈繘琛屽彇鏁?
		姣忎竴鍙ヨ瘽寮€濮嬨€佺粨鏉熺殑鏃堕棿锛岄渶瑕佸悜涓嬪彇鏁?
	*/
	subStartTimeFloor := pkg.MakeFloor10msMultipleFromFloat(pkg.Time2SecondNumber(fileInfo.GetStartTime()))
	subEndTimeFloor := pkg.MakeFloor10msMultipleFromFloat(pkg.Time2SecondNumber(fileInfo.GetEndTime()))
	// 濡傛灉鎯宠浠?0 鏃堕棿鐐瑰紑濮嬬畻锛岄偅涔?subStartTimeFloor 杩欎釜鍊煎氨闇€瑕侀噸缃埌0
	subStartTimeFloor = 0
	subFullSecondTimeFloor := subEndTimeFloor - subStartTimeFloor
	// 鏍规嵁杩欎釜鏃堕暱灏辫兘澶熷緱鍒颁竴涓畬鏁寸殑 VAD List锛岀劧鍚庡啀閫氳繃姣忎竴鍙ュ鐧借繘琛?VAD 鍊肩殑璋冩暣鍗冲彲锛岃繖鏍峰氨鑳藉淇濊瘉
	// 鐩稿悓鐨勪竴涓瓧骞曞洜涓轰娇鐢?ffmpeg 瀵煎嚭 srt 鍜?ass 鍚庣殑锛屽彲鑳藉瓨鍦ㄦ€讳綋鏃堕棿杞翠笉涓€鑷寸殑闂
	// 123.450 - > 12345
	vadLen := int(subFullSecondTimeFloor*100) + 2
	subVADs := make([]vad.VADInfo, vadLen)
	subStartTimeFloor10ms := subStartTimeFloor * 100
	for i := 0; i < vadLen; i++ {
		subVADs[i] = *vad.NewVADInfoBase(false, time.Duration((subStartTimeFloor10ms+float64(i))*math.Pow10(7)))
	}
	// 璁＄畻鍑洪渶瑕佹埅鍙栫殑鐗囨,璧峰鍜岀粨鏉?
	skipLen := int(float64(vadLen) * SkipFrontAndEndPer)
	skipStartIndex := skipLen
	skipEndIndex := vadLen - skipLen
	// 鐜板湪闇€瑕佷粠 fileInfo 鐨勬瘡涓€鍙ュ鐧戒篃灏卞搴斾竴娈佃繛缁殑 VAD active = true 鏉ヨ繘琛屾敼鍐欙紝璁板緱鍚戜笅鍙栨暣
	lastDialogueIndex := 0
	for _, dialogue := range fileInfo.Dialogues {

		if dialogue.Lines == nil || len(dialogue.Lines) == 0 {
			continue
		}
		// 濡傛灉褰撳墠鐨勮繖涓€鍙ヨ瘽锛屼负绌猴紝鎴栬€呰繘杩囨鍒欒〃杈惧紡鍓旈櫎鐗规畩瀛楃鍚庝负绌猴紝鍒欒烦杩?
		if pkg.ReplaceSpecString(dialogue.Lines[0], "") == "" {
			continue
		}
		// 瀛楀箷鐨勫紑濮嬫椂闂?
		oneDialogueStartTime, err := pkg.ParseTime(dialogue.StartTime)
		if err != nil {
			return nil, err
		}
		// 瀛楀箷鐨勭粨鏉熸椂闂?
		oneDialogueEndTime, err := pkg.ParseTime(dialogue.EndTime)
		if err != nil {
			return nil, err
		}
		// 瀛楀箷鐨勬椂闀匡紝瀵规椂闂磋繘琛屽悜涓嬪彇鏁?
		oneDialogueStartTimeFloor := pkg.MakeCeil10msMultipleFromFloat(pkg.Time2SecondNumber(oneDialogueStartTime))
		oneDialogueEndTimeFloor := pkg.MakeFloor10msMultipleFromFloat(pkg.Time2SecondNumber(oneDialogueEndTime))
		// 寰楀埌涓€鍙ュ鐧界殑鏃堕暱
		changeVADStartIndex := int(oneDialogueStartTimeFloor * 100)
		changeVADEndIndex := int(oneDialogueEndTimeFloor * 100)
		// 涓嶈兘瓒呰繃 鏈€鍚庝竴鍙ヨ瘽鐨勬椂甯?
		if changeVADStartIndex > int(subEndTimeFloor*100) {
			continue
		}
		// 涔熶笉鑳芥瘮璧峰鐨勭涓€鍙ヨ瘽鏃堕棿杞存洿浣?
		if changeVADStartIndex < int(subStartTimeFloor10ms) {
			continue
		}
		// 褰撳墠杩欏彞璇濈殑寮€濮嬪拰缁撴潫淇℃伅
		changerStartIndex := changeVADStartIndex - int(subStartTimeFloor10ms)
		if changerStartIndex < 0 {
			continue
		}
		changerEndIndex := changeVADEndIndex - int(subStartTimeFloor10ms)
		if changerEndIndex < 0 {
			continue
		}
		// 濡傛灉涓婁竴涓鐧界殑鏈€鍚庝竴涓?OffsetIndex 杩炴帴鐫€褰撳墠杩欎竴鍙ョ殑绱㈠紩鐨?VAD 淇℃伅 active 鏄?true 灏辫缃负 false
		if lastDialogueIndex == changerStartIndex {
			for i := 1; i <= 2; i++ {
				if lastDialogueIndex-i >= 0 && subVADs[lastDialogueIndex-i].Active == true {
					subVADs[lastDialogueIndex-i].Active = false
				}
			}
		}
		// 寮€濮嬫牴鎹綋鍓嶈繖鍙ヨ瘽杩涜 VAD 淇℃伅鐨勮缃?
		// 璋冩暣涔嬪墠鍋氬ソ鐨勬暣浣?VAD 鐨勪俊鎭紝绗﹀悎 VAD active = true
		if changerEndIndex >= vadLen {
			changerEndIndex = vadLen - 1
		}
		for i := changerStartIndex; i <= changerEndIndex; i++ {
			subVADs[i].Active = true
		}
		lastDialogueIndex = changerEndIndex
	}

	// 鎴彇鍑烘潵褰撳墠杩欎竴娈?
	tmpVADList := subVADs[skipStartIndex:skipEndIndex]
	outSubUnits.VADList = tmpVADList

	tmpStartTime := time.Time{}
	tmpStartTime = tmpStartTime.Add(tmpVADList[0].Time)
	tmpEndTime := time.Time{}
	tmpEndTime = tmpEndTime.Add(tmpVADList[len(tmpVADList)-1].Time)

	outSubUnits.SetBaseTime(tmpStartTime)
	outSubUnits.SetOffsetStartTime(tmpStartTime)
	outSubUnits.SetOffsetEndTime(tmpEndTime)

	return outSubUnits, nil
}
