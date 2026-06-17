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

	cardSelection := doc.Find("div.bg-white.shadow-sm.rounded-3.mb-4")
	if len(cardSelection.Nodes) > 0 {
		results := make([]searchResultItem, 0, len(cardSelection.Nodes))
		seen := make(map[string]struct{})
		cardSelection.Each(func(_ int, card *goquery.Selection) {
			link := card.Find(".view-text a.link-dark").First()
			if link.Length() == 0 {
				link = card.Find(".float-start a.link-dark").First()
			}
			if link.Length() == 0 {
				link = card.Find("a.link-dark").First()
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

			title := normalizeSearchResultTitle(link.Text())
			if title == "" {
				return
			}

			seen[href] = struct{}{}
			results = append(results, searchResultItem{
				Title: title,
				URL:   href,
			})
		})
		if len(results) > 0 {
			return results, len(results), nil
		}
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

func parseSubtitleRows(pageHTML string, siteRoot string, detailPageURL string, isMovie bool, topic int) ([]HdListItem, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if err != nil {
		return nil, err
	}

	if single := parseSingleSubtitleDetail(doc, siteRoot, detailPageURL); len(single) > 0 {
		return single, nil
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

func parseSingleSubtitleDetail(doc *goquery.Document, siteRoot string, detailPageURL string) []HdListItem {
	if doc == nil {
		return nil
	}

	card := doc.Find("div.bg-white.shadow-sm.rounded-3.mt-3.mb-3").First()
	if card.Length() == 0 {
		return nil
	}

	downloadButton := card.Parent().Find("a.btn.btn-danger.down").First()
	if downloadButton.Length() == 0 {
		downloadButton = doc.Find("a.btn.btn-danger.down").First()
	}
	if downloadButton.Length() == 0 {
		return nil
	}

	title := strings.Join(strings.Fields(card.Find(".f16.fw-bold.mb-2").First().Text()), " ")
	if title == "" {
		return nil
	}

	insideSubType := strings.Join(strings.Fields(card.Find(".p-3.my-2.bg-light.clearfix .float-start").First().Text()), " ")
	if sub_parser_hub.IsSubTypeWanted(insideSubType) == false {
		return nil
	}

	downCount := 0
	statValues := make([]int, 0, 3)
	stats := card.Find(".p-3.my-2.bg-light.clearfix .float-end span")
	stats.Each(func(_ int, span *goquery.Selection) {
		value := strings.TrimSpace(span.Text())
		if value == "" {
			return
		}
		parsed, err := decode.GetNumber2int(value)
		if err != nil {
			return
		}
		statValues = append(statValues, parsed)
	})
	if len(statValues) >= 2 {
		downCount = statValues[1]
	} else if len(statValues) == 1 {
		downCount = statValues[0]
	}

	return []HdListItem{
		{
			Url:       pkg.AddBaseUrl(siteRoot, detailPageURL),
			BaseUrl:   siteRoot,
			Title:     title,
			DownCount: downCount,
		},
	}
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
