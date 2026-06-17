package pkg

import (
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/unit_test_helper"
	"github.com/sirupsen/logrus"
)

func TestCloseChrome(t *testing.T) {

	// BUG: will produce Logs under this dir
	CloseChrome(logrus.New())
}

func TestCloseChromeIgnoresNoProcessExit(t *testing.T) {
	exit1 := exitErrorWithCode(t, 1)
	exit2 := exitErrorWithCode(t, 2)

	if isCloseChromeNoProcessErr("linux", exit1) == false {
		t.Fatal("expected linux exit code 1 to be ignored")
	}
	if isCloseChromeNoProcessErr("darwin", exit1) == false {
		t.Fatal("expected darwin exit code 1 to be ignored")
	}
	if isCloseChromeNoProcessErr("windows", exit1) == true {
		t.Fatal("did not expect windows exit code 1 to be ignored")
	}
	if isCloseChromeNoProcessErr("windows", exitErrorWithCode(t, 128)) == false {
		t.Fatal("expected windows exit code 128 to be ignored")
	}
	if isCloseChromeNoProcessErr("linux", exit2) == true {
		t.Fatal("did not expect linux exit code 2 to be ignored")
	}
	if isCloseChromeNoProcessErr("linux", errors.New("boom")) == true {
		t.Fatal("did not expect plain error to be ignored")
	}
	if isCloseChromeNoProcessErr("linux", errors.New("exit status 1")) == false {
		t.Fatal("expected linux exit status text to be ignored")
	}
	if isCloseChromeNoProcessErr("windows", errors.New("exit status 128")) == false {
		t.Fatal("expected windows exit status text to be ignored")
	}
}

func exitErrorWithCode(t *testing.T, code int) error {
	t.Helper()

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd.exe", "/c", "exit", "/b", strconv.Itoa(code))
	} else {
		cmd = exec.Command("/bin/sh", "-c", "exit "+strconv.Itoa(code))
	}

	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected exit code %d command to fail", code)
	}
	return err
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

func TestResolvePublicIPFromSitesSkipsEmptyAndNonIPResponses(t *testing.T) {
	log := logrus.New()
	seen := make([]string, 0)
	fetch := func(site string) string {
		seen = append(seen, site)
		switch site {
		case "site-a":
			return ""
		case "site-b":
			return "not-an-ip"
		case "site-c":
			return " 203.0.113.8 \n"
		default:
			return ""
		}
	}

	got := resolvePublicIPFromSites(log, []string{"site-a", "site-b", "site-c", "site-d"}, fetch)
	if got != "203.0.113.8" {
		t.Fatalf("resolvePublicIPFromSites() = %q", got)
	}
	if len(seen) != 3 {
		t.Fatalf("resolvePublicIPFromSites() fetch count = %d; want 3", len(seen))
	}
}

func TestResolvePublicIPFromSitesAcceptsIPv6Responses(t *testing.T) {
	log := logrus.New()
	fetch := func(site string) string {
		switch site {
		case "site-a":
			return " 2407:cdc0:b00a:0:103:135:100:1156 \n"
		default:
			return ""
		}
	}

	got := resolvePublicIPFromSites(log, []string{"site-a", "site-b"}, fetch)
	if got != "2407:cdc0:b00a:0:103:135:100:1156" {
		t.Fatalf("resolvePublicIPFromSites() = %q", got)
	}
}

func TestExtractIPFromText(t *testing.T) {
	testCases := []struct {
		name string
		in   string
		want string
	}{
		{name: "ipv4", in: "203.0.113.8", want: "203.0.113.8"},
		{name: "ipv6", in: "2407:cdc0:b00a:0:103:135:100:1156", want: "2407:cdc0:b00a:0:103:135:100:1156"},
		{name: "whitespace", in: " \n2407:cdc0:b00a:0:103:135:100:1156\t", want: "2407:cdc0:b00a:0:103:135:100:1156"},
		{name: "prefixed text", in: "Current IP: 198.51.100.7", want: "198.51.100.7"},
		{name: "non ip", in: "not-an-ip", want: ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractIPFromText(tc.in); got != tc.want {
				t.Fatalf("extractIPFromText() = %q; want %q", got, tc.want)
			}
		})
	}
}

func TestPublicIPCacheRoundTrip(t *testing.T) {
	clearPublicIPCache()
	t.Cleanup(clearPublicIPCache)

	if got := getCachedPublicIP(); got != "" {
		t.Fatalf("getCachedPublicIP() = %q; want empty before cache", got)
	}

	cachePublicIP("198.51.100.7")
	if got := getCachedPublicIP(); got != "198.51.100.7" {
		t.Fatalf("getCachedPublicIP() = %q; want cached ip", got)
	}
}

func TestPublicIPCacheExpires(t *testing.T) {
	clearPublicIPCache()
	t.Cleanup(clearPublicIPCache)

	publicIPCacheMu.Lock()
	publicIPCacheValue = "198.51.100.9"
	publicIPCacheExpires = time.Now().Add(-time.Second)
	publicIPCacheMu.Unlock()

	if got := getCachedPublicIP(); got != "" {
		t.Fatalf("getCachedPublicIP() = %q; want empty after expiry", got)
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
