package cache_center

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center/models"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/sirupsen/logrus"
)

func TestCacheCenter_DownloadFileAdd(t *testing.T) {
	cc, cleanup := newTestCacheCenter(t)
	defer cleanup()

	subInfo := newTestSubInfo("url123123", []byte{1, 2, 3, 4, 5})
	err := cc.DownloadFileAdd(subInfo)
	if err != nil {
		t.Fatal(err)
	}

	bok, getSubInfo, err := cc.DownloadFileGet(subInfo.GetUID())
	if err != nil {
		t.Fatal(err)
	}
	if bok == false {
		t.Fatal("bok == false")
	}

	if subInfo.FileUrl != getSubInfo.FileUrl {
		t.Fatal("subInfo.FileUrl != getSubInfo.FileUrl")
	}
}

func TestCacheCenter_DownloadFileGetExpiredCache(t *testing.T) {
	cc, cleanup := newTestCacheCenter(t)
	defer cleanup()

	subInfo := newTestSubInfo("expired-url", []byte{1, 2, 3})
	if err := cc.DownloadFileAdd(subInfo); err != nil {
		t.Fatal(err)
	}

	var df models.DownloadFileInfo
	if err := cc.db.Where("uid = ?", subInfo.GetUID()).First(&df).Error; err != nil {
		t.Fatal(err)
	}
	df.ExpirationTime = time.Now().Add(-time.Minute)
	if err := cc.db.Model(&models.DownloadFileInfo{}).Where("uid = ?", subInfo.GetUID()).Update("expiration_time", df.ExpirationTime).Error; err != nil {
		t.Fatal(err)
	}

	bok, getSubInfo, err := cc.DownloadFileGet(subInfo.GetUID())
	if err != nil {
		t.Fatal(err)
	}
	if bok {
		t.Fatal("expected expired cache miss")
	}
	if getSubInfo != nil {
		t.Fatal("expected nil sub info for expired cache")
	}
	assertCacheDeleted(t, cc, subInfo.GetUID())
}

func TestCacheCenter_DownloadFileGetInvalidCacheByValidator(t *testing.T) {
	cc, cleanup := newTestCacheCenter(t)
	defer cleanup()

	subInfo := newTestSubInfo("invalid-url", []byte{1, 2, 3})
	if err := cc.DownloadFileAdd(subInfo); err != nil {
		t.Fatal(err)
	}

	bok, getSubInfo, err := cc.DownloadFileGet(subInfo.GetUID(), func(subInfo *supplier.SubInfo) error {
		return errors.New("invalid cached payload")
	})
	if err != nil {
		t.Fatal(err)
	}
	if bok {
		t.Fatal("expected invalid cache miss")
	}
	if getSubInfo != nil {
		t.Fatal("expected nil sub info for invalid cache")
	}
	assertCacheDeleted(t, cc, subInfo.GetUID())
}

func TestCacheCenter_DownloadFileGetMissingCacheFile(t *testing.T) {
	cc, cleanup := newTestCacheCenter(t)
	defer cleanup()

	subInfo := newTestSubInfo("missing-file-url", []byte{1, 2, 3})
	if err := cc.DownloadFileAdd(subInfo); err != nil {
		t.Fatal(err)
	}

	cachePath := cacheFilePath(t, cc, subInfo.GetUID())
	if err := os.Remove(cachePath); err != nil {
		t.Fatal(err)
	}

	bok, getSubInfo, err := cc.DownloadFileGet(subInfo.GetUID())
	if err != nil {
		t.Fatal(err)
	}
	if bok {
		t.Fatal("expected missing-file cache miss")
	}
	if getSubInfo != nil {
		t.Fatal("expected nil sub info for missing-file cache")
	}
	assertCacheDeleted(t, cc, subInfo.GetUID())
}

func newTestCacheCenter(t *testing.T) (*CacheCenter, func()) {
	t.Helper()

	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())
	cacheName := "test_download_file_" + time.Now().Format("20060102150405.000000000")
	DelDb(cacheName)
	cc := newCacheCenterOrSkip(t, cacheName)

	cleanup := func() {
		cc.Close()
		DelDb(cacheName)
	}
	return cc, cleanup
}

func newCacheCenterOrSkip(t *testing.T, cacheName string) (cc *CacheCenter) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			msg := fmt.Sprint(r)
			if strings.Contains(msg, "go-sqlite3 requires cgo to work") {
				t.Skip("skip cache_center tests: sqlite driver requires cgo in this environment")
			}
			panic(r)
		}
	}()

	return NewCacheCenter(cacheName, logrus.New())
}

func newTestSubInfo(fileURL string, data []byte) *supplier.SubInfo {
	return supplier.NewSubInfo(
		"test",
		1,
		"name",
		language.ChineseSimple,
		fileURL,
		0,
		0,
		".srt",
		data,
	)
}

func cacheFilePath(t *testing.T, cc *CacheCenter, uid string) string {
	t.Helper()

	var df models.DownloadFileInfo
	if err := cc.db.Where("uid = ?", uid).First(&df).Error; err != nil {
		t.Fatal(err)
	}
	return filepath.Join(cc.downloadFileSaveRootPath, df.RelPath)
}

func assertCacheDeleted(t *testing.T, cc *CacheCenter, uid string) {
	t.Helper()

	var count int64
	if err := cc.db.Model(&models.DownloadFileInfo{}).Where("uid = ?", uid).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected cache metadata deleted, count = %d", count)
	}

	cachePath := filepath.Join(cc.downloadFileSaveRootPath, time.Now().Format("2006-01-02"), uid)
	if _, err := os.Stat(cachePath); err == nil {
		t.Fatalf("expected cache file deleted: %s", cachePath)
	}
}
