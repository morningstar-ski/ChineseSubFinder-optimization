package pkg

import (
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
)

func TestNewHttpClientUsesDefaultTimeoutAndRetry(t *testing.T) {
	client, err := NewHttpClient()
	if err != nil {
		t.Fatalf("NewHttpClient() error = %v", err)
	}

	if got := client.GetClient().Timeout; got != common.HTMLTimeOut {
		t.Fatalf("NewHttpClient timeout = %v, want %v", got, common.HTMLTimeOut)
	}
	if got := client.RetryCount; got != 1 {
		t.Fatalf("NewHttpClient retry count = %d, want 1", got)
	}
}

func TestNewSubtitleDownloadHTTPClientUsesExtendedTimeoutAndRetry(t *testing.T) {
	client, err := newSubtitleDownloadHTTPClient()
	if err != nil {
		t.Fatalf("newSubtitleDownloadHTTPClient() error = %v", err)
	}

	if got := client.GetClient().Timeout; got != subtitleDownloadTimeout {
		t.Fatalf("download client timeout = %v, want %v", got, subtitleDownloadTimeout)
	}
	if got := client.RetryCount; got != subtitleDownloadRetryCount {
		t.Fatalf("download client retry count = %d, want %d", got, subtitleDownloadRetryCount)
	}
}
