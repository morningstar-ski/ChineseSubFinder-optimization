package subhd

import (
	"fmt"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
)

func (s *Supplier) httpGetPage(pageURL string) (string, error) {
	httpClient, err := pkg.NewHttpClient(s.rootURL())
	if err != nil {
		return "", err
	}

	resp, err := httpClient.R().Get(pageURL)
	if err != nil {
		return "", err
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return "", fmt.Errorf("unexpected http status %d for %s", resp.StatusCode(), pageURL)
	}

	return resp.String(), nil
}
