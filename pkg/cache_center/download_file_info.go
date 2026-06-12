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

	return c.downloadFileAddLocked(subInfo)
}

func (c *CacheCenter) downloadFileAddLocked(subInfo *supplier.SubInfo) error {
	if subInfo.FileUrl == "" {
		return errors.New("subInfo FileUrl is empty")
	}

	// 只支持秒或者小时为单位
	tmpTTL := downloadFileCacheTTL()

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
		recoveredSubInfo, recovered, err := c.recoverDownloadFileCacheLocked(fileUrlUID, validators...)
		if err != nil {
			return false, nil, err
		}
		if recovered {
			c.Log.Infoln("DownloadFileGet", fileUrlUID, "cache_recovered")
			return true, recoveredSubInfo, nil
		}
		c.Log.Debugln("DownloadFileGet", fileUrlUID, "cache_miss")
		return false, nil, nil
	}

	df := dfs[0]
	localFileFPath := filepath.Join(c.downloadFileSaveRootPath, df.RelPath)
	if df.ExpirationTime.Before(time.Now()) {
		subInfo, err := c.readDownloadSubInfoLocked(localFileFPath, validators...)
		if err != nil {
			c.Log.Warningln("DownloadFileGet", fileUrlUID, "cache_invalid", err)
			c.deleteDownloadFileCacheLocked(df, localFileFPath)
			return false, nil, nil
		}
		df.ExpirationTime = time.Now().Add(downloadFileCacheTTL())
		c.db.Model(&models.DownloadFileInfo{}).Where("uid = ?", df.UID).Update("expiration_time", df.ExpirationTime)
		c.Log.Infoln("DownloadFileGet", fileUrlUID, "cache_revalidated")
		return true, subInfo, nil
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

func (c *CacheCenter) recoverDownloadFileCacheLocked(fileUrlUID string, validators ...DownloadFileCacheValidator) (*supplier.SubInfo, bool, error) {
	pattern := filepath.Join(c.centerFolder, downloadFilesFolderName, "*", "*", fileUrlUID)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, false, err
	}
	if len(matches) == 0 {
		return nil, false, nil
	}

	var latestPath string
	var latestModTime time.Time
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil || info.IsDir() {
			continue
		}
		if latestPath == "" || info.ModTime().After(latestModTime) {
			latestPath = match
			latestModTime = info.ModTime()
		}
	}
	if latestPath == "" {
		return nil, false, nil
	}

	subInfo, err := c.readDownloadSubInfoLocked(latestPath, validators...)
	if err != nil {
		return nil, false, nil
	}
	if err := c.downloadFileAddLocked(subInfo); err != nil {
		return nil, false, err
	}
	return subInfo, true, nil
}

func (c *CacheCenter) readDownloadSubInfoLocked(localFileFPath string, validators ...DownloadFileCacheValidator) (*supplier.SubInfo, error) {
	if pkg.IsFile(localFileFPath) == false {
		return nil, errors.New("cache file missing")
	}

	bytes, err := os.ReadFile(localFileFPath)
	if err != nil {
		return nil, err
	}

	var subInfo supplier.SubInfo
	err = json.Unmarshal(bytes, &subInfo)
	if err != nil {
		return nil, err
	}
	if subInfo.FileUrl == "" || len(subInfo.Data) == 0 {
		return nil, errors.New("empty sub info data")
	}
	for _, validate := range validators {
		if validate == nil {
			continue
		}
		err = validate(&subInfo)
		if err != nil {
			return nil, err
		}
	}

	return &subInfo, nil
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

func downloadFileCacheTTL() time.Duration {
	cache := settings.Get().AdvancedSettings.DownloadFileCache
	if cache == nil {
		return 4320 * time.Hour
	}
	if cache.Unit == "second" {
		ttl := cache.TTL
		if ttl < 259200 || ttl > 525600 {
			ttl = 259200
		}
		return time.Duration(ttl) * time.Second
	}
	ttl := cache.TTL
	if ttl < 4320 || ttl > 8760 {
		ttl = 4320
	}
	return time.Duration(ttl) * time.Hour
}
