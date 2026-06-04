package notify_center

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/log_helper"
)

func TestNewNotifyCenter(t *testing.T) {

	requestPaths := make([]string, 0, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestPaths = append(requestPaths, r.URL.EscapedPath())
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	center := NewNotifyCenter(log_helper.GetLogger4Tester(), server.URL+"/")
	center.Add("group-a", "Info asd")
	center.Add("group-b", "hello world")
	center.Send()

	if len(requestPaths) != 2 {
		t.Fatalf("expected 2 webhook requests, got %d", len(requestPaths))
	}
	if !containsPath(requestPaths, "/group-a/Info+asd") {
		t.Fatalf("missing request path for group-a: %v", requestPaths)
	}
	if !containsPath(requestPaths, "/group-b/hello+world") {
		t.Fatalf("missing request path for group-b: %v", requestPaths)
	}
}

func containsPath(paths []string, target string) bool {
	for _, path := range paths {
		if path == target {
			return true
		}
	}
	return false
}
