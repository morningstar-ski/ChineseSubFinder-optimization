package xunlei

import (
	"crypto/sha1"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/random_auth_key"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/unit_test_helper"
)

func TestGetList(t *testing.T) {
	t.Skip("integration test depends on external test data and xunlei availability")
	//movie1 := "X:\\电影\\The Devil All the Time (2020)\\The Devil All the Time (2020) WEBDL-1080p.mkv"
	//movie1 := "X:\\电影\\龙猫 (1988)\\龙猫 (1988) 1080p DTS.mkv"
	//movie1 := "X:\\电影\\消失爱人 (2016)\\消失爱人 (2016) 720p AAC.rmvb"
	//movie1 := "X:\\电影\\Spiral From the Book of Saw (2021)\\Spiral From the Book of Saw (2021) WEBDL-1080p.mkv"
	//movie1 := "X:\\电影\\机动战士Z高达：星之继承者 (2005)\\机动战士Z高达：星之继承者 (2005) 1080p TrueHD.mkv"
	//movie1 := "X:\\连续剧\\The Bad Batch\\Season 1\\The Bad Batch - S01E01 - Aftermath WEBDL-1080p.mkv"
	//movie1 := "X:\\连续剧\\黄石 (2018)\\Season 4\\Yellowstone (2018) - S04E05 - Under a Blanket of Red WEBDL-2160p.mkv"
	//movie1 := "X:\\动漫\\碧蓝之海 (2018)\\Season 1\\碧蓝之海 - S01E01 - [UHA-WINGS][Grand Blue][01][BDRIP 1920x1080 x264 FLACx2].mkv"
	//movie1 := "X:\\连续剧\\瑞克和莫蒂 (2013)\\Season 5\\Rick and Morty - S05E01 - Mort Dinner Rick Andre WEBDL-1080p.mkv"
	//movie1 := "X:\\电影\\手机 (2003)\\手机 (2003) 720p Cooker.rmvb"

	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())
	rootDir := unit_test_helper.GetTestDataResourceRootPath([]string{"sub_spplier"}, 5, true)
	rootDir = filepath.Join(rootDir, "xunlei")

	gVideoFPath, err := unit_test_helper.GenerateXunleiVideoFile(rootDir)
	if err != nil {
		t.Fatal(err)
	}

	defInstance()
	outList, err := xunleiInstance.getSubListFromFile(gVideoFPath)
	if err != nil {
		t.Error(err)
	}
	println(outList)

	for i, sublist := range outList {
		println(i, sublist.Name, sublist.Ext, sublist.Language.String(), sublist.Score, len(sublist.Data))
	}

	alive, _ := xunleiInstance.CheckAlive()
	if alive == false {
		t.Fatal("CheckAlive == false")
	}
}

var xunleiInstance *Supplier

func defInstance() {

	pkg.ReadCustomAuthFile(log_helper.GetLogger4Tester())

	authKey := random_auth_key.AuthKey{
		BaseKey:  pkg.BaseKey(),
		AESKey16: pkg.AESKey16(),
		AESIv16:  pkg.AESIv16(),
	}

	xunleiInstance = NewSupplier(file_downloader.NewFileDownloader(
		cache_center.NewCacheCenter("test", log_helper.GetLogger4Tester()), authKey))
}

func TestFilterAndSelectSubtitlesPrioritizesChineseWantedTypes(t *testing.T) {
	subtitles := []SublistXunLei{
		{Scid: "", Sname: "ignored.ass", Language: "简体", Surl: "https://example.com/empty"},
		{Scid: "cn-srt", Sname: "episode.zh.srt", Language: "简体", Surl: "https://example.com/cn-srt"},
		{Scid: "cn-ass", Sname: "episode.zh.ass", Language: "简体", Surl: "https://example.com/cn-ass"},
		{Scid: "en-srt", Sname: "episode.en.srt", Language: "英语", Surl: "https://example.com/en-srt"},
		{Scid: "cn-bad", Sname: "episode.zh.exe", Language: "简体", Surl: "https://example.com/cn-bad"},
	}

	got := filterAndSelectSubtitles(subtitles, 3)
	want := []SublistXunLei{
		{Scid: "cn-srt", Sname: "episode.zh.srt", Language: "简体", Surl: "https://example.com/cn-srt"},
		{Scid: "cn-ass", Sname: "episode.zh.ass", Language: "简体", Surl: "https://example.com/cn-ass"},
		{Scid: "en-srt", Sname: "episode.en.srt", Language: "英语", Surl: "https://example.com/en-srt"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected subtitles\nwant: %#v\ngot:  %#v", want, got)
	}
}

func TestGetCidReturnsExpectedHash(t *testing.T) {
	filePath := filepath.Join(t.TempDir(), "sample-video.mkv")
	content := make([]byte, 0x14000)
	for i := range content {
		content[i] = byte((i*31 + 7) % 253)
	}
	if err := os.WriteFile(filePath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	supplier := &Supplier{}
	got, err := supplier.getCid(filePath)
	if err != nil {
		t.Fatal(err)
	}

	want := expectedXunleiCID(content)
	if got != want {
		t.Fatalf("unexpected cid\nwant: %s\ngot:  %s", want, got)
	}
}

func expectedXunleiCID(content []byte) string {
	sha1Ctx := sha1.New()
	fileLength := int64(len(content))
	bufferSize := int64(0x5000)
	positions := []int64{0, int64(math.Floor(float64(fileLength) / 3)), fileLength - bufferSize}
	for _, position := range positions {
		sha1Ctx.Write(content[position : position+bufferSize])
	}
	return fmt.Sprintf("%X", sha1Ctx.Sum(nil))
}
