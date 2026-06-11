package subhd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	neturl "net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/rod_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	common2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/go-resty/resty/v2"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

var (
	captchaTextPattern  = regexp.MustCompile(`[A-Za-z0-9]{4}`)
	captchaCleanPattern = regexp.MustCompile(`[^A-Za-z0-9]+`)
	captchaNodePattern  = regexp.MustCompile(`(?is)<(?:text|tspan)\b([^>]*)>(.*?)</(?:text|tspan)>`)
	captchaAttrPattern  = regexp.MustCompile(`(?i)\b(x|y)\s*=\s*['"]?([0-9.+-]+)`)
)

var captchaOCRSymbolReplacer = strings.NewReplacer(
	"|", "1",
	"!", "1",
	"$", "S",
)

const maxCaptchaCandidateCount = 8
const maxCaptchaVerifyCandidatesPerAttempt = 3
const maxSubHDGateSubtitleCandidates = 6
const expectedCaptchaTextLength = 4

var captchaAmbiguousChars = map[rune][]rune{
	'0': {'O'},
	'O': {'0'},
	'o': {'0'},
	'1': {'I', 'l'},
	'I': {'1', 'l'},
	'l': {'1', 'I'},
	'2': {'Z'},
	'Z': {'2'},
	'5': {'S'},
	'S': {'5'},
	'6': {'G'},
	'G': {'6'},
	'8': {'B'},
	'B': {'8'},
}

var captchaGlyphAmbiguousChars = map[rune][]rune{
	'4': {'7', 'A'},
	'6': {'G'},
	'7': {'T', 'Y'},
	'8': {'B', 'g'},
	'9': {'g', 'q'},
	'A': {'4'},
	'C': {'O', 'Q', 'G'},
	'D': {'O', 'Q'},
	'G': {'6', 'C', 'Q'},
	'H': {'N', 'M'},
	'K': {'X', 'R'},
	'M': {'W', 'N', 'H'},
	'N': {'H', 'M', 'W'},
	'O': {'C', 'Q', 'D'},
	'Q': {'O', 'G', 'C', 'd', 'g'},
	'R': {'K'},
	'T': {'7', 'Y', 'f'},
	'V': {'Y', 'U'},
	'W': {'M', 'N', 'H'},
	'X': {'K', 'Y', 'V'},
	'Y': {'V', 'T', 'X', '7'},
	'a': {'A'},
	'c': {'C', 'O'},
	'd': {'Q', 'O', 'D'},
	'f': {'T', 'I', '1', 't'},
	'g': {'Q', '9', '6', '8'},
	'h': {'H', 'N', 'M'},
	'i': {'I', 'l', '1', 'j'},
	'j': {'J', 'i', '1'},
	'l': {'I', '1', 'i'},
	'm': {'M', 'W', 'N'},
	'n': {'N', 'H'},
	'o': {'O', 'Q', '0'},
	'q': {'Q', 'g', '9'},
	't': {'T', 'f', 'Y'},
	'v': {'V', 'Y'},
	'w': {'W', 'M'},
	'x': {'X', 'K'},
	'y': {'Y', 'V', 'T'},
	'z': {'Z', '2'},
}

var captchaGlyphPositionPreferredChars = map[int]map[rune][]rune{
	0: {
		'C': {'Q', 'G', 'O'},
		'c': {'C', 'O', 'Q'},
		'F': {'H'},
		'f': {'T', 'F', 'I', '1'},
		'g': {'Q', 'G', '9', '6'},
		'H': {'N', 'M', 'W'},
		'k': {'K', 'X', 'R'},
		'L': {'g', 'Q', 'G'},
		'M': {'N', 'H', 'W'},
		'm': {'M', 'W', 'N'},
		'N': {'H', 'M', 'W'},
		'n': {'N', 'H', 'M'},
		'Q': {'C', 'G', 'O'},
		'q': {'Q', '9', 'g'},
		'r': {'R', 'K'},
		'T': {'Y', 'V', '7'},
		'V': {'Y', 'T', 'U'},
		'W': {'H', 'M', 'N'},
		'w': {'W', 'M', 'N'},
		'X': {'K', 'I', 'T', 'H'},
		'x': {'X', 'K'},
		'Y': {'T', 'V', 'X', '7'},
		'y': {'Y', 'V', 'T'},
		'Z': {'2'},
		'z': {'Z', '2'},
	},
	1: {
		'1': {'I', 'l'},
		'6': {'G'},
		'7': {'T', 'Y'},
		'8': {'B', 'g'},
		'H': {'W', 'M', 'N'},
		'I': {'P', 'J', '1', 'l'},
		'J': {'I', '1'},
		'R': {'K', 'P'},
		'W': {'H', 'M', 'N'},
		'X': {'H', 'K', 'Y', 'V'},
		'Y': {'V', 'T'},
		'Z': {'2'},
		'c': {'C', 'O', 'Q'},
		'f': {'T', 'F', 'I', '1'},
		'g': {'Q', 'G', '9'},
		'i': {'I', '1', 'l', 'j'},
		'q': {'Q', 'g', '9'},
		'w': {'W', 'M'},
		'x': {'X', 'K'},
		'z': {'Z', '2'},
	},
	2: {
		'1': {'I', 'l'},
		'2': {'Z'},
		'6': {'G'},
		'7': {'2', 'Z', 'T', 'Y'},
		'W': {'N', 'H', 'M'},
		'X': {'K', 'I', 'T', 'H'},
		'Y': {'V', 'T', 'W'},
		'8': {'B', 'g'},
		'c': {'C', 'O', 'Q'},
		'f': {'F', 'T', 'I', '1'},
		'g': {'Q', 'G', '9', '6'},
		'h': {'H', 'N', 'M'},
		'i': {'I', '1', 'l', 'j'},
		'k': {'K', 'X', 'R'},
		'q': {'Q', 'g', '9'},
		'w': {'W', 'M'},
		'x': {'X', 'K'},
		'z': {'Z', '2'},
	},
	3: {
		'1': {'I', 'l'},
		'4': {'7', 'A'},
		'7': {'T', 'Y'},
		'8': {'B', 'g'},
		'S': {'5', '9'},
		'T': {'F', 'I', 'Y', '1'},
		'W': {'M', 'N', 'H'},
		'X': {'K', 'I', 'T'},
		'Y': {'V', 'T', '7'},
		'Z': {'2'},
		'b': {'B', 'P', '8'},
		'f': {'T', 'F', 'I', '1'},
		'i': {'I', '1', 'l', 'j'},
		'j': {'J', 'I', '1'},
		'k': {'K', 'X', 'R'},
		'p': {'P', 'F', 'R'},
		'w': {'W', 'M'},
		'x': {'X', 'K'},
		'y': {'Y', 'V', 'T'},
		'z': {'Z', '2'},
	},
}

type CaptchaSolver interface {
	Solve(s *Supplier, page *rod.Page, svgText string) (string, error)
}

type CaptchaCandidateSolver interface {
	SolveCandidates(s *Supplier, page *rod.Page, svgText string) ([]string, error)
}

type CaptchaBundleSolver interface {
	SolveBundle(s *Supplier, page *rod.Page, svgText string) (*captchaCandidateBundle, error)
}

type captchaCandidateBundle struct {
	Primary  []string
	Fallback []string
	Simple   bool
}

type captchaVerifyAttempt struct {
	Text       string
	Phase      string
	SourceRank int
}

func newCaptchaSolver(name string) CaptchaSolver {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "glyph_ocr", "local_ocr":
		return glyphOCRCaptchaSolver{}
	default:
		return glyphOCRCaptchaSolver{}
	}
}

func buildCaptchaCandidateBundle(s *Supplier, page *rod.Page, svgText string) (*captchaCandidateBundle, error) {
	if s.captchaSolver == nil {
		s.captchaSolver = newCaptchaSolver(settings.Get().SubtitleSources.SubHDSettings.CaptchaSolver)
	}
	if solver, ok := s.captchaSolver.(CaptchaBundleSolver); ok {
		bundle, err := solver.SolveBundle(s, page, svgText)
		if err != nil {
			return nil, err
		}
		return preferSharedSubHDCodeCandidate(bundle), nil
	}
	bundle, err := glyphOCRCaptchaSolver{}.SolveBundle(s, page, svgText)
	if err != nil {
		return nil, err
	}
	return preferSharedSubHDCodeCandidate(bundle), nil
}

func preferSharedSubHDCodeCandidate(bundle *captchaCandidateBundle) *captchaCandidateBundle {
	if bundle == nil {
		return nil
	}

	sharedCode := strings.TrimSpace(common2.SubhdCode)
	if sharedCode == "" {
		return bundle
	}

	normalized, err := normalizeCaptchaText(sharedCode)
	if err != nil || len([]rune(normalized)) != expectedCaptchaTextLength {
		return bundle
	}
	normalized = uppercaseCaptchaCandidate(normalized)

	next := &captchaCandidateBundle{
		Primary:  append([]string(nil), bundle.Primary...),
		Fallback: append([]string(nil), bundle.Fallback...),
		Simple:   true,
	}
	next.Primary = appendUniqueStringCandidates([]string{normalized}, next.Primary...)
	next.Fallback = removeCaptchaCandidateValue(next.Fallback, normalized)
	return next
}

func removeCaptchaCandidateValue(candidates []string, target string) []string {
	target = strings.TrimSpace(target)
	if target == "" || len(candidates) == 0 {
		return candidates
	}

	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == target {
			continue
		}
		out = append(out, candidate)
	}
	return out
}

type downloadGateResponse struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
	Pass    bool   `json:"pass"`
	URL     string `json:"url"`
}

type downloadGateClickProbe struct {
	Before downloadGateClickProbePageState `json:"before"`
	After  downloadGateClickProbePageState `json:"after"`
	Alerts []string                        `json:"alerts"`
	Calls  []downloadGateClickProbeCall    `json:"calls"`
}

type downloadGateClickProbePageState struct {
	Href           string `json:"href"`
	Referrer       string `json:"referrer"`
	Webdriver      string `json:"webdriver"`
	UserAgent      string `json:"ua"`
	ReadyState     string `json:"readyState"`
	SID            string `json:"sidAttr"`
	ButtonHref     string `json:"btnHref"`
	ButtonOnclick  string `json:"btnOnclick"`
	ButtonText     string `json:"btnText"`
	ButtonDisabled bool   `json:"btnDisabled"`
	CapText        string `json:"capText"`
	SVGVisible     bool   `json:"svgVisible"`
	SVGLen         int    `json:"svgLen"`
	ScriptCount    int    `json:"scriptCount"`
	JQueryPresent  bool   `json:"jqueryPresent"`
	JQueryClicks   int    `json:"jqueryClickHandlers"`
	HasCapInput    bool   `json:"hasCapInput"`
	HasSVGCap      bool   `json:"hasSvgCap"`
	HasAPIDownText bool   `json:"hasApiDownText"`
}

type downloadGateClickProbeCall struct {
	URL          string `json:"url"`
	Method       string `json:"method"`
	Headers      string `json:"headers"`
	Body         string `json:"body"`
	Status       int    `json:"status"`
	OK           bool   `json:"ok"`
	ResponseURL  string `json:"responseUrl"`
	ResponseText string `json:"responseText"`
	Error        string `json:"error"`
}

type fetchedDownload struct {
	StatusCode         int    `json:"statusCode"`
	DataURL            string `json:"dataUrl"`
	ContentType        string `json:"contentType"`
	ContentDisposition string `json:"contentDisposition"`
	FinalURL           string `json:"finalUrl"`
}

func (s *Supplier) downloadSubFileViaGate(browser *rod.Browser, httpClient interface {
	R() *resty.Request
}, downloadPageURL string) (*supplier.SubInfo, error) {
	currentGateURL := downloadPageURL
	sid, err := subHDSIDFromURL(currentGateURL)
	if err != nil {
		return nil, wrapReason(ReasonDownloadGateChanged, err)
	}

	var lastErr error
	deadGateLoopCount := 0
	for attempt := 1; attempt <= s.maxCaptchaAttempts(); attempt++ {
		s.log.Infoln(s.GetSupplierName(), "download gate page ready sid:", sid, "url:", currentGateURL, "attempt:", attempt)

		var gateProbeErr error
		page, pageErr := s.openDownloadGatePage(browser, currentGateURL)
		if pageErr == nil {
			s.syncPageCookiesToHTTPClientIfPossible(page, httpClient, currentGateURL)
			subInfo, tryErr := s.tryDownloadFromPage(page, sid, currentGateURL)
			_ = page.Close()
			if tryErr == nil {
				return subInfo, nil
			}
			effectiveErr := tryErr
			if shouldRetryDetailPageContextAfterPageError(tryErr) {
				subInfo, detailErr := s.tryDownloadFromDetailPageContext(browser, sid, currentGateURL)
				if detailErr == nil {
					return subInfo, nil
				}
				effectiveErr = detailErr
				gateProbeErr = detailErr
				s.log.Warningln(s.GetSupplierName(), "download gate detail page context failed:", detailErr)
			}
			s.logDownloadGateClickProbeByBrowser(browser, currentGateURL, sid, attempt, effectiveErr)
			lastErr = effectiveErr
			if shouldUseHTTPFallbackAfterPageError(effectiveErr) == false {
				s.log.Warningln(s.GetSupplierName(), "download gate browser flow failed, refresh gate before http fallback:", effectiveErr)
				if refreshedGateURL, refreshErr := s.refreshDownloadGateURLByError(httpClient, currentGateURL, effectiveErr); refreshErr == nil && refreshedGateURL != "" {
					currentGateURL = refreshedGateURL
					sid, err = subHDSIDFromURL(currentGateURL)
					if err != nil {
						return nil, wrapReason(ReasonDownloadGateChanged, err)
					}
					s.log.Infoln(s.GetSupplierName(), "download gate refreshed after page failure", "attempt:", attempt, "gate:", currentGateURL)
				}
				continue
			}
			if shouldRefreshGateBeforeHTTPFallbackAfterPageError(effectiveErr) {
				if refreshedGateURL, refreshErr := s.resolveFreshDownloadGateURLBeforeHTTPFallback(browser, httpClient, currentGateURL); refreshErr == nil && refreshedGateURL != "" {
					currentGateURL = refreshedGateURL
					sid, err = subHDSIDFromURL(currentGateURL)
					if err != nil {
						return nil, wrapReason(ReasonDownloadGateChanged, err)
					}
					s.log.Infoln(s.GetSupplierName(), "download gate refreshed before http fallback", "attempt:", attempt, "gate:", currentGateURL)
				}
			}
			s.log.Warningln(s.GetSupplierName(), "download gate browser flow failed, fallback to http:", effectiveErr)
		} else {
			lastErr = wrapReason(ReasonProbeFailed, pageErr)
			gateProbeErr = lastErr
			s.log.Warningln(s.GetSupplierName(), "download gate browser open failed, fallback to http:", pageErr)
		}

		subInfo, attemptErr := s.tryDownloadFromHTTP(browser, httpClient, sid, currentGateURL, attempt)
		if attemptErr == nil {
			return subInfo, nil
		}

		lastErr = attemptErr
		s.log.Warningln(s.GetSupplierName(), "captcha attempt", attempt, "failed:", attemptErr)
		if refreshedGateURL, refreshErr := s.refreshDownloadGateURLByError(httpClient, currentGateURL, attemptErr); refreshErr == nil && refreshedGateURL != "" {
			deadGateLoopCount = nextRepeatedDeadGateLoopCount(deadGateLoopCount, gateProbeErr, attemptErr, currentGateURL, refreshedGateURL)
			currentGateURL = refreshedGateURL
			sid, err = subHDSIDFromURL(currentGateURL)
			if err != nil {
				return nil, wrapReason(ReasonDownloadGateChanged, err)
			}
			s.log.Infoln(s.GetSupplierName(), "download gate refreshed after failure", "attempt:", attempt, "gate:", currentGateURL)
			if deadGateLoopCount >= 2 {
				lastErr = wrapReason(ReasonDownloadGateChanged, fmt.Errorf("subhd dead gate loop detected sid %s: %w", sid, attemptErr))
				s.log.Warningln(s.GetSupplierName(), "download gate dead loop detected, stop retrying", "sid:", sid, "attempt:", attempt)
				break
			}
			continue
		}
		deadGateLoopCount = 0
		if shouldRetrySubHDGateAttempt(attemptErr) == false {
			break
		}
	}

	if lastErr == nil {
		lastErr = wrapReason(ReasonDownloadFailed, fmt.Errorf("subhd captcha attempts exhausted"))
	}

	return nil, lastErr
}

func shouldUseHTTPFallbackAfterPageError(err error) bool {
	switch reasonOf(err) {
	case ReasonProbeFailed:
		return true
	case ReasonCaptchaOcrFailed:
		return true
	case ReasonDownloadGateChanged:
		return false
	default:
		return shouldRefreshSubHDGate(err) == false
	}
}

func shouldRefreshGateBeforeHTTPFallbackAfterPageError(err error) bool {
	return reasonOf(err) == ReasonCaptchaOcrFailed
}

func nextRepeatedDeadGateLoopCount(current int, gateProbeErr error, attemptErr error, previousGateURL string, refreshedGateURL string) int {
	if sameSubHDSID(previousGateURL, refreshedGateURL) == false {
		return 0
	}
	if isSubHDGateStatus500Error(gateProbeErr) == false {
		return 0
	}
	if isSubHDGateExpiredError(attemptErr) == false {
		return 0
	}
	return current + 1
}

func isSubHDDeadGateLoopError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "subhd dead gate loop detected")
}

func shouldRetryDetailPageContextAfterPageError(err error) bool {
	return reasonOf(err) == ReasonProbeFailed
}

func (s *Supplier) openDownloadGatePage(browser *rod.Browser, downloadPageURL string) (*rod.Page, error) {
	page, _, _, err := rod_helper.NewPageNavigate(browser, downloadPageURL, s.tt)
	if err != nil {
		return nil, err
	}
	return page, nil
}

func (s *Supplier) syncPageCookiesToHTTPClientIfPossible(page *rod.Page, httpClient interface {
	R() *resty.Request
}, rawURL string) {
	client, ok := httpClient.(*resty.Client)
	if ok == false || client == nil {
		return
	}
	if err := s.syncPageCookiesToHTTPClient(page, client, rawURL); err != nil {
		s.log.Warningln(s.GetSupplierName(), "sync gate page cookies failed:", err)
	}
}

func (s *Supplier) tryDownloadFromDetailPageContext(browser *rod.Browser, sid string, sourcePageURL string) (*supplier.SubInfo, error) {
	detailPageURL, err := detailPageURLFromDownloadGateURL(sourcePageURL)
	if err != nil {
		return nil, wrapReason(ReasonDownloadGateChanged, err)
	}
	s.log.Infoln(s.GetSupplierName(), "download gate retry detail context sid:", sid, "url:", detailPageURL)

	page, err := s.openDownloadGatePage(browser, detailPageURL)
	if err != nil {
		return nil, wrapReason(ReasonProbeFailed, err)
	}
	defer func() {
		_ = page.Close()
	}()

	pageHTML, err := page.HTML()
	if err != nil {
		s.log.Warningln(s.GetSupplierName(), "download gate detail context html failed:", err)
	} else {
		refreshedGateURL, refreshedSID, parseErr := s.resolveDownloadGateContextFromHTML(pageHTML)
		if parseErr != nil {
			s.log.Warningln(s.GetSupplierName(), "download gate detail context parse failed:", parseErr)
		} else {
			sid = refreshedSID
			sourcePageURL = refreshedGateURL
			s.log.Infoln(s.GetSupplierName(), "download gate detail context refreshed sid:", sid, "gate:", sourcePageURL)
		}
	}

	return s.tryDownloadFromPage(page, sid, sourcePageURL)
}

func (s *Supplier) tryDownloadFromHTTP(browser *rod.Browser, httpClient interface {
	R() *resty.Request
}, sid string, sourcePageURL string, attempt int) (*supplier.SubInfo, error) {
	firstResp, err := s.fetchDownloadGateResponseHTTP(httpClient, sid, "")
	if err != nil {
		return nil, wrapReason(ReasonProbeFailed, err)
	}
	s.logSubHDGateResponse("initial", sid, attempt, "", firstResp)
	if firstResp.Success && firstResp.Pass && firstResp.URL != "" {
		return s.subInfoFromDownloadURLHTTP(httpClient, firstResp.URL, sourcePageURL)
	}
	if firstResp.Success == false {
		return nil, wrapReason(ReasonDownloadGateChanged, fmt.Errorf(strings.TrimSpace(firstResp.Msg)))
	}
	if strings.TrimSpace(firstResp.Msg) == "" {
		return nil, wrapReason(ReasonDownloadGateChanged, fmt.Errorf("subhd captcha svg is empty"))
	}

	page, err := browser.Page(proto.TargetCreateTarget{URL: ""})
	if err != nil {
		return nil, wrapReason(ReasonProbeFailed, err)
	}
	page = page.Timeout(s.tt)
	if err := page.Navigate("about:blank"); err != nil {
		_ = page.Close()
		return nil, wrapReason(ReasonProbeFailed, err)
	}
	defer func() {
		_ = page.Close()
	}()

	bundle, err := buildCaptchaCandidateBundle(s, page, firstResp.Msg)
	if err != nil {
		return nil, wrapReason(ReasonCaptchaOcrFailed, err)
	}
	verifyPlan := buildCaptchaVerifyPlanForBundle(bundle, s.maxCaptchaVerifyCandidates())
	captchaTexts := captchaVerifyPlanTexts(verifyPlan, "primary")
	s.log.Infoln(s.GetSupplierName(), "captcha candidates:", strings.Join(captchaTexts, ","), "generated:", len(bundle.Primary), "used:", len(captchaTexts))

	fallbackTexts := captchaVerifyPlanTexts(verifyPlan, "fallback")
	if len(fallbackTexts) > 0 {
		s.log.Infoln(s.GetSupplierName(), "captcha fallback candidates:", strings.Join(fallbackTexts, ","), "generated:", len(bundle.Fallback), "used:", len(fallbackTexts))
	}

	return s.executeCaptchaVerifyPlan(
		verifyPlan,
		sid,
		attempt,
		"http",
		func(captchaText string) (*downloadGateResponse, error) {
			return s.fetchDownloadGateResponseHTTP(httpClient, sid, captchaText)
		},
		func(downloadURL string) (*supplier.SubInfo, error) {
			return s.subInfoFromDownloadURLHTTP(httpClient, downloadURL, sourcePageURL)
		},
	)
}

func (s *Supplier) tryDownloadFromPage(page *rod.Page, sid string, sourcePageURL string) (*supplier.SubInfo, error) {
	pageHTML, pageHTMLErr := page.HTML()
	if pageHTMLErr != nil {
		s.log.Warningln(s.GetSupplierName(), "download gate page html failed:", pageHTMLErr)
	} else {
		refreshedSID, parseErr := parseDownloadGateSID(pageHTML)
		if parseErr != nil {
			s.log.Warningln(s.GetSupplierName(), "download gate page sid parse failed:", parseErr)
		} else if refreshedSID != "" && refreshedSID != sid {
			s.log.Infoln(s.GetSupplierName(), "download gate page sid refreshed from page", "old:", sid, "new:", refreshedSID)
			sid = refreshedSID
		}
	}

	firstResp, err := s.fetchDownloadGateResponse(page, sid, "")
	if err != nil {
		return nil, wrapReason(ReasonProbeFailed, err)
	}
	s.logSubHDGateResponse("initial_page", sid, 1, "", firstResp)
	if firstResp.Success && firstResp.Pass && firstResp.URL != "" {
		return s.subInfoFromDownloadURL(page, firstResp.URL, sourcePageURL)
	}
	if firstResp.Success == false {
		return nil, wrapReason(ReasonDownloadGateChanged, fmt.Errorf(strings.TrimSpace(firstResp.Msg)))
	}
	if strings.TrimSpace(firstResp.Msg) == "" {
		return nil, wrapReason(ReasonDownloadGateChanged, fmt.Errorf("subhd captcha svg is empty"))
	}

	bundle, err := buildCaptchaCandidateBundle(s, page, firstResp.Msg)
	if err != nil {
		return nil, wrapReason(ReasonCaptchaOcrFailed, err)
	}
	verifyPlan := buildCaptchaVerifyPlanForBundle(bundle, s.maxCaptchaVerifyCandidates())
	captchaTexts := captchaVerifyPlanTexts(verifyPlan, "primary")
	s.log.Infoln(s.GetSupplierName(), "captcha candidates:", strings.Join(captchaTexts, ","), "generated:", len(bundle.Primary), "used:", len(captchaTexts))

	fallbackTexts := captchaVerifyPlanTexts(verifyPlan, "fallback")
	if len(fallbackTexts) > 0 {
		s.log.Infoln(s.GetSupplierName(), "captcha fallback candidates:", strings.Join(fallbackTexts, ","), "generated:", len(bundle.Fallback), "used:", len(fallbackTexts))
	}

	return s.executeCaptchaVerifyPlan(
		verifyPlan,
		sid,
		1,
		"page",
		func(captchaText string) (*downloadGateResponse, error) {
			return s.fetchDownloadGateResponse(page, sid, captchaText)
		},
		func(downloadURL string) (*supplier.SubInfo, error) {
			return s.subInfoFromDownloadURL(page, downloadURL, sourcePageURL)
		},
	)
}

func (s *Supplier) maxCaptchaAttempts() int {
	attempts := settings.Get().SubtitleSources.SubHDSettings.MaxCaptchaAttempts
	if attempts <= 0 {
		return 4
	}
	if attempts > 4 {
		return 4
	}
	return attempts
}

func (s *Supplier) maxCaptchaVerifyCandidates() int {
	limit := settings.Get().SubtitleSources.SubHDSettings.MaxVerifyCandidates
	if limit <= 0 {
		return maxCaptchaVerifyCandidatesPerAttempt
	}
	if limit > maxCaptchaCandidateCount {
		return maxCaptchaCandidateCount
	}
	return limit
}

func (s *Supplier) subInfoFromDownloadURL(page *rod.Page, downloadURL string, sourcePageURL string) (*supplier.SubInfo, error) {
	payload, err := s.fetchDownloadPayload(page, downloadURL)
	if err != nil {
		return nil, wrapReason(ReasonDownloadFailed, err)
	}
	if payload.StatusCode < 200 || payload.StatusCode >= 300 {
		return nil, wrapReason(ReasonDownloadFailed, fmt.Errorf("unexpected subhd download status %d", payload.StatusCode))
	}

	fileData, err := decodeDataURLBody(payload.DataURL)
	if err != nil {
		return nil, wrapReason(ReasonDownloadFailed, err)
	}

	fileName := fileNameFromDownloadMeta(payload.ContentDisposition, payload.FinalURL, "subhd-subtitle")
	ext := filepath.Ext(fileName)
	if ext == "" {
		if payloadURL, parseErr := pathFromURL(payload.FinalURL); parseErr == nil {
			ext = path.Ext(payloadURL)
		}
		if ext != "" {
			fileName += ext
		}
	}

	return supplier.NewSubInfo(
		s.GetSupplierName(),
		1,
		fileName,
		language.ChineseSimple,
		sourcePageURL,
		0,
		0,
		ext,
		fileData,
	), nil
}

func (s *Supplier) fetchDownloadGateResponse(page *rod.Page, sid string, cap string) (*downloadGateResponse, error) {
	var jsonBody string
	err := rod.Try(func() {
		jsonBody = page.MustEval(`async (sid, cap) => {
			const res = await fetch("/api/sub/down", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ sid, cap }),
			});
			return JSON.stringify({ status: res.status, body: await res.text() });
		}`, sid, cap).Str()
	})
	if err != nil {
		return nil, err
	}

	var gateEnvelope struct {
		Status int    `json:"status"`
		Body   string `json:"body"`
	}
	if err := json.Unmarshal([]byte(jsonBody), &gateEnvelope); err != nil {
		return nil, err
	}
	return decodeDownloadGateResponseBody(gateEnvelope.Status, []byte(gateEnvelope.Body))
}

func (s *Supplier) logDownloadGateClickProbeByBrowser(browser *rod.Browser, gateURL string, sid string, attempt int, cause error) {
	if browser == nil || shouldLogDownloadGateClickProbe(cause) == false {
		return
	}

	page, err := s.openDownloadGatePage(browser, gateURL)
	if err != nil {
		s.log.Warningln(s.GetSupplierName(), "download gate click probe open failed:", err)
		return
	}
	defer func() {
		_ = page.Close()
	}()

	probe, err := s.probeDownloadGateClick(page, sid, "")
	if err != nil {
		s.log.Warningln(s.GetSupplierName(), "download gate click probe failed:", err)
		return
	}
	s.logDownloadGateClickProbe(gateURL, sid, attempt, cause, probe)
}

func shouldLogDownloadGateClickProbe(err error) bool {
	return isSubHDGateStatus500Error(err) || isSubHDGateExpiredError(err)
}

func (s *Supplier) probeDownloadGateClick(page *rod.Page, sid string, cap string) (*downloadGateClickProbe, error) {
	var jsonBody string
	err := rod.Try(func() {
		jsonBody = page.MustEval(`async (sid, cap) => {
			const toText = (value) => {
				if (value == null) return "";
				if (typeof value === "string") return value;
				try { return JSON.stringify(value); } catch (_) { return String(value); }
			};
			const btn = document.querySelector(".down");
			const capInput = document.querySelector("#capIn");
			const svgBox = document.querySelector("#svgBox");
			const svgCap = document.querySelector("#svgCap");
			const fetchCalls = [];
			const alerts = [];
			const originalFetch = window.fetch ? window.fetch.bind(window) : null;
			const originalAlert = window.alert ? window.alert.bind(window) : null;
			const countJQueryClickHandlers = () => {
				try {
					if (!window.jQuery || !window.jQuery._data || !btn) {
						return 0;
					}
					const events = window.jQuery._data(btn, "events") || {};
					const clicks = events.click || [];
					return Array.isArray(clicks) ? clicks.length : 0;
				} catch (_) {
					return 0;
				}
			};
			const snapshot = () => ({
				href: window.location.href || "",
				referrer: document.referrer || "",
				webdriver: navigator.webdriver === undefined ? "undefined" : String(navigator.webdriver),
				ua: navigator.userAgent || "",
				readyState: document.readyState || "",
				sidAttr: btn ? (btn.getAttribute("sid") || "") : "",
				btnHref: btn ? (btn.getAttribute("href") || "") : "",
				btnOnclick: btn ? (btn.getAttribute("onclick") || "") : "",
				btnText: btn ? ((btn.textContent || "").trim()) : "",
				btnDisabled: btn ? !!btn.disabled : false,
				capText: capInput ? String(capInput.value || "") : "",
				svgVisible: !!(svgBox && !svgBox.classList.contains("d-none")),
				svgLen: svgCap ? ((svgCap.innerHTML || "").length) : 0,
				scriptCount: document.scripts ? document.scripts.length : 0,
				jqueryPresent: !!window.jQuery,
				jqueryClickHandlers: countJQueryClickHandlers(),
				hasCapInput: !!capInput,
				hasSvgCap: !!svgCap,
				hasApiDownText: document.documentElement ? document.documentElement.outerHTML.includes("/api/sub/down") : false,
			});

			if (!btn || !originalFetch) {
				return JSON.stringify({
					before: snapshot(),
					after: snapshot(),
					alerts: ["button_or_fetch_missing"],
					calls: fetchCalls,
				});
			}

			window.alert = (message) => {
				alerts.push(String(message || ""));
			};
			window.fetch = async (...args) => {
				const input = args[0];
				const init = args[1] || {};
				const entry = {
					url: typeof input === "string" ? input : ((input && input.url) || ""),
					method: init.method || "GET",
					headers: toText(init.headers || ""),
					body: typeof init.body === "string" ? init.body : toText(init.body || ""),
					status: 0,
					ok: false,
					responseUrl: "",
					responseText: "",
					error: "",
				};
				try {
					const res = await originalFetch(...args);
					const text = await res.clone().text();
					entry.status = res.status || 0;
					entry.ok = !!res.ok;
					entry.responseUrl = res.url || "";
					entry.responseText = text.slice(0, 240);
					fetchCalls.push(entry);
					return res;
				} catch (err) {
					entry.error = String(err || "");
					fetchCalls.push(entry);
					throw err;
				}
			};

			const before = snapshot();
			try {
				if (typeof cap === "string" && capInput) {
					capInput.value = cap;
				}
				if (typeof sid === "string" && sid && btn.getAttribute("sid") !== sid) {
					btn.setAttribute("sid", sid);
				}
				btn.click();
				await new Promise((resolve) => setTimeout(resolve, 1200));
				return JSON.stringify({
					before,
					after: snapshot(),
					alerts,
					calls: fetchCalls,
				});
			} finally {
				window.fetch = originalFetch;
				window.alert = originalAlert;
			}
		}`, sid, cap).Str()
	})
	if err != nil {
		return nil, err
	}

	out := downloadGateClickProbe{}
	if err := json.Unmarshal([]byte(jsonBody), &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (s *Supplier) fetchDownloadGateResponseHTTP(httpClient interface {
	R() *resty.Request
}, sid string, cap string) (*downloadGateResponse, error) {
	gateURL := pkg.AddBaseUrl(s.rootURL(), "/api/sub/down")
	s.logHTTPClientCookies("gate_request", httpClient, gateURL)
	resp, err := httpClient.R().
		SetHeader("Content-Type", "application/json").
		SetBody(map[string]string{
			"sid": sid,
			"cap": cap,
		}).
		Post(gateURL)
	if err != nil {
		return nil, err
	}
	s.logHTTPResponseCookies("gate_response", gateURL, resp.RawResponse)
	s.logHTTPClientCookies("gate_response", httpClient, gateURL)
	return decodeDownloadGateResponseBody(resp.StatusCode(), resp.Body())
}

func (s *Supplier) logDownloadGateClickProbe(gateURL string, sid string, attempt int, cause error, probe *downloadGateClickProbe) {
	if probe == nil {
		return
	}
	if dumpPath, err := dumpDownloadGateClickProbeJSON(sid, attempt, probe); err != nil {
		s.log.Warningln(s.GetSupplierName(), "download gate click probe dump failed:", err)
	} else {
		s.log.Infoln(s.GetSupplierName(), "download gate click probe dump:", dumpPath)
	}

	s.log.Infoln(
		s.GetSupplierName(),
		"download gate click probe",
		"sid:", sid,
		"attempt:", attempt,
		"gate:", gateURL,
		"cause:", truncateSubHDLogText(errorText(cause), 120),
		"webdriver_before:", probe.Before.Webdriver,
		"webdriver_after:", probe.After.Webdriver,
		"button_sid_before:", probe.Before.SID,
		"button_sid_after:", probe.After.SID,
		"button_href_after:", truncateSubHDLogText(probe.After.ButtonHref, 80),
		"button_onclick_after:", truncateSubHDLogText(probe.After.ButtonOnclick, 80),
		"button_disabled_after:", probe.After.ButtonDisabled,
		"script_count_after:", probe.After.ScriptCount,
		"jquery_present_after:", probe.After.JQueryPresent,
		"jquery_clicks_after:", probe.After.JQueryClicks,
		"has_cap_input_after:", probe.After.HasCapInput,
		"has_svg_cap_after:", probe.After.HasSVGCap,
		"has_api_down_text_after:", probe.After.HasAPIDownText,
		"svg_visible_after:", probe.After.SVGVisible,
		"svg_len_after:", probe.After.SVGLen,
		"alerts:", strings.Join(limitProbeLogSlice(probe.Alerts, 3), " | "),
		"call_count:", len(probe.Calls),
	)
	for index, call := range probe.Calls {
		s.log.Infoln(
			s.GetSupplierName(),
			"download gate click probe call",
			"index:", index,
			"method:", truncateSubHDLogText(call.Method, 16),
			"url:", truncateSubHDLogText(call.URL, 80),
			"status:", call.Status,
			"ok:", call.OK,
			"headers:", truncateSubHDLogText(call.Headers, 160),
			"body:", truncateSubHDLogText(call.Body, 160),
			"resp_url:", truncateSubHDLogText(call.ResponseURL, 80),
			"resp_text:", truncateSubHDLogText(call.ResponseText, 160),
			"error:", truncateSubHDLogText(call.Error, 120),
		)
	}
}

func dumpDownloadGateClickProbeJSON(sid string, attempt int, probe *downloadGateClickProbe) (string, error) {
	if probe == nil {
		return "", fmt.Errorf("probe is nil")
	}
	sid = strings.TrimSpace(sid)
	if sid == "" {
		sid = "unknown"
	}
	fileName := fmt.Sprintf("subhd-click-probe-%s-attempt-%d.json", sid, attempt)
	savePath := filepath.Join(pkg.DefTmpFolder(), fileName)
	payload, err := json.MarshalIndent(probe, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(savePath, payload, 0o600); err != nil {
		return "", err
	}
	return savePath, nil
}

func truncateSubHDLogText(text string, limit int) string {
	text = strings.TrimSpace(text)
	if limit <= 0 || len(text) <= limit {
		return text
	}
	return text[:limit]
}

func limitProbeLogSlice(values []string, limit int) []string {
	if len(values) == 0 || limit <= 0 || len(values) <= limit {
		return values
	}
	return values[:limit]
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func decodeDownloadGateResponseBody(statusCode int, body []byte) (*downloadGateResponse, error) {
	out := downloadGateResponse{}
	if err := json.Unmarshal(body, &out); err == nil {
		if statusCode >= 200 && statusCode < 300 {
			return &out, nil
		}
		if out.Success || strings.TrimSpace(out.Msg) != "" || strings.TrimSpace(out.URL) != "" {
			return &out, nil
		}
	}

	bodyPreview := strings.TrimSpace(string(body))
	if len(bodyPreview) > 160 {
		bodyPreview = bodyPreview[:160]
	}
	return nil, fmt.Errorf("unexpected gate status %d body %q", statusCode, bodyPreview)
}

func (s *Supplier) fetchDownloadPayload(page *rod.Page, downloadURL string) (*fetchedDownload, error) {
	var jsonBody string
	err := rod.Try(func() {
		jsonBody = page.MustEval(`async (downloadURL) => {
			const res = await fetch(downloadURL, {
				method: "GET",
				credentials: "same-origin",
			});
			const blob = await res.blob();
			const dataUrl = await new Promise((resolve, reject) => {
				const reader = new FileReader();
				reader.onload = () => resolve(reader.result);
				reader.onerror = () => reject(reader.error || new Error("failed to read blob"));
				reader.readAsDataURL(blob);
			});
			return JSON.stringify({
				statusCode: res.status,
				dataUrl,
				contentType: res.headers.get("content-type") || blob.type || "",
				contentDisposition: res.headers.get("content-disposition") || "",
				finalUrl: res.url || downloadURL,
			});
		}`, downloadURL).Str()
	})
	if err != nil {
		return nil, err
	}

	payload := fetchedDownload{}
	if err := json.Unmarshal([]byte(jsonBody), &payload); err != nil {
		return nil, err
	}

	return &payload, nil
}

func (s *Supplier) subInfoFromDownloadURLHTTP(httpClient interface {
	R() *resty.Request
}, downloadURL string, sourcePageURL string) (*supplier.SubInfo, error) {
	resp, err := httpClient.R().SetDoNotParseResponse(true).Get(downloadURL)
	if err != nil {
		return nil, wrapReason(ReasonDownloadFailed, err)
	}
	defer func() {
		if resp.RawBody() != nil {
			_ = resp.RawBody().Close()
		}
	}()

	body, err := io.ReadAll(resp.RawBody())
	if err != nil {
		return nil, wrapReason(ReasonDownloadFailed, err)
	}

	fileName := pkg.GetFileName(s.log, resp.RawResponse)
	if err := pkg.ValidateSubtitleDownloadPayload(s.log, s.inspectSubtitlePayload, downloadURL, fileName, resp.Header().Get("Content-Type"), resp.StatusCode(), body); err != nil {
		return nil, wrapReason(ReasonDownloadFailed, err)
	}

	ext := filepath.Ext(fileName)
	return supplier.NewSubInfo(
		s.GetSupplierName(),
		1,
		fileName,
		language.ChineseSimple,
		sourcePageURL,
		0,
		0,
		ext,
		body,
	), nil
}

func (s *Supplier) inspectSubtitlePayload(body []byte, ext string) (bool, error) {
	if s.fileDownloader == nil || s.fileDownloader.SubParserHub == nil {
		return false, fmt.Errorf("subtitle parser hub is nil")
	}
	found, _, err := s.fileDownloader.SubParserHub.DetermineFileTypeFromBytes(body, ext)
	return found, err
}

func (s *Supplier) solveCaptcha(page *rod.Page, svgText string) (string, error) {
	if s.captchaSolver == nil {
		s.captchaSolver = newCaptchaSolver(settings.Get().SubtitleSources.SubHDSettings.CaptchaSolver)
	}
	return s.captchaSolver.Solve(s, page, svgText)
}

func buildCaptchaVerifyPlanForBundle(bundle *captchaCandidateBundle, perPhaseLimit int) []captchaVerifyAttempt {
	if bundle == nil {
		return nil
	}
	return buildSimpleCaptchaVerifyPlan(bundle.Primary, bundle.Fallback, perPhaseLimit)
}

func buildSimpleCaptchaVerifyPlan(primaryCandidates []string, fallbackCandidates []string, perPhaseLimit int) []captchaVerifyAttempt {
	primaryCandidates = limitSimpleCaptchaCandidates(primaryCandidates, perPhaseLimit)
	fallbackCandidates = limitSimpleFallbackCaptchaCandidates(primaryCandidates, fallbackCandidates, perPhaseLimit)

	plan := make([]captchaVerifyAttempt, 0, len(primaryCandidates)+len(fallbackCandidates))
	for index, candidate := range primaryCandidates {
		plan = append(plan, captchaVerifyAttempt{
			Text:       candidate,
			Phase:      "primary",
			SourceRank: index,
		})
	}
	for index, candidate := range fallbackCandidates {
		plan = append(plan, captchaVerifyAttempt{
			Text:       candidate,
			Phase:      "fallback",
			SourceRank: index,
		})
	}
	return plan
}

func limitSimpleCaptchaCandidates(candidates []string, limit int) []string {
	if limit <= 0 {
		limit = maxCaptchaVerifyCandidatesPerAttempt
	}
	out := make([]string, 0, limit)
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func limitSimpleFallbackCaptchaCandidates(primaryCandidates []string, fallbackCandidates []string, limit int) []string {
	if limit <= 0 {
		limit = maxCaptchaVerifyCandidatesPerAttempt
	}
	seen := make(map[string]struct{}, len(primaryCandidates))
	for _, candidate := range primaryCandidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		seen[candidate] = struct{}{}
	}

	out := make([]string, 0, limit)
	for _, candidate := range fallbackCandidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
		if len(out) >= limit {
			break
		}
	}
	return out
}

func captchaVerifyPlanTexts(plan []captchaVerifyAttempt, phase string) []string {
	out := make([]string, 0, len(plan))
	for _, item := range plan {
		if item.Phase != phase {
			continue
		}
		out = append(out, item.Text)
	}
	return out
}

func (s *Supplier) executeCaptchaVerifyPlan(
	plan []captchaVerifyAttempt,
	sid string,
	attempt int,
	mode string,
	verify func(string) (*downloadGateResponse, error),
	download func(string) (*supplier.SubInfo, error),
) (*supplier.SubInfo, error) {
	if len(plan) == 0 {
		return nil, wrapReason(ReasonDownloadFailed, fmt.Errorf("subhd download url missing after captcha verify"))
	}

	var lastErr error
	for index, item := range plan {
		verifyResp, err := verify(item.Text)
		if err != nil {
			return nil, wrapReason(ReasonDownloadFailed, err)
		}
		s.logSubHDGateResponse(captchaVerifyResponseLogLabel(mode, item.Phase), sid, attempt, item.Text, verifyResp)
		if verifyResp.Success && verifyResp.Pass && verifyResp.URL != "" {
			return download(verifyResp.URL)
		}

		lastErr = captchaVerifyError(item.Text, verifyResp)
		if lastErr == nil {
			return nil, wrapReason(ReasonDownloadFailed, fmt.Errorf("subhd download url missing after captcha verify"))
		}
		if isRetryableCaptchaVerifyResponse(verifyResp) == false {
			return nil, lastErr
		}
		if index == len(plan)-1 {
			break
		}

		s.log.Infoln(s.GetSupplierName(), captchaVerifyRetryLogLabel(item.Phase), index+1, "failed, trying next candidate")
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, wrapReason(ReasonDownloadFailed, fmt.Errorf("subhd download url missing after captcha verify"))
}

func captchaVerifyResponseLogLabel(mode string, phase string) string {
	switch mode {
	case "page":
		if phase == "fallback" {
			return "verify_page_fallback"
		}
		return "verify_page"
	default:
		if phase == "fallback" {
			return "verify_fallback"
		}
		return "verify"
	}
}

func captchaVerifyRetryLogLabel(phase string) string {
	if phase == "fallback" {
		return "captcha fallback candidate retry"
	}
	return "captcha candidate retry"
}

func sortVerifyTailCandidates(candidates []string) {
	if len(candidates) < 2 {
		return
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left := verifyTailCandidateRank(candidates[i])
		right := verifyTailCandidateRank(candidates[j])
		if left != right {
			return left > right
		}
		return candidates[i] < candidates[j]
	})
}

func verifyTailCandidateRank(candidate string) int {
	candidate = strings.TrimSpace(candidate)
	score := captchaCandidateQualityScore(candidate) * 10
	score -= countLowercaseASCII(candidate) * 14
	if isStrongCaptchaCandidate(candidate) {
		score += 30
	}
	return score
}

func isStrongCaptchaCandidate(candidate string) bool {
	candidate = strings.TrimSpace(candidate)
	if len([]rune(candidate)) != expectedCaptchaTextLength {
		return false
	}
	if countLowercaseASCII(candidate) > 0 {
		return false
	}
	return captchaCandidateQualityScore(candidate) >= 8
}

func extractCaptchaTextFromSVG(svgText string) string {
	matches := captchaNodePattern.FindAllStringSubmatch(svgText, -1)
	if len(matches) == 0 {
		return ""
	}

	type captchaNode struct {
		text  string
		x     float64
		y     float64
		index int
	}

	nodes := make([]captchaNode, 0, len(matches))
	for index, match := range matches {
		if len(match) < 3 {
			continue
		}
		cleaned := captchaCleanPattern.ReplaceAllString(strings.TrimSpace(match[2]), "")
		if cleaned == "" {
			continue
		}
		x, y := parseCaptchaNodePosition(match[1])
		nodes = append(nodes, captchaNode{
			text:  cleaned,
			x:     x,
			y:     y,
			index: index,
		})
	}
	if len(nodes) == 0 {
		return ""
	}

	sort.SliceStable(nodes, func(i, j int) bool {
		left := nodes[i]
		right := nodes[j]
		if left.y != right.y {
			return left.y < right.y
		}
		if left.x != right.x {
			return left.x < right.x
		}
		return left.index < right.index
	})

	var builder strings.Builder
	for _, node := range nodes {
		builder.WriteString(node.text)
	}
	directText := builder.String()
	if len(directText) < 4 {
		return ""
	}
	if len(directText) > 5 {
		directText = directText[:5]
	}

	return directText
}

func parseCaptchaNodePosition(attrs string) (float64, float64) {
	x := 0.0
	y := 0.0
	for _, match := range captchaAttrPattern.FindAllStringSubmatch(attrs, -1) {
		if len(match) < 3 {
			continue
		}
		value, err := strconv.ParseFloat(strings.TrimSpace(match[2]), 64)
		if err != nil {
			continue
		}
		switch strings.ToLower(strings.TrimSpace(match[1])) {
		case "x":
			x = value
		case "y":
			y = value
		}
	}
	return x, y
}

func (s *Supplier) renderCaptchaPNG(page *rod.Page, svgText string) (string, error) {
	var pngDataURL string
	err := rod.Try(func() {
		pngDataURL = page.MustEval(`async (svgText) => {
			const wrapper = document.createElement("div");
			wrapper.innerHTML = svgText;
			const svg = wrapper.querySelector("svg");
			if (!svg) {
				throw new Error("captcha svg not found");
			}
			if (!svg.getAttribute("xmlns")) {
				svg.setAttribute("xmlns", "http://www.w3.org/2000/svg");
			}
			for (const noisy of svg.querySelectorAll("[stroke]")) {
				noisy.remove();
			}
			for (const path of svg.querySelectorAll("path")) {
				if (path.getAttribute("fill")) {
					path.setAttribute("fill", "#000000");
				}
			}

			const widthAttr = Number.parseFloat(svg.getAttribute("width") || "");
			const heightAttr = Number.parseFloat(svg.getAttribute("height") || "");
			const viewBox = svg.viewBox && svg.viewBox.baseVal ? svg.viewBox.baseVal : null;
			const baseWidth = Math.max(
				Math.round((viewBox && viewBox.width) || widthAttr || 180),
				120,
			);
			const baseHeight = Math.max(
				Math.round((viewBox && viewBox.height) || heightAttr || 60),
				40,
			);
			const scale = 4;

			const canvas = document.createElement("canvas");
			canvas.width = baseWidth * scale;
			canvas.height = baseHeight * scale;

			const ctx = canvas.getContext("2d", { willReadFrequently: true });
			if (!ctx) {
				throw new Error("captcha canvas context unavailable");
			}
			ctx.fillStyle = "#ffffff";
			ctx.fillRect(0, 0, canvas.width, canvas.height);
			ctx.imageSmoothingEnabled = false;

			const serialized = new XMLSerializer().serializeToString(svg);
			const blob = new Blob([serialized], { type: "image/svg+xml;charset=utf-8" });
			const objectURL = URL.createObjectURL(blob);
			try {
				const img = await new Promise((resolve, reject) => {
					const image = new Image();
					image.onload = () => resolve(image);
					image.onerror = () => reject(new Error("captcha svg load failed"));
					image.src = objectURL;
				});
				ctx.drawImage(img, 0, 0, canvas.width, canvas.height);

				const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
				for (let i = 0; i < imageData.data.length; i += 4) {
					const r = imageData.data[i];
					const g = imageData.data[i + 1];
					const b = imageData.data[i + 2];
					const lum = 0.299 * r + 0.587 * g + 0.114 * b;
					const v = lum > 110 ? 255 : 0;
					imageData.data[i] = v;
					imageData.data[i + 1] = v;
					imageData.data[i + 2] = v;
					imageData.data[i + 3] = 255;
				}
				ctx.putImageData(imageData, 0, 0);

				let minX = canvas.width;
				let minY = canvas.height;
				let maxX = -1;
				let maxY = -1;
				for (let y = 0; y < canvas.height; y++) {
					for (let x = 0; x < canvas.width; x++) {
						const offset = (y * canvas.width + x) * 4;
						if (imageData.data[offset] !== 0) {
							continue;
						}
						if (x < minX) minX = x;
						if (y < minY) minY = y;
						if (x > maxX) maxX = x;
						if (y > maxY) maxY = y;
					}
				}
				if (maxX >= minX && maxY >= minY) {
					const pad = 8;
					const cropX = Math.max(minX - pad, 0);
					const cropY = Math.max(minY - pad, 0);
					const cropW = Math.min(maxX - minX + 1 + pad * 2, canvas.width - cropX);
					const cropH = Math.min(maxY - minY + 1 + pad * 2, canvas.height - cropY);
					const cropped = document.createElement("canvas");
					cropped.width = cropW;
					cropped.height = cropH;
					const croppedCtx = cropped.getContext("2d", { willReadFrequently: true });
					if (!croppedCtx) {
						throw new Error("captcha cropped canvas context unavailable");
					}
					croppedCtx.fillStyle = "#ffffff";
					croppedCtx.fillRect(0, 0, cropW, cropH);
					croppedCtx.drawImage(canvas, cropX, cropY, cropW, cropH, 0, 0, cropW, cropH);
					return cropped.toDataURL("image/png");
				}

				return canvas.toDataURL("image/png");
			} finally {
				URL.revokeObjectURL(objectURL);
			}
		}`, svgText).Str()
	})
	if err != nil {
		return "", err
	}

	return pngDataURL, nil
}

func (s *Supplier) renderCaptchaGlyphPNGs(page *rod.Page, svgText string) ([]string, error) {
	glyphsJSON := ""
	err := rod.Try(func() {
		glyphsJSON = page.MustEval(`async (svgText) => {
			const wrapper = document.createElement("div");
			wrapper.style.position = "fixed";
			wrapper.style.left = "-99999px";
			wrapper.style.top = "-99999px";
			wrapper.innerHTML = svgText;
			document.body.appendChild(wrapper);
			try {
				const svg = wrapper.querySelector("svg");
				if (!svg) {
					throw new Error("captcha svg not found");
				}
				if (!svg.getAttribute("xmlns")) {
					svg.setAttribute("xmlns", "http://www.w3.org/2000/svg");
				}

				const scale = 4;
				const entries = [];
				for (const path of svg.querySelectorAll("path")) {
					const fill = (path.getAttribute("fill") || "").trim().toLowerCase();
					if (!fill || fill === "none") {
						continue;
					}
					const bbox = path.getBBox();
					if (!Number.isFinite(bbox.width) || !Number.isFinite(bbox.height) || bbox.width <= 0 || bbox.height <= 0) {
						continue;
					}
					entries.push({
						d: path.getAttribute("d") || "",
						fill: path.getAttribute("fill") || "#000000",
						x: bbox.x,
						y: bbox.y,
						width: bbox.width,
						height: bbox.height,
					});
				}
				entries.sort((left, right) => left.x - right.x || left.y - right.y);
				const dataURLs = [];
				for (const entry of entries) {
					const pad = 4;
					const cropX = Math.max(Math.floor(entry.x - pad), 0);
					const cropY = Math.max(Math.floor(entry.y - pad), 0);
					const cropW = Math.max(Math.ceil(entry.width + pad * 2), 12);
					const cropH = Math.max(Math.ceil(entry.height + pad * 2), 18);
					const glyphSVG = document.createElementNS("http://www.w3.org/2000/svg", "svg");
					glyphSVG.setAttribute("xmlns", "http://www.w3.org/2000/svg");
					glyphSVG.setAttribute("viewBox", "0 0 " + cropW + " " + cropH);
					glyphSVG.setAttribute("width", String(cropW));
					glyphSVG.setAttribute("height", String(cropH));
					const glyphPath = document.createElementNS("http://www.w3.org/2000/svg", "path");
					glyphPath.setAttribute("d", entry.d);
					glyphPath.setAttribute("fill", "#000000");
					glyphPath.setAttribute("transform", "translate(" + (-cropX) + ", " + (-cropY) + ")");
					glyphSVG.appendChild(glyphPath);

					const canvas = document.createElement("canvas");
					canvas.width = cropW * scale;
					canvas.height = cropH * scale;
					const ctx = canvas.getContext("2d", { willReadFrequently: true });
					if (!ctx) {
						throw new Error("captcha glyph canvas context unavailable");
					}
					ctx.fillStyle = "#ffffff";
					ctx.fillRect(0, 0, canvas.width, canvas.height);
					ctx.imageSmoothingEnabled = false;

					const serialized = new XMLSerializer().serializeToString(glyphSVG);
					const blob = new Blob([serialized], { type: "image/svg+xml;charset=utf-8" });
					const objectURL = URL.createObjectURL(blob);
					try {
						const img = await new Promise((resolve, reject) => {
							const image = new Image();
							image.onload = () => resolve(image);
							image.onerror = () => reject(new Error("captcha glyph svg load failed"));
							image.src = objectURL;
						});
						ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
						const imageData = ctx.getImageData(0, 0, canvas.width, canvas.height);
						for (let i = 0; i < imageData.data.length; i += 4) {
							const r = imageData.data[i];
							const g = imageData.data[i + 1];
							const b = imageData.data[i + 2];
							const lum = 0.299 * r + 0.587 * g + 0.114 * b;
							const v = lum > 110 ? 255 : 0;
							imageData.data[i] = v;
							imageData.data[i + 1] = v;
							imageData.data[i + 2] = v;
							imageData.data[i + 3] = 255;
						}
						ctx.putImageData(imageData, 0, 0);
						dataURLs.push(canvas.toDataURL("image/png"));
					} finally {
						URL.revokeObjectURL(objectURL);
					}
				}
				return JSON.stringify(dataURLs);
			} finally {
				wrapper.remove();
			}
		}`, svgText).Str()
	})
	if err != nil {
		return nil, err
	}

	var glyphDataURLs []string
	if err := json.Unmarshal([]byte(glyphsJSON), &glyphDataURLs); err != nil {
		return nil, err
	}
	return glyphDataURLs, nil
}

func (s *Supplier) solveCaptchaByGlyphs(page *rod.Page, svgText string, debugPrefix string) ([]string, error) {
	glyphDataURLs, err := s.renderCaptchaGlyphPNGs(page, svgText)
	if err != nil {
		return nil, err
	}
	if len(glyphDataURLs) < 4 {
		return nil, fmt.Errorf("captcha glyph count too small: %d", len(glyphDataURLs))
	}

	recognized := make([][]string, 0, len(glyphDataURLs))
	rawSummary := make([]string, 0, len(glyphDataURLs))
	for index, glyphDataURL := range glyphDataURLs {
		glyphBytes, err := decodeDataURLBody(glyphDataURL)
		if err != nil {
			return nil, err
		}
		tmpFile, err := os.CreateTemp(pkg.DefTmpFolder(), "subhd-captcha-glyph-*.png")
		if err != nil {
			return nil, err
		}
		tmpPath := tmpFile.Name()
		_ = tmpFile.Close()
		if err := os.WriteFile(tmpPath, glyphBytes, 0o600); err != nil {
			_ = os.Remove(tmpPath)
			return nil, err
		}
		rawText, err := runTesseract(tmpPath, 10)
		if debugPrefix != "" {
			_ = os.WriteFile(fmt.Sprintf("%s-glyph-%d.png", debugPrefix, index+1), glyphBytes, 0o600)
			_ = os.WriteFile(fmt.Sprintf("%s-glyph-%d-ocr.txt", debugPrefix, index+1), []byte(strings.TrimSpace(rawText)), 0o600)
		}
		if err != nil {
			_ = os.Remove(tmpPath)
			return nil, fmt.Errorf("glyph %d: %w", index+1, err)
		}
		glyphTexts, err := recognizeSingleCaptchaGlyph(tmpPath, rawText)
		_ = os.Remove(tmpPath)
		if err != nil {
			return nil, fmt.Errorf("glyph %d: %w", index+1, err)
		}
		recognized = append(recognized, glyphTexts)
		rawSummary = append(rawSummary, glyphTexts[0])
	}

	combined := strings.Join(rawSummary, "")
	s.log.Infoln(s.GetSupplierName(), "captcha glyph ocr raw:", strings.Join(rawSummary, ","), "combined:", combined)
	return combineGlyphCaptchaCandidates(recognized, maxCaptchaCandidateCount), nil
}

func combineGlyphCaptchaCandidates(recognized [][]string, limit int) []string {
	if limit <= 0 {
		limit = maxCaptchaCandidateCount
	}

	glyphOptions := make([][]string, 0, len(recognized))
	for _, glyphTexts := range recognized {
		options := make([]string, 0, 3)
		for _, glyphText := range glyphTexts {
			glyphText = strings.TrimSpace(glyphText)
			if len([]rune(glyphText)) != 1 {
				continue
			}
			options = appendUniqueStringCandidates(options, glyphText)
			upper := uppercaseCaptchaCandidate(glyphText)
			if upper != glyphText {
				options = appendUniqueStringCandidates(options, upper)
			}
			if len(options) >= 3 {
				break
			}
		}
		if len(options) == 0 {
			return nil
		}
		glyphOptions = append(glyphOptions, options)
	}

	out := make([]string, 0, limit)
	var build func(index int, parts []string)
	build = func(index int, parts []string) {
		if len(out) >= limit {
			return
		}
		if index >= len(glyphOptions) {
			candidate := strings.Join(parts, "")
			if len([]rune(candidate)) != expectedCaptchaTextLength {
				return
			}
			out = appendUniqueStringCandidates(out, candidate)
			upper := uppercaseCaptchaCandidate(candidate)
			if upper != candidate {
				out = appendUniqueStringCandidates(out, upper)
			}
			return
		}
		for _, option := range glyphOptions[index] {
			build(index+1, append(parts, option))
			if len(out) >= limit {
				return
			}
		}
	}

	build(0, nil)
	return out
}

func recognizeSingleCaptchaGlyph(imagePath string, firstRaw string) ([]string, error) {
	return recognizeSingleCaptchaGlyphWithRunner(imagePath, firstRaw, buildEnhancedGlyphImage, runTesseract)
}

func recognizeCaptchaTextWithRunner(
	imagePath string,
	firstRaw string,
	buildEnhanced func(string, int, int) (string, func(), error),
	runOCR func(string, int) (string, error),
) ([]string, error) {
	candidates := make([]string, 0, 4)
	normalizedText, err := normalizeCaptchaText(firstRaw)
	if err == nil {
		candidates = appendUniqueStringCandidates(candidates, normalizedText)
	}

	enhancedConfigs := []struct {
		padding int
		scale   int
		psm     int
	}{
		{padding: 8, scale: 2, psm: 7},
		{padding: 8, scale: 2, psm: 8},
		{padding: 12, scale: 3, psm: 7},
	}

	var lastErr error
	if err != nil {
		lastErr = err
	}
	for _, cfg := range enhancedConfigs {
		enhancedPath, cleanup, buildErr := buildEnhanced(imagePath, cfg.padding, cfg.scale)
		if buildErr != nil {
			lastErr = buildErr
			continue
		}
		rawText, ocrErr := runOCR(enhancedPath, cfg.psm)
		cleanup()
		if ocrErr != nil {
			lastErr = ocrErr
			continue
		}
		normalizedText, normErr := normalizeCaptchaText(rawText)
		if normErr != nil {
			lastErr = normErr
			continue
		}
		candidates = appendUniqueStringCandidates(candidates, normalizedText)
	}

	if len(candidates) > 0 {
		return candidates, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("unexpected captcha OCR output %q", strings.TrimSpace(firstRaw))
	}
	return nil, lastErr
}

func recognizeSingleCaptchaGlyphWithRunner(
	imagePath string,
	firstRaw string,
	buildEnhanced func(string, int, int) (string, func(), error),
	runOCR func(string, int) (string, error),
) ([]string, error) {
	firstCandidates := collectSingleGlyphCandidates(firstRaw)
	rawRetryCandidates := make([]string, 0, 4)
	enhancedCandidates := make([]string, 0, 4)
	candidates := append([]string(nil), firstCandidates...)
	var lastErr error
	for _, psm := range []int{6, 7, 8, 13} {
		rawText, err := runOCR(imagePath, psm)
		if err != nil {
			lastErr = err
			continue
		}
		newCandidates := collectSingleGlyphCandidates(rawText)
		rawRetryCandidates = appendUniqueGlyphCandidates(rawRetryCandidates, newCandidates...)
		candidates = appendUniqueGlyphCandidates(candidates, newCandidates...)
	}

	enhancedConfigs := []struct {
		padding int
		scale   int
		psm     int
	}{
		{padding: 8, scale: 4, psm: 6},
		{padding: 8, scale: 4, psm: 10},
		{padding: 12, scale: 6, psm: 6},
	}
	for _, cfg := range enhancedConfigs {
		enhancedPath, cleanup, err := buildEnhanced(imagePath, cfg.padding, cfg.scale)
		if err != nil {
			lastErr = err
			continue
		}
		rawText, err := runOCR(enhancedPath, cfg.psm)
		cleanup()
		if err != nil {
			lastErr = err
			continue
		}
		newCandidates := collectSingleGlyphCandidates(rawText)
		enhancedCandidates = appendUniqueGlyphCandidates(enhancedCandidates, newCandidates...)
		candidates = appendUniqueGlyphCandidates(candidates, newCandidates...)
		if len(candidates) == 0 {
			lastErr = fmt.Errorf("unexpected captcha glyph OCR output %q", strings.TrimSpace(rawText))
		}
	}
	candidates = promoteStrongSingleGlyphCandidateHead(candidates, firstCandidates, rawRetryCandidates, enhancedCandidates)

	if len(candidates) > 0 {
		return candidates, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("unexpected captcha glyph OCR output %q", strings.TrimSpace(firstRaw))
	}
	return nil, lastErr
}

func promoteStrongSingleGlyphCandidateHead(candidates []string, firstCandidates []string, rawRetryCandidates []string, enhancedCandidates []string) []string {
	if len(candidates) < 2 {
		return candidates
	}
	head := strings.TrimSpace(candidates[0])
	headRunes := []rune(head)
	if len(headRunes) != 1 || (headRunes[0] < 'a' || headRunes[0] > 'z') {
		return candidates
	}
	if len(firstCandidates) > 0 {
		for _, candidate := range rawRetryCandidates {
			runes := []rune(strings.TrimSpace(candidate))
			if len(runes) != 1 {
				continue
			}
			if isUpperOrDigitRune(runes[0]) == false {
				continue
			}
			return promoteSingleGlyphCandidateToHead(candidates, candidate)
		}
	}
	for index := 1; index < len(candidates); index++ {
		candidate := strings.TrimSpace(candidates[index])
		if containsStringCandidate(rawRetryCandidates, candidate) == false {
			continue
		}
		runes := []rune(candidate)
		if len(runes) == 1 && isUpperOrDigitRune(runes[0]) {
			return promoteSingleGlyphCandidateToHead(candidates, candidate)
		}
	}
	for _, candidate := range enhancedCandidates {
		runes := []rune(strings.TrimSpace(candidate))
		if len(runes) != 1 {
			continue
		}
		if isUpperOrDigitRune(runes[0]) == false {
			continue
		}
		if len(firstCandidates) == 0 {
			continue
		}
		return promoteSingleGlyphCandidateToHead(candidates, candidate)
	}
	return candidates
}

func promoteSingleGlyphCandidateToHead(candidates []string, candidate string) []string {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return candidates
	}
	for index := 1; index < len(candidates); index++ {
		if strings.TrimSpace(candidates[index]) != candidate {
			continue
		}
		out := append([]string(nil), candidates...)
		out[0], out[index] = out[index], out[0]
		return out
	}
	return candidates
}

func buildEnhancedGlyphImage(srcPath string, padding int, scale int) (string, func(), error) {
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return "", func() {}, err
	}
	defer srcFile.Close()

	srcImage, _, err := image.Decode(srcFile)
	if err != nil {
		return "", func() {}, err
	}

	srcBounds := srcImage.Bounds()
	dstW := srcBounds.Dx()*scale + padding*2
	dstH := srcBounds.Dy()*scale + padding*2
	if dstW <= 0 || dstH <= 0 {
		return "", func() {}, fmt.Errorf("invalid glyph bounds")
	}

	dstImage := image.NewRGBA(image.Rect(0, 0, dstW, dstH))
	draw.Draw(dstImage, dstImage.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	for y := 0; y < srcBounds.Dy(); y++ {
		for x := 0; x < srcBounds.Dx(); x++ {
			c := srcImage.At(srcBounds.Min.X+x, srcBounds.Min.Y+y)
			for sy := 0; sy < scale; sy++ {
				for sx := 0; sx < scale; sx++ {
					dstImage.Set(padding+x*scale+sx, padding+y*scale+sy, c)
				}
			}
		}
	}

	tmpFile, err := os.CreateTemp(pkg.DefTmpFolder(), "subhd-captcha-glyph-enhanced-*.png")
	if err != nil {
		return "", func() {}, err
	}
	tmpPath := tmpFile.Name()
	cleanup := func() {
		_ = os.Remove(tmpPath)
	}
	if err := png.Encode(tmpFile, dstImage); err != nil {
		_ = tmpFile.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := tmpFile.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}

	return tmpPath, cleanup, nil
}

func runTesseract(imagePath string, psm int) (string, error) {
	tesseractPath, err := resolveTesseractBinary()
	if err != nil {
		return "", err
	}

	args, outputBase, cleanup, err := buildTesseractArgs(imagePath, psm)
	if err != nil {
		return "", err
	}
	defer cleanup()
	cmd := exec.Command(tesseractPath, args...)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	stderrText := strings.TrimSpace(stderr.String())
	textBytes, readErr := os.ReadFile(outputBase + ".txt")
	if err != nil && readErr != nil {
		return "", fmt.Errorf("tesseract failed: %w: %s", err, stderrText)
	}
	if readErr == nil && strings.TrimSpace(string(textBytes)) != "" {
		return string(textBytes), nil
	}
	if strings.TrimSpace(string(out)) != "" {
		return string(out), nil
	}
	if err != nil {
		return "", fmt.Errorf("tesseract failed: %w: %s", err, stderrText)
	}
	if readErr != nil {
		return "", readErr
	}
	return string(textBytes), nil
}

func dumpCaptchaDebugArtifacts(svgText string, pngBytes []byte, rawText string) (string, error) {
	baseFile, err := os.CreateTemp(pkg.DefTmpFolder(), "subhd-captcha-debug-")
	if err != nil {
		return "", err
	}
	basePath := baseFile.Name()
	_ = baseFile.Close()
	_ = os.Remove(basePath)

	if err := os.WriteFile(basePath+".svg", []byte(svgText), 0o600); err != nil {
		return "", err
	}
	if len(pngBytes) > 0 {
		if err := os.WriteFile(basePath+".png", pngBytes, 0o600); err != nil {
			return "", err
		}
	}
	if rawText != "" {
		if err := os.WriteFile(basePath+"-ocr.txt", []byte(rawText), 0o600); err != nil {
			return "", err
		}
	}

	return basePath, nil
}

func buildTesseractArgs(imagePath string, psm int) ([]string, string, func(), error) {
	configFile, err := os.CreateTemp(pkg.DefTmpFolder(), "subhd-tesseract-*.cfg")
	if err != nil {
		return nil, "", nil, err
	}
	configPath := configFile.Name()
	configBody := "tessedit_char_whitelist ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789\n"
	if _, err := configFile.WriteString(configBody); err != nil {
		_ = configFile.Close()
		_ = os.Remove(configPath)
		return nil, "", nil, err
	}
	if err := configFile.Close(); err != nil {
		_ = os.Remove(configPath)
		return nil, "", nil, err
	}
	outputFile, err := os.CreateTemp(pkg.DefTmpFolder(), "subhd-tesseract-out-")
	if err != nil {
		_ = os.Remove(configPath)
		return nil, "", nil, err
	}
	outputBase := outputFile.Name()
	_ = outputFile.Close()
	_ = os.Remove(outputBase)

	cleanup := func() {
		_ = os.Remove(configPath)
		_ = os.Remove(outputBase)
		_ = os.Remove(outputBase + ".txt")
	}

	return []string{
		imagePath,
		outputBase,
		"-l", "eng",
		"--psm", strconv.Itoa(psm),
		configPath,
	}, outputBase, cleanup, nil
}

func (s *Supplier) logSubHDGateResponse(stage string, sid string, attempt int, captchaText string, resp *downloadGateResponse) {
	if resp == nil {
		s.log.Warningln(s.GetSupplierName(), "download gate", stage, "sid:", sid, "attempt:", attempt, "captcha:", captchaText, "resp:nil")
		return
	}

	msg := strings.TrimSpace(resp.Msg)
	msgPreview := msg
	if len(msgPreview) > 80 {
		msgPreview = msgPreview[:80]
	}

	msgKind := "empty"
	switch {
	case resp.Pass && resp.URL != "":
		msgKind = "download_url"
	case strings.Contains(msg, "<svg"):
		msgKind = "captcha_svg"
	case isSubHDGateExpiredError(errors.New(msg)):
		msgKind = "expired"
	case msg != "":
		msgKind = "text"
	}

	s.log.Infoln(
		s.GetSupplierName(),
		"download gate", stage,
		"sid:", sid,
		"attempt:", attempt,
		"captcha:", captchaText,
		"success:", resp.Success,
		"pass:", resp.Pass,
		"url:", resp.URL != "",
		"msg_kind:", msgKind,
		"msg_len:", len(msg),
		"msg_preview:", msgPreview,
	)
}

func shouldRetrySubHDGateAttempt(err error) bool {
	if err == nil {
		return false
	}
	if isSubHDGateExpiredError(err) {
		return false
	}
	if isSubHDGateStatus500Error(err) {
		return false
	}
	return true
}

func isSubHDGateExpiredError(err error) bool {
	if err == nil {
		return false
	}
	errText := strings.ToLower(err.Error())
	if strings.Contains(errText, "expired gate") {
		return true
	}
	return strings.Contains(errText, "闂備礁鎼崯顐︽偉閻撳宫娑㈠箰鎼搭澀姘﹂梺鎼炲労閸撴艾袙閹扮増鐓涢柛顐亜閸樼顭跨捄铏剐х€殿噮鍣ｅ畷濂稿即濡嘲浜惧鑸靛姈椤ュ牓鏌曡箛鏇炐㈤柛銈嗗笧缁辨挻鎷呴崫銉ь唶闂侀潧娲﹂崹鍨嚕?") ||
		strings.Contains(errText, "闂傚倷绀侀幖顐﹀疮椤愶附鍋夐柣鎾冲濞戙垹绠伴幖鎼線濮橈箓姊洪幖鐐插姶闁告挻鑹捐闁规壆澧楅悡娑㈡煕椤愵偄浜滈柛妯碱焾椤法鎹勯搹鍓愌呪偓娈垮櫘閸ｏ絽鐣锋總绋垮嵆婵☆垰鍢叉禍鎯ь熆閼搁潧濮堟い銉ョ墦閺屾洝绠涢弴鐐愩垽鏌涢妶鍡楃缂佽鲸鎸婚幏鍛村传閵壯屽敹闂備線娼уú锕傚垂閸噮鍤?")
}

func isSubHDGateStatus500Error(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "unexpected gate status 500")
}

func resolveTesseractBinary() (string, error) {
	return resolveTesseractBinaryWith(func(name string) (string, error) {
		return exec.LookPath(name)
	}, []string{
		`C:\Program Files\Subtitle Edit\Tesseract302\tesseract.exe`,
		`C:\Program Files\Tesseract-OCR\tesseract.exe`,
		`C:\Program Files (x86)\Tesseract-OCR\tesseract.exe`,
	})
}

func resolveTesseractBinaryWith(lookPath func(string) (string, error), candidates []string) (string, error) {
	if bin, err := lookPath("tesseract"); err == nil && strings.TrimSpace(bin) != "" {
		return bin, nil
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("tesseract not found in PATH or known install locations")
}

func normalizeCaptchaOCRSymbols(text string) string {
	return captchaOCRSymbolReplacer.Replace(text)
}

func normalizeCaptchaText(raw string) (string, error) {
	raw = normalizeCaptchaOCRSymbols(strings.TrimSpace(raw))
	if matched := captchaTextPattern.FindString(raw); matched != "" {
		return matched, nil
	}

	cleaned := captchaCleanPattern.ReplaceAllString(raw, "")
	if len(cleaned) >= expectedCaptchaTextLength {
		if len(cleaned) > expectedCaptchaTextLength {
			cleaned = cleaned[:expectedCaptchaTextLength]
		}
		return cleaned, nil
	}

	return "", fmt.Errorf("unexpected captcha OCR output %q", strings.TrimSpace(raw))
}

func normalizeSingleCaptchaGlyph(raw string) (string, error) {
	normalized := normalizeCaptchaOCRSymbols(strings.TrimSpace(raw))
	normalized = captchaCleanPattern.ReplaceAllString(normalized, "")
	if normalized == "" {
		return "", fmt.Errorf("unexpected captcha glyph OCR output %q", strings.TrimSpace(raw))
	}
	runes := []rune(normalized)
	return string(runes[:1]), nil
}

func collectSingleGlyphCandidates(raw string) []string {
	normalized := normalizeCaptchaOCRSymbols(strings.TrimSpace(raw))
	normalized = captchaCleanPattern.ReplaceAllString(normalized, "")
	if normalized == "" {
		return nil
	}
	out := make([]string, 0, len(normalized))
	seen := make(map[rune]struct{}, len(normalized))
	for _, char := range normalized {
		if _, ok := seen[char]; ok {
			continue
		}
		seen[char] = struct{}{}
		out = append(out, string(char))
	}
	return out
}

func appendUniqueGlyphCandidates(base []string, extra ...string) []string {
	if len(extra) == 0 {
		return base
	}
	seen := make(map[string]struct{}, len(base)+len(extra))
	for _, candidate := range base {
		seen[candidate] = struct{}{}
	}
	for _, candidate := range extra {
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		base = append(base, candidate)
	}
	return base
}

func appendUniqueStringCandidates(base []string, extra ...string) []string {
	if len(extra) == 0 {
		return base
	}
	seen := make(map[string]struct{}, len(base)+len(extra))
	for _, candidate := range base {
		seen[candidate] = struct{}{}
	}
	for _, candidate := range extra {
		if candidate == "" {
			continue
		}
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		base = append(base, candidate)
	}
	return base
}

func captchaCandidateQualityScore(candidate string) int {
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return 0
	}
	score := 0
	for _, char := range candidate {
		switch {
		case char >= 'A' && char <= 'Z':
			score += 3
		case char >= '0' && char <= '9':
			score += 2
		case char >= 'a' && char <= 'z':
			score += 1
		default:
			score -= 2
		}
	}
	if len([]rune(candidate)) != expectedCaptchaTextLength {
		score -= 4
	}
	score -= countLowercaseASCII(candidate)
	return score
}

type captchaGlyphOption struct {
	Char             rune
	OCRRank          int
	FromGlyphOCR     bool
	PreferredRank    int
	FromPosPreferred bool
}

func newCaptchaGlyphOption(base rune, position int, char rune, ocrRank int) captchaGlyphOption {
	option := captchaGlyphOption{
		Char:         char,
		OCRRank:      ocrRank,
		FromGlyphOCR: true,
	}
	if preferredRank, ok := glyphPreferredRank(position, base, char); ok {
		option.PreferredRank = preferredRank
		option.FromPosPreferred = true
	}
	return option
}

func newCaptchaGlyphFallbackOption(base rune, char rune, position int) captchaGlyphOption {
	option := captchaGlyphOption{
		Char:    char,
		OCRRank: -1,
	}
	if preferredRank, ok := glyphPreferredRank(position, base, char); ok {
		option.PreferredRank = preferredRank
		option.FromPosPreferred = true
	}
	return option
}

func appendUniqueRuneCandidates(base []rune, extra ...rune) []rune {
	if len(extra) == 0 {
		return base
	}
	seen := make(map[rune]struct{}, len(base)+len(extra))
	for _, candidate := range base {
		seen[candidate] = struct{}{}
	}
	for _, candidate := range extra {
		if _, ok := seen[candidate]; ok {
			continue
		}
		seen[candidate] = struct{}{}
		base = append(base, candidate)
	}
	return base
}

func appendUniqueGlyphOptions(base []captchaGlyphOption, extra ...captchaGlyphOption) []captchaGlyphOption {
	if len(extra) == 0 {
		return base
	}
	seen := make(map[rune]struct{}, len(base)+len(extra))
	for _, candidate := range base {
		seen[candidate.Char] = struct{}{}
	}
	for _, candidate := range extra {
		if _, ok := seen[candidate.Char]; ok {
			continue
		}
		seen[candidate.Char] = struct{}{}
		base = append(base, candidate)
	}
	return base
}

func glyphRuneAlternatives(char rune) []rune {
	return glyphRuneAlternativesAt(-1, char)
}

func glyphRuneAlternativesAt(position int, char rune) []rune {
	out := []rune{char}
	seen := map[rune]struct{}{char: {}}

	appendAlt := func(r rune) {
		if _, ok := seen[r]; ok {
			return
		}
		seen[r] = struct{}{}
		out = append(out, r)
	}

	if preferredByPos, ok := captchaGlyphPositionPreferredChars[position]; ok {
		if preferred, ok := preferredByPos[char]; ok {
			for _, alt := range preferred {
				appendAlt(alt)
			}
		}
	}
	if alts, ok := captchaGlyphAmbiguousChars[char]; ok {
		for _, alt := range alts {
			appendAlt(alt)
		}
	}
	if char >= 'a' && char <= 'z' {
		appendAlt(char - 'a' + 'A')
	}
	if char >= 'A' && char <= 'Z' {
		appendAlt(char - 'A' + 'a')
	}

	return out
}

func sortGlyphCaptchaCandidates(baseRunes []rune, perPosAlts [][]captchaGlyphOption, candidates []string) {
	sort.SliceStable(candidates, func(i, j int) bool {
		left := scoreGlyphCaptchaCandidate(baseRunes, perPosAlts, candidates[i])
		right := scoreGlyphCaptchaCandidate(baseRunes, perPosAlts, candidates[j])
		if left != right {
			return left > right
		}
		return candidates[i] < candidates[j]
	})
}

func promoteNearBaseGlyphCandidate(baseRunes []rune, perPosAlts [][]captchaGlyphOption, candidates []string) {
	if len(candidates) < 2 {
		return
	}
	baseCandidate := string(baseRunes)
	baseIndex := -1
	for index, candidate := range candidates {
		if candidate == baseCandidate {
			baseIndex = index
			break
		}
	}
	if baseIndex <= 0 {
		return
	}

	topCandidate := candidates[0]
	if countLowercaseASCII(baseCandidate) > countLowercaseASCII(topCandidate) {
		return
	}

	topScore := scoreGlyphCaptchaCandidate(baseRunes, perPosAlts, topCandidate)
	baseScore := scoreGlyphCaptchaCandidate(baseRunes, perPosAlts, baseCandidate)
	if topScore-baseScore > 2 {
		return
	}
	if shouldPromoteNearBaseGlyphCandidate(baseCandidate, topCandidate) == false {
		return
	}

	copy(candidates[1:baseIndex+1], candidates[0:baseIndex])
	candidates[0] = baseCandidate
}

func shouldPromoteNearBaseGlyphCandidate(baseCandidate string, topCandidate string) bool {
	if countLowercaseASCII(baseCandidate) > countLowercaseASCII(topCandidate) {
		return false
	}
	baseRunes := []rune(baseCandidate)
	topRunes := []rune(topCandidate)
	if len(baseRunes) != len(topRunes) {
		return false
	}

	changed := 0
	for index, baseRune := range baseRunes {
		current := topRunes[index]
		if current == baseRune {
			continue
		}
		changed++
		if changed > 1 {
			return false
		}
		if isNearBaseShapePreference(baseRune, current) == false {
			return false
		}
	}
	return changed == 1
}

func isNearBaseShapePreference(base rune, current rune) bool {
	switch base {
	case 'W', 'w', 'M', 'm', 'N', 'n', 'H', 'h':
		return strings.ContainsRune("WMNHwmnh", current)
	case 'Q', 'q', 'C', 'c', 'G', 'g', 'O', 'o':
		return strings.ContainsRune("QCGOqcgo", current)
	default:
		return false
	}
}

func scoreGlyphCaptchaCandidate(baseRunes []rune, perPosAlts [][]captchaGlyphOption, candidate string) int {
	runes := []rune(candidate)
	score := 0
	if len(runes) == expectedCaptchaTextLength {
		score += 40
	}
	if len(runes) != len(baseRunes) {
		score -= 50
	}
	edits := 0
	for index, char := range runes {
		if index >= len(baseRunes) {
			score -= 10
			continue
		}
		if char == baseRunes[index] {
			score += 6
			continue
		}
		edits++
		score += glyphPositionReplacementScore(index, baseRunes[index], char)
		score += glyphOptionSourceScore(perPosAlts, index, char)
	}
	score -= edits * 12
	score -= countLowercaseASCII(candidate) * 5
	return score
}

func glyphOptionSourceScore(perPosAlts [][]captchaGlyphOption, position int, current rune) int {
	if position < 0 || position >= len(perPosAlts) {
		return 0
	}
	for _, option := range perPosAlts[position] {
		if option.Char != current {
			continue
		}
		score := 0
		if option.FromGlyphOCR {
			switch option.OCRRank {
			case 1:
				score += 12
			case 2:
				score += 9
			default:
				if option.OCRRank > 0 {
					score += 6
				}
			}
		}
		return score
	}
	return 0
}

func glyphPositionReplacementScore(position int, base rune, current rune) int {
	if preferredByPos, ok := captchaGlyphPositionPreferredChars[position]; ok {
		if preferred, ok := preferredByPos[base]; ok {
			for index, alt := range preferred {
				if alt == current {
					score := 19 - index
					return score
				}
			}
		}
	}
	if current == toUpperASCII(base) {
		return 11
	}
	if current >= 'A' && current <= 'Z' {
		return 8
	}
	if current >= '0' && current <= '9' {
		return 6
	}
	if current >= 'a' && current <= 'z' {
		return 3
	}
	return 1
}

func glyphPreferredRank(position int, base rune, current rune) (int, bool) {
	if preferredByPos, ok := captchaGlyphPositionPreferredChars[position]; ok {
		if preferred, ok := preferredByPos[base]; ok {
			for index, alt := range preferred {
				if alt == current {
					return index, true
				}
			}
		}
	}
	return 0, false
}

func toUpperASCII(char rune) rune {
	if char >= 'a' && char <= 'z' {
		return char - 'a' + 'A'
	}
	return char
}

func toLowerASCII(char rune) rune {
	if char >= 'A' && char <= 'Z' {
		return char - 'A' + 'a'
	}
	return char
}

func countLowercaseASCII(text string) int {
	count := 0
	for _, char := range text {
		if char >= 'a' && char <= 'z' {
			count++
		}
	}
	return count
}

func containsStringCandidate(candidates []string, target string) bool {
	for _, candidate := range candidates {
		if candidate == target {
			return true
		}
	}
	return false
}

func containsTrimmedStringCandidate(candidates []string, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == target {
			return true
		}
	}
	return false
}

func isRedundantWholeVariant(base string, candidate string) bool {
	baseRunes := []rune(strings.TrimSpace(base))
	candidateRunes := []rune(strings.TrimSpace(candidate))
	if len(baseRunes) != expectedCaptchaTextLength || len(candidateRunes) != expectedCaptchaTextLength {
		return false
	}

	diffCount := 0
	for index := 0; index < expectedCaptchaTextLength; index++ {
		left := baseRunes[index]
		right := candidateRunes[index]
		if left == right {
			continue
		}
		if toUpperASCII(left) == toUpperASCII(right) {
			diffCount++
			continue
		}
		if isOneILClusterRune(left) && isOneILClusterRune(right) {
			diffCount++
			continue
		}
		if areAmbiguousCaptchaRunes(left, right) {
			diffCount++
			continue
		}
		return false
	}

	return diffCount == 1
}

func isOneILClusterRune(char rune) bool {
	switch char {
	case '1', 'I', 'i', 'L', 'l':
		return true
	default:
		return false
	}
}

func areAmbiguousCaptchaRunes(left rune, right rune) bool {
	if left == right {
		return true
	}
	leftUpper := toUpperASCII(left)
	rightUpper := toUpperASCII(right)
	if leftUpper == rightUpper {
		return true
	}
	if alts, ok := captchaAmbiguousChars[leftUpper]; ok {
		for _, alt := range alts {
			if toUpperASCII(alt) == rightUpper {
				return true
			}
		}
	}
	if alts, ok := captchaAmbiguousChars[rightUpper]; ok {
		for _, alt := range alts {
			if toUpperASCII(alt) == leftUpper {
				return true
			}
		}
	}
	return false
}

func shouldUseGlyphRuneForWholeHybrid(whole rune, glyph rune) bool {
	if whole == glyph {
		return false
	}
	if isUpperOrDigitRune(glyph) == false {
		return false
	}
	if whole >= 'a' && whole <= 'z' {
		return true
	}
	return areAmbiguousCaptchaRunes(whole, glyph)
}

func isUpperOrDigitRune(char rune) bool {
	return (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')
}

func appendUniqueCaptchaRunes(base []rune, extra ...rune) []rune {
	if len(extra) == 0 {
		return base
	}
	seen := make(map[rune]struct{}, len(base)+len(extra))
	for _, one := range base {
		seen[one] = struct{}{}
	}
	for _, one := range extra {
		if _, ok := seen[one]; ok {
			continue
		}
		seen[one] = struct{}{}
		base = append(base, one)
	}
	return base
}

func uppercaseCaptchaCandidate(candidate string) string {
	if candidate == "" {
		return ""
	}
	runes := []rune(candidate)
	for index, char := range runes {
		runes[index] = toUpperASCII(char)
	}
	return string(runes)
}

func isRetryableCaptchaVerifyResponse(resp *downloadGateResponse) bool {
	if resp == nil {
		return false
	}
	if resp.Success == false {
		return true
	}
	return strings.TrimSpace(resp.Msg) != ""
}

func captchaVerifyError(captchaText string, resp *downloadGateResponse) error {
	if resp == nil {
		return wrapReason(ReasonDownloadFailed, fmt.Errorf("subhd captcha verify response is nil"))
	}
	if resp.Success == false {
		return wrapReason(ReasonCaptchaOcrFailed, fmt.Errorf(strings.TrimSpace(resp.Msg)))
	}
	if strings.TrimSpace(resp.Msg) != "" {
		return wrapReason(ReasonCaptchaOcrFailed, fmt.Errorf("subhd captcha rejected: %s", captchaText))
	}
	return nil
}

func decodeDataURLBody(dataURL string) ([]byte, error) {
	idx := strings.Index(dataURL, ",")
	if idx < 0 || idx+1 >= len(dataURL) {
		return nil, fmt.Errorf("invalid data url")
	}

	return base64.StdEncoding.DecodeString(dataURL[idx+1:])
}

func fileNameFromDownloadMeta(contentDisposition string, finalURL string, fallback string) string {
	if fileName := contentDispositionFilename(contentDisposition); fileName != "" {
		return fileName
	}

	if finalPath, err := pathFromURL(finalURL); err == nil {
		if baseName := path.Base(finalPath); baseName != "" && baseName != "." && baseName != "/" {
			return baseName
		}
	}

	if fallback != "" {
		return fallback
	}

	return "subhd-subtitle"
}

func contentDispositionFilename(contentDisposition string) string {
	if strings.TrimSpace(contentDisposition) == "" {
		return ""
	}

	matched := regexp.MustCompile(`filename=["]*([^"]+)["]*`).FindStringSubmatch(contentDisposition)
	if len(matched) >= 2 {
		return matched[1]
	}

	return ""
}

func subHDSIDFromURL(downloadPageURL string) (string, error) {
	downloadPath, err := pathFromURL(downloadPageURL)
	if err != nil {
		return "", err
	}

	sid := strings.TrimSpace(path.Base(strings.TrimRight(downloadPath, "/")))
	if sid == "" || sid == "." || sid == "/" {
		return "", fmt.Errorf("subhd sid missing in %s", downloadPageURL)
	}

	return sid, nil
}

func pathFromURL(rawURL string) (string, error) {
	parsedURL, err := neturl.Parse(rawURL)
	if err == nil && parsedURL.Path != "" {
		return parsedURL.Path, nil
	}

	return path.Clean(rawURL), nil
}
