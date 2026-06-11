package shooter

import (
	"crypto/md5"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/random_auth_key"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/unit_test_helper"
)

func TestComputeFileHashReturnsExpectedHash(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "sample-video.mkv")
	content := make([]byte, 0x12000)
	for i := range content {
		content[i] = byte((i*17 + 29) % 251)
	}
	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	hash, err := ComputeFileHash(filePath)
	if err != nil {
		t.Fatal(err)
	}

	expected := expectedShooterHash(content)
	if hash != expected {
		t.Fatalf("unexpected hash\nexpected: %s\nactual:   %s", expected, hash)
	}
}

func TestComputeFileHashRejectsSmallFile(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "small-video.mkv")
	if err := os.WriteFile(filePath, make([]byte, 0xEFFF), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := ComputeFileHash(filePath)
	if err != common.VideoFileIsTooSmall {
		t.Fatalf("expected %v, got %v", common.VideoFileIsTooSmall, err)
	}
}

func expectedShooterHash(content []byte) string {
	size := float64(len(content))
	samplePositions := []int{
		4 * 1024,
		int(math.Floor(size / 3 * 2)),
		int(math.Floor(size / 3)),
		len(content) - 8*1024,
	}
	hash := ""
	for _, position := range samplePositions {
		if hash != "" {
			hash += ";"
		}
		hash += fmt.Sprintf("%x", md5.Sum(content[position:position+4*1024]))
	}
	return hash
}

func TestNewSupplier(t *testing.T) {
	t.Skip("integration test depends on external test data and shooter availability")

	pkg.ReadCustomAuthFile(log_helper.GetLogger4Tester())
	authKey := random_auth_key.AuthKey{
		BaseKey:  pkg.BaseKey(),
		AESKey16: pkg.AESKey16(),
		AESIv16:  pkg.AESIv16(),
	}

	//movie1 := "X:\\电影\\The Devil All the Time (2020)\\The Devil All the Time (2020) WEBDL-1080p.mkv"
	//movie1 := "X:\\电影\\龙猫 (1988)\\龙猫 (1988) 1080p DTS.mkv"
	//movie1 := "X:\\电影\\消失爱人 (2016)\\消失爱人 (2016) 720p AAC.rmvb"
	//movie1 := "X:\\电影\\机动战士Z高达：星之继承者 (2005)\\机动战士Z高达：星之继承者 (2005) 1080p TrueHD.mkv"
	//movie1 := "X:\\连续剧\\The Bad Batch\\Season 1\\The Bad Batch - S01E01 - Aftermath WEBDL-1080p.mkv"
	//movie1 := "X:\\电影\\An Invisible Sign (2010)\\An Invisible Sign (2010) 720p AAC.mp4"
	//movie1 := "X:\\连续剧\\少年间谍 (2020)\\Season 2\\Alex Rider - S02E01 - Episode One WEBDL-1080p.mkv"
	//movie1 := "X:\\连续剧\\黄石 (2018)\\Season 4\\Yellowstone (2018) - S04E05 - Under a Blanket of Red WEBDL-2160p.mkv"
	//movie1 := "X:\\连续剧\\瑞克和莫蒂 (2013)\\Season 5\\Rick and Morty - S05E09 - Forgetting Sarick Mortshall WEBRip-1080p.mkv"

	rootDir := unit_test_helper.GetTestDataResourceRootPath([]string{"sub_spplier"}, 5, true)
	rootDir = filepath.Join(rootDir, "shooter")

	gVideoFPath, err := unit_test_helper.GenerateShooterVideoFile(rootDir)
	if err != nil {
		t.Fatal(err)
	}
	shooter := NewSupplier(file_downloader.NewFileDownloader(cache_center.NewCacheCenter("test", log_helper.GetLogger4Tester()), authKey))
	outList, err := shooter.getSubListFromFile(gVideoFPath)
	if err != nil {
		t.Error(err)
	}
	println(outList)

	for i, sublist := range outList {
		println(i, sublist.Name, sublist.Ext, sublist.Language.String(), sublist.Score, sublist.FileUrl, len(sublist.Data))
	}

	alive, _ := shooter.CheckAlive()
	if alive == false {
		t.Fatal("CheckAlive == false")
	}
}
