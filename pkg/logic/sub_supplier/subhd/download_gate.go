package subhd

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	neturl "net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/rod_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/go-rod/rod"
)

const maxCaptchaAttempts = 5

var (
	captchaTextPattern  = regexp.MustCompile(`[A-Za-z0-9]{4,5}`)
	captchaCleanPattern = regexp.MustCompile(`[^A-Za-z0-9]+`)
	captchaNodePattern  = regexp.MustCompile(`(?is)<(?:text|tspan)\b[^>]*>(.*?)</(?:text|tspan)>`)
)

type downloadGateResponse struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
	Pass    bool   `json:"pass"`
	URL     string `json:"url"`
}

type fetchedDownload struct {
	StatusCode         int    `json:"statusCode"`
	DataURL            string `json:"dataUrl"`
	ContentType        string `json:"contentType"`
	ContentDisposition string `json:"contentDisposition"`
	FinalURL           string `json:"finalUrl"`
}

func (s *Supplier) downloadSubFileViaGate(browser *rod.Browser, downloadPageURL string) (*supplier.SubInfo, error) {
	sid, err := subHDSIDFromURL(downloadPageURL)
	if err != nil {
		return nil, wrapReason(ReasonDownloadGateChanged, err)
	}

	page, _, _, err := rod_helper.NewPageNavigate(browser, downloadPageURL, s.tt)
	if err != nil {
		return nil, wrapReason(ReasonProbeFailed, err)
	}
	defer func() {
		_ = page.Close()
	}()

	var lastErr error
	for attempt := 1; attempt <= maxCaptchaAttempts; attempt++ {
		subInfo, attemptErr := s.tryDownloadFromPage(page, sid, downloadPageURL)
		if attemptErr == nil {
			return subInfo, nil
		}

		lastErr = attemptErr
		s.log.Warningln(s.GetSupplierName(), "captcha attempt", attempt, "failed:", attemptErr)
	}

	if lastErr == nil {
		lastErr = wrapReason(ReasonDownloadFailed, fmt.Errorf("subhd captcha attempts exhausted"))
	}

	return nil, lastErr
}

func (s *Supplier) tryDownloadFromPage(page *rod.Page, sid string, sourcePageURL string) (*supplier.SubInfo, error) {
	firstResp, err := s.fetchDownloadGateResponse(page, sid, "")
	if err != nil {
		return nil, wrapReason(ReasonProbeFailed, err)
	}
	if firstResp.Success && firstResp.Pass && firstResp.URL != "" {
		return s.subInfoFromDownloadURL(page, firstResp.URL, sourcePageURL)
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
		return s.subInfoFromDownloadURL(page, verifyResp.URL, sourcePageURL)
	}
	if verifyResp.Success == false {
		return nil, wrapReason(ReasonCaptchaOcrFailed, fmt.Errorf(strings.TrimSpace(verifyResp.Msg)))
	}
	if strings.TrimSpace(verifyResp.Msg) != "" {
		return nil, wrapReason(ReasonCaptchaOcrFailed, fmt.Errorf("subhd captcha rejected: %s", captchaText))
	}

	return nil, wrapReason(ReasonDownloadFailed, fmt.Errorf("subhd download url missing after captcha verify"))
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
	if gateEnvelope.Status < 200 || gateEnvelope.Status >= 300 {
		return nil, fmt.Errorf("unexpected gate status %d", gateEnvelope.Status)
	}

	resp := downloadGateResponse{}
	if err := json.Unmarshal([]byte(gateEnvelope.Body), &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

func (s *Supplier) fetchDownloadPayload(page *rod.Page, downloadURL string) (*fetchedDownload, error) {
	var jsonBody string
	err := rod.Try(func() {
		jsonBody = page.MustEval(`async (downloadURL) => {
			const res = await fetch(downloadURL, {
				method: "GET",
				credentials: "omit",
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

func (s *Supplier) solveCaptcha(page *rod.Page, svgText string) (string, error) {
	if directText := extractCaptchaTextFromSVG(svgText); directText != "" {
		s.log.Infoln(s.GetSupplierName(), "captcha svg extracted:", directText)
		return directText, nil
	}

	tmpFile, err := os.CreateTemp(pkg.DefTmpFolder(), "subhd-captcha-*.png")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	defer func() { _ = os.Remove(tmpPath) }()

	pngDataURL, err := s.renderCaptchaPNG(page, svgText)
	if err != nil {
		return "", err
	}
	pngBytes, err := decodeDataURLBody(pngDataURL)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(tmpPath, pngBytes, 0o600); err != nil {
		return "", err
	}

	rawText, err := runTesseract(tmpPath)
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

func extractCaptchaTextFromSVG(svgText string) string {
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
	})
	if err != nil {
		return "", err
	}

	return pngDataURL, nil
}

func runTesseract(imagePath string) (string, error) {
	if _, err := exec.LookPath("tesseract"); err != nil {
		return "", fmt.Errorf("tesseract not found in PATH: %w", err)
	}

	cmd := exec.Command(
		"tesseract",
		imagePath,
		"stdout",
		"--psm", "7",
		"-c", "tessedit_char_whitelist=ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789",
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("tesseract failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	return string(out), nil
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
