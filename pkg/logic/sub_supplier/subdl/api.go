package subdl

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/go-resty/resty/v2"
)

var errSubdlStatusFalse = errors.New("subdl search returned status=false")

type Api struct {
	apiKey  string
	rootURL string
}

func NewApi(apiKey string) *Api {
	return &Api{apiKey: apiKey, rootURL: common.SubSubDLRootUrlDef}
}

func (a *Api) SearchSubtitles(client *resty.Client, queryParams map[string]string) (*SearchResponse, error) {
	resp, err := client.R().
		SetQueryParams(queryParams).
		Get(a.searchURL())
	if err != nil {
		return nil, err
	}

	var searchResponse SearchResponse
	if err = json.Unmarshal(resp.Body(), &searchResponse); err != nil {
		return nil, err
	}
	searchResponse.populateLegacyResults()
	if searchResponse.Status == false {
		return nil, fmt.Errorf("%w", errSubdlStatusFalse)
	}

	return &searchResponse, nil
}

func (a *Api) searchURL() string {
	rootURL := a.rootURL
	if rootURL == "" {
		rootURL = common.SubSubDLRootUrlDef
	}
	return rootURL + common.SubSubDLSearchUrl
}
