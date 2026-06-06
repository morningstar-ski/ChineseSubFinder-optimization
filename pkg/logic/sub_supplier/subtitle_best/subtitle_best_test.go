package subtitle_best

import (
	"path/filepath"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/random_auth_key"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
)

var sbInstance *Supplier

func defInstance() {

	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())

	pkg.ReadCustomAuthFile(log_helper.GetLogger4Tester())

	authKey := random_auth_key.AuthKey{
		BaseKey:  pkg.BaseKey(),
		AESKey16: pkg.AESKey16(),
		AESIv16:  pkg.AESIv16(),
	}

	sbInstance = NewSupplier(file_downloader.NewFileDownloader(
		cache_center.NewCacheCenter("test", log_helper.GetLogger4Tester()), authKey))
}

func TestSupplier_CheckAlive(t *testing.T) {
	t.Skip("integration test depends on subtitle.best availability")

	defInstance()

	bok, speed := sbInstance.CheckAlive()
	println(bok, speed)
}

func TestSupplier_GetSubListFromFile4Movie(t *testing.T) {
	t.Skip("integration test depends on local media files and subtitle.best availability")

	defInstance()

	subInfos, err := sbInstance.GetSubListFromFile4Movie("X:\\电影\\Avatar (2009)\\Avatar (2009) Bluray-1080p.mp4")
	if err != nil {
		t.Fatal(err)
		return
	}
	for i, subInfo := range subInfos {
		println(i, subInfo.Name, subInfo.GetUID())
	}
}

func TestSupplier_GetSubListFromFile4Series(t *testing.T) {
	t.Skip("integration test depends on local media files and subtitle.best availability")

	defInstance()

	eps := "X:\\连续剧\\曼达洛人 (2019)\\Season 1\\曼达洛人 - S01E01 - 第1章：曼达洛人.mp4"
	subInfos, err := sbInstance.getSubListFromFile(eps, false, 1, 1)
	if err != nil {
		t.Fatal(err)
		return
	}

	for i, subInfo := range subInfos {
		println(i, subInfo.Name, subInfo.GetUID())
	}
}

func TestSupplier_OverDailyDownloadLimitBeforeFirstCheck(t *testing.T) {
	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())

	oldKey := settings.Get().SubtitleSources.SubtitleBestSettings.ApiKey
	settings.Get().SubtitleSources.SubtitleBestSettings.ApiKey = "test-key"
	defer func() {
		settings.Get().SubtitleSources.SubtitleBestSettings.ApiKey = oldKey
	}()

	supplier := &Supplier{}
	if supplier.OverDailyDownloadLimit() {
		t.Fatal("expected unknown limit state to stay usable before first health check")
	}

	supplier.limitInfoReady = true
	supplier.dailyDownloadCount = 95
	supplier.dailyDownloadLimit = 100
	if supplier.OverDailyDownloadLimit() == false {
		t.Fatal("expected known near-limit state to block downloads")
	}
}

func TestSortSubtitleBestSubtitlesPrefersMatchingMetadata(t *testing.T) {
	subtitles := []Subtitle{
		{
			SubSha256: "a",
			Title:     "My Show S01E03 720p HDTV-OTHER",
			Ext:       ".srt",
			IsMovie:   false,
			Season:    1,
			Episode:   3,
		},
		{
			SubSha256: "b",
			Title:     "My Show S01E03 1080p WEB-DL-GROUP",
			Ext:       ".srt",
			IsMovie:   false,
			Season:    1,
			Episode:   3,
		},
	}

	sortSubtitleBestSubtitles(subtitles, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false, 1, 3)
	if subtitles[0].SubSha256 != "b" {
		t.Fatalf("expected metadata-matching subtitle first, got %#v", subtitles[0])
	}
}

func TestSortSubtitleBestSubtitlesPrefersExactEpisode(t *testing.T) {
	subtitles := []Subtitle{
		{
			SubSha256: "a",
			Title:     "My Show S01E04 1080p WEB-DL-GROUP",
			Ext:       ".srt",
			IsMovie:   false,
			Season:    1,
			Episode:   4,
		},
		{
			SubSha256: "b",
			Title:     "My Show S01E03 1080p WEB-DL-GROUP",
			Ext:       ".srt",
			IsMovie:   false,
			Season:    1,
			Episode:   3,
		},
	}

	sortSubtitleBestSubtitles(subtitles, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false, 1, 3)
	if subtitles[0].SubSha256 != "b" {
		t.Fatalf("expected exact episode subtitle first, got %#v", subtitles[0])
	}
}

func TestSortSubtitleBestSubtitlesPenalizesWrongEpisodeDespiteBetterRelease(t *testing.T) {
	subtitles := []Subtitle{
		{
			SubSha256: "a",
			Title:     "My Show S01E04 1080p WEB-DL-GROUP",
			Ext:       ".srt",
			IsMovie:   false,
			Season:    1,
			Episode:   4,
		},
		{
			SubSha256: "b",
			Title:     "My Show S01E03 720p HDTV-OTHER",
			Ext:       ".srt",
			IsMovie:   false,
			Season:    1,
			Episode:   3,
		},
	}

	sortSubtitleBestSubtitles(subtitles, filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false, 1, 3)
	if subtitles[0].SubSha256 != "b" {
		t.Fatalf("expected exact episode subtitle first, got %#v", subtitles[0])
	}
}

func TestSubtitleBestCandidateMetadata(t *testing.T) {
	sub := Subtitle{
		Title:   "My Show S01E03 1080p WEB-DL-GROUP",
		Ext:     ".srt",
		Season:  1,
		Episode: 3,
	}

	metadata := subtitleBestCandidateMetadata(sub)
	if metadata.Name != sub.Title || metadata.SubtitleExt != sub.Ext || metadata.Season != 1 || metadata.Episode != 3 {
		t.Fatalf("unexpected metadata %#v", metadata)
	}
}
