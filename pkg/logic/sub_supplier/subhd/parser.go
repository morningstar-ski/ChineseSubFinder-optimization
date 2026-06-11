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

	capHint := 0
	if resultCount > 0 {
		capHint = resultCount
	}
	results := make([]searchResultItem, 0, capHint)
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

	if directSubtitle, ok := parseDirectSubtitleDetail(doc); ok {
		return []HdListItem{directSubtitle}, nil
	}

	lists := make([]HdListItem, 0)
	rows := doc.Find("div.row.pt-2.mb-2, div.row.pt-2")
	if rows.Length() == 0 {
		rows = doc.Find(".pt-2")
	}
	rows.EachWithBreak(func(_ int, tr *goquery.Selection) bool {
		titleNode := tr.Find("div.view-text a.link-dark[href], a.link-dark[href]").First()
		if titleNode.Size() == 0 {
			return true
		}

		downURL, exists := titleNode.Attr("href")
		if !exists {
			return true
		}

		title := strings.TrimSpace(titleNode.Text())
		insideSubType := tr.Find(".pt-1.f11, .text-secondary").Text()
		if sub_parser_hub.IsSubTypeWanted(insideSubType) == false {
			return true
		}

		downCountText := strings.TrimSpace(tr.Find("div.col-2 div.px-3.py-2.text-end.text-secondary").First().Text())
		if downCountText == "" {
			downCountText = strings.TrimSpace(tr.Find("div.px-3").Eq(1).Text())
		}
		downCount, err := decode.GetNumber2int(downCountText)
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

func parseDirectSubtitleDetail(doc *goquery.Document) (HdListItem, bool) {
	downloadLink := doc.Find("a.btn.btn-danger.down").First()
	if downloadLink.Length() == 0 {
		return HdListItem{}, false
	}

	downURL, exists := downloadLink.Attr("href")
	if exists == false || strings.TrimSpace(downURL) == "" {
		return HdListItem{}, false
	}

	title := strings.TrimSpace(doc.Find("div.f16.fw-bold.mb-2").First().Text())
	if title == "" {
		title = strings.TrimSpace(doc.Find("h1 .link-light").First().Text())
	}
	if title == "" {
		return HdListItem{}, false
	}

	insideSubType := strings.TrimSpace(doc.Find("div.p-3.my-2.bg-light.clearfix .float-start").First().Text())
	if sub_parser_hub.IsSubTypeWanted(insideSubType) == false {
		return HdListItem{}, false
	}

	downCount := 0
	doc.Find("div.p-3.my-2.bg-light.clearfix .float-end span").EachWithBreak(func(_ int, selection *goquery.Selection) bool {
		text := strings.TrimSpace(selection.Text())
		if text == "" {
			return true
		}
		lowerText := strings.ToLower(text)
		if strings.Contains(lowerText, "k") || strings.Contains(lowerText, "m") || strings.Contains(text, "-") || strings.Contains(text, ":") {
			return true
		}
		value, err := decode.GetNumber2int(text)
		if err != nil {
			return true
		}
		downCount = value
		return false
	})

	return HdListItem{
		Url:       strings.TrimSpace(downURL),
		Title:     title,
		DownCount: downCount,
	}, true
}

type downloadGateButton struct {
	SID  string
	Href string
}

func parseDownloadGateButton(pageHTML string) (*downloadGateButton, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if err != nil {
		return nil, err
	}

	button := doc.Find(".btn.btn-danger.down").First()
	if button.Length() == 0 {
		return nil, common2.SubHDStep2ExCannotFindDownloadBtn
	}

	out := &downloadGateButton{}
	if sid, exists := button.Attr("sid"); exists {
		out.SID = strings.TrimSpace(sid)
	}
	if href, exists := button.Attr("href"); exists {
		out.Href = strings.TrimSpace(href)
	}
	if out.SID == "" && out.Href == "" {
		return nil, common2.SubHDStep2ExCannotFindDownloadBtn
	}
	return out, nil
}

func parseDownloadGateSID(pageHTML string) (string, error) {
	button, err := parseDownloadGateButton(pageHTML)
	if err != nil {
		return "", err
	}
	if button.SID == "" {
		return "", common2.SubHDStep2ExCannotFindDownloadBtn
	}
	return button.SID, nil
}

func parseDownloadPage(pageHTML string, siteRoot string) (string, error) {
	button, err := parseDownloadGateButton(pageHTML)
	if err != nil {
		return "", err
	}
	if button.Href == "" {
		return "", common2.SubHDStep2ExCannotFindDownloadBtn
	}

	return pkg.AddBaseUrl(siteRoot, button.Href), nil
}

func normalizeSearchResultTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return ""
	}
	return strings.Join(strings.Fields(title), " ")
}
