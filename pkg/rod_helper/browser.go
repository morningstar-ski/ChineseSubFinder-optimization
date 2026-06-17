package rod_helper

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/local_http_proxy_server"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/random_useragent"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

const newPageNavigateRetryAttempts = 2

func NewBrowserEx(opt *BrowserOptions) (*rod.Browser, error) {
	if opt == nil || opt.Settings == nil {
		return nil, fmt.Errorf("browser options are required")
	}

	if opt.Settings.ExperimentalFunction.RemoteChromeSettings.Enable {
		return newRemoteBrowser(opt)
	}
	return newLocalBrowser(opt)
}

func HttpGetFromBrowser(browser *rod.Browser, inputURL string, timeout time.Duration, debugMode ...bool) (string, *rod.Page, error) {
	page, _, _, err := NewPageNavigate(browser, inputURL, timeout, debugMode...)
	if err != nil {
		return "", nil, err
	}
	pageString, err := page.HTML()
	if err != nil {
		if page != nil {
			_ = page.Close()
		}
		return "", nil, err
	}

	if len(debugMode) == 0 || debugMode[0] == false {
		time.Sleep(pkg.RandomSecondDuration(1, 3))
	}

	return pageString, page, nil
}

func NewPageNavigate(browser *rod.Browser, destURL string, timeout time.Duration, debugMode ...bool) (*rod.Page, int, string, error) {
	var lastErr error
	for attempt := 1; attempt <= newPageNavigateRetryAttempts; attempt++ {
		page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
		if err != nil {
			return nil, 0, "", err
		}

		page = page.Timeout(timeout)
		if err := page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
			UserAgent: random_useragent.RandomUserAgent(true),
		}); err != nil {
			_ = page.Close()
			return nil, 0, "", err
		}

		var event proto.NetworkResponseReceived
		wait := page.WaitEvent(&event)
		err = rod.Try(func() {
			page.MustNavigate(destURL)
			wait()
			page.MustWaitLoad()
		})
		err = normalizeNewPageNavigateError(err)
		if err == nil {
			page = page.CancelTimeout()
			return page, event.Response.Status, event.Response.URL, nil
		}

		lastErr = err
		_ = page.Close()
		if shouldRetryNewPageNavigate(err) == false || attempt >= newPageNavigateRetryAttempts {
			return nil, 0, "", err
		}
	}

	return nil, 0, "", lastErr
}

func normalizeNewPageNavigateError(err error) error {
	if shouldRetryNewPageNavigate(err) == false {
		return err
	}
	return fmt.Errorf("object reference chain is too long")
}

func shouldRetryNewPageNavigate(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "object reference chain is too long")
}

func newLocalBrowser(opt *BrowserOptions) (*rod.Browser, error) {
	userDataDir := filepath.Join(pkg.DefRodTmpRootFolder(), pkg.RandStringBytesMaskImprSrcSB(20))
	_ = os.MkdirAll(userDataDir, os.ModePerm)

	controlURL, err := launchLocalBrowser(opt, userDataDir)
	if err != nil {
		return nil, err
	}

	browser := rod.New().ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		return nil, err
	}

	if preload := opt.PreLoadUrl(); preload != "" {
		if _, page, err := HttpGetFromBrowser(browser, preload, 15*time.Second); err != nil {
			_ = browser.Close()
			return nil, err
		} else if page != nil {
			_ = page.Close()
		}
	}

	return browser, nil
}

func launchLocalBrowser(opt *BrowserOptions, userDataDir string) (string, error) {
	chromePath := resolveChromePath(opt.Settings)
	if chromePath == "" {
		chromePath = "chromium"
	}
	configHome := filepath.Join(userDataDir, "xdg-config")
	cacheHome := filepath.Join(userDataDir, "xdg-cache")
	_ = os.MkdirAll(configHome, os.ModePerm)
	_ = os.MkdirAll(cacheHome, os.ModePerm)

	args := []string{
		"--headless=new",
		"--no-sandbox",
		"--disable-dev-shm-usage",
		"--user-data-dir=" + userDataDir,
		"--remote-debugging-address=127.0.0.1",
		"--remote-debugging-port=0",
		"about:blank",
	}
	if proxyURL := local_http_proxy_server.GetProxyUrl(); proxyURL != "" {
		args = append(args, "--proxy-server="+proxyURL)
	}

	cmd := exec.Command(chromePath, args...)
	cmd.Env = append(os.Environ(),
		"HOME="+userDataDir,
		"XDG_CONFIG_HOME="+configHome,
		"XDG_CACHE_HOME="+cacheHome,
	)

	reader, writer := io.Pipe()
	cmd.Stdout = writer
	cmd.Stderr = writer

	if err := cmd.Start(); err != nil {
		_ = writer.Close()
		_ = reader.Close()
		return "", err
	}

	wsURLCh := make(chan string, 1)
	errCh := make(chan error, 1)

	go func() {
		defer reader.Close()
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		lastLines := make([]string, 0, 12)
		found := false
		for scanner.Scan() {
			line := scanner.Text()
			lastLines = append(lastLines, line)
			if len(lastLines) > 12 {
				lastLines = lastLines[1:]
			}
			if opt.Log != nil {
				opt.Log.Debugln("[chromium]", line)
			}
			if found {
				continue
			}
			const prefix = "DevTools listening on "
			if strings.Contains(line, prefix) {
				wsURL := strings.TrimSpace(line[strings.Index(line, prefix)+len(prefix):])
				wsURLCh <- wsURL
				found = true
			}
		}
		if scanErr := scanner.Err(); scanErr != nil && found == false {
			errCh <- fmt.Errorf("%w; chromium output: %s", scanErr, strings.Join(lastLines, " | "))
			return
		}
		if found == false {
			errCh <- fmt.Errorf("chrome exited before devtools url was available; chromium output: %s", strings.Join(lastLines, " | "))
		}
	}()

	go func() {
		_ = cmd.Wait()
		_ = writer.Close()
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	select {
	case wsURL := <-wsURLCh:
		return wsURL, nil
	case err := <-errCh:
		return "", err
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return "", fmt.Errorf("wait devtools websocket url timeout: %w", ctx.Err())
	}
}

func newRemoteBrowser(opt *BrowserOptions) (*rod.Browser, error) {
	launch := launcher.MustNewManaged(opt.Settings.ExperimentalFunction.RemoteChromeSettings.RemoteDockerURL).
		Headless(true).
		NoSandbox(true)

	if proxyURL := local_http_proxy_server.GetProxyUrl(); proxyURL != "" {
		launch = launch.Proxy(proxyURL)
	}

	if userDataDir := opt.Settings.ExperimentalFunction.RemoteChromeSettings.ReMoteUserDataDir; userDataDir != "" {
		launch = launch.UserDataDir(userDataDir)
	}

	controlURL, err := launch.Launch()
	if err != nil {
		return nil, err
	}

	browser := rod.New().Client(launch.MustClient()).ControlURL(controlURL)
	if err := browser.Connect(); err != nil {
		return nil, err
	}

	if preload := opt.PreLoadUrl(); preload != "" {
		if _, page, err := HttpGetFromBrowser(browser, preload, 15*time.Second); err != nil {
			_ = browser.Close()
			return nil, err
		} else if page != nil {
			_ = page.Close()
		}
	}

	return browser, nil
}

func resolveChromePath(cfg *settings.Settings) string {
	if cfg.ExperimentalFunction.LocalChromeSettings.Enabled && cfg.ExperimentalFunction.LocalChromeSettings.LocalChromeExeFPath != "" {
		return cfg.ExperimentalFunction.LocalChromeSettings.LocalChromeExeFPath
	}

	candidates := chromePathCandidates()

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}

func chromePathCandidates() []string {
	candidates := []string{
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/google-chrome",
		"/usr/bin/chrome",
	}

	if runtime.GOOS == "windows" {
		programFiles := []string{
			os.Getenv("ProgramFiles"),
			os.Getenv("ProgramFiles(x86)"),
			os.Getenv("LocalAppData"),
		}
		for _, root := range programFiles {
			if root == "" {
				continue
			}
			candidates = append(candidates, filepath.Join(root, "Google", "Chrome", "Application", "chrome.exe"))
		}
	}

	if runtime.GOOS == "darwin" {
		candidates = append(candidates, "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome")
	}

	return candidates
}
