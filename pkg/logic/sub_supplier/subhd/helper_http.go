package subhd

import (
	"fmt"
	"strings"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/go-resty/resty/v2"
)

const (
	subHDPageRequestTimeout    = 30 * time.Second
	subHDPageRequestRetryCount = 0
	subHDPageRetryAttempts     = 2
	subHDPageRetryDelay        = 800 * time.Millisecond
)

func (s *Supplier) newPageHTTPClient() (*resty.Client, error) {
	httpClient, err := pkg.NewHttpClient(s.rootURL())
	if err != nil {
		return nil, err
	}
	httpClient.SetTimeout(subHDPageRequestTimeout)
	httpClient.SetRetryCount(subHDPageRequestRetryCount)
	return httpClient, nil
}

func (s *Supplier) httpGetPage(pageURL string) (string, error) {
	httpClient, err := s.newPageHTTPClient()
	if err != nil {
		return "", err
	}

	var lastErr error
	for attempt := 1; attempt <= subHDPageRetryAttempts; attempt++ {
		resp, err := httpClient.R().Get(pageURL)
		if err == nil && resp.StatusCode() >= 200 && resp.StatusCode() < 300 {
			return resp.String(), nil
		}

		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("unexpected http status %d for %s", resp.StatusCode(), pageURL)
		}

		if shouldRetryPageFetch(lastErr) == false || attempt == subHDPageRetryAttempts {
			return "", lastErr
		}

		s.log.Warningln(s.GetSupplierName(), "http page fetch transient failure, retry", attempt, pageURL, lastErr)
		time.Sleep(subHDPageRetryDelay)
	}

	return "", lastErr
}

func shouldRetryPageFetch(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "eof") ||
		strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "unexpected http status 502") ||
		strings.Contains(msg, "unexpected http status 503") ||
		strings.Contains(msg, "unexpected http status 504")
}
