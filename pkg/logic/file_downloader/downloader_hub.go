package file_downloader

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/media_info_dealers"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/ass"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/srt"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_parser_hub"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/random_auth_key"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/subtitle_best_api"
	"github.com/go-rod/rod"
	"github.com/sirupsen/logrus"
)

type FileDownloader struct {
	Log              *logrus.Logger
	CacheCenter      *cache_center.CacheCenter
	SubParserHub     *sub_parser_hub.SubParserHub
	MediaInfoDealers *media_info_dealers.Dealers
}

func NewFileDownloader(cacheCenter *cache_center.CacheCenter, authKey random_auth_key.AuthKey) *FileDownloader {

	f := FileDownloader{
		Log:              cacheCenter.Log,
		CacheCenter:      cacheCenter,
		SubParserHub:     sub_parser_hub.NewSubParserHub(cacheCenter.Log, ass.NewParser(cacheCenter.Log), srt.NewParser(cacheCenter.Log)),
		MediaInfoDealers: media_info_dealers.NewDealers(cacheCenter.Log, subtitle_best_api.NewSubtitleBestApi(cacheCenter.Log, authKey)),
	}
	return &f
}

func (f *FileDownloader) GetName() string {
	return f.CacheCenter.GetName()
}

func (f *FileDownloader) validateDownloadedPayload(fileName string, fileData []byte) error {
	if err := pkg.ValidateDownloadedPayload(fileName, fileData); err != nil {
		return err
	}

	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".zip", ".tar", ".rar", ".7z":
		return nil
	}

	if f.SubParserHub == nil {
		return nil
	}

	ok, _, err := f.SubParserHub.DetermineFileTypeFromBytes(fileData, ext)
	if err != nil {
		return err
	}
	if ok == false {
		return fmt.Errorf("download payload is not a supported subtitle: %s", fileName)
	}

	return nil
}

// Get supplierName 杩欎釜鍙傛暟涓€瀹氬緱鏄瓧骞曟簮鐨勫悕绉帮紝閫氳繃 s.GetSupplierName() 鑾峰彇锛屽惁鍒欏悗缁殑瀛楀箷婧愪粖鏃ヤ笅杞介噺灏嗕笉鑳芥纭粺璁″拰鍒ゆ柇
// xunlei銆乻hooter 浣跨敤杩欎釜
func (f *FileDownloader) Get(supplierName string, topN int64, videoFileName string,
	fileDownloadUrl string, score int64, offset int64, cacheString ...string) (*supplier.SubInfo, error) {

	var fileUID string

	if len(cacheString) < 1 {
		fileUID = fmt.Sprintf("%x", sha256.Sum256([]byte(fileDownloadUrl)))
	} else {
		fileUID = cacheString[0]
	}

	found, subInfo, err := f.CacheCenter.DownloadFileGet(fileUID)
	if err != nil {
		return nil, err
	}
	// 濡傛灉涓嶅瓨鍦ㄩ偅涔堝氨鍏堜笅杞斤紝鐒跺悗鍐嶅瓨鍏ョ紦瀛樹腑
	if found == false {
		fileData, downloadFileName, err := pkg.DownFile(f.Log, fileDownloadUrl)
		if err != nil {
			return nil, err
		}
		validationName := downloadFileName
		if validationName == "" {
			validationName = fileDownloadUrl
		}
		if err := f.validateDownloadedPayload(validationName, fileData); err != nil {
			return nil, err
		}
		// 涓嬭浇鎴愬姛闇€瑕佺粺璁″埌浠婂ぉ鐨勬鏁颁腑
		_, err = f.CacheCenter.DailyDownloadCountAdd(supplierName,
			pkg.GetPublicIP(f.Log, settings.Get().AdvancedSettings.TaskQueue))
		if err != nil {
			f.Log.Warningln(supplierName, "FileDownloader.Get.DailyDownloadCountAdd", err)
		}
		// 闇€瑕佽幏鍙栦笅杞芥枃浠剁殑鍚庣紑鍚嶏紝鍚庣画鎵嶆寚瀵兼槸瑕佽В鍘嬭繕鏄洿鎺ヨВ鏋愬瓧骞?
		ext := ""
		if downloadFileName == "" {
			ext = filepath.Ext(fileDownloadUrl)
		} else {
			ext = filepath.Ext(downloadFileName)
		}
		// 榛樿瀛樺叆閮芥槸绠€浣撲腑鏂囩殑璇█绫诲瀷锛屽悗缁彇鍑烘潵鐨勬椂鍊欓渶瑕佸啀娆¤皟鐢?SubParser 杩涜瑙ｆ瀽
		inSubInfo := supplier.NewSubInfo(supplierName, topN, videoFileName, language.ChineseSimple, fileDownloadUrl, score, offset, ext, fileData)

		if len(cacheString) > 0 {
			// 涓撻棬涓?ASSRT 杩欑涓嬭浇杩炴帴鏄复鏃舵儏鍐佃€屽畾鍒剁殑
			inSubInfo.SetFileUrlSha256(fileUID)
		}

		err = f.CacheCenter.DownloadFileAdd(inSubInfo)
		if err != nil {
			return nil, err
		}

		return inSubInfo, nil
	} else {
		// 濡傛灉宸茬粡瀛樺湪缂撳瓨涓紝閭ｄ箞灏辩洿鎺ヨ繑鍥?
		return subInfo, nil
	}
}

// GetA4k supplierName 杩欎釜鍙傛暟涓€瀹氬緱鏄瓧骞曟簮鐨勫悕绉帮紝閫氳繃 s.GetSupplierName() 鑾峰彇锛屽惁鍒欏悗缁殑瀛楀箷婧愪粖鏃ヤ笅杞介噺灏嗕笉鑳芥纭粺璁″拰鍒ゆ柇
func (f *FileDownloader) GetA4k(supplierName string, topN int64, season, eps int,
	videoFileName string, fileDownloadUrl string) (*supplier.SubInfo, error) {

	var fileUID string
	fileUID = fmt.Sprintf("%x", sha256.Sum256([]byte(fileDownloadUrl)))

	found, subInfo, err := f.CacheCenter.DownloadFileGet(fileUID)
	if err != nil {
		return nil, err
	}
	// 濡傛灉涓嶅瓨鍦ㄩ偅涔堝氨鍏堜笅杞斤紝鐒跺悗鍐嶅瓨鍏ョ紦瀛樹腑
	if found == false {
		fileData, downloadFileName, err := pkg.DownFile(f.Log, fileDownloadUrl)
		if err != nil {
			return nil, err
		}
		validationName := downloadFileName
		if validationName == "" {
			validationName = fileDownloadUrl
		}
		if err := f.validateDownloadedPayload(validationName, fileData); err != nil {
			return nil, err
		}
		// 涓嬭浇鎴愬姛闇€瑕佺粺璁″埌浠婂ぉ鐨勬鏁颁腑
		_, err = f.CacheCenter.DailyDownloadCountAdd(supplierName,
			pkg.GetPublicIP(f.Log, settings.Get().AdvancedSettings.TaskQueue))
		if err != nil {
			f.Log.Warningln(supplierName, "FileDownloader.Get.DailyDownloadCountAdd", err)
		}
		// 闇€瑕佽幏鍙栦笅杞芥枃浠剁殑鍚庣紑鍚嶏紝鍚庣画鎵嶆寚瀵兼槸瑕佽В鍘嬭繕鏄洿鎺ヨВ鏋愬瓧骞?
		ext := ""
		if downloadFileName == "" {
			ext = filepath.Ext(fileDownloadUrl)
		} else {
			ext = filepath.Ext(downloadFileName)
		}
		// 榛樿瀛樺叆閮芥槸绠€浣撲腑鏂囩殑璇█绫诲瀷锛屽悗缁彇鍑烘潵鐨勬椂鍊欓渶瑕佸啀娆¤皟鐢?SubParser 杩涜瑙ｆ瀽
		inSubInfo := supplier.NewSubInfo(supplierName, topN, videoFileName, language.ChineseSimple, fileDownloadUrl, 0, 0, ext, fileData)
		inSubInfo.Season = season
		inSubInfo.Episode = eps
		inSubInfo.GetUID()

		err = f.CacheCenter.DownloadFileAdd(inSubInfo)
		if err != nil {
			return nil, err
		}

		return inSubInfo, nil
	} else {
		// 濡傛灉宸茬粡瀛樺湪缂撳瓨涓紝閭ｄ箞灏辩洿鎺ヨ繑鍥?
		return subInfo, nil
	}
}

// GetEx supplierName 杩欎釜鍙傛暟涓€瀹氬緱鏄瓧骞曟簮鐨勫悕绉帮紝閫氳繃 s.GetSupplierName() 鑾峰彇锛屽惁鍒欏悗缁殑瀛楀箷婧愪粖鏃ヤ笅杞介噺灏嗕笉鑳芥纭粺璁″拰鍒ゆ柇
// zimuku銆乻ubhd 浣跨敤杩欎釜
func (f *FileDownloader) GetEx(supplierName string, browser *rod.Browser, subDownloadPageUrl string, TopN int64, Season, Episode int, downFileFunc func(browser *rod.Browser, subDownloadPageUrl string, TopN int64, Season, Episode int) (*supplier.SubInfo, error)) (*supplier.SubInfo, error) {

	fileUID := fmt.Sprintf("%x", sha256.Sum256([]byte(subDownloadPageUrl)))
	found, subInfo, err := f.CacheCenter.DownloadFileGet(fileUID)
	if err != nil {
		return nil, err
	}
	// 濡傛灉涓嶅瓨鍦ㄩ偅涔堝氨鍏堜笅杞斤紝鐒跺悗鍐嶅瓨鍏ョ紦瀛樹腑
	if found == false {

		subInfo, err = downFileFunc(browser, subDownloadPageUrl, TopN, Season, Episode)
		if err != nil {
			return nil, err
		}
		validationName := subInfo.Name
		if validationName == "" {
			validationName = subDownloadPageUrl
		}
		if subInfo.Ext != "" && filepath.Ext(validationName) == "" {
			validationName += subInfo.Ext
		}
		if err := f.validateDownloadedPayload(validationName, subInfo.Data); err != nil {
			return nil, err
		}
		// 涓嬭浇鎴愬姛闇€瑕佺粺璁″埌浠婂ぉ鐨勬鏁颁腑
		_, err = f.CacheCenter.DailyDownloadCountAdd(supplierName,
			pkg.GetPublicIP(f.Log, settings.Get().AdvancedSettings.TaskQueue))
		if err != nil {
			f.Log.Warningln(supplierName, "FileDownloader.GetEx.DailyDownloadCountAdd", err)
		}
		// 榛樿瀛樺叆閮芥槸绠€浣撲腑鏂囩殑璇█绫诲瀷锛屽悗缁彇鍑烘潵鐨勬椂鍊欓渶瑕佸啀娆¤皟鐢?SubParser 杩涜瑙ｆ瀽
		err = f.CacheCenter.DownloadFileAdd(subInfo)
		if err != nil {
			return nil, err
		}

		return subInfo, nil
	} else {
		// 濡傛灉宸茬粡瀛樺湪缂撳瓨涓紝閭ｄ箞灏辩洿鎺ヨ繑鍥?
		return subInfo, nil
	}
}

// GetSubtitleBest supplierName 杩欎釜鍙傛暟涓€瀹氬緱鏄瓧骞曟簮鐨勫悕绉帮紝閫氳繃 s.GetSupplierName() 鑾峰彇锛屽惁鍒欏悗缁殑瀛楀箷婧愪粖鏃ヤ笅杞介噺灏嗕笉鑳芥纭粺璁″拰鍒ゆ柇
func (f *FileDownloader) GetSubtitleBest(supplierName string, topN int64, season, eps int,
	title, ext, subSha256, fileDownloadUrl string) (*supplier.SubInfo, error) {

	found, subInfo, err := f.CacheCenter.DownloadFileGet(subSha256)
	if err != nil {
		return nil, err
	}
	// 濡傛灉涓嶅瓨鍦ㄩ偅涔堝氨鍏堜笅杞斤紝鐒跺悗鍐嶅瓨鍏ョ紦瀛樹腑
	if found == false {

		fileData, _, err := pkg.DownFile(f.Log, fileDownloadUrl)
		if err != nil {
			return nil, err
		}
		validationName := title
		if validationName == "" {
			validationName = fileDownloadUrl
		}
		if ext != "" && filepath.Ext(validationName) == "" {
			validationName += ext
		}
		if err := f.validateDownloadedPayload(validationName, fileData); err != nil {
			return nil, err
		}
		// 涓嬭浇鎴愬姛闇€瑕佺粺璁″埌浠婂ぉ鐨勬鏁颁腑
		_, err = f.CacheCenter.DailyDownloadCountAdd(supplierName,
			pkg.GetPublicIP(f.Log, settings.Get().AdvancedSettings.TaskQueue))
		if err != nil {
			f.Log.Warningln(supplierName, "FileDownloader.Get.DailyDownloadCountAdd", err)
		}
		// 榛樿瀛樺叆閮芥槸绠€浣撲腑鏂囩殑璇█绫诲瀷锛屽悗缁彇鍑烘潵鐨勬椂鍊欓渶瑕佸啀娆¤皟鐢?SubParser 杩涜瑙ｆ瀽
		inSubInfo := supplier.NewSubInfo(supplierName, topN, title, language.ChineseSimple, fileDownloadUrl, 0, 0, ext, fileData)
		inSubInfo.Season = season
		inSubInfo.Episode = eps
		inSubInfo.SetFileUrlSha256(subSha256)
		inSubInfo.GetUID()

		err = f.CacheCenter.DownloadFileAdd(inSubInfo)
		if err != nil {
			return nil, err
		}

		return inSubInfo, nil
	} else {
		// 濡傛灉宸茬粡瀛樺湪缂撳瓨涓紝閭ｄ箞灏辩洿鎺ヨ繑鍥?
		return subInfo, nil
	}
}
