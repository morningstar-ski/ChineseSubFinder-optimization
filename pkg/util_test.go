package pkg

import (
	"bytes"
	"errors"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/unit_test_helper"
	"github.com/sirupsen/logrus"
)

func TestCloseChrome(t *testing.T) {
	logger, logOutput := newCloseChromeTestLogger()

	closeChromeWithRunner(logger, "linux", func(command *exec.Cmd) ([]byte, error) {
		return []byte("no process found"), errors.New("exit status 1")
	})

	if strings.Contains(logOutput.String(), "level=warning") == true {
		t.Fatalf("expected benign close error to be ignored, got logs: %s", logOutput.String())
	}
}

func TestCloseChromeLogsRealError(t *testing.T) {
	logger, logOutput := newCloseChromeTestLogger()

	closeChromeWithRunner(logger, "linux", func(command *exec.Cmd) ([]byte, error) {
		return nil, errors.New("fork/exec /bin/sh: access denied")
	})

	if strings.Contains(logOutput.String(), "level=warning") == false {
		t.Fatalf("expected real close error to log warning, got logs: %s", logOutput.String())
	}
}

func TestIsIgnorableCloseChromeError(t *testing.T) {
	if isIgnorableCloseChromeError(errors.New("exit status 1"), []byte("No such process")) == false {
		t.Fatal("expected no-process close error to be ignored")
	}
	if isIgnorableCloseChromeError(errors.New("fork/exec /bin/sh: access denied"), nil) == true {
		t.Fatal("expected real process execution error to be preserved")
	}
}

func newCloseChromeTestLogger() (*logrus.Logger, *bytes.Buffer) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	logger.SetFormatter(&logrus.TextFormatter{DisableTimestamp: true})

	var logOutput bytes.Buffer
	logger.SetOutput(&logOutput)

	return logger, &logOutput
}

func TestFileNameIsBDMV(t *testing.T) {

	rootDir := unit_test_helper.SkipIfTestDataResourceAbsent(t, []string{"movies", "澶辨帶鐜╁ (2021)"}, 4, false)
	dbmvFPath := filepath.Join(rootDir, "CERTIFICATE", "id.bdmv")
	bok, fakeVideoFPath := FileNameIsBDMV(dbmvFPath)
	if bok == false {
		t.Fatal("FileNameIsBDMV error")
	}
	println(fakeVideoFPath)
}

func TestGetRestOfDaySec(t *testing.T) {

	rest := GetRestOfDaySec()
	println(rest)
}

func TestGetPublicIP(t *testing.T) {

	//got := GetPublicIP(log_helper.GetLogger4Tester(), settings.NewTaskQueue())
	//println("NoProxy:", got)
	//
	//sock5ProxySettings := settings.NewProxySettings(true, "socks5", local_http_proxy_server.LocalHttpProxyPort,
	//	"127.0.0.1", "10808", "", "")
	//
	//got = GetPublicIP(log_helper.GetLogger4Tester(), settings.NewTaskQueue(), sock5ProxySettings)
	//println("UseProxy socks5:", got)
	//err := sock5ProxySettings.CloseLocalHttpProxyServer()
	//if err != nil {
	//	t.Fatal(err)
	//}
	//
	//httpProxySettings := settings.NewProxySettings(true, "http", local_http_proxy_server.LocalHttpProxyPort,
	//	"127.0.0.1", "10809", "", "")
	//got = GetPublicIP(log_helper.GetLogger4Tester(), settings.NewTaskQueue(), httpProxySettings)
	//println("UseProxy http:", got)
	//err = httpProxySettings.CloseLocalHttpProxyServer()
	//if err != nil {
	//	t.Fatal(err)
	//}
}

func TestNewHttpClientProvidesCookieJar(t *testing.T) {
	client, err := NewHttpClient("https://subhd.me")
	if err != nil {
		t.Fatalf("NewHttpClient() error = %v", err)
	}
	if client.GetClient().Jar == nil {
		t.Fatal("expected NewHttpClient() to attach cookie jar")
	}

	u, err := url.Parse("https://subhd.me/a/test")
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	client.GetClient().Jar.SetCookies(u, []*http.Cookie{
		{Name: "csrftoken", Value: "1"},
	})
	got := client.GetClient().Jar.Cookies(u)
	if len(got) != 1 || got[0].Name != "csrftoken" {
		t.Fatalf("cookie jar round-trip failed, got %#v", got)
	}
}

func TestSortByModTime(t *testing.T) {
	//type args struct {
	//	fileList []string
	//}
	//tests := []struct {
	//	name string
	//	args args
	//	want []string
	//}{
	//	{name: "001", args: args{fileList: []string{
	//		"X:\\鐢靛奖\\21搴фˉ (2019)\\21搴фˉ (2019) 720p AAC.mp4",
	//		"X:\\鐢靛奖\\Texas Chainsaw Massacre (2022)\\Texas Chainsaw Massacre (2022) WEBDL-1080p.mkv",
	//		"X:\\鐢靛奖\\76 Days (2020)\\76 Days (2020) WEBDL-1080p.mkv"}},
	//		want: []string{
	//			"a",
	//			"b",
	//			"c"}},
	//	}
	//for _, tt := range tests {
	//	t.Run(tt.name, func(t *testing.T) {
	//		if got := SortByModTime(tt.args.fileList); !reflect.DeepEqual(got, tt.want) {
	//			t.Errorf("SortByModTime() = %v, want %v", got, tt.want)
	//		}
	//	})
	//}
}
