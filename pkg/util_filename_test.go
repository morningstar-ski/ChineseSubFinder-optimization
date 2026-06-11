package pkg

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestGetFileNameSupportsRFC5987FilenameStar(t *testing.T) {
	log := logrus.New()
	u, err := url.Parse("https://dl.subdl.com/subtitle/9n1K1x5VGn6/DWHbrmyocr")
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	resp := &http.Response{
		Header: http.Header{
			"Content-Disposition": []string{"attachment; filename*=UTF-8''Euphoria.US.S01E02.1080p.AMZN.10bit.DDP.5.1.x265%20%E7%B9%81%E9%AB%94.srt"},
		},
		Request: &http.Request{URL: u},
	}

	got := GetFileName(log, resp)
	want := "Euphoria.US.S01E02.1080p.AMZN.10bit.DDP.5.1.x265 繁體.srt"
	if got != want {
		t.Fatalf("GetFileName() = %q, want %q", got, want)
	}
}

func TestGetFileNameFallsBackToURLWhenContentDispositionIsBareAttachment(t *testing.T) {
	log := logrus.New()
	u, err := url.Parse("https://www.opensubtitles.com/download/example/subfile/Holiday.Dreaming.Chs.srt")
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	resp := &http.Response{
		Header:  http.Header{"Content-Disposition": []string{"attachment"}},
		Request: &http.Request{URL: u},
	}

	got := GetFileName(log, resp)
	want := "Holiday.Dreaming.Chs.srt"
	if got != want {
		t.Fatalf("GetFileName() = %q, want %q", got, want)
	}
}
