package subhd

import (
	"fmt"
	"net/http"
	neturl "net/url"
	"sort"
	"strings"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/rod_helper"
	"github.com/go-resty/resty/v2"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

const downloadGateRefreshTimeout = 8 * time.Second

func (s *Supplier) httpGetPage(pageURL string) (string, error) {
	pageHTML, _, err := s.httpGetPageWithFinalURL(pageURL)
	return pageHTML, err
}

func (s *Supplier) browserGetPage(browser *rod.Browser, pageURL string) (string, error) {
	page, _, _, err := rod_helper.NewPageNavigate(browser, pageURL, s.tt)
	if err != nil {
		return "", err
	}
	defer func() { _ = page.Close() }()

	httpClient, clientErr := s.httpClientOrNew()
	if clientErr == nil {
		if syncErr := s.syncPageCookiesToHTTPClient(page, httpClient, pageURL); syncErr != nil {
			s.log.Warningln(s.GetSupplierName(), "sync page cookies after browser page fetch failed:", syncErr)
		}
	}

	return page.HTML()
}

func (s *Supplier) browserGetSearchPage(browser *rod.Browser, pageURL string) (string, error) {
	page, _, _, err := rod_helper.NewPageNavigate(browser, pageURL, s.tt)
	if err != nil {
		return "", err
	}
	defer func() { _ = page.Close() }()

	httpClient, clientErr := s.httpClientOrNew()
	if clientErr == nil {
		if syncErr := s.syncPageCookiesToHTTPClient(page, httpClient, pageURL); syncErr != nil {
			s.log.Warningln(s.GetSupplierName(), "sync search page cookies after browser fetch failed:", syncErr)
		}
	}

	var pageHTML string
	for attempt := 1; attempt <= 5; attempt++ {
		pageHTML, err = page.HTML()
		if err != nil {
			return "", err
		}
		if parseSearchResultCount(pageHTML) >= 0 ||
			strings.Contains(pageHTML, "link-dark align-middle") ||
			strings.Contains(pageHTML, "href=\"/a/") {
			return pageHTML, nil
		}
		if attempt < 5 {
			time.Sleep(5 * time.Second)
			page.MustWaitIdle()
		}
	}

	title := strings.TrimSpace(page.MustEval(`() => document.title`).String())
	snippet := strings.TrimSpace(pageHTML)
	snippet = strings.ReplaceAll(snippet, "\r", " ")
	snippet = strings.ReplaceAll(snippet, "\n", " ")
	if len(snippet) > 240 {
		snippet = snippet[:240]
	}
	s.log.Warningln(s.GetSupplierName(), "browser search fallback unresolved", "url:", pageURL, "title:", title, "html:", snippet)

	return pageHTML, nil
}

func (s *Supplier) httpGetPageWithFinalURL(pageURL string) (string, string, error) {
	httpClient, err := s.httpClientOrNew()
	if err != nil {
		return "", "", err
	}
	return s.httpGetPageWithFinalURLByClient(httpClient, pageURL)
}

func (s *Supplier) httpGetPageWithFinalURLByClient(httpClient interface {
	R() *resty.Request
}, pageURL string) (string, string, error) {
	return s.httpGetPageWithFinalURLByClientTimeout(httpClient, pageURL, 0)
}

func (s *Supplier) httpGetPageWithFinalURLByClientTimeout(httpClient interface {
	R() *resty.Request
}, pageURL string, timeout time.Duration) (string, string, error) {
	if timeout > 0 {
		if client, ok := httpClient.(*resty.Client); ok && client != nil {
			originalTimeout := client.GetClient().Timeout
			client.SetTimeout(timeout)
			defer client.SetTimeout(originalTimeout)
		}
	}
	req := httpClient.R()
	resp, err := req.Get(pageURL)
	if err != nil {
		return "", "", err
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return "", "", fmt.Errorf("unexpected http status %d for %s", resp.StatusCode(), pageURL)
	}

	finalURL := pageURL
	if resp.RawResponse != nil && resp.RawResponse.Request != nil && resp.RawResponse.Request.URL != nil {
		finalURL = resp.RawResponse.Request.URL.String()
	}
	s.logHTTPResponseCookies("page", finalURL, resp.RawResponse)
	s.logHTTPClientCookies("page", httpClient, finalURL)

	return resp.String(), finalURL, nil
}

func sameSubHDSID(rawLeft string, rawRight string) bool {
	left, leftErr := subHDSIDFromURL(rawLeft)
	right, rightErr := subHDSIDFromURL(rawRight)
	if leftErr == nil && rightErr == nil {
		return left == right
	}

	leftURL, leftErr := neturl.Parse(rawLeft)
	rightURL, rightErr := neturl.Parse(rawRight)
	if leftErr == nil && rightErr == nil {
		return leftURL.Path == rightURL.Path
	}
	return false
}

func (s *Supplier) httpClientOrNew() (*resty.Client, error) {
	if s.httpClient != nil {
		return s.httpClient, nil
	}
	httpClient, err := s.newSessionHTTPClient()
	if err != nil {
		return nil, err
	}
	s.httpClient = httpClient
	return httpClient, nil
}

func (s *Supplier) newSessionHTTPClient() (*resty.Client, error) {
	return pkg.NewHttpClient(s.rootURL())
}

func (s *Supplier) normalizeDownloadGateURL(rawURL string) (string, error) {
	sid, err := subHDSIDFromURL(rawURL)
	if err != nil {
		return "", err
	}
	return pkg.AddBaseUrl(s.rootURL(), "/down/"+sid), nil
}

func (s *Supplier) resolveDownloadGateContextFromHTML(pageHTML string) (string, string, error) {
	refreshedGateURL, err := parseDownloadPage(pageHTML, s.rootURL())
	if err != nil {
		return "", "", err
	}
	normalizedGateURL, err := s.normalizeDownloadGateURL(refreshedGateURL)
	if err != nil {
		return "", "", err
	}
	sid, err := subHDSIDFromURL(normalizedGateURL)
	if err != nil {
		return "", "", err
	}
	return normalizedGateURL, sid, nil
}

func (s *Supplier) resolveFreshDownloadGateURL(httpClient interface {
	R() *resty.Request
}, rawURL string) (string, error) {
	return s.resolveFreshDownloadGateURLWithTimeout(httpClient, rawURL, 0)
}

func (s *Supplier) resolveFreshDownloadGateURLWithTimeout(httpClient interface {
	R() *resty.Request
}, rawURL string, timeout time.Duration) (string, error) {
	detailPageURL, err := detailPageURLFromDownloadGateURL(rawURL)
	if err != nil {
		return "", err
	}
	pageHTML, _, err := s.httpGetPageWithFinalURLByClientTimeout(httpClient, detailPageURL, timeout)
	if err != nil {
		return "", err
	}
	refreshedGateURL, _, err := s.resolveDownloadGateContextFromHTML(pageHTML)
	return refreshedGateURL, err
}

func (s *Supplier) resolveFreshDownloadGateURLByBrowser(browser *rod.Browser, httpClient *resty.Client, rawURL string) (string, error) {
	return s.resolveFreshDownloadGateURLByBrowserWithTimeout(browser, httpClient, rawURL, 0)
}

func (s *Supplier) resolveFreshDownloadGateURLByBrowserWithTimeout(browser *rod.Browser, httpClient *resty.Client, rawURL string, timeout time.Duration) (string, error) {
	detailPageURL, err := detailPageURLFromDownloadGateURL(rawURL)
	if err != nil {
		return "", err
	}
	pageTimeout := s.tt
	if timeout > 0 {
		pageTimeout = timeout
	}
	page, _, _, err := rod_helper.NewPageNavigate(browser, detailPageURL, pageTimeout)
	if err != nil {
		return "", err
	}
	defer func() { _ = page.Close() }()

	if err := s.syncPageCookiesToHTTPClient(page, httpClient, detailPageURL); err != nil {
		s.log.Warningln(s.GetSupplierName(), "sync detail page cookies failed:", err)
	}
	pageHTML, err := page.HTML()
	if err != nil {
		return "", err
	}
	refreshedGateURL, _, err := s.resolveDownloadGateContextFromHTML(pageHTML)
	return refreshedGateURL, err
}

func (s *Supplier) resolveFreshDownloadGateURLBeforeHTTPFallback(browser *rod.Browser, httpClient interface {
	R() *resty.Request
}, rawURL string) (string, error) {
	if client, ok := httpClient.(*resty.Client); ok && client != nil && browser != nil {
		refreshedGateURL, err := s.resolveFreshDownloadGateURLByBrowserWithTimeout(browser, client, rawURL, downloadGateRefreshTimeout)
		if err == nil {
			return refreshedGateURL, nil
		}
		s.log.Warningln(s.GetSupplierName(), "refresh download gate by browser before http fallback failed:", err)
	}
	return s.resolveFreshDownloadGateURLWithTimeout(httpClient, rawURL, downloadGateRefreshTimeout)
}

func (s *Supplier) refreshDownloadGateURLByError(httpClient interface {
	R() *resty.Request
}, currentGateURL string, attemptErr error) (string, error) {
	if shouldRefreshSubHDGate(attemptErr) == false {
		return "", attemptErr
	}

	return s.resolveFreshDownloadGateURL(httpClient, currentGateURL)
}

func (s *Supplier) syncPageCookiesToHTTPClient(page *rod.Page, httpClient *resty.Client, rawURL string) error {
	if page == nil || httpClient == nil {
		return nil
	}
	parsedURL, err := neturl.Parse(rawURL)
	if err != nil || parsedURL == nil {
		return err
	}
	jar := httpClient.GetClient().Jar
	if jar == nil {
		return nil
	}

	pageCookies, err := page.Cookies([]string{rawURL})
	if err != nil {
		return err
	}
	httpCookies := networkCookiesToHTTPCookies(pageCookies)
	if len(httpCookies) == 0 {
		return nil
	}
	jar.SetCookies(parsedURL, httpCookies)
	s.log.Infoln(s.GetSupplierName(), "http cookies browser_sync", "url:", rawURL, "cookie_count:", len(httpCookies), "cookies:", strings.Join(cookieNames(httpCookies), ","))
	s.logHTTPClientCookies("browser_sync", httpClient, rawURL)
	return nil
}

func shouldRefreshSubHDGate(err error) bool {
	if err == nil {
		return false
	}
	return isSubHDGateExpiredError(err) || isSubHDGateStatus500Error(err)
}

func shouldFallbackToBrowserPageFetch(err error) bool {
	if err == nil {
		return false
	}
	errText := strings.ToLower(err.Error())
	return strings.Contains(errText, "unexpected http status 522") ||
		strings.Contains(errText, "unexpected http status 500") ||
		strings.Contains(errText, "deadline exceeded") ||
		strings.Contains(errText, "client.timeout exceeded") ||
		strings.Contains(errText, "timeout")
}

func detailPageURLFromDownloadGateURL(rawURL string) (string, error) {
	parsedURL, err := neturl.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return "", err
	}
	sid, err := subHDSIDFromURL(rawURL)
	if err != nil {
		return "", err
	}
	parsedURL.Path = "/a/" + sid
	parsedURL.RawQuery = ""
	parsedURL.Fragment = ""
	return parsedURL.String(), nil
}

func (s *Supplier) logHTTPResponseCookies(stage string, rawURL string, resp *http.Response) {
	if resp == nil {
		return
	}
	cookies := resp.Cookies()
	s.log.Infoln(
		s.GetSupplierName(),
		"http cookies", stage,
		"url:", rawURL,
		"set_cookie_count:", len(cookies),
		"set_cookies:", strings.Join(cookieNames(cookies), ","),
	)
}

func (s *Supplier) logHTTPClientCookies(stage string, httpClient interface {
	R() *resty.Request
}, rawURL string) {
	client, ok := httpClient.(*resty.Client)
	if ok == false || client == nil {
		return
	}
	parsedURL, err := neturl.Parse(rawURL)
	if err != nil || parsedURL == nil {
		return
	}

	jar := client.GetClient().Jar
	if jar == nil {
		s.log.Infoln(s.GetSupplierName(), "http cookies", stage, "url:", rawURL, "jar:nil")
		return
	}

	cookies := jar.Cookies(parsedURL)
	s.log.Infoln(
		s.GetSupplierName(),
		"http cookies", stage,
		"url:", rawURL,
		"jar:", "present",
		"cookie_count:", len(cookies),
		"cookies:", strings.Join(cookieNames(cookies), ","),
	)
}

func cookieNames(cookies []*http.Cookie) []string {
	if len(cookies) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(cookies))
	out := make([]string, 0, len(cookies))
	for _, cookie := range cookies {
		if cookie == nil {
			continue
		}
		name := strings.TrimSpace(cookie.Name)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func networkCookiesToHTTPCookies(cookies []*proto.NetworkCookie) []*http.Cookie {
	if len(cookies) == 0 {
		return nil
	}
	out := make([]*http.Cookie, 0, len(cookies))
	for _, cookie := range cookies {
		if cookie == nil || strings.TrimSpace(cookie.Name) == "" {
			continue
		}
		httpCookie := &http.Cookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Path:     cookie.Path,
			Domain:   strings.TrimPrefix(cookie.Domain, "."),
			Secure:   cookie.Secure,
			HttpOnly: cookie.HTTPOnly,
		}
		if cookie.Expires > 0 {
			httpCookie.Expires = time.Unix(int64(cookie.Expires), 0)
		}
		out = append(out, httpCookie)
	}
	return out
}
