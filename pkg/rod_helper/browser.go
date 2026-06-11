package rod_helper

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/local_http_proxy_server"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/random_useragent"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

const minimalStealthBootstrap = `(() => {
	const defineGetter = (obj, prop, getter) => {
		try {
			Object.defineProperty(obj, prop, { get: getter, configurable: true });
		} catch (_) {}
	};

	defineGetter(Navigator.prototype, "webdriver", () => undefined);
	defineGetter(Navigator.prototype, "languages", () => ["zh-CN", "zh", "en-US", "en"]);
	defineGetter(Navigator.prototype, "plugins", () => [1, 2, 3, 4, 5]);

	if (window.chrome === undefined) {
		Object.defineProperty(window, "chrome", {
			value: { runtime: {} },
			configurable: true,
		});
	}

	if (navigator.permissions && navigator.permissions.query) {
		const originalQuery = navigator.permissions.query.bind(navigator.permissions);
		navigator.permissions.query = (parameters) => {
			if (parameters && parameters.name === "notifications") {
				return Promise.resolve({ state: Notification.permission });
			}
			return originalQuery(parameters);
		};
	}
})();`

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
	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		return nil, 0, "", err
	}

	page = page.Timeout(timeout)
	if _, err := page.EvalOnNewDocument(minimalStealthBootstrap); err != nil {
		_ = page.Close()
		return nil, 0, "", err
	}
	if err := page.SetUserAgent(&proto.NetworkSetUserAgentOverride{
		UserAgent: random_useragent.RandomUserAgent(true),
	}); err != nil {
		_ = page.Close()
		return nil, 0, "", err
	}

	var event proto.NetworkResponseReceived
	wait := page.WaitEvent(&event)
	if err := rod.Try(func() {
		page.MustNavigate(destURL)
		wait()
		page.MustWaitIdle()
	}); err != nil {
		_ = page.Close()
		return nil, 0, "", err
	}

	page = page.CancelTimeout()
	return page, event.Response.Status, event.Response.URL, nil
}

func newLocalBrowser(opt *BrowserOptions) (*rod.Browser, error) {
	launch := launcher.New().
		Headless(true).
		NoSandbox(true).
		Delete("enable-automation").
		Append("disable-blink-features", "AutomationControlled").
		UserDataDir(filepath.Join(pkg.DefRodTmpRootFolder(), pkg.RandStringBytesMaskImprSrcSB(20)))

	if runtime.GOOS == "windows" {
		launch = launch.Leakless(false)
	}

	if proxyURL := local_http_proxy_server.GetProxyUrl(); proxyURL != "" {
		launch = launch.Proxy(proxyURL)
	}

	if chromePath := resolveChromePath(opt.Settings); chromePath != "" {
		launch = launch.Bin(chromePath)
	}

	controlURL, err := launch.Launch()
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

func newRemoteBrowser(opt *BrowserOptions) (*rod.Browser, error) {
	launch := launcher.MustNewManaged(opt.Settings.ExperimentalFunction.RemoteChromeSettings.RemoteDockerURL).
		Headless(true).
		NoSandbox(true).
		Delete("enable-automation").
		Append("disable-blink-features", "AutomationControlled")

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

	candidates := []string{
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/google-chrome",
		"/usr/bin/chrome",
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return ""
}
