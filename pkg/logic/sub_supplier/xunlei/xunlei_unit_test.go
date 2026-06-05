package xunlei

import (
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/ass"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/srt"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_parser_hub"
	supplier2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
)

const chineseSRTContent = "1\n00:00:01,000 --> 00:00:02,000\n你好，世界\n"
const indonesianSRTContent = "1\n00:00:01,000 --> 00:00:02,000\nHarta tersembunyi di Brisbane.\n"

func newTestSupplier(t *testing.T) *Supplier {
	t.Helper()

	settings.SetConfigRootPath(t.TempDir())
	log := log_helper.GetLogger4Tester()
	return &Supplier{
		log: log,
		fileDownloader: &file_downloader.FileDownloader{
			Log:          log,
			SubParserHub: sub_parser_hub.NewSubParserHub(log, ass.NewParser(log), srt.NewParser(log)),
		},
	}
}

func TestBuildDownloadCandidatesPrefersChineseMetadataButKeepsFallback(t *testing.T) {
	t.Parallel()

	supplier := newTestSupplier(t)
	got := supplier.buildDownloadCandidates([]SublistXunLei{
		{Scid: "cn-1", Sname: "first.srt", Surl: "https://example.com/1.srt", Language: "Chinese"},
		{Scid: "dup-1", Sname: "dup.srt", Surl: "https://example.com/dup.srt", Language: "Chinese"},
		{Scid: "dup-1", Sname: "dup.srt", Surl: "https://example.com/dup.srt", Language: "English"},
		{Scid: "fallback-1", Sname: "fallback.srt", Surl: "https://example.com/fallback.srt", Language: "English"},
		{Scid: "skip-ext", Sname: "skip.txt", Surl: "https://example.com/skip.txt", Language: "Chinese"},
	})

	if len(got) != 3 {
		t.Fatalf("candidate count = %d; want 3", len(got))
	}
	if got[0].Scid != "cn-1" {
		t.Fatalf("first candidate = %q; want cn-1", got[0].Scid)
	}
	if got[1].Scid != "dup-1" {
		t.Fatalf("second candidate = %q; want dup-1", got[1].Scid)
	}
	if got[2].Scid != "fallback-1" {
		t.Fatalf("third candidate = %q; want fallback-1", got[2].Scid)
	}
}

func TestIsChineseSubtitlePayloadUsesActualContent(t *testing.T) {
	t.Parallel()

	supplier := newTestSupplier(t)

	if supplier.isChineseSubtitlePayload(&supplier2.SubInfo{
		Name: "good.srt",
		Ext:  ".srt",
		Data: []byte(chineseSRTContent),
	}) == false {
		t.Fatal("expected Chinese subtitle payload to be accepted")
	}

	if supplier.isChineseSubtitlePayload(&supplier2.SubInfo{
		Name: "bad.srt",
		Ext:  ".srt",
		Data: []byte(indonesianSRTContent),
	}) == true {
		t.Fatal("expected non-Chinese subtitle payload to be rejected")
	}
}
