package file_downloader

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/ass"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/srt"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/media_info_dealers"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/random_auth_key"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_parser_hub"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
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
	_ = authKey

	f := FileDownloader{
		Log:              cacheCenter.Log,
		CacheCenter:      cacheCenter,
		SubParserHub:     sub_parser_hub.NewSubParserHub(cacheCenter.Log, ass.NewParser(cacheCenter.Log), srt.NewParser(cacheCenter.Log)),
		MediaInfoDealers: media_info_dealers.NewDealers(cacheCenter.Log),
	}
	return &f
}

func (f *FileDownloader) GetName() string {
	return f.CacheCenter.GetName()
}

func (f *FileDownloader) inspectSubtitlePayload(body []byte, ext string) (bool, error) {
	if f.SubParserHub == nil {
		return false, fmt.Errorf("subtitle parser hub is nil")
	}
	found, _, err := f.SubParserHub.DetermineFileTypeFromBytes(body, ext)
	return found, err
}

func (f *FileDownloader) ValidateCachedSubInfo(subInfo *supplier.SubInfo) error {
	if subInfo == nil {
		return fmt.Errorf("subInfo is nil")
	}

	fileName := subInfo.Name
	if fileName == "" {
		fileName = "subtitle" + subInfo.Ext
	} else if subInfo.Ext != "" && filepath.Ext(fileName) == "" && strings.HasSuffix(fileName, subInfo.Ext) == false {
		fileName += subInfo.Ext
	}

	return pkg.ValidateSubtitleDownloadPayload(f.Log, f.inspectSubtitlePayload, subInfo.FileUrl, fileName, "", 0, subInfo.Data)
}

// Get supplierName 这个参数一定得是字幕源的名称，通过 s.GetSupplierName() 获取，否则后续的字幕源今日下载量将不能正确统计和判断
// xunlei、shooter 使用这个
func (f *FileDownloader) Get(supplierName string, topN int64, videoFileName string,
	fileDownloadUrl string, score int64, offset int64, cacheString ...string) (*supplier.SubInfo, error) {

	var fileUID string

	if len(cacheString) < 1 {
		fileUID = fmt.Sprintf("%x", sha256.Sum256([]byte(fileDownloadUrl)))
	} else {
		fileUID = cacheString[0]
	}

	found, subInfo, err := f.CacheCenter.DownloadFileGet(fileUID, f.ValidateCachedSubInfo)
	if err != nil {
		return nil, err
	}
	if found == false {
		fileData, downloadFileName, err := pkg.DownSubtitleFile(f.Log, f.inspectSubtitlePayload, fileDownloadUrl)
		if err != nil {
			return nil, err
		}
		_, err = f.CacheCenter.DailyDownloadCountAdd(supplierName,
			pkg.GetPublicIP(f.Log, settings.Get().AdvancedSettings.TaskQueue))
		if err != nil {
			f.Log.Warningln(supplierName, "FileDownloader.Get.DailyDownloadCountAdd", err)
		}
		ext := ""
		if downloadFileName == "" {
			ext = filepath.Ext(fileDownloadUrl)
		} else {
			ext = filepath.Ext(downloadFileName)
		}
		inSubInfo := supplier.NewSubInfo(supplierName, topN, videoFileName, language.ChineseSimple, fileDownloadUrl, score, offset, ext, fileData)

		if len(cacheString) > 0 {
			inSubInfo.SetFileUrlSha256(fileUID)
		}

		err = f.CacheCenter.DownloadFileAdd(inSubInfo)
		if err != nil {
			return nil, err
		}

		return inSubInfo, nil
	}

	return subInfo, nil
}

// GetEx supplierName 这个参数一定得是字幕源的名称，通过 s.GetSupplierName() 获取，否则后续的字幕源今日下载量将不能正确统计和判断
// zimuku、subhd 使用这个
func (f *FileDownloader) GetEx(supplierName string, browser *rod.Browser, subDownloadPageUrl string, TopN int64, Season, Episode int, downFileFunc func(browser *rod.Browser, subDownloadPageUrl string, TopN int64, Season, Episode int) (*supplier.SubInfo, error)) (*supplier.SubInfo, error) {

	fileUID := fmt.Sprintf("%x", sha256.Sum256([]byte(subDownloadPageUrl)))
	found, subInfo, err := f.CacheCenter.DownloadFileGet(fileUID, f.ValidateCachedSubInfo)
	if err != nil {
		return nil, err
	}
	if found == false {

		subInfo, err = downFileFunc(browser, subDownloadPageUrl, TopN, Season, Episode)
		if err != nil {
			return nil, err
		}
		if subInfo == nil {
			return nil, fmt.Errorf("downFileFunc returned nil subInfo")
		}
		err = pkg.ValidateSubtitleDownloadPayload(f.Log, f.inspectSubtitlePayload, subDownloadPageUrl, subInfo.Name, "", 0, subInfo.Data)
		if err != nil {
			return nil, err
		}
		_, err = f.CacheCenter.DailyDownloadCountAdd(supplierName,
			pkg.GetPublicIP(f.Log, settings.Get().AdvancedSettings.TaskQueue))
		if err != nil {
			f.Log.Warningln(supplierName, "FileDownloader.GetEx.DailyDownloadCountAdd", err)
		}
		err = f.CacheCenter.DownloadFileAdd(subInfo)
		if err != nil {
			return nil, err
		}

		return subInfo, nil
	}

	return subInfo, nil
}
