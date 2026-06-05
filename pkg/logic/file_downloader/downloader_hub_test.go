package file_downloader

import (
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/ass"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_parser/srt"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_parser_hub"
	"github.com/sirupsen/logrus"
)

func TestResolveDownloadedExtFromZipPayload(t *testing.T) {
	log := logrus.New()
	downloader := &FileDownloader{
		Log:          log,
		SubParserHub: sub_parser_hub.NewSubParserHub(log, ass.NewParser(log), srt.NewParser(log)),
	}

	ext := downloader.resolveDownloadedExt("https://example.com/download-123.html", "", append([]byte("PK\x03\x04"), []byte("fake zip")...))
	if ext != ".zip" {
		t.Fatalf("resolveDownloadedExt() = %q, want .zip", ext)
	}
}

func TestResolveDownloadedExtFromSubtitlePayload(t *testing.T) {
	log := logrus.New()
	downloader := &FileDownloader{
		Log:          log,
		SubParserHub: sub_parser_hub.NewSubParserHub(log, ass.NewParser(log), srt.NewParser(log)),
	}

	body := []byte("1\n00:00:01,000 --> 00:00:02,000\nhello\n")
	ext := downloader.resolveDownloadedExt("https://example.com/download-123.html", "", body)
	if ext != ".srt" {
		t.Fatalf("resolveDownloadedExt() = %q, want .srt", ext)
	}
}
