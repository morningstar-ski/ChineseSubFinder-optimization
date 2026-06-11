package subdl

import (
	"encoding/json"
	"errors"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/go-resty/resty/v2"
)

var errSubdlStatusFalse = errors.New("subdl status false")

type Api struct {
	apiKey string
}

func NewApi(apiKey string) *Api {
	return &Api{apiKey: apiKey}
}

func (a *Api) SearchSubtitles(client *resty.Client, queryParams map[string]string) (*SearchResponse, error) {
	resp, err := client.R().
		SetQueryParams(queryParams).
		Get(common.SubSubDLRootUrlDef + common.SubSubDLSearchUrl)
	if err != nil {
		return nil, err
	}

	var searchResponse SearchResponse
	if err = json.Unmarshal(resp.Body(), &searchResponse); err != nil {
		return nil, err
	}
	if searchResponse.Status == false {
		return &SearchResponse{}, errSubdlStatusFalse
	}

	return &searchResponse, nil
}
