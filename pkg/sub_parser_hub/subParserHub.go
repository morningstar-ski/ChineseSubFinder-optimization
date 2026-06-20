package sub_parser_hub

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/filter"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/ifaces"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	languageConst "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/subparser"
	"github.com/sirupsen/logrus"
)

type SubParserHub struct {
	log    *logrus.Logger
	Parser []ifaces.ISubParser
}

// NewSubParserHub 澶勭悊鐨勫瓧骞曟枃浠堕渶瑕佺鍚?[siteName]_ 鐨勫墠缂€鎻忚堪锛屾槸鏈▼搴忎笓鐢ㄧ殑
func NewSubParserHub(log *logrus.Logger, parser ifaces.ISubParser, _parser ...ifaces.ISubParser) *SubParserHub {
	s := SubParserHub{}
	s.log = log
	s.Parser = make([]ifaces.ISubParser, 0)
	s.Parser = append(s.Parser, parser)
	if len(_parser) > 0 {
		for _, one := range _parser {
			s.Parser = append(s.Parser, one)
		}
	}
	return &s
}

// DetermineFileTypeFromFile 纭畾瀛楀箷鏂囦欢鐨勭被鍨嬶紝鏄弻璇瓧骞曟垨鑰呮煇涓€绉嶈瑷€绛夌瓑淇℃伅锛屽鏋滆繑鍥?nil 锛岄偅涔堝氨璇存槑閮芥病鏈夊瓧骞曠殑鏍煎紡鍖归厤涓?
func (p SubParserHub) DetermineFileTypeFromFile(filePath string) (bool, *subparser.FileInfo, error) {
	for _, parser := range p.Parser {
		bFind, subFileInfo, err := parser.DetermineFileTypeFromFile(filePath)
		if err != nil {
			return false, nil, err
		}
		if bFind == false {
			continue
		}
		// 姝ｅ父鑷冲皯搴旇鍖归厤涓€涓惂锛屼笉鐒跺氨鏄渶澶栧眰缁х画杩斿洖 nil 鍑哄幓浜?
		// 绠€浣撳拰绻佷綋瀛楀箷鐨勫垽鏂紝閫氳繃鏂囦欢鍚嶆潵鍋氬埌鐨勶紝鍩烘湰灏辩畻涓ˉ鍒よ€屽凡
		//newLang := IsChineseSimpleOrTraditional(filePath, subFileInfo.Lang)
		subFileInfo.Name = filepath.Base(filePath)
		//subFileInfo.Lang = newLang
		subFileInfo.FileFullPath = filePath
		subFileInfo.FromWhereSite = p.getFromWhereSite(filePath)
		return true, subFileInfo, nil
	}
	// 濡傛灉杩斿洖 nil 锛岄偅涔堝氨璇存槑閮芥病鏈夊瓧骞曠殑鏍煎紡鍖归厤涓?
	return false, nil, nil
}

// DetermineFileTypeFromBytes 纭畾瀛楀箷鏂囦欢鐨勭被鍨嬶紝鏄弻璇瓧骞曟垨鑰呮煇涓€绉嶈瑷€绛夌瓑淇℃伅锛屽鏋滆繑鍥?nil 锛岄偅涔堝氨璇存槑閮芥病鏈夊瓧骞曠殑鏍煎紡鍖归厤涓?
// 濡傛灉瑕佸仛瀛楀箷鐨勬椂闂磋酱鍖归厤锛屽緢鍙兘闇€瑕佷竴涓姛鑳?sub_helper.MergeMultiDialogue4EngSubtitle锛屼絾鏄粎浠呮槸鍚堝苟浜?English 瀛楀箷鏃堕棿杞?
func (p SubParserHub) DetermineFileTypeFromBytes(inBytes []byte, nowExt string) (bool, *subparser.FileInfo, error) {
	normalizedBytes, err := language.ChangeFileCoding2UTF8(inBytes)
	if err != nil {
		return p.determineFileTypeFromBytesWithPayload(inBytes, nowExt)
	}

	found, subFileInfo, err := p.determineFileTypeFromBytesWithPayload(normalizedBytes, nowExt)
	if err != nil {
		return false, nil, err
	}
	if found == true {
		return true, subFileInfo, nil
	}
	if string(normalizedBytes) == string(inBytes) {
		return false, nil, nil
	}

	return p.determineFileTypeFromBytesWithPayload(inBytes, nowExt)
}

func (p SubParserHub) determineFileTypeFromBytesWithPayload(inBytes []byte, nowExt string) (bool, *subparser.FileInfo, error) {
	for _, parser := range p.Parser {
		bFind, subFileInfo, err := parser.DetermineFileTypeFromBytes(inBytes, nowExt)
		if err != nil {
			return false, nil, err
		}
		if bFind == false {
			continue
		}
		return true, subFileInfo, nil
	}
	return false, nil, nil
}

// IsSubHasChinese 瀛楀箷鏂囦欢鏄惁鍖呭惈涓枃
func (p SubParserHub) IsSubHasChinese(fileInfo *subparser.FileInfo) bool {
	if fileInfo == nil {
		return false
	}

	// 澧炲姞鍒ゆ柇宸插瓨鍦ㄧ殑瀛楀箷鏄惁鏈変腑鏂?
	if language.HasChineseLang(fileInfo.Lang) == false {
		if p.log != nil {
			p.log.Warnln("IsSubHasChinese.HasChineseLang", fileInfo.FileFullPath, "not chinese sub, is ", fileInfo.Lang.String())
		}
		return false
	}

	if fileInfoHasChineseContent(fileInfo) == false {
		if p.log != nil {
			p.log.Warnln("IsSubHasChinese.NoChineseContent", fileInfo.FileFullPath, "lang", fileInfo.Lang.String())
		}
		return false
	}

	return true
}

func fileInfoHasChineseContent(fileInfo *subparser.FileInfo) bool {
	if fileInfo == nil {
		return false
	}
	for _, line := range fileInfo.CHLines {
		if containsChineseRune(line) {
			return true
		}
	}
	for _, dialogueEx := range fileInfo.DialoguesFilterEx {
		if containsChineseRune(dialogueEx.ChLine) {
			return true
		}
	}
	for _, dialogue := range fileInfo.DialoguesFilter {
		for _, line := range dialogue.Lines {
			if containsChineseRune(line) {
				return true
			}
		}
	}
	return containsChineseRune(fileInfo.Content)
}

func containsChineseRune(input string) bool {
	for _, r := range input {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

// getFromWhereSite 浠庢枃浠跺悕鎵惧嚭鏄粠閭ｄ釜缃戠珯涓嬭浇鐨勩€傝繖閲岀殑鏂囦欢鍚嶇殑鍓嶇紑鏄笅杞芥椂鍊欐爣璁板ソ鐨勶紝姣旇緝鐗规畩
func (p SubParserHub) getFromWhereSite(filePath string) string {
	fileName := filepath.Base(filePath)
	var re = regexp.MustCompile(`^\[(\w+)\]_`)
	matched := re.FindStringSubmatch(fileName)
	if matched == nil || len(matched) < 1 {
		return ""
	}
	return matched[1]
}

// IsSubTypeWanted 杩欓噷鍖归厤鐨勫瓧骞曠殑鏍煎紡锛屼笉鍖呭惈 Ext 鐨?. 灏忔暟鐐癸紝娉ㄦ剰锛屼粎浠呮槸鍖呭惈鍏崇郴
func IsSubTypeWanted(subName string) bool {
	nowLowerName := strings.ToLower(subName)
	if strings.Contains(nowLowerName, common.SubTypeASS) ||
		strings.Contains(nowLowerName, common.SubTypeSSA) ||
		strings.Contains(nowLowerName, common.SubTypeSRT) {
		return true
	}

	return false
}

// IsSubExtWanted 杈撳叆鐨勫瓧骞曟枃浠跺悕锛屽垽鏂悗缂€鍚嶆槸鍚︾鍚堟湡鏈涚殑瀛楀箷鍚庣紑鍚嶅垪琛?
func IsSubExtWanted(subName string) bool {
	inExt := filepath.Ext(subName)
	switch strings.ToLower(inExt) {
	case common.SubExtSSA, common.SubExtASS, common.SubExtSRT:
		return true
	default:
		return false
	}
}

// IsEmbySubCodecWanted 浠?Emby api 鎷垮埌瀛楀箷鐨?sub 绫诲瀷 string (Codec) 鏄惁鏄鍚堟湰绋嬪簭瑕佹眰鐨?
func IsEmbySubCodecWanted(inSubCodec string) bool {

	tmpString := strings.ToLower(inSubCodec)
	if tmpString == common.SubTypeSRT ||
		tmpString == common.SubTypeASS ||
		tmpString == common.SubTypeSSA {
		return true
	}

	return false
}

// IsEmbySubChineseLangStringWanted 鏄惁鏄?Emby 鑷繁瑙ｆ瀽鍑烘潵鐨勪腑鏂囪瑷€绫诲瀷
func IsEmbySubChineseLangStringWanted(inLangString string) bool {

	isWanted := false

	tmpString := strings.ToLower(inLangString)
	nextString := tmpString
	spStrings := strings.Split(tmpString, "[")
	if len(spStrings) > 1 {
		// 鍘婚櫎 chi[xunlie] 绫讳技鐨勬爣璁?
		nextString = spStrings[0]
	} else {
		// 鍘婚櫎 chinese锛堢畝鑻?zimuku锛?
		spStrings = strings.Split(tmpString, "(")
		if len(spStrings) > 1 {
			nextString = spStrings[0]
		}
	}

	// 鍏堝垽鏂?ISO 鏍囧噯鐨勫拰鍙樼鐨勬敮鎸佸垪琛紝浠呬粎鏄腑鏂囩殑
	if language.IsSupportISOChineseString(nextString) {
		isWanted = true
	}

	// 鍐嶅垽鏂箣鍓嶆敮鎸佺殑鍒楄〃
	switch nextString {
	case languageConst.Emby_chinese_chs,
		languageConst.Emby_chinese_cht,
		languageConst.Emby_chinese_chi:
		// chi chs cht
		isWanted = true
	case replaceLangString(languageConst.Emby_chinese):
		// chinese锛岃繖涓瘮杈冪壒娈婏紝鏄湰绋嬪簭瀹氫箟鐨?chinese 鐨勫瓧娈碉紝鍐?Emby API 涓嬬壒娈婄殑瀛楀箷鍛藉悕瀛楁
		isWanted = true
	}

	return isWanted
}

// SearchMatchedSubFile 鎼滅储绗﹀悎鍚庣紑鍚嶇殑瀛楀箷鏂囦欢
func SearchMatchedSubFile(log *logrus.Logger, dir string) ([]string, error) {

	var fileFullPathList = make([]string, 0)
	pathSep := string(os.PathSeparator)
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, curFile := range files {
		fullPath := dir + pathSep + curFile.Name()
		if curFile.IsDir() {
			// 鍐呭眰鐨勯敊璇氨鏃犺浜?
			oneList, _ := SearchMatchedSubFile(log, fullPath)
			if oneList != nil {
				fileFullPathList = append(fileFullPathList, oneList...)
			}
		} else {
			// 杩欓噷灏辨槸鏂囦欢浜?
			if IsSubExtWanted(curFile.Name()) == false {
				continue
			} else {

				// 璺宠繃涓嶇鍚堢殑鏂囦欢锛屾瘮濡?MAC OS 涓嬪彲鑳芥湁缂撳瓨鏂囦欢锛岃 #138
				if filter.SkipFileInfo(log, curFile, fullPath) == true {
					continue
				}

				fileFullPathList = append(fileFullPathList, fullPath)
			}
		}
	}
	return fileFullPathList, nil
}

func replaceLangString(inString string) string {
	tmpString := strings.ToLower(inString)
	one := strings.ReplaceAll(tmpString, ".", "")
	two := strings.ReplaceAll(one, "_", "")
	return two
}
