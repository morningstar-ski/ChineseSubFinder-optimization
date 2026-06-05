package file_downloader

import (
	"bytes"
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
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/subtitle_best_api"
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

// Get downloads and caches a subtitle payload by URL.
func (f *FileDownloader) Get(supplierName string, topN int64, videoFileName string,
	fileDownloadUrl string, score int64, offset int64, cacheString ...string) (*supplier.SubInfo, error) {

	fileUID := cacheUID(fileDownloadUrl, cacheString...)

	found, subInfo, err := f.CacheCenter.DownloadFileGet(fileUID, f.ValidateCachedSubInfo)
	if err != nil {
		return nil, err
	}
	if found == true {
		return subInfo, nil
	}

	fileData, downloadFileName, err := pkg.DownSubtitleFile(f.Log, f.inspectSubtitlePayload, fileDownloadUrl)
	if err != nil {
		return nil, err
	}

	return f.saveDownloadedSub(fileUID, len(cacheString) > 0, supplierName, topN, videoFileName, fileDownloadUrl, score, offset, fileData, downloadFileName)
}

// GetByData stores subtitle bytes that were already fetched by the supplier.
func (f *FileDownloader) GetByData(supplierName string, topN int64, videoFileName string,
	fileDownloadUrl string, score int64, offset int64, fileData []byte, downloadFileName string, cacheString ...string) (*supplier.SubInfo, error) {

	fileUID := cacheUID(fileDownloadUrl, cacheString...)

	found, subInfo, err := f.CacheCenter.DownloadFileGet(fileUID, f.ValidateCachedSubInfo)
	if err != nil {
		return nil, err
	}
	if found == true {
		return subInfo, nil
	}

	return f.saveDownloadedSub(fileUID, len(cacheString) > 0, supplierName, topN, videoFileName, fileDownloadUrl, score, offset, fileData, downloadFileName)
}

// GetA4k downloads and caches an A4k subtitle payload.
func (f *FileDownloader) GetA4k(supplierName string, topN int64, season, eps int,
	videoFileName string, fileDownloadUrl string) (*supplier.SubInfo, error) {

	fileUID := fmt.Sprintf("%x", sha256.Sum256([]byte(fileDownloadUrl)))

	found, subInfo, err := f.CacheCenter.DownloadFileGet(fileUID, f.ValidateCachedSubInfo)
	if err != nil {
		return nil, err
	}
	if found == true {
		return subInfo, nil
	}

	fileData, downloadFileName, err := pkg.DownSubtitleFile(f.Log, f.inspectSubtitlePayload, fileDownloadUrl)
	if err != nil {
		return nil, err
	}

	inSubInfo, err := f.saveDownloadedSub(fileUID, false, supplierName, topN, videoFileName, fileDownloadUrl, 0, 0, fileData, downloadFileName)
	if err != nil {
		return nil, err
	}
	inSubInfo.Season = season
	inSubInfo.Episode = eps
	inSubInfo.GetUID()

	return inSubInfo, nil
}

// GetEx lets suppliers download content through a browser-backed callback.
func (f *FileDownloader) GetEx(supplierName string, browser *rod.Browser, subDownloadPageUrl string, topN int64, season, episode int, downFileFunc func(browser *rod.Browser, subDownloadPageUrl string, topN int64, season, episode int) (*supplier.SubInfo, error)) (*supplier.SubInfo, error) {

	fileUID := fmt.Sprintf("%x", sha256.Sum256([]byte(subDownloadPageUrl)))
	found, subInfo, err := f.CacheCenter.DownloadFileGet(fileUID, f.ValidateCachedSubInfo)
	if err != nil {
		return nil, err
	}
	if found == true {
		return subInfo, nil
	}

	subInfo, err = downFileFunc(browser, subDownloadPageUrl, topN, season, episode)
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

// GetSubtitleBest downloads and caches a subtitle.best payload keyed by subtitle hash.
func (f *FileDownloader) GetSubtitleBest(supplierName string, topN int64, season, eps int,
	title, ext, subSha256, fileDownloadUrl string) (*supplier.SubInfo, error) {

	found, subInfo, err := f.CacheCenter.DownloadFileGet(subSha256, f.ValidateCachedSubInfo)
	if err != nil {
		return nil, err
	}
	if found == true {
		return subInfo, nil
	}

	fileData, _, err := pkg.DownSubtitleFile(f.Log, f.inspectSubtitlePayload, fileDownloadUrl)
	if err != nil {
		return nil, err
	}
	_, err = f.CacheCenter.DailyDownloadCountAdd(supplierName,
		pkg.GetPublicIP(f.Log, settings.Get().AdvancedSettings.TaskQueue))
	if err != nil {
		f.Log.Warningln(supplierName, "FileDownloader.Get.DailyDownloadCountAdd", err)
	}

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
}

func cacheUID(fileDownloadURL string, cacheString ...string) string {
	if len(cacheString) > 0 {
		return cacheString[0]
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(fileDownloadURL)))
}

func (f *FileDownloader) saveDownloadedSub(fileUID string, setCustomUID bool, supplierName string, topN int64, videoFileName string,
	fileDownloadUrl string, score int64, offset int64, fileData []byte, downloadFileName string) (*supplier.SubInfo, error) {

	fileNameForValidation := downloadFileName
	if fileNameForValidation == "" {
		fileNameForValidation = filepath.Base(fileDownloadUrl)
	}
	err := pkg.ValidateSubtitleDownloadPayload(f.Log, f.inspectSubtitlePayload, fileDownloadUrl, fileNameForValidation, "", 0, fileData)
	if err != nil {
		return nil, err
	}

	_, err = f.CacheCenter.DailyDownloadCountAdd(supplierName,
		pkg.GetPublicIP(f.Log, settings.Get().AdvancedSettings.TaskQueue))
	if err != nil {
		f.Log.Warningln(supplierName, "FileDownloader.saveDownloadedSub.DailyDownloadCountAdd", err)
	}

	ext := f.resolveDownloadedExt(fileDownloadUrl, downloadFileName, fileData)
	inSubInfo := supplier.NewSubInfo(supplierName, topN, videoFileName, language.ChineseSimple, fileDownloadUrl, score, offset, ext, fileData)
	if setCustomUID == true {
		inSubInfo.SetFileUrlSha256(fileUID)
	}

	err = f.CacheCenter.DownloadFileAdd(inSubInfo)
	if err != nil {
		return nil, err
	}

	return inSubInfo, nil
}

func (f *FileDownloader) resolveDownloadedExt(fileDownloadUrl string, downloadFileName string, fileData []byte) string {
	nameExt := strings.ToLower(filepath.Ext(downloadFileName))
	if isLikelyPayloadExt(nameExt) == true {
		return nameExt
	}

	urlExt := strings.ToLower(filepath.Ext(fileDownloadUrl))
	if isLikelyPayloadExt(urlExt) == true {
		return urlExt
	}

	if bytes.HasPrefix(fileData, []byte("PK\x03\x04")) || bytes.HasPrefix(fileData, []byte("PK\x05\x06")) {
		return ".zip"
	}
	if bytes.HasPrefix(fileData, []byte("Rar!\x1A\x07")) {
		return ".rar"
	}
	if bytes.HasPrefix(fileData, []byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C}) {
		return ".7z"
	}

	if f.SubParserHub != nil {
		for _, candidateExt := range []string{".srt", ".ass", ".ssa"} {
			found, _, err := f.SubParserHub.DetermineFileTypeFromBytes(fileData, candidateExt)
			if err == nil && found == true {
				return candidateExt
			}
		}
	}

	if nameExt != "" {
		return nameExt
	}
	return urlExt
}

func isLikelyPayloadExt(ext string) bool {
	switch ext {
	case ".zip", ".tar", ".rar", ".7z", ".ass", ".ssa", ".srt":
		return true
	default:
		return false
	}
}
