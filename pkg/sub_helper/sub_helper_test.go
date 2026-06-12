package sub_helper

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/ass"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/srt"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_parser_hub"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/unit_test_helper"
)

func TestDeleteOneSeasonSubCacheFolder(t *testing.T) {
	const testSerName = "XXX"
	const needDelFolderName = "Sub_S1E0"
	testRootDir := unit_test_helper.SkipIfTestDataResourceAbsent(t, []string{"sub_helper", "org", needDelFolderName}, 4, false)
	desSerFullPath, err := pkg.GetDebugFolderByName([]string{testSerName})
	if err != nil {
		t.Fatal(err)
	}
	desSeasonFullPath, err := pkg.GetDebugFolderByName([]string{testSerName, filepath.Base(testRootDir)})
	if err != nil {
		t.Fatal(err)
	}
	err = pkg.CopyDir(testRootDir, desSeasonFullPath)
	if err != nil {
		t.Fatal(err)
	}
	err = DeleteOneSeasonSubCacheFolder(desSerFullPath)
	if err != nil {
		t.Fatal(err)
	}
	if pkg.IsDir(desSeasonFullPath) == true {
		t.Fatal("Sub_S1E0 not delete")
	}
}

func TestGetVADInfosFromSub(t *testing.T) {

	log := log_helper.GetLogger4Tester()
	// 这两个字幕是一样的，只不过是格式不同而已
	subParserHub := sub_parser_hub.NewSubParserHub(log, ass.NewParser(log), srt.NewParser(log))

	testRootDir := unit_test_helper.SkipIfTestDataResourceAbsent(t, []string{"sub_helper", "org", "R&M-S05E10"}, 4, false)

	baseSubFile := filepath.Join(testRootDir, "Rick and Morty - S05E10 - Rickmurai Jack WEBRip-1080p.chinese(简,zimuku).default.srt")
	srcSubFile := filepath.Join(testRootDir, "Rick and Morty - S05E10 - Rickmurai Jack WEBRip-1080p.chinese(简英,zimuku).ass")

	bFind, infoBase, err := subParserHub.DetermineFileTypeFromFile(baseSubFile)
	if err != nil {
		t.Fatal(err)
	}
	if bFind == false {
		t.Fatal("sub not match")
	}
	bFind, infoSrc, err := subParserHub.DetermineFileTypeFromFile(srcSubFile)
	if err != nil {
		t.Fatal(err)
	}
	if bFind == false {
		t.Fatal("sub not match")
	}

	if len(infoBase.Dialogues) != len(infoSrc.Dialogues) {
		t.Fatal(fmt.Sprintf("info Base And Src Parse Error, infoBase.DialoguesFilterEx Len = %v, infoSrc.DialoguesFilterEx Len = %v",
			len(infoBase.Dialogues), len(infoSrc.Dialogues)))
	}

	baseSubUnit, err := GetVADInfoFeatureFromSubNew(infoBase, FrontAndEndPerBase)
	if err != nil {
		t.Fatal(err)
	}
	srcSubUnit, err := GetVADInfoFeatureFromSubNew(infoSrc, FrontAndEndPerBase)
	if err != nil {
		t.Fatal(err)
	}
	if len(baseSubUnit.VADList) != len(srcSubUnit.VADList) {
		t.Fatal(fmt.Sprintf("info Base And Src Parse Error, infoBase.VADList Len = %v, infoSrc.VADList Len = %v",
			len(baseSubUnit.VADList), len(srcSubUnit.VADList)))
	}

	for i := 0; i < len(baseSubUnit.VADList); i++ {
		if baseSubUnit.VADList[i] != srcSubUnit.VADList[i] {
			println(fmt.Sprintf("base src VADList i=%v, not the same", i))
		}
	}
}

const FrontAndEndPerBase = 0

func TestOrganizeDlSubFilesFallsBackToKnownEpisodeForSingleArchiveEntry(t *testing.T) {
	log := log_helper.GetLogger4Tester()
	tmpFolderName := "sub_helper_episode_fallback_test"
	_ = pkg.ClearTmpFolderByName(tmpFolderName)
	defer func() {
		_ = pkg.ClearTmpFolderByName(tmpFolderName)
	}()

	subInfo := supplier.NewSubInfo(
		"assrt",
		0,
		"generic.zip",
		language.ChineseSimple,
		"https://example.com/generic.zip",
		0,
		0,
		".zip",
		mustBuildSubZipBytes(t, map[string]string{
			"subtitle.srt": strings.Repeat("1\n00:00:01,000 --> 00:00:02,000\nhello world subtitle line\n\n", 40),
		}),
	)
	subInfo.Season = 1
	subInfo.Episode = 2

	organized, err := OrganizeDlSubFiles(log, tmpFolderName, []supplier.SubInfo{*subInfo}, false)
	if err != nil {
		t.Fatalf("OrganizeDlSubFiles() error = %v", err)
	}

	epsKey := pkg.GetEpisodeKeyName(1, 2)
	if len(organized[epsKey]) != 1 {
		t.Fatalf("OrganizeDlSubFiles() organized[%q] = %#v; want one file", epsKey, organized[epsKey])
	}
}

func TestSearchMatchedSubFileByOneVideoIncludesSmallSubtitleFile(t *testing.T) {
	log := log_helper.GetLogger4Tester()
	dir := t.TempDir()
	videoPath := filepath.Join(dir, "Audit.Movie.2024.mp4")
	subPath := filepath.Join(dir, "Audit.Movie.2024.chinese(简,manual).default.srt")

	if err := os.WriteFile(videoPath, []byte(strings.Repeat("v", 2048)), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(subPath, []byte("1\n00:00:01,000 --> 00:00:03,000\n测试字幕上传\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := SearchMatchedSubFileByOneVideo(log, videoPath)
	if err != nil {
		t.Fatalf("SearchMatchedSubFileByOneVideo() error = %v", err)
	}
	if len(got) != 1 || got[0] != subPath {
		t.Fatalf("SearchMatchedSubFileByOneVideo() = %#v", got)
	}
}

func mustBuildSubZipBytes(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		writer, err := zw.Create(name)
		if err != nil {
			t.Fatalf("zip.Create(%q) error = %v", name, err)
		}
		if _, err = writer.Write([]byte(body)); err != nil {
			t.Fatalf("zip.Write(%q) error = %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("zip.Close() error = %v", err)
	}

	return buf.Bytes()
}
