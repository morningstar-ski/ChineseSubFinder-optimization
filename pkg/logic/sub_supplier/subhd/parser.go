package subhd

import (
	"fmt"
	"regexp"
	"strconv"
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

func parseSearchResults(pageHTML string) ([]searchResultItem, int, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if err != nil {
		return nil, 0, err
	}

	imgSelection := doc.Find("img.rounded-start")
	if len(imgSelection.Nodes) < 1 {
		if count, ok := parseSearchResultCount(doc); ok && count == 0 {
			return []searchResultItem{}, 0, nil
		}
		return nil, 0, common2.SubHDStep0ImgParentLessThan1
	}

	results := make([]searchResultItem, 0, len(imgSelection.Nodes))
	seen := make(map[string]struct{})
	imgSelection.Each(func(_ int, selection *goquery.Selection) {
		link := selection.ParentsFiltered("a").First()
		if link.Length() == 0 {
			link = selection.ParentFiltered("a")
		}
		if link.Length() == 0 {
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

		results = append(results, searchResultItem{
			Title: normalizeSearchResultTitle(link.Text()),
			URL:   href,
		})
	})
	if len(results) == 0 {
		return nil, 0, common2.SubHDStep0HrefIsNull
	}

	return results, len(results), nil
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

var searchResultCountPattern = regexp.MustCompile(`共\s*(\d+)\s*条`)

func parseSearchResultCount(doc *goquery.Document) (int, bool) {
	if doc == nil {
		return 0, false
	}
	headerText := strings.Join(strings.Fields(doc.Find("h4").First().Text()), " ")
	if headerText == "" {
		return 0, false
	}
	matches := searchResultCountPattern.FindStringSubmatch(headerText)
	if len(matches) != 2 {
		return 0, false
	}
	count, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, false
	}
	return count, true
}
