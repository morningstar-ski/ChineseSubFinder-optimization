package pkg

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
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

func TestExtractChromeUserDataDir(t *testing.T) {
	testCases := []struct {
		name string
		cmd  string
		want string
	}{
		{
			name: "quoted value",
			cmd:  `"C:\Program Files\Google\Chrome\Application\chrome.exe" --headless=new --user-data-dir="C:\work\rod\a1b2" about:blank`,
			want: `C:\work\rod\a1b2`,
		},
		{
			name: "unquoted value",
			cmd:  `"C:\Program Files\Google\Chrome\Application\chrome.exe" --user-data-dir=C:\work\rod\a1b2 --remote-debugging-port=0`,
			want: `C:\work\rod\a1b2`,
		},
		{
			name: "missing",
			cmd:  `"C:\Program Files\Google\Chrome\Application\chrome.exe"`,
			want: ``,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractChromeUserDataDir(tc.cmd); got != tc.want {
				t.Fatalf("extractChromeUserDataDir() = %q; want %q", got, tc.want)
			}
		})
	}
}

func TestShouldCloseOwnedChromeProcessWindows(t *testing.T) {
	rodRoot := filepath.Clean(`C:\csf\cache\rod`)
	testCases := []struct {
		name string
		cmd  string
		want bool
	}{
		{
			name: "owned root child",
			cmd:  `"C:\Program Files\Google\Chrome\Application\chrome.exe" --headless=new --user-data-dir="C:\csf\cache\rod\abc123" about:blank`,
			want: true,
		},
		{
			name: "owned exact root",
			cmd:  `"C:\Program Files\Google\Chrome\Application\chrome.exe" --user-data-dir=C:\csf\cache\rod`,
			want: true,
		},
		{
			name: "normal user chrome",
			cmd:  `"C:\Program Files\Google\Chrome\Application\chrome.exe" --type=renderer --user-data-dir="C:\Users\yang\AppData\Local\Google\Chrome\User Data"`,
			want: false,
		},
		{
			name: "no user data dir",
			cmd:  `"C:\Program Files\Google\Chrome\Application\chrome.exe"`,
			want: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldCloseOwnedChromeProcessWindows(tc.cmd, rodRoot); got != tc.want {
				t.Fatalf("shouldCloseOwnedChromeProcessWindows() = %v; want %v", got, tc.want)
			}
		})
	}
}

func TestParseWindowsChromeProcessList(t *testing.T) {
	many := `[{"ProcessId":1,"ParentProcessId":2,"CommandLine":"a"},{"ProcessId":3,"ParentProcessId":4,"CommandLine":"b"}]`
	gotMany, err := parseWindowsChromeProcessList([]byte(many))
	if err != nil {
		t.Fatal(err)
	}
	if len(gotMany) != 2 {
		t.Fatalf("parseWindowsChromeProcessList() len = %d; want 2", len(gotMany))
	}

	one := `{"ProcessId":5,"ParentProcessId":6,"CommandLine":"c"}`
	gotOne, err := parseWindowsChromeProcessList([]byte(one))
	if err != nil {
		t.Fatal(err)
	}
	if len(gotOne) != 1 || gotOne[0].ProcessId != 5 {
		t.Fatalf("parseWindowsChromeProcessList() single = %+v", gotOne)
	}

	none, err := parseWindowsChromeProcessList([]byte(`null`))
	if err != nil {
		t.Fatal(err)
	}
	if len(none) != 0 {
		t.Fatalf("parseWindowsChromeProcessList() null len = %d; want 0", len(none))
	}
}

func TestCloseChromeWindowsOnlyClosesOwnedBrowser(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows only")
	}

	chromePath := testChromePath(t)
	rodOwnedDir := filepath.Join(DefRodTmpRootFolder(), "test-owned-browser")
	externalDir := filepath.Join(t.TempDir(), "test-external-browser")
	if err := os.MkdirAll(rodOwnedDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(externalDir, os.ModePerm); err != nil {
		t.Fatal(err)
	}

	ownedCmd := exec.Command(chromePath,
		"--headless=new",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--user-data-dir="+rodOwnedDir,
		"--remote-debugging-address=127.0.0.1",
		"--remote-debugging-port=0",
		"about:blank",
	)
	if err := ownedCmd.Start(); err != nil {
		t.Fatal(err)
	}

	externalCmd := exec.Command(chromePath,
		"--headless=new",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--user-data-dir="+externalDir,
		"--remote-debugging-address=127.0.0.1",
		"--remote-debugging-port=0",
		"about:blank",
	)
	if err := externalCmd.Start(); err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		killChromeByUserDataDir(t, rodOwnedDir)
		killChromeByUserDataDir(t, externalDir)
	})

	waitForChromeUserDataDir(t, rodOwnedDir, true, 10*time.Second)
	waitForChromeUserDataDir(t, externalDir, true, 10*time.Second)

	CloseChrome(logrus.New())

	waitForChromeUserDataDir(t, rodOwnedDir, false, 10*time.Second)
	waitForChromeUserDataDir(t, externalDir, true, 10*time.Second)
}

func waitForChromeUserDataDir(t *testing.T, userDataDir string, want bool, timeout time.Duration) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		found, err := hasChromeUserDataDir(userDataDir)
		if err == nil && found == want {
			return
		}
		time.Sleep(300 * time.Millisecond)
	}

	found, err := hasChromeUserDataDir(userDataDir)
	if err != nil {
		t.Fatal(err)
	}
	t.Fatalf("hasChromeUserDataDir(%q) = %v; want %v", userDataDir, found, want)
}

func hasChromeUserDataDir(userDataDir string) (bool, error) {
	processes, err := listWindowsChromeProcesses()
	if err != nil {
		return false, err
	}
	normalizedWant := strings.ToLower(filepath.Clean(userDataDir))
	for _, proc := range processes {
		gotDir := extractChromeUserDataDir(proc.CommandLine)
		if gotDir == "" {
			continue
		}
		if strings.ToLower(filepath.Clean(gotDir)) == normalizedWant {
			return true, nil
		}
	}
	return false, nil
}

func listWindowsChromeProcesses() ([]windowsChromeProcessInfo, error) {
	script := `$ErrorActionPreference = 'Stop'
$procs = @(Get-CimInstance Win32_Process -Filter "Name='chrome.exe'" | Select-Object ProcessId,ParentProcessId,CommandLine)
$procs | ConvertTo-Json -Depth 3 -Compress`
	output, err := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script).Output()
	if err != nil {
		return nil, err
	}
	return parseWindowsChromeProcessList(output)
}

func killChromeByUserDataDir(t *testing.T, userDataDir string) {
	t.Helper()

	processes, err := listWindowsChromeProcesses()
	if err != nil {
		return
	}
	normalizedWant := strings.ToLower(filepath.Clean(userDataDir))
	for _, proc := range processes {
		gotDir := extractChromeUserDataDir(proc.CommandLine)
		if gotDir == "" {
			continue
		}
		if strings.ToLower(filepath.Clean(gotDir)) != normalizedWant {
			continue
		}
		_ = exec.Command("taskkill.exe", "/F", "/T", "/PID", strconv.Itoa(proc.ProcessId)).Run()
	}
}

func testChromePath(t *testing.T) string {
	t.Helper()

	candidates := []string{
		filepath.Join(os.Getenv("ProgramFiles"), "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(os.Getenv("ProgramFiles(x86)"), "Google", "Chrome", "Application", "chrome.exe"),
		filepath.Join(os.Getenv("LocalAppData"), "Google", "Chrome", "Application", "chrome.exe"),
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	t.Fatal("chrome.exe not found")
	return ""
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
