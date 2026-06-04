package opensubtitles

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	subCommon "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/go-resty/resty/v2"
)

type Api struct {
	baseURL  string
	apiKey   string
	username string
	password string

	locker      sync.Mutex
	bearerToken string
}

func NewApi(baseURL, apiKey, username, password string) *Api {
	return &Api{
		baseURL:  strings.TrimRight(baseURL, "/"),
		apiKey:   apiKey,
		username: username,
		password: password,
	}
}

func (a *Api) CheckAlive(client *resty.Client) error {
	return a.Login(client)
}

func (a *Api) Login(client *resty.Client) error {
	a.locker.Lock()
	defer a.locker.Unlock()

	req := client.R().
		SetHeader("Api-Key", a.apiKey).
		SetHeader("Accept", "application/json").
		SetBody(map[string]string{
			"username": a.username,
			"password": a.password,
		})

	resp, err := req.Post(a.baseURL + subCommon.SubOpenSubtitlesLoginUrl)
	if err != nil {
		return err
	}
	if resp.StatusCode() >= http.StatusBadRequest {
		return a.httpError(resp)
	}

	var loginResp LoginResponse
	if err := json.Unmarshal(resp.Body(), &loginResp); err != nil {
		return err
	}
	if loginResp.Token == "" {
		return fmt.Errorf("opensubtitles login returned empty token")
	}

	a.bearerToken = loginResp.Token
	return nil
}

func (a *Api) SearchSubtitles(client *resty.Client, queryParams map[string]string) (*SearchResponse, error) {
	var out SearchResponse
	if err := a.doAuthenticatedJSON(client, http.MethodGet, subCommon.SubOpenSubtitlesSearchUrl, queryParams, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (a *Api) DownloadByFileID(client *resty.Client, fileID int64) (*DownloadResponse, error) {
	var out DownloadResponse
	if err := a.doAuthenticatedJSON(
		client,
		http.MethodPost,
		subCommon.SubOpenSubtitlesDownloadUrl,
		nil,
		map[string]int64{"file_id": fileID},
		&out,
	); err != nil {
		return nil, err
	}
	if out.Link == "" {
		return nil, fmt.Errorf("opensubtitles download returned empty link")
	}
	return &out, nil
}

func (a *Api) doAuthenticatedJSON(client *resty.Client, method, path string, queryParams map[string]string, body interface{}, target interface{}) error {
	if client == nil {
		return fmt.Errorf("http client is nil")
	}
	if err := a.ensureToken(client); err != nil {
		return err
	}

	resp, err := a.doRequest(client, method, path, queryParams, body, true)
	if err != nil {
		return err
	}
	if resp.StatusCode() == http.StatusUnauthorized {
		a.clearToken()
		if err := a.ensureToken(client); err != nil {
			return err
		}
		resp, err = a.doRequest(client, method, path, queryParams, body, true)
		if err != nil {
			return err
		}
	}
	if resp.StatusCode() >= http.StatusBadRequest {
		return a.httpError(resp)
	}
	if target == nil {
		return nil
	}
	return json.Unmarshal(resp.Body(), target)
}

func (a *Api) doRequest(client *resty.Client, method, path string, queryParams map[string]string, body interface{}, withAuth bool) (*resty.Response, error) {
	req := client.R().
		SetHeader("Api-Key", a.apiKey).
		SetHeader("Accept", "application/json")

	if withAuth {
		token := a.token()
		if token == "" {
			return nil, fmt.Errorf("opensubtitles bearer token is empty")
		}
		req.SetHeader("Authorization", "Bearer "+token)
	}
	if len(queryParams) > 0 {
		req.SetQueryParams(queryParams)
	}
	if body != nil {
		req.SetBody(body)
	}

	return req.Execute(method, a.baseURL+path)
}

func (a *Api) ensureToken(client *resty.Client) error {
	if a.token() != "" {
		return nil
	}
	return a.Login(client)
}

func (a *Api) token() string {
	a.locker.Lock()
	defer a.locker.Unlock()
	return a.bearerToken
}

func (a *Api) clearToken() {
	a.locker.Lock()
	defer a.locker.Unlock()
	a.bearerToken = ""
}

func (a *Api) httpError(resp *resty.Response) error {
	var apiErr ErrorResponse
	if err := json.Unmarshal(resp.Body(), &apiErr); err == nil {
		if apiErr.Message != "" {
			return fmt.Errorf("opensubtitles http %d: %s", resp.StatusCode(), apiErr.Message)
		}
		for _, item := range apiErr.Errors {
			parts := make([]string, 0, 2)
			if item.Title != "" {
				parts = append(parts, item.Title)
			}
			if item.Detail != "" {
				parts = append(parts, item.Detail)
			}
			if len(parts) > 0 {
				return fmt.Errorf("opensubtitles http %d: %s", resp.StatusCode(), strings.Join(parts, " - "))
			}
		}
	}

	body := strings.TrimSpace(resp.String())
	if body == "" {
		body = http.StatusText(resp.StatusCode())
	}
	return fmt.Errorf("opensubtitles http %d: %s", resp.StatusCode(), body)
}
