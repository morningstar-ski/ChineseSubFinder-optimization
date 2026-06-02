package cache_center

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center/models"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
)

func TestCacheCenter_DownloadFileAdd(t *testing.T) {
	cc := NewCacheCenter("testFile", log_helper.GetLogger4Tester())

	subInfo := supplier.NewSubInfo(
		"test",
		1,
		"name",
		language.ChineseSimple,
		"url123123",
		0,
		0,
		"ext",
		[]byte{1, 2, 3, 4, 5},
	)
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

func TestCacheCenter_DownloadFileGetExpiredEntry(t *testing.T) {
	cacheName := "testFileExpired"
	t.Cleanup(func() {
		DelDb(cacheName)
	})

	cc := NewCacheCenter(cacheName, log_helper.GetLogger4Tester())
	uid := "expired-uid"
	relPath := filepath.Join("2026-06-02", uid)
	filePath := filepath.Join(cc.downloadFileSaveRootPath, relPath)
	if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("payload"), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	if err := cc.db.Create(&models.DownloadFileInfo{
		UID:            uid,
		RelPath:        relPath,
		ExpirationTime: time.Now().Add(-time.Hour),
	}).Error; err != nil {
		t.Fatal(err)
	}

	bok, getSubInfo, err := cc.DownloadFileGet(uid)
	if err != nil {
		t.Fatal(err)
	}
	if bok || getSubInfo != nil {
		t.Fatal("expected expired cache entry to miss")
	}

	var dfs []models.DownloadFileInfo
	cc.db.Where("uid = ?", uid).Find(&dfs)
	if len(dfs) != 0 {
		t.Fatal("expired cache entry was not removed")
	}
}
