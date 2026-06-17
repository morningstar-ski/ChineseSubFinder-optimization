package subhd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"image"
	"image/color"
	"image/png"
	neturl "net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/rod_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/go-rod/rod"
)

const maxCaptchaAttempts = 5

const subHDGateRetryBaseDelay = 1500 * time.Millisecond
const subHDPageEvalRetryAttempts = 2
const subHDPageEvalRetryDelay = 250 * time.Millisecond

const (
	captchaForegroundThreshold = 210
	captchaCropPadding         = 4
	captchaScaleFactor         = 2
)

var (
	captchaTextPattern      = regexp.MustCompile(`[A-Za-z0-9]{4,5}`)
	captchaCleanPattern     = regexp.MustCompile(`[^A-Za-z0-9]+`)
	captchaNodePattern      = regexp.MustCompile(`(?is)<(?:text|tspan)\b[^>]*>(.*?)</(?:text|tspan)>`)
	captchaBackendWhitelist = map[string]struct{}{
		"ddddocr":  {},
		"external": {},
	}
)

type downloadGateResponse struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
	Pass    bool   `json:"pass"`
	URL     string `json:"url"`
}

type externalCaptchaRequest struct {
	ImageBase64 string `json:"image_base64"`
	MimeType    string `json:"mime_type"`
}

type externalCaptchaResponse struct {
	Text string `json:"text"`
}

func (s *Supplier) downloadSubFileViaGate(browser *rod.Browser, downloadPageURL string) (*supplier.SubInfo, error) {
	sid, err := subHDSIDFromURL(downloadPageURL)
	if err != nil {
		return nil, wrapReason(ReasonDownloadGateChanged, err)
	}

	var lastErr error
	for challengeAttempt := 1; challengeAttempt <= maxCaptchaAttempts; challengeAttempt++ {
		subInfo, attemptErr := s.tryDownloadChallengeFromFreshPage(browser, sid, downloadPageURL)
		if attemptErr == nil {
			return subInfo, nil
		}

		lastErr = attemptErr
		s.log.Warningln(s.GetSupplierName(), "captcha attempt", challengeAttempt, "failed:", attemptErr)
		retryProbeFailure := shouldRetryDownloadGateProbe(attemptErr)
		retryCaptchaFailure := shouldRetryCaptchaOCRFailure(attemptErr)
		if retryProbeFailure == false && retryCaptchaFailure == false {
			return nil, attemptErr
		}
		if challengeAttempt < maxCaptchaAttempts {
			retryDelay := nextDownloadGateRetryDelay(challengeAttempt)
			if retryCaptchaFailure {
				s.log.Warningln(s.GetSupplierName(), "captcha ocr failure, open fresh page before retry", "delay", retryDelay)
			} else {
				s.log.Warningln(s.GetSupplierName(), "download gate transient failure, open fresh page before retry", "delay", retryDelay)
			}
			time.Sleep(retryDelay)
		}
	}

	if lastErr == nil {
		lastErr = wrapReason(ReasonDownloadFailed, fmt.Errorf("subhd captcha attempts exhausted"))
	}

	return nil, lastErr
}

func (s *Supplier) tryDownloadChallengeFromFreshPage(browser *rod.Browser, sid string, sourcePageURL string) (*supplier.SubInfo, error) {
	page, _, _, err := rod_helper.NewPageNavigate(browser, sourcePageURL, s.tt)
	if err != nil {
		return nil, wrapReason(ReasonProbeFailed, err)
	}
	defer func() {
		_ = page.Close()
	}()

	return s.tryDownloadChallenge(browser, page, sid, sourcePageURL)
}

func (s *Supplier) tryDownloadChallenge(browser *rod.Browser, page *rod.Page, sid string, sourcePageURL string) (*supplier.SubInfo, error) {
	firstResp, err := s.fetchDownloadGateResponse(page, sid, "")
	if err != nil {
		return nil, wrapReason(ReasonProbeFailed, err)
	}
	if firstResp.Success && firstResp.Pass && firstResp.URL != "" {
		s.log.Infoln(s.GetSupplierName(), "download gate passed without captcha")
		return s.subInfoFromDownloadURL(browser, page, firstResp.URL, sourcePageURL)
	}
	if firstResp.Success == false {
		return nil, wrapReason(ReasonDownloadGateChanged, fmt.Errorf(strings.TrimSpace(firstResp.Msg)))
	}
	if strings.TrimSpace(firstResp.Msg) == "" {
		return nil, wrapReason(ReasonDownloadGateChanged, fmt.Errorf("subhd captcha svg is empty"))
	}

	captchaText, err := s.solveCaptcha(page, firstResp.Msg)
	if err != nil {
		return nil, wrapReason(ReasonCaptchaOcrFailed, err)
	}

	verifyResp, err := s.fetchDownloadGateResponse(page, sid, captchaText)
	if err != nil {
		return nil, wrapReason(ReasonDownloadFailed, err)
	}
	if verifyResp.Success && verifyResp.Pass && verifyResp.URL != "" {
		s.log.Infoln(s.GetSupplierName(), "captcha accepted")
		return s.subInfoFromDownloadURL(browser, page, verifyResp.URL, sourcePageURL)
	}
	if verifyResp.Success == false {
		return nil, wrapReason(ReasonCaptchaOcrFailed, fmt.Errorf(strings.TrimSpace(verifyResp.Msg)))
	}
	if refreshedChallenge := captchaChallengeMessage(verifyResp); refreshedChallenge != "" {
		s.log.Warningln(s.GetSupplierName(), "captcha rejected, retry with refreshed challenge")
		secondCaptchaText, err := s.solveCaptcha(page, refreshedChallenge)
		if err != nil {
			return nil, wrapReason(ReasonCaptchaOcrFailed, err)
		}
		secondVerifyResp, err := s.fetchDownloadGateResponse(page, sid, secondCaptchaText)
		if err != nil {
			return nil, wrapReason(ReasonDownloadFailed, err)
		}
		if secondVerifyResp.Success && secondVerifyResp.Pass && secondVerifyResp.URL != "" {
			s.log.Infoln(s.GetSupplierName(), "captcha accepted after refreshed challenge")
			return s.subInfoFromDownloadURL(browser, page, secondVerifyResp.URL, sourcePageURL)
		}
		if secondVerifyResp.Success == false {
			return nil, wrapReason(ReasonCaptchaOcrFailed, fmt.Errorf(strings.TrimSpace(secondVerifyResp.Msg)))
		}
		if strings.TrimSpace(secondVerifyResp.Msg) != "" {
			return nil, wrapReason(ReasonCaptchaOcrFailed, fmt.Errorf("subhd captcha rejected after refreshed challenge: %s", secondCaptchaText))
		}
		return nil, wrapReason(ReasonDownloadFailed, fmt.Errorf("subhd download url missing after refreshed captcha verify"))
	}
	if strings.TrimSpace(verifyResp.Msg) != "" {
		return nil, wrapReason(ReasonCaptchaOcrFailed, fmt.Errorf("subhd captcha rejected: %s", captchaText))
	}
	return nil, wrapReason(ReasonDownloadFailed, fmt.Errorf("subhd download url missing after captcha verify"))
}

func (s *Supplier) subInfoFromDownloadURL(browser *rod.Browser, page *rod.Page, downloadURL string, sourcePageURL string) (*supplier.SubInfo, error) {
	fileData, fileName, err := s.downloadSubtitleFileThroughBrowser(browser, page, downloadURL)
	if err != nil {
		return nil, wrapReason(ReasonDownloadFailed, err)
	}
	ext := filepath.Ext(fileName)
	if ext == "" {
		if payloadURL, parseErr := pathFromURL(downloadURL); parseErr == nil {
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

func (s *Supplier) downloadSubtitleFileThroughBrowser(browser *rod.Browser, page *rod.Page, downloadURL string) ([]byte, string, error) {
	tmpDir := filepath.Join(pkg.DefTmpFolder(), "downloads")
	if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
		return nil, "", err
	}

	wait := browser.Timeout(30 * time.Second).WaitDownload(tmpDir)
	err := rod.Try(func() {
		if navigateErr := page.Navigate(downloadURL); navigateErr != nil {
			panic(navigateErr)
		}
	})
	if err != nil && shouldIgnoreSubHDDownloadNavigateError(err) == false {
		return nil, "", err
	}

	info := wait()
	if info == nil {
		return nil, "", fmt.Errorf("subhd browser download timed out")
	}

	downloadPath := filepath.Join(tmpDir, info.GUID)
	defer func() {
		_ = os.Remove(downloadPath)
	}()

	fileData, err := os.ReadFile(downloadPath)
	if err != nil {
		return nil, "", err
	}

	fileName := strings.TrimSpace(info.SuggestedFilename)
	if fileName == "" {
		fileName = fileNameFromDownloadMeta("", downloadURL, "subhd-subtitle")
	}

	return fileData, fileName, nil
}

func (s *Supplier) fetchDownloadGateResponse(page *rod.Page, sid string, cap string) (*downloadGateResponse, error) {
	var jsonBody string
	err := withTransientPageEvalRetry(func() error {
		jsonBody = page.MustEval(`async (sid, cap) => {
			const res = await fetch("/api/sub/down", {
				method: "POST",
				headers: { "Content-Type": "application/json" },
				body: JSON.stringify({ sid, cap }),
			});
			return JSON.stringify({ status: res.status, body: await res.text() });
		}`, sid, cap).Str()
		return nil
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
	if gateEnvelope.Status < 200 || gateEnvelope.Status >= 300 {
		return nil, fmt.Errorf("unexpected gate status %d", gateEnvelope.Status)
	}

	resp := downloadGateResponse{}
	if err := json.Unmarshal([]byte(gateEnvelope.Body), &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *Supplier) solveCaptcha(page *rod.Page, svgText string) (string, error) {
	if directText := extractCaptchaTextFromSVG(svgText); directText != "" {
		s.log.Infoln(s.GetSupplierName(), "captcha svg extracted:", directText)
		return directText, nil
	}

	pngDataURL, err := s.renderCaptchaPNG(page, svgText)
	if err != nil {
		return "", err
	}
	pngBytes, err := decodeDataURLBody(pngDataURL)
	if err != nil {
		return "", err
	}
	preparedPNGBytes, err := prepareCaptchaPNGForOCR(pngBytes)
	if err != nil {
		return "", err
	}
	rawText, err := s.runConfiguredCaptchaOCR(preparedPNGBytes)
	if err != nil {
		return "", err
	}
	captchaText, err := normalizeCaptchaText(rawText)
	if err != nil {
		return "", err
	}

	s.log.Infoln(s.GetSupplierName(), "captcha ocr raw:", strings.TrimSpace(rawText), "normalized:", captchaText)

	return captchaText, nil
}

func (s *Supplier) runConfiguredCaptchaOCR(pngBytes []byte) (string, error) {
	backend := configuredCaptchaBackend()
	s.log.Infoln(s.GetSupplierName(), "captcha using ocr backend:", backend)
	switch backend {
	case "external":
		return runExternalCaptchaOCR(pngBytes)
	case "ddddocr":
		return runDDDDOCR(pngBytes)
	default:
		return "", fmt.Errorf("unsupported subhd ocr backend %q", backend)
	}
}

func configuredCaptchaBackend() string {
	cfg := settings.Get().SubtitleSources.SubHDSettings
	backend := strings.ToLower(strings.TrimSpace(cfg.OCRBackend))
	if _, ok := captchaBackendWhitelist[backend]; ok {
		return backend
	}
	return "ddddocr"
}

func extractCaptchaTextFromSVG(svgText string) string {
	if directText := extractCaptchaTextFromSVGXML(svgText); directText != "" {
		return directText
	}

	matches := captchaNodePattern.FindAllStringSubmatch(svgText, -1)
	if len(matches) == 0 {
		return ""
	}

	var builder strings.Builder
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		cleaned := captchaCleanPattern.ReplaceAllString(strings.TrimSpace(match[1]), "")
		if cleaned == "" {
			continue
		}
		builder.WriteString(cleaned)
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

func extractCaptchaTextFromSVGXML(svgText string) string {
	decoder := xml.NewDecoder(strings.NewReader(svgText))
	decoder.Strict = false

	var builder strings.Builder
	var textDepth int

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch value := token.(type) {
		case xml.StartElement:
			if isSVGTextElement(value.Name.Local) {
				textDepth++
			}
		case xml.EndElement:
			if isSVGTextElement(value.Name.Local) && textDepth > 0 {
				textDepth--
			}
		case xml.CharData:
			if textDepth == 0 {
				continue
			}
			cleaned := captchaCleanPattern.ReplaceAllString(
				strings.TrimSpace(html.UnescapeString(string(value))),
				"",
			)
			if cleaned == "" {
				continue
			}
			builder.WriteString(cleaned)
		}
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

func isSVGTextElement(localName string) bool {
	switch strings.ToLower(strings.TrimSpace(localName)) {
	case "text", "tspan":
		return true
	default:
		return false
	}
}

func shouldRetryDownloadGateProbe(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unexpected gate status 500") ||
		strings.Contains(msg, "unexpected gate status 502") ||
		strings.Contains(msg, "unexpected gate status 503") ||
		strings.Contains(msg, "unexpected gate status 504") ||
		isTransientDownloadGateTransportError(msg) ||
		strings.Contains(msg, "object reference chain is too long") ||
		isExpiredTemporaryDownloadGateError(err.Error())
}

func isTransientDownloadGateTransportError(msg string) bool {
	if msg == "" {
		return false
	}

	return strings.Contains(msg, "err_connection_closed") ||
		strings.Contains(msg, "connection reset by peer") ||
		strings.Contains(msg, "\": eof") ||
		strings.HasSuffix(msg, ": eof") ||
		strings.Contains(msg, " eof")
}

func shouldRetryCaptchaOCRFailure(err error) bool {
	if err == nil {
		return false
	}

	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	if (reasonOf(err) == ReasonCaptchaOcrFailed || strings.Contains(msg, "captcha_ocr_failed")) &&
		strings.Contains(msg, "unexpected captcha ocr output") {
		return true
	}
	if (reasonOf(err) == ReasonCaptchaOcrFailed || strings.Contains(msg, "captcha_ocr_failed")) &&
		strings.Contains(msg, "captcha rejected after refreshed challenge") {
		return true
	}

	unwrapped := errors.Unwrap(err)
	if unwrapped == nil {
		return false
	}
	unwrappedMsg := strings.ToLower(strings.TrimSpace(unwrapped.Error()))
	return strings.Contains(unwrappedMsg, "unexpected captcha ocr output") ||
		strings.Contains(unwrappedMsg, "captcha rejected after refreshed challenge")
}

func isExpiredTemporaryDownloadGateError(msg string) bool {
	normalized := strings.TrimSpace(msg)
	if normalized == "" {
		return false
	}
	if strings.Contains(normalized, "时间过长本临时页面已经失效") {
		return true
	}
	if strings.Contains(normalized, "鏃堕棿杩囬暱") {
		return true
	}

	lower := strings.ToLower(normalized)
	return strings.Contains(lower, "temporary page") && strings.Contains(lower, "expired")
}

func nextDownloadGateRetryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	return time.Duration(attempt) * subHDGateRetryBaseDelay
}

func captchaChallengeMessage(resp *downloadGateResponse) string {
	if resp == nil || resp.Success == false || resp.Pass {
		return ""
	}

	msg := strings.TrimSpace(resp.Msg)
	if msg == "" {
		return ""
	}
	if strings.Contains(strings.ToLower(msg), "<svg") {
		return msg
	}

	return ""
}

func (s *Supplier) renderCaptchaPNG(page *rod.Page, svgText string) (string, error) {
	var pngDataURL string
	err := withTransientPageEvalRetry(func() error {
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
					const v = lum > 210 ? 255 : 0;
					imageData.data[i] = v;
					imageData.data[i + 1] = v;
					imageData.data[i + 2] = v;
					imageData.data[i + 3] = 255;
				}
				ctx.putImageData(imageData, 0, 0);

				return canvas.toDataURL("image/png");
			} finally {
				URL.revokeObjectURL(objectURL);
			}
		}`, svgText).Str()
		return nil
	})
	if err != nil {
		return "", err
	}

	return pngDataURL, nil
}

func withTransientPageEvalRetry(run func() error) error {
	var lastErr error
	for attempt := 1; attempt <= subHDPageEvalRetryAttempts; attempt++ {
		lastErr = rod.Try(func() {
			if err := run(); err != nil {
				panic(err)
			}
		})
		lastErr = normalizeTransientPageEvalError(lastErr)
		if lastErr == nil {
			return nil
		}
		if shouldRetryTransientPageEval(lastErr) == false || attempt >= subHDPageEvalRetryAttempts {
			return lastErr
		}
		time.Sleep(subHDPageEvalRetryDelay)
	}

	return lastErr
}

func normalizeTransientPageEvalError(err error) error {
	if shouldRetryTransientPageEval(err) == false {
		return err
	}
	return fmt.Errorf("object reference chain is too long")
}

func shouldRetryTransientPageEval(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "object reference chain is too long")
}

func shouldIgnoreSubHDDownloadNavigateError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "net::err_aborted") ||
		strings.Contains(msg, "err_aborted") ||
		strings.Contains(msg, "object reference chain is too long")
}

func runDDDDOCR(imageBytes []byte) (string, error) {
	pythonExe := strings.TrimSpace(os.Getenv("CSF_DDDDOCR_PYTHON"))
	if pythonExe == "" {
		pythonExe = "python3"
	}
	if _, err := exec.LookPath(pythonExe); err != nil {
		return "", fmt.Errorf("%s not found in PATH: %w", pythonExe, err)
	}

	cmd := exec.Command(
		pythonExe,
		"-c",
		`import sys
import ddddocr

ocr = ddddocr.DdddOcr(show_ad=False)
sys.stdout.write(ocr.classification(sys.stdin.buffer.read()))
`,
	)
	cmd.Stdin = bytes.NewReader(imageBytes)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("ddddocr failed: %w: %s", err, strings.TrimSpace(string(out)))
	}

	return string(out), nil
}

func prepareCaptchaPNGForOCR(pngBytes []byte) ([]byte, error) {
	srcImg, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, err
	}

	bounds := srcImg.Bounds()
	binaryImg := image.NewNRGBA(bounds)
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X-1, bounds.Min.Y-1
	hasForeground := false

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := srcImg.At(x, y).RGBA()
			lum := 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
			value := uint8(255)
			if lum <= captchaForegroundThreshold {
				value = 0
				hasForeground = true
				if x < minX {
					minX = x
				}
				if y < minY {
					minY = y
				}
				if x > maxX {
					maxX = x
				}
				if y > maxY {
					maxY = y
				}
			}
			binaryImg.Set(x, y, color.NRGBA{R: value, G: value, B: value, A: 255})
		}
	}

	if hasForeground == false {
		return pngBytes, nil
	}

	cropRect := image.Rect(
		maxIntLocal(bounds.Min.X, minX-captchaCropPadding),
		maxIntLocal(bounds.Min.Y, minY-captchaCropPadding),
		minIntLocal(bounds.Max.X, maxX+captchaCropPadding+1),
		minIntLocal(bounds.Max.Y, maxY+captchaCropPadding+1),
	)
	if cropRect.Dx() <= 0 || cropRect.Dy() <= 0 {
		return pngBytes, nil
	}

	scaled := image.NewNRGBA(image.Rect(0, 0, cropRect.Dx()*captchaScaleFactor, cropRect.Dy()*captchaScaleFactor))
	for y := 0; y < scaled.Bounds().Dy(); y++ {
		srcY := cropRect.Min.Y + y/captchaScaleFactor
		for x := 0; x < scaled.Bounds().Dx(); x++ {
			srcX := cropRect.Min.X + x/captchaScaleFactor
			scaled.Set(x, y, binaryImg.At(srcX, srcY))
		}
	}

	var out bytes.Buffer
	if err := png.Encode(&out, scaled); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

func runExternalCaptchaOCR(imageBytes []byte) (string, error) {
	cfg := settings.Get().SubtitleSources.SubHDSettings
	if strings.TrimSpace(cfg.ExternalOCRURL) == "" {
		return "", fmt.Errorf("subhd external ocr url is empty")
	}

	client, err := pkg.NewHttpClient()
	if err != nil {
		return "", err
	}

	reqBody := externalCaptchaRequest{
		ImageBase64: base64.StdEncoding.EncodeToString(imageBytes),
		MimeType:    "image/png",
	}
	var respBody externalCaptchaResponse
	resp, err := client.R().
		SetBody(reqBody).
		SetResult(&respBody).
		Post(strings.TrimSpace(cfg.ExternalOCRURL))
	if err != nil {
		return "", err
	}
	if resp.StatusCode() < 200 || resp.StatusCode() >= 300 {
		return "", fmt.Errorf("external ocr status %d: %s", resp.StatusCode(), strings.TrimSpace(resp.String()))
	}
	return respBody.Text, nil
}

func normalizeCaptchaText(raw string) (string, error) {
	if matched := captchaTextPattern.FindString(raw); matched != "" {
		return matched, nil
	}

	cleaned := captchaCleanPattern.ReplaceAllString(strings.TrimSpace(raw), "")
	if len(cleaned) >= 4 {
		if len(cleaned) > 5 {
			cleaned = cleaned[:5]
		}
		return cleaned, nil
	}

	return "", fmt.Errorf("unexpected captcha OCR output %q", strings.TrimSpace(raw))
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

func minIntLocal(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxIntLocal(a, b int) int {
	if a > b {
		return a
	}
	return b
}
