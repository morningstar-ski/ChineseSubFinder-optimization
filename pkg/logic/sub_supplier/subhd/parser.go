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

func parseSearchResult(pageHTML string) (string, int, error) {
	re := regexp.MustCompile(`共\s*(\d+)\s*条`)
	matched := re.FindAllStringSubmatch(pageHTML, -1)
	if matched == nil || len(matched) < 1 {
		return "", 0, common2.SubHDStep0SubCountElementNotFound
	}

	subCount, err := decode.GetNumber2int(matched[0][0])
	if err != nil {
		return "", 0, err
	}
	if subCount < 1 {
		return "", 0, nil
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(pageHTML))
	if err != nil {
		return "", 0, err
	}
	imgSelection := doc.Find("img.rounded-start")
	if _, ok := imgSelection.Attr("src"); ok == false {
		return "", 0, common2.SubHDStep0HrefIsNull
	}
	if len(imgSelection.Nodes) < 1 {
		return "", 0, common2.SubHDStep0ImgParentLessThan1
	}

	step1URL := ""
	if imgSelection.Nodes[0].Parent.Data == "a" {
		for _, attribute := range imgSelection.Nodes[0].Parent.Attr {
			if attribute.Key == "href" {
				step1URL = attribute.Val
				break
			}
		}
	} else if imgSelection.Nodes[0].Parent.Parent != nil && imgSelection.Nodes[0].Parent.Parent.Data == "a" {
		for _, attribute := range imgSelection.Nodes[0].Parent.Parent.Attr {
			if attribute.Key == "href" {
				step1URL = attribute.Val
				break
			}
		}
	}
	if step1URL == "" {
		return "", 0, common2.SubHDStep0HrefIsNull
	}

	return step1URL, subCount, nil
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
