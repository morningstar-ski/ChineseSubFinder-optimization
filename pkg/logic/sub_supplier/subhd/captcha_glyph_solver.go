package subhd

import (
	"fmt"
	"os"
	"strings"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/go-rod/rod"
)

type glyphOCRCaptchaSolver struct{}

func (glyphOCRCaptchaSolver) Solve(s *Supplier, page *rod.Page, svgText string) (string, error) {
	bundle, err := glyphOCRCaptchaSolver{}.SolveBundle(s, page, svgText)
	if err != nil {
		return "", err
	}
	if len(bundle.Primary) > 0 {
		return bundle.Primary[0], nil
	}
	if len(bundle.Fallback) > 0 {
		return bundle.Fallback[0], nil
	}
	return "", fmt.Errorf("unexpected empty captcha candidate bundle")
}

func (glyphOCRCaptchaSolver) SolveCandidates(s *Supplier, page *rod.Page, svgText string) ([]string, error) {
	bundle, err := glyphOCRCaptchaSolver{}.SolveBundle(s, page, svgText)
	if err != nil {
		return nil, err
	}
	if len(bundle.Primary) > 0 {
		return append([]string(nil), bundle.Primary...), nil
	}
	return append([]string(nil), bundle.Fallback...), nil
}

func (glyphOCRCaptchaSolver) SolveBundle(s *Supplier, page *rod.Page, svgText string) (*captchaCandidateBundle, error) {
	if directText := extractCaptchaTextFromSVG(svgText); directText != "" {
		s.log.Infoln(s.GetSupplierName(), "captcha svg extracted:", directText)
		return &captchaCandidateBundle{
			Primary: []string{directText},
			Simple:  true,
		}, nil
	}

	tmpFile, err := os.CreateTemp(pkg.DefTmpFolder(), "subhd-captcha-*.png")
	if err != nil {
		return nil, err
	}
	tmpPath := tmpFile.Name()
	_ = tmpFile.Close()
	defer func() { _ = os.Remove(tmpPath) }()

	pngDataURL, err := s.renderCaptchaPNG(page, svgText)
	if err != nil {
		return nil, err
	}
	pngBytes, err := decodeDataURLBody(pngDataURL)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(tmpPath, pngBytes, 0o600); err != nil {
		return nil, err
	}

	debugPrefix, dumpErr := dumpCaptchaDebugArtifacts(svgText, pngBytes, "")
	if dumpErr != nil {
		s.log.Warningln(s.GetSupplierName(), "captcha debug dump init failed:", dumpErr)
	} else {
		s.log.Infoln(s.GetSupplierName(), "captcha debug dump:", debugPrefix)
	}

	wholeCandidates := make([]string, 0, 4)
	glyphCandidates := make([]string, 0, maxCaptchaCandidateCount)
	var firstErr error

	rawText, wholeErr := runTesseract(tmpPath, 8)
	if dumpErr == nil {
		if writeErr := os.WriteFile(debugPrefix+"-ocr.txt", []byte(strings.TrimSpace(rawText)), 0o600); writeErr != nil {
			s.log.Warningln(s.GetSupplierName(), "captcha debug dump ocr failed:", writeErr)
		}
	}
	if wholeErr != nil {
		s.log.Warningln(s.GetSupplierName(), "captcha ocr failed raw:", strings.TrimSpace(rawText), "err:", wholeErr)
		firstErr = wholeErr
	} else {
		normalized, normErr := recognizeCaptchaTextWithRunner(tmpPath, rawText, buildEnhancedGlyphImage, runTesseract)
		if normErr != nil {
			s.log.Warningln(s.GetSupplierName(), "captcha ocr normalize failed raw:", strings.TrimSpace(rawText), "dump:", debugPrefix)
			firstErr = normErr
		} else {
			wholeCandidates = collectLightweightWholeCandidates(normalized)
			if len(wholeCandidates) > 0 {
				s.log.Infoln(s.GetSupplierName(), "captcha ocr raw:", strings.TrimSpace(rawText), "normalized:", wholeCandidates[0], "candidates:", strings.Join(wholeCandidates, ","))
			}
		}
	}

	glyphTexts, glyphErr := s.solveCaptchaByGlyphs(page, svgText, debugPrefix)
	if glyphErr != nil {
		s.log.Warningln(s.GetSupplierName(), "captcha glyph ocr failed:", glyphErr)
		if firstErr == nil && len(wholeCandidates) == 0 {
			firstErr = glyphErr
		}
	} else {
		glyphCandidates = collectLightweightGlyphCandidates(glyphTexts)
	}

	primaryCandidates, fallbackCandidates := assembleLightweightCaptchaCandidateLanes(glyphCandidates, wholeCandidates, s.maxCaptchaVerifyCandidates())
	if len(primaryCandidates) == 0 && len(fallbackCandidates) == 0 {
		if firstErr != nil {
			return nil, firstErr
		}
		return nil, fmt.Errorf("unexpected captcha OCR output %q", strings.TrimSpace(rawText))
	}

	return &captchaCandidateBundle{
		Primary:  primaryCandidates,
		Fallback: fallbackCandidates,
		Simple:   true,
	}, nil
}

func collectLightweightWholeCandidates(candidates []string) []string {
	out := make([]string, 0, len(candidates)*2)
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if len([]rune(candidate)) < expectedCaptchaTextLength {
			continue
		}
		runes := []rune(candidate)
		if len(runes) > expectedCaptchaTextLength {
			candidate = string(runes[:expectedCaptchaTextLength])
		}
		out = appendUniqueStringCandidates(out, candidate)
		upper := uppercaseCaptchaCandidate(candidate)
		if upper != candidate {
			out = appendUniqueStringCandidates(out, upper)
		}
	}
	return out
}

func collectLightweightGlyphCandidates(candidates []string) []string {
	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if len([]rune(candidate)) != expectedCaptchaTextLength {
			continue
		}
		out = appendUniqueStringCandidates(out, candidate)
		upper := uppercaseCaptchaCandidate(candidate)
		if upper != candidate && countLowercaseASCII(candidate) >= 2 {
			out = appendUniqueStringCandidates(out, upper)
		}
	}
	return out
}

func assembleLightweightCaptchaCandidateLanes(glyphCandidates []string, wholeCandidates []string, verifyLimit int) ([]string, []string) {
	primary := make([]string, 0, maxCaptchaCandidateCount)
	fallback := make([]string, 0, maxCaptchaCandidateCount)

	if len(glyphCandidates) > 0 {
		primary = appendUniqueStringCandidates(primary, glyphCandidates...)
	}

	for _, candidate := range wholeCandidates {
		if len(primary) == 0 {
			primary = appendUniqueStringCandidates(primary, candidate)
			continue
		}
		fallback = appendUniqueStringCandidates(fallback, candidate)
	}

	if len(primary) == 0 {
		primary = appendUniqueStringCandidates(primary, fallback...)
		fallback = nil
	}

	primary = limitSimpleCaptchaCandidates(primary, maxCaptchaCandidateCount)
	fallback = limitSimpleFallbackCaptchaCandidates(primary, fallback, maxCaptchaCandidateCount)

	if len(primary) > 0 && verifyLimit > 0 && len(primary) < verifyLimit && len(fallback) > 0 {
		need := verifyLimit - len(primary)
		if need > len(fallback) {
			need = len(fallback)
		}
		primary = append(primary, fallback[:need]...)
		fallback = fallback[need:]
	}

	return primary, fallback
}
