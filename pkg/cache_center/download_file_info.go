package cache_center

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center/models"
)

type DownloadFileCacheValidator func(subInfo *supplier.SubInfo) error

func (c *CacheCenter) DownloadFileAdd(subInfo *supplier.SubInfo) error {
	defer c.locker.Unlock()
	c.locker.Lock()

	if subInfo.FileUrl == "" {
		return errors.New("subInfo FileUrl is empty")
	}

	// 只支持秒或者小时为单位
	tmpTTL := time.Duration(settings.Get().AdvancedSettings.DownloadFileCache.TTL) * time.Second
	if settings.Get().AdvancedSettings.DownloadFileCache.Unit == "hour" {
		tmpTTL = time.Duration(settings.Get().AdvancedSettings.DownloadFileCache.TTL) * time.Hour
	} else {
		tmpTTL = time.Duration(settings.Get().AdvancedSettings.DownloadFileCache.TTL) * time.Second
	}

	b, err := json.Marshal(subInfo)
	if err != nil {
		return err
	}

	// 保存到本地文件
	todayString := time.Now().Format("2006-01-02")
	saveFPath := filepath.Join(c.downloadFileSaveRootPath, todayString, subInfo.GetUID())
	err = pkg.WriteFile(saveFPath, b)
	if err != nil {
		return err
	}
	relPath, err := filepath.Rel(c.downloadFileSaveRootPath, saveFPath)
	if err != nil {
		return err
	}

	df := models.DownloadFileInfo{
		UID:            subInfo.GetUID(),
		RelPath:        relPath,
		ExpirationTime: time.Now().Add(tmpTTL),
	}

	c.db.Save(&df)

	return nil
}

func (c *CacheCenter) DownloadFileGet(fileUrlUID string, validators ...DownloadFileCacheValidator) (bool, *supplier.SubInfo, error) {
	defer c.locker.Unlock()
	c.locker.Lock()

	var dfs []models.DownloadFileInfo
	c.db.Where("uid = ?", fileUrlUID).Find(&dfs)

	if len(dfs) == 0 {
		c.Log.Debugln("DownloadFileGet", fileUrlUID, "cache_miss")
		return false, nil, nil
	}

	df := dfs[0]
	localFileFPath := filepath.Join(c.downloadFileSaveRootPath, df.RelPath)
	if df.ExpirationTime.Before(time.Now()) {
		c.Log.Infoln("DownloadFileGet", fileUrlUID, "cache_expired")
		c.deleteDownloadFileCacheLocked(df, localFileFPath)
		return false, nil, nil
	}
	if pkg.IsFile(localFileFPath) == false {
		c.Log.Warningln("DownloadFileGet", fileUrlUID, "cache_invalid", "file missing")
		c.deleteDownloadFileCacheLocked(df, localFileFPath)
		return false, nil, nil
	}

	bytes, err := os.ReadFile(localFileFPath)
	if err != nil {
		c.Log.Warningln("DownloadFileGet", fileUrlUID, "cache_invalid", "read file", err)
		c.deleteDownloadFileCacheLocked(df, localFileFPath)
		return false, nil, nil
	}

	var subInfo supplier.SubInfo
	err = json.Unmarshal(bytes, &subInfo)
	if err != nil {
		c.Log.Warningln("DownloadFileGet", fileUrlUID, "cache_invalid", "unmarshal", err)
		c.deleteDownloadFileCacheLocked(df, localFileFPath)
		return false, nil, nil
	}
	if subInfo.FileUrl == "" || len(subInfo.Data) == 0 {
		c.Log.Warningln("DownloadFileGet", fileUrlUID, "cache_invalid", "empty sub info data")
		c.deleteDownloadFileCacheLocked(df, localFileFPath)
		return false, nil, nil
	}
	for _, validate := range validators {
		if validate == nil {
			continue
		}
		err = validate(&subInfo)
		if err != nil {
			c.Log.Warningln("DownloadFileGet", fileUrlUID, "cache_invalid", err)
			c.deleteDownloadFileCacheLocked(df, localFileFPath)
			return false, nil, nil
		}
	}

	c.Log.Debugln("DownloadFileGet", fileUrlUID, "cache_hit")
	return true, &subInfo, nil
}

func (c *CacheCenter) deleteDownloadFileCacheLocked(df models.DownloadFileInfo, localFileFPath string) {
	if localFileFPath != "" && pkg.IsFile(localFileFPath) == true {
		err := os.Remove(localFileFPath)
		if err != nil && os.IsNotExist(err) == false {
			c.Log.Warningln("deleteDownloadFileCacheLocked", df.UID, "remove file", err)
		}
	}
	c.db.Where("uid = ?", df.UID).Delete(&models.DownloadFileInfo{})
}
