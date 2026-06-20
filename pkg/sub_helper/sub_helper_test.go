package sub_helper

import (
	"archive/zip"
	"bytes"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/ass"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/srt"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_parser_hub"
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

func TestOrganizeDlSubFilesMaps1xPatternArchiveEntry(t *testing.T) {
	srtPayload := string(bytes.Repeat([]byte("1\n00:00:01,000 --> 00:00:02,000\nWinter is coming.\n\n"), 32))
	zipPayload, err := buildTestZip(map[string]string{
		"Game of Thrones - 1x01 - Winter is Coming.720p HDTV.cn.srt": srtPayload,
	})
	if err != nil {
		t.Fatalf("buildTestZip() error = %v", err)
	}

	subInfos := []supplier.SubInfo{
		{
			FromWhere: "tvsubtitles",
			TopN:      0,
			Name:      "download-247152.html",
			FileUrl:   "https://www.tvsubtitles.net/download-247152.html",
			Ext:       ".zip",
			Data:      zipPayload,
			Season:    1,
			Episode:   0,
		},
	}

	organized, err := OrganizeDlSubFiles(log_helper.GetLogger4Tester(), "TestOrganizeDlSubFilesMaps1xPatternArchiveEntry", subInfos, false)
	if err != nil {
		t.Fatalf("OrganizeDlSubFiles() error = %v", err)
	}

	epsKey := pkg.GetEpisodeKeyName(1, 1)
	got := organized[epsKey]
	if len(got) != 1 {
		t.Fatalf("organized[%q] len = %d, want 1", epsKey, len(got))
	}
	if filepath.Base(got[0]) != "[tvsubtitles]_0_Game of Thrones - 1x01 - Winter is Coming.720p HDTV.cn.srt" {
		t.Fatalf("organized subtitle path = %q", got[0])
	}
}

func TestOrganizeDlSubFilesFallsBackToOuterEpisodeForSingleSeriesArchiveEntry(t *testing.T) {
	srtPayload := string(bytes.Repeat([]byte("1\n00:00:01,000 --> 00:00:02,000\nForgetting Sarick Mortshall.\n\n"), 32))
	zipPayload, err := buildTestZip(map[string]string{
		"1630928156508.srt": srtPayload,
	})
	if err != nil {
		t.Fatalf("buildTestZip() error = %v", err)
	}

	subInfos := []supplier.SubInfo{
		{
			FromWhere: "subhd",
			TopN:      0,
			Name:      "1630928156508.zip",
			FileUrl:   "https://subhd.me/a/9Lbt7F",
			Ext:       ".zip",
			Data:      zipPayload,
			Season:    5,
			Episode:   9,
		},
	}

	organized, err := OrganizeDlSubFiles(log_helper.GetLogger4Tester(), "TestOrganizeDlSubFilesFallsBackToOuterEpisodeForSingleSeriesArchiveEntry", subInfos, false)
	if err != nil {
		t.Fatalf("OrganizeDlSubFiles() error = %v", err)
	}

	epsKey := pkg.GetEpisodeKeyName(5, 9)
	got := organized[epsKey]
	if len(got) != 1 {
		t.Fatalf("organized[%q] len = %d, want 1", epsKey, len(got))
	}
	if filepath.Base(got[0]) != "[subhd]_0_1630928156508.srt" {
		t.Fatalf("organized subtitle path = %q", got[0])
	}
}

func TestOrganizeDlSubFilesMapsNumericFullSeasonArchiveEntries(t *testing.T) {
	zipPayload, err := buildTestZip(map[string]string{
		"01.ass": string(bytes.Repeat([]byte("Dialogue: 0,0:00:01.00,0:00:02.00,Default,,0,0,0,,Hello boys\n"), 24)),
		"02.ass": string(bytes.Repeat([]byte("Dialogue: 0,0:00:01.00,0:00:02.00,Default,,0,0,0,,Hello again\n"), 24)),
	})
	if err != nil {
		t.Fatalf("buildTestZip() error = %v", err)
	}

	subInfos := []supplier.SubInfo{
		{
			FromWhere:    "subhd",
			TopN:         0,
			Name:         "season-pack.zip",
			FileUrl:      "https://subhd.me/a/RHptLe",
			Ext:          ".zip",
			Data:         zipPayload,
			Season:       1,
			Episode:      0,
			IsFullSeason: true,
		},
	}

	organized, err := OrganizeDlSubFiles(log_helper.GetLogger4Tester(), "TestOrganizeDlSubFilesMapsNumericFullSeasonArchiveEntries", subInfos, false)
	if err != nil {
		t.Fatalf("OrganizeDlSubFiles() error = %v", err)
	}

	if got := organized[pkg.GetEpisodeKeyName(1, 1)]; len(got) != 1 || filepath.Base(got[0]) != "[subhd]_0_01.ass" {
		t.Fatalf("organized S1E1 = %#v", got)
	}
	if got := organized[pkg.GetEpisodeKeyName(1, 2)]; len(got) != 1 || filepath.Base(got[0]) != "[subhd]_0_02.ass" {
		t.Fatalf("organized S1E2 = %#v", got)
	}
}

func TestOrganizeDlSubFilesMapsEpisodeOnlyArchiveEntries(t *testing.T) {
	zipPayload, err := buildTestZip(map[string]string{
		"E01.chs&eng.ass": string(bytes.Repeat([]byte("Dialogue: 0,0:00:01.00,0:00:02.00,Default,,0,0,0,,Hello boys\n"), 24)),
		"E02.chs&eng.ass": string(bytes.Repeat([]byte("Dialogue: 0,0:00:01.00,0:00:02.00,Default,,0,0,0,,Hello again\n"), 24)),
	})
	if err != nil {
		t.Fatalf("buildTestZip() error = %v", err)
	}

	subInfos := []supplier.SubInfo{
		{
			FromWhere: "subhd",
			TopN:      0,
			Name:      "season-pack.zip",
			FileUrl:   "https://subhd.me/a/s6uTM8",
			Ext:       ".zip",
			Data:      zipPayload,
			Season:    2,
			Episode:   0,
		},
	}

	organized, err := OrganizeDlSubFiles(log_helper.GetLogger4Tester(), "TestOrganizeDlSubFilesMapsEpisodeOnlyArchiveEntries", subInfos, false)
	if err != nil {
		t.Fatalf("OrganizeDlSubFiles() error = %v", err)
	}

	if got := organized[pkg.GetEpisodeKeyName(2, 1)]; len(got) != 1 || filepath.Base(got[0]) != "[subhd]_0_E01.chs&eng.ass" {
		t.Fatalf("organized S2E1 = %#v", got)
	}
	if got := organized[pkg.GetEpisodeKeyName(2, 2)]; len(got) != 1 || filepath.Base(got[0]) != "[subhd]_0_E02.chs&eng.ass" {
		t.Fatalf("organized S2E2 = %#v", got)
	}
}

func buildTestZip(files map[string]string) ([]byte, error) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err = entry.Write([]byte(content)); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
