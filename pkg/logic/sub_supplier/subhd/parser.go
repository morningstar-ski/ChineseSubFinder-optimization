package subhd

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/decode"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_parser_hub"
	common2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/PuerkitoBio/goquery"
)

const (
	ReasonCodeUnavailable        = "code_unavailable"
	ReasonSearchLayoutChanged    = "search_layout_changed"
	ReasonDetailLayoutChanged    = "detail_layout_changed"
	ReasonDownloadGateChanged    = "download_gate_changed"
	ReasonWaterWallFailed        = "waterwall_failed"
	ReasonCaptchaOcrFailed       = "captcha_ocr_failed"
	ReasonDownloadFailed         = "download_failed"
	ReasonBrowserRuntimeRequired = "browser_runtime_required"
	ReasonCredentialMissing      = "credential_missing"
	ReasonProbeFailed            = "probe_failed"
	ReasonDisabled               = "disabled"
)

type searchResultItem struct {
	Title string
	URL   string
}

var searchResultCountPattern = regexp.MustCompile(`共\s*(\d+)\s*条`)

func parseSearchResults(pageHTML string) ([]searchResultItem, int, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if err != nil {
		return nil, 0, err
	}

	resultCount := parseSearchResultCount(pageHTML)
	if resultCount == 0 {
		return []searchResultItem{}, 0, nil
	}

	results := make([]searchResultItem, 0, resultCount)
	seen := make(map[string]struct{})

	appendResult := func(link *goquery.Selection) {
		if link == nil || link.Length() == 0 {
			return
		}

		href, ok := link.Attr("href")
		if ok == false || strings.TrimSpace(href) == "" {
			return
		}
		href = strings.TrimSpace(href)
		if _, ok = seen[href]; ok {
			return
		}
		seen[href] = struct{}{}

		title := normalizeSearchResultTitle(link.Text())
		if title == "" {
			return
		}

		results = append(results, searchResultItem{
			Title: title,
			URL:   href,
		})
	}

	doc.Find("img.rounded-start").Each(func(_ int, selection *goquery.Selection) {
		link := selection.ParentsFiltered("a").First()
		if link.Length() == 0 {
			link = selection.ParentFiltered("a")
		}
		appendResult(link)
	})

	doc.Find("a.link-dark.align-middle[href], a[href^='/a/']").Each(func(_ int, selection *goquery.Selection) {
		link := selection
		if strings.EqualFold(goquery.NodeName(selection), "a") == false {
			link = selection.Closest("a")
		}
		appendResult(link)
	})
	if len(results) == 0 {
		if resultCount > 0 {
			return nil, resultCount, common2.SubHDStep0HrefIsNull
		}
		return []searchResultItem{}, 0, nil
	}

	return results, len(results), nil
}

func parseSearchResultCount(pageHTML string) int {
	matches := searchResultCountPattern.FindStringSubmatch(pageHTML)
	if len(matches) != 2 {
		return -1
	}
	count, err := decode.GetNumber2int(matches[1])
	if err != nil {
		return -1
	}
	return count
}

func parseSubtitleRows(pageHTML string, siteRoot string, isMovie bool, topic int) ([]HdListItem, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if err != nil {
		return nil, err
	}

	lists := make([]HdListItem, 0)
	doc.Find(".pt-2").EachWithBreak(func(_ int, tr *goquery.Selection) bool {
		titleNode := tr.Find("a.link-dark")
		if titleNode.Size() == 0 {
			return true
		}

		downURL, exists := titleNode.Eq(0).Attr("href")
		if !exists {
			return true
		}

		title := strings.TrimSpace(titleNode.Text())
		insideSubType := tr.Find(".text-secondary").Text()
		if sub_parser_hub.IsSubTypeWanted(insideSubType) == false {
			return true
		}

		downCount, err := decode.GetNumber2int(tr.Find("div.px-3").Eq(1).Text())
		if err != nil {
			return true
		}

		lists = append(lists, HdListItem{
			Url:       downURL,
			BaseUrl:   siteRoot,
			Title:     title,
			DownCount: downCount,
		})

		if isMovie && len(lists) >= topic {
			return false
		}
		return true
	})

	if len(lists) == 0 {
		return nil, fmt.Errorf("%w: no subtitle rows", common2.SubHDStep0HrefIsNull)
	}

	return lists, nil
}

func parseDownloadPage(pageHTML string, siteRoot string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if err != nil {
		return "", err
	}

	downloadURL, exists := doc.Find("a.btn.btn-danger.down").First().Attr("href")
	if exists == false || strings.TrimSpace(downloadURL) == "" {
		return "", common2.SubHDStep2ExCannotFindDownloadBtn
	}

	return pkg.AddBaseUrl(siteRoot, strings.TrimSpace(downloadURL)), nil
}

func normalizeSearchResultTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	return strings.Join(strings.Fields(title), " ")
}
