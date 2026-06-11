package subhd

import (
	"errors"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/internal/models"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	common2 "github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/go-resty/resty/v2"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/sirupsen/logrus"
)

type stubCaptchaSolver struct {
	gotSVG string
	text   string
	err    error
}

func (s *stubCaptchaSolver) Solve(_ *Supplier, _ *rod.Page, svgText string) (string, error) {
	s.gotSVG = svgText
	return s.text, s.err
}

type stubCaptchaBundleSolver struct {
	gotSVG string
	bundle *captchaCandidateBundle
	err    error
}

func (s *stubCaptchaBundleSolver) Solve(_ *Supplier, _ *rod.Page, svgText string) (string, error) {
	s.gotSVG = svgText
	if s.err != nil {
		return "", s.err
	}
	if s.bundle != nil && len(s.bundle.Primary) > 0 {
		return s.bundle.Primary[0], nil
	}
	return "", nil
}

func (s *stubCaptchaBundleSolver) SolveBundle(_ *Supplier, _ *rod.Page, svgText string) (*captchaCandidateBundle, error) {
	s.gotSVG = svgText
	return s.bundle, s.err
}

func TestOverDailyDownloadLimitTreatsNegativeLimitAsUnlimited(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.SuppliersSettings.SubHD.DailyDownloadLimit = -1

	supplier := &Supplier{log: logrus.New()}
	if supplier.OverDailyDownloadLimit() {
		t.Fatal("OverDailyDownloadLimit() = true; want false for unlimited limit")
	}
}

func TestMaxCaptchaAttemptsCapsConfiguredValue(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.SubtitleSources.SubHDSettings.MaxCaptchaAttempts = 7

	supplier := &Supplier{log: logrus.New()}
	if got := supplier.maxCaptchaAttempts(); got != 4 {
		t.Fatalf("maxCaptchaAttempts() = %d; want 4", got)
	}
}

func TestMaxCaptchaAttemptsFallsBackToDefault(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.SubtitleSources.SubHDSettings.MaxCaptchaAttempts = 0

	supplier := &Supplier{log: logrus.New()}
	if got := supplier.maxCaptchaAttempts(); got != 4 {
		t.Fatalf("maxCaptchaAttempts() = %d; want 4", got)
	}
}

func TestFinalizeDownloadAttemptsReturnsLastDownloadError(t *testing.T) {
	wantErr := wrapReason(ReasonCaptchaOcrFailed, errors.New("subhd captcha rejected"))

	gotSubs, err := finalizeDownloadAttempts(nil, wantErr, false, "movie keyword")
	if err == nil {
		t.Fatal("finalizeDownloadAttempts() error = nil; want captcha error")
	}
	if err.Error() != wantErr.Error() {
		t.Fatalf("finalizeDownloadAttempts() error = %q; want %q", err.Error(), wantErr.Error())
	}
	if gotSubs != nil {
		t.Fatalf("finalizeDownloadAttempts() subs = %#v; want nil", gotSubs)
	}
}

func TestFinalizeDownloadAttemptsPrefersDeadGateLoopContext(t *testing.T) {
	lastErr := wrapReason(ReasonCaptchaOcrFailed, errors.New("subhd captcha rejected"))

	_, err := finalizeDownloadAttempts(nil, lastErr, true, "movie keyword")
	if err == nil {
		t.Fatal("finalizeDownloadAttempts() error = nil; want dead gate error")
	}
	if strings.Contains(err.Error(), ReasonDownloadGateChanged) == false {
		t.Fatalf("finalizeDownloadAttempts() error = %q; want %q marker", err.Error(), ReasonDownloadGateChanged)
	}
	if strings.Contains(err.Error(), "movie keyword") == false {
		t.Fatalf("finalizeDownloadAttempts() error = %q; want context", err.Error())
	}
}

func TestMaxCaptchaAttemptsKeepsSmallerConfiguredValue(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.SubtitleSources.SubHDSettings.MaxCaptchaAttempts = 3

	supplier := &Supplier{log: logrus.New()}
	if got := supplier.maxCaptchaAttempts(); got != 3 {
		t.Fatalf("maxCaptchaAttempts() = %d; want 3", got)
	}
}

func TestMaxCaptchaVerifyCandidatesUsesConfiguredPositiveValue(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.SubtitleSources.SubHDSettings.MaxVerifyCandidates = 2

	supplier := &Supplier{log: logrus.New()}
	if got := supplier.maxCaptchaVerifyCandidates(); got != 2 {
		t.Fatalf("maxCaptchaVerifyCandidates() = %d; want 2", got)
	}
}

func TestMaxCaptchaVerifyCandidatesCapsAtCandidateCount(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.SubtitleSources.SubHDSettings.MaxVerifyCandidates = maxCaptchaCandidateCount + 5

	supplier := &Supplier{log: logrus.New()}
	if got := supplier.maxCaptchaVerifyCandidates(); got != maxCaptchaCandidateCount {
		t.Fatalf("maxCaptchaVerifyCandidates() = %d; want %d", got, maxCaptchaCandidateCount)
	}
}

func TestSolveCaptchaUsesConfiguredSolver(t *testing.T) {
	solver := &stubCaptchaSolver{text: "AB12"}
	supplier := &Supplier{
		log:           logrus.New(),
		captchaSolver: solver,
	}

	got, err := supplier.solveCaptcha(nil, "<svg>code</svg>")
	if err != nil {
		t.Fatalf("solveCaptcha() error = %v", err)
	}
	if got != "AB12" {
		t.Fatalf("solveCaptcha() = %q; want %q", got, "AB12")
	}
	if solver.gotSVG != "<svg>code</svg>" {
		t.Fatalf("solver got svg = %q", solver.gotSVG)
	}
}

func TestBuildCaptchaCandidateBundleUsesBundleSolver(t *testing.T) {
	solver := &stubCaptchaBundleSolver{
		bundle: &captchaCandidateBundle{
			Primary:  []string{"AB12", "A812"},
			Fallback: []string{"AB1Z"},
			Simple:   true,
		},
	}
	supplier := &Supplier{
		log:           logrus.New(),
		captchaSolver: solver,
	}

	got, err := buildCaptchaCandidateBundle(supplier, nil, "<svg></svg>")
	if err != nil {
		t.Fatalf("buildCaptchaCandidateBundle() error = %v", err)
	}
	if got == nil || got.Simple == false {
		t.Fatalf("buildCaptchaCandidateBundle() = %#v; want simple bundle", got)
	}
	if strings.Join(got.Primary, ",") != "AB12,A812" {
		t.Fatalf("buildCaptchaCandidateBundle() primary = %v", got.Primary)
	}
	if strings.Join(got.Fallback, ",") != "AB1Z" {
		t.Fatalf("buildCaptchaCandidateBundle() fallback = %v", got.Fallback)
	}
	if solver.gotSVG != "<svg></svg>" {
		t.Fatalf("bundle solver got svg = %q", solver.gotSVG)
	}
}

func TestBuildCaptchaVerifyPlanForBundleKeepsSimplePrimaryFallbackOrder(t *testing.T) {
	plan := buildCaptchaVerifyPlanForBundle(&captchaCandidateBundle{
		Primary:  []string{"A0B1", "AOB1"},
		Fallback: []string{"ZTVQ", "ABCD", "ZTVQ"},
	}, 3)

	if got := strings.Join(captchaVerifyPlanTexts(plan, "primary"), ","); got != "A0B1,AOB1" {
		t.Fatalf("primary plan = %q; want %q", got, "A0B1,AOB1")
	}
	if got := strings.Join(captchaVerifyPlanTexts(plan, "fallback"), ","); got != "ZTVQ,ABCD" {
		t.Fatalf("fallback plan = %q; want %q", got, "ZTVQ,ABCD")
	}
}

func TestPreferSharedSubHDCodeCandidatePrependsSharedCode(t *testing.T) {
	original := common2.SubhdCode
	common2.SubhdCode = "z9q1"
	t.Cleanup(func() {
		common2.SubhdCode = original
	})

	got := preferSharedSubHDCodeCandidate(&captchaCandidateBundle{
		Primary:  []string{"A0B1", "Z9Q1"},
		Fallback: []string{"ABCD", "Z9Q1"},
	})

	if strings.Join(got.Primary, ",") != "Z9Q1,A0B1" {
		t.Fatalf("primary = %v; want [Z9Q1 A0B1]", got.Primary)
	}
	if strings.Join(got.Fallback, ",") != "ABCD" {
		t.Fatalf("fallback = %v; want [ABCD]", got.Fallback)
	}
}

func TestDumpCaptchaDebugArtifactsWritesFiles(t *testing.T) {
	t.Setenv("TEMP", t.TempDir())
	t.Setenv("TMP", t.TempDir())

	basePath, err := dumpCaptchaDebugArtifacts("<svg>code</svg>", []byte{1, 2, 3}, "AB12")
	if err != nil {
		t.Fatalf("dumpCaptchaDebugArtifacts() error = %v", err)
	}

	for _, suffix := range []string{".svg", ".png", "-ocr.txt"} {
		if _, err := os.Stat(basePath + suffix); err != nil {
			t.Fatalf("expected %s to exist: %v", suffix, err)
		}
	}
}

func TestShouldRetrySubHDGateAttemptStopsOnExpiredOr500(t *testing.T) {
	if shouldRetrySubHDGateAttempt(errors.New("download_gate_changed: expired gate")) {
		t.Fatal("expected expired gate error to stop retries")
	}
	if shouldRetrySubHDGateAttempt(errors.New("probe_failed: unexpected gate status 500 body \"oops\"")) {
		t.Fatal("expected gate 500 to stop retries")
	}
	if shouldRetrySubHDGateAttempt(errors.New("captcha_ocr_failed: subhd captcha rejected")) == false {
		t.Fatal("expected retryable captcha error to continue retries")
	}
}

func TestShouldRefreshSubHDGateTreatsExpiredAnd500AsRefreshable(t *testing.T) {
	if shouldRefreshSubHDGate(errors.New("download_gate_changed: expired gate")) == false {
		t.Fatal("expected expired gate error to refresh")
	}
	if shouldRefreshSubHDGate(errors.New("probe_failed: unexpected gate status 500")) == false {
		t.Fatal("expected gate 500 to refresh")
	}
	if shouldRefreshSubHDGate(errors.New("captcha_ocr_failed: subhd captcha rejected")) == true {
		t.Fatal("unexpected refresh for generic captcha reject")
	}
}

func TestShouldUseHTTPFallbackAfterPageError(t *testing.T) {
	if shouldUseHTTPFallbackAfterPageError(wrapReason(ReasonProbeFailed, errors.New("unexpected gate status 500"))) == false {
		t.Fatal("expected probe failure to allow http fallback")
	}
	if shouldUseHTTPFallbackAfterPageError(wrapReason(ReasonCaptchaOcrFailed, errors.New("subhd captcha rejected: ABCD"))) == false {
		t.Fatal("expected http fallback after browser captcha flow failure")
	}
	if shouldUseHTTPFallbackAfterPageError(wrapReason(ReasonDownloadGateChanged, errors.New("expired gate"))) == true {
		t.Fatal("unexpected http fallback after stale gate failure")
	}
}

func TestShouldRefreshGateBeforeHTTPFallbackAfterPageError(t *testing.T) {
	if shouldRefreshGateBeforeHTTPFallbackAfterPageError(wrapReason(ReasonCaptchaOcrFailed, errors.New("subhd captcha rejected: ABCD"))) == false {
		t.Fatal("expected captcha OCR failure to refresh gate before http fallback")
	}
	if shouldRefreshGateBeforeHTTPFallbackAfterPageError(wrapReason(ReasonProbeFailed, errors.New("unexpected gate status 500"))) == true {
		t.Fatal("unexpected pre-fallback refresh for generic probe failure")
	}
}

func TestShouldRetryDetailPageContextAfterPageError(t *testing.T) {
	if shouldRetryDetailPageContextAfterPageError(wrapReason(ReasonProbeFailed, errors.New("unexpected gate status 500"))) == false {
		t.Fatal("expected probe failure to retry detail page context")
	}
	if shouldRetryDetailPageContextAfterPageError(wrapReason(ReasonCaptchaOcrFailed, errors.New("subhd captcha rejected"))) == true {
		t.Fatal("unexpected detail page retry after captcha reject")
	}
}

func TestDecodeDownloadGateResponseBodyAcceptsStructured500JSON(t *testing.T) {
	body := []byte(`{"success":false,"error":"Internal Server Error","message":"structured 500","msg":"structured 500","pass":false,"url":""}`)
	got, err := decodeDownloadGateResponseBody(500, body)
	if err != nil {
		t.Fatalf("decodeDownloadGateResponseBody() error = %v", err)
	}
	if got.Success != false || strings.TrimSpace(got.Msg) == "" {
		t.Fatalf("decodeDownloadGateResponseBody() = %#v; want structured failure with message", got)
	}
}

func TestDecodeDownloadGateResponseBodyRejectsOpaque500Body(t *testing.T) {
	if _, err := decodeDownloadGateResponseBody(500, []byte("oops")); err == nil {
		t.Fatal("decodeDownloadGateResponseBody() error = nil; want opaque 500 failure")
	}
}

func TestDetailPageURLFromDownloadGateURL(t *testing.T) {
	got, err := detailPageURLFromDownloadGateURL("https://subhd.me/down/Ks9kp5")
	if err != nil {
		t.Fatalf("detailPageURLFromDownloadGateURL() error = %v", err)
	}
	if got != "https://subhd.me/a/Ks9kp5" {
		t.Fatalf("detailPageURLFromDownloadGateURL() = %q; want %q", got, "https://subhd.me/a/Ks9kp5")
	}
}

func TestRefreshDownloadGateURLByErrorLoadsDetailPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/a/Ks9kp5" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`
<html><body>
  <a class="btn btn-danger down" href="/down/N2qlBz">婵犵數鍋為崹鍫曞箰閹间緡鏁勯柛顐ｇ贩瑜版帒鐐婇柍瑙勫劤娴滅偓绻涢幋鐐垫噽婵炲牊姊荤槐鎺楁偐闂堟稑骞嬮悗娈垮櫘閸撶喎鐣烽崼鏇熸櫆缂佹稑顑呮禒?/a>
</body></html>`))
	}))
	defer server.Close()

	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.SuppliersSettings.SubHD.RootUrl = server.URL

	supplier := &Supplier{log: logrus.New()}
	httpClient := resty.New()

	got, err := supplier.refreshDownloadGateURLByError(httpClient, server.URL+"/down/Ks9kp5", errors.New("download_gate_changed: expired gate"))
	if err != nil {
		t.Fatalf("refreshDownloadGateURLByError() error = %v", err)
	}
	if got != server.URL+"/down/N2qlBz" {
		t.Fatalf("refreshDownloadGateURLByError() = %q; want %q", got, server.URL+"/down/N2qlBz")
	}
}

func TestResolveFreshDownloadGateURLLoadsDetailPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/a/Ks9kp5" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`
<html><body>
  <a class="btn btn-danger down" href="/down/N2qlBz">濠电姷鏁搁崑鐐哄垂閸洖绠伴柟闂寸贰閺佸嫰鏌涢锝囪穿鐟滅増甯掗悙濠囨煃鐟欏嫬鍔ゅù婊呭亾缁绘盯骞嬮悙鍨櫧濠电偛鐗婂鑽ゆ閹烘鍋愰梻鍫熺☉楠炲鎮楀▓鍨珮闁告挾鍠庨悾鐑藉醇閺囩喐娅嗙紓浣圭☉椤戝懏绂?/a>
</body></html>`))
	}))
	defer server.Close()

	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.SuppliersSettings.SubHD.RootUrl = server.URL

	supplier := &Supplier{log: logrus.New()}
	httpClient := resty.New()

	got, err := supplier.resolveFreshDownloadGateURL(httpClient, server.URL+"/down/Ks9kp5")
	if err != nil {
		t.Fatalf("resolveFreshDownloadGateURL() error = %v", err)
	}
	if got != server.URL+"/down/N2qlBz" {
		t.Fatalf("resolveFreshDownloadGateURL() = %q; want %q", got, server.URL+"/down/N2qlBz")
	}
}

func TestResolveFreshDownloadGateURLWithTimeoutStopsWaitingForSlowDetailPage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/a/Ks9kp5" {
			http.NotFound(w, r)
			return
		}
		time.Sleep(200 * time.Millisecond)
		_, _ = w.Write([]byte(`
<html><body>
  <a class="btn btn-danger down" href="/down/N2qlBz">download</a>
</body></html>`))
	}))
	defer server.Close()

	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.SuppliersSettings.SubHD.RootUrl = server.URL

	supplier := &Supplier{log: logrus.New()}
	httpClient := resty.New()

	start := time.Now()
	_, err := supplier.resolveFreshDownloadGateURLWithTimeout(httpClient, server.URL+"/down/Ks9kp5", 50*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error for slow detail page")
	}
	if elapsed := time.Since(start); elapsed > 500*time.Millisecond {
		t.Fatalf("resolveFreshDownloadGateURLWithTimeout() took too long: %v", elapsed)
	}
}

func TestResolveDownloadGateContextFromHTMLReturnsFreshSID(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.SuppliersSettings.SubHD.RootUrl = "https://subhd.me"

	supplier := &Supplier{log: logrus.New()}
	html := `
<html><body>
  <a class="btn btn-danger down" sid="N2qlBz" href="/down/N2qlBz" target="_blank">婵犵數鍋為崹鍫曞箰閹间緡鏁勯柛顐ｇ贩瑜版帒鐐婇柍瑙勫劤娴滅偓绻涢幋鐐垫噽婵炲牊姊荤槐鎺楁偐闂堟稑骞嬮悗娈垮櫘閸撶喎鐣烽崼鏇熸櫆缂佹稑顑呮禒?/a>
</body></html>`

	gateURL, sid, err := supplier.resolveDownloadGateContextFromHTML(html)
	if err != nil {
		t.Fatalf("resolveDownloadGateContextFromHTML() error = %v", err)
	}
	if gateURL != "https://subhd.me/down/N2qlBz" {
		t.Fatalf("resolveDownloadGateContextFromHTML() gate = %q; want %q", gateURL, "https://subhd.me/down/N2qlBz")
	}
	if sid != "N2qlBz" {
		t.Fatalf("resolveDownloadGateContextFromHTML() sid = %q; want %q", sid, "N2qlBz")
	}
}

func TestParseDownloadGateSIDSupportsGatePageButton(t *testing.T) {
	html := `
<html><body>
  <button class="btn btn-danger down" sid="58cNCQ">婵犵數鍋為崹鍫曞箰閹间緡鏁勯柛顐ｇ贩瑜版帒鐐婇柍瑙勫劤娴滅偓绻涢幋鐐垫噽婵炲牊姊荤槐鎺楁偐闂堟稑骞嬮悗娈垮櫘閸撶喎鐣烽崼鏇熸櫆缂佹稑顑呮禒?/button>
</body></html>`

	sid, err := parseDownloadGateSID(html)
	if err != nil {
		t.Fatalf("parseDownloadGateSID() error = %v", err)
	}
	if sid != "58cNCQ" {
		t.Fatalf("parseDownloadGateSID() = %q; want %q", sid, "58cNCQ")
	}
}

func TestExtractCaptchaTextFromSVGSortsByVisualPosition(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg">
		<text x="44" y="20">1</text>
		<text x="12" y="20">A</text>
		<text x="32" y="20">B</text>
		<text x="22" y="20">0</text>
	</svg>`

	got := extractCaptchaTextFromSVG(svg)
	if got != "A0B1" {
		t.Fatalf("extractCaptchaTextFromSVG() = %q; want %q", got, "A0B1")
	}
}

func TestExtractCaptchaTextFromSVGSupportsMultipleRows(t *testing.T) {
	svg := `<svg xmlns="http://www.w3.org/2000/svg">
		<text x="22" y="30">D</text>
		<text x="10" y="10">A</text>
		<text x="20" y="10">B</text>
		<text x="12" y="30">C</text>
	</svg>`

	got := extractCaptchaTextFromSVG(svg)
	if got != "ABCD" {
		t.Fatalf("extractCaptchaTextFromSVG() = %q; want %q", got, "ABCD")
	}
}

func TestNormalizeCaptchaTextRepairsCommonOCRSymbols(t *testing.T) {
	got, err := normalizeCaptchaText("|A$!")
	if err != nil {
		t.Fatalf("normalizeCaptchaText() error = %v", err)
	}
	if got != "1AS1" {
		t.Fatalf("normalizeCaptchaText() = %q; want %q", got, "1AS1")
	}
}

func TestNormalizeCaptchaTextTrimsAfterRepair(t *testing.T) {
	got, err := normalizeCaptchaText("  |AB$!X  ")
	if err != nil {
		t.Fatalf("normalizeCaptchaText() error = %v", err)
	}
	if got != "1ABS" {
		t.Fatalf("normalizeCaptchaText() = %q; want %q", got, "1ABS")
	}
}

func TestNormalizeSingleCaptchaGlyphRepairsCommonOCRSymbols(t *testing.T) {
	got, err := normalizeSingleCaptchaGlyph("| ")
	if err != nil {
		t.Fatalf("normalizeSingleCaptchaGlyph() error = %v", err)
	}
	if got != "1" {
		t.Fatalf("normalizeSingleCaptchaGlyph() = %q; want %q", got, "1")
	}
}

func TestCollectSingleGlyphCandidatesKeepsDistinctCharacters(t *testing.T) {
	got := collectSingleGlyphCandidates("jdl")
	if strings.Join(got, ",") != "j,d,l" {
		t.Fatalf("collectSingleGlyphCandidates() = %v; want [j d l]", got)
	}
}

func TestAppendUniqueGlyphCandidatesDeduplicates(t *testing.T) {
	got := appendUniqueGlyphCandidates([]string{"K"}, "K", "H", "G")
	if strings.Join(got, ",") != "K,H,G" {
		t.Fatalf("appendUniqueGlyphCandidates() = %v; want [K H G]", got)
	}
}

func TestAppendUniqueStringCandidatesDeduplicates(t *testing.T) {
	got := appendUniqueStringCandidates([]string{"AB12"}, "AB12", "A812", "AB1Z")
	if strings.Join(got, ",") != "AB12,A812,AB1Z" {
		t.Fatalf("appendUniqueStringCandidates() = %v; want [AB12 A812 AB1Z]", got)
	}
}

func TestRecognizeCaptchaTextWithRunnerAccumulatesEnhancedCandidates(t *testing.T) {
	enhancedPaths := []string{"enhanced-1", "enhanced-2", "enhanced-3"}
	buildCalls := 0
	got, err := recognizeCaptchaTextWithRunner(
		"captcha.png",
		"QIKWJ",
		func(string, int, int) (string, func(), error) {
			path := enhancedPaths[buildCalls]
			buildCalls++
			return path, func() {}, nil
		},
		func(path string, psm int) (string, error) {
			switch path {
			case "enhanced-1":
				return "Q1KW", nil
			case "enhanced-2":
				return "QIKW", nil
			default:
				return "QI|W", nil
			}
		},
	)
	if err != nil {
		t.Fatalf("recognizeCaptchaTextWithRunner() error = %v", err)
	}
	if strings.Join(got, ",") != "QIKW,Q1KW,QI1W" {
		t.Fatalf("recognizeCaptchaTextWithRunner() = %v; want [QIKW Q1KW QI1W]", got)
	}
}

func TestRecognizeCaptchaTextWithRunnerFallsBackToEnhancedWhenFirstRawInvalid(t *testing.T) {
	enhancedPaths := []string{"enhanced-1", "enhanced-2", "enhanced-3"}
	buildCalls := 0
	got, err := recognizeCaptchaTextWithRunner(
		"captcha.png",
		"",
		func(string, int, int) (string, func(), error) {
			path := enhancedPaths[buildCalls]
			buildCalls++
			return path, func() {}, nil
		},
		func(path string, psm int) (string, error) {
			switch path {
			case "enhanced-1":
				return "", nil
			case "enhanced-2":
				return "AB12", nil
			default:
				return "ABI2", nil
			}
		},
	)
	if err != nil {
		t.Fatalf("recognizeCaptchaTextWithRunner() error = %v", err)
	}
	if strings.Join(got, ",") != "AB12,ABI2" {
		t.Fatalf("recognizeCaptchaTextWithRunner() = %v; want [AB12 ABI2]", got)
	}
}

func TestRecognizeSingleCaptchaGlyphWithRunnerAccumulatesEnhancedCandidates(t *testing.T) {
	enhancedPaths := []string{"enhanced-1", "enhanced-2", "enhanced-3"}
	buildCalls := 0
	got, err := recognizeSingleCaptchaGlyphWithRunner(
		"glyph.png",
		"Z",
		func(string, int, int) (string, func(), error) {
			path := enhancedPaths[buildCalls]
			buildCalls++
			return path, func() {}, nil
		},
		func(path string, psm int) (string, error) {
			switch path {
			case "enhanced-1":
				return "I", nil
			case "enhanced-2":
				return "P", nil
			default:
				return "", nil
			}
		},
	)
	if err != nil {
		t.Fatalf("recognizeSingleCaptchaGlyphWithRunner() error = %v", err)
	}
	if strings.Join(got, ",") != "Z,I,P" {
		t.Fatalf("recognizeSingleCaptchaGlyphWithRunner() = %v; want [Z I P]", got)
	}
}

func glyphSets(values ...string) [][]string {
	out := make([][]string, 0, len(values))
	for _, value := range values {
		out = append(out, []string{value})
	}
	return out
}

func TestNetworkCookiesToHTTPCookiesCopiesKeyFields(t *testing.T) {
	got := networkCookiesToHTTPCookies([]*proto.NetworkCookie{
		{
			Name:     "tk_123",
			Value:    "abc",
			Domain:   ".subhd.me",
			Path:     "/",
			Secure:   true,
			HTTPOnly: true,
		},
	})
	if len(got) != 1 {
		t.Fatalf("networkCookiesToHTTPCookies() len = %d; want 1", len(got))
	}
	if got[0].Name != "tk_123" || got[0].Value != "abc" {
		t.Fatalf("networkCookiesToHTTPCookies() cookie = %#v", got[0])
	}
	if got[0].Domain != "subhd.me" || got[0].Path != "/" {
		t.Fatalf("networkCookiesToHTTPCookies() domain/path = %q %q", got[0].Domain, got[0].Path)
	}
	if got[0].Secure == false || got[0].HttpOnly == false {
		t.Fatalf("networkCookiesToHTTPCookies() secure/httpOnly = %v %v", got[0].Secure, got[0].HttpOnly)
	}
}

func TestGlyphRuneAlternativesIncludesShapeVariantsAndCase(t *testing.T) {
	got := glyphRuneAlternatives('g')
	joined := string(got)
	for _, want := range []rune{'g', 'Q', '9', 'G'} {
		if strings.ContainsRune(joined, want) == false {
			t.Fatalf("glyphRuneAlternatives() missing %q in %q", string(want), joined)
		}
	}
}

func TestParseSubtitleRowsSupportsDirectSubtitleDetailPage(t *testing.T) {
	html := `
<html><body>
  <div class="p-3">
    <div class="f16 fw-bold mb-2">Wonder Show S01E06 2160p NF WEB-DL</div>
  </div>
  <div class="p-3 my-2 bg-light clearfix">
    <div class="float-start">
      <span class="p-1 fw-bold">闂傚倷绀侀幉锟犳偡閵夆晜鍎楀〒姘ｅ亾妞?/span>
      <span class="p-1 fw-bold">缂傚倸鍊烽懗鑸垫叏椤撱垹纾婚柟鎹愮М?/span><span class="p-1 fw-bold">闂傚倷绀侀崥瀣ｉ幒鏂垮灊濡わ絽鍠氶弫?/span>
      <span class="p-1 text-secondary">SRT</span>
    </div>
    <div class="float-end">
      <span class='align-text-top me-3'>299k</span>
      <span class="align-text-top me-3">181</span>
      <span class="align-text-top">2026-5-29 18:55</span>
    </div>
  </div>
  <div class="mb-3 clearfix">
    <a class="btn btn-danger down" sid="Ks9kp5" href="/down/Ks9kp5" target="_blank">婵犵數鍋為崹鍫曞箰閹间緡鏁勯柛顐ｇ贩瑜版帒鐐婇柍瑙勫劤娴滅偓绻涢幋鐐垫噽婵炲牊姊荤槐鎺楁偐闂堟稑骞嬮悗娈垮櫘閸撶喎鐣烽崼鏇熸櫆缂佹稑顑呮禒?/a>
  </div>
</body></html>`

	got, err := parseSubtitleRows(html, "https://subhd.me", false, 3)
	if err != nil {
		t.Fatalf("parseSubtitleRows() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("parseSubtitleRows() len = %d; want 1", len(got))
	}
	if got[0].Url != "/down/Ks9kp5" {
		t.Fatalf("parseSubtitleRows() url = %q; want %q", got[0].Url, "/down/Ks9kp5")
	}
	if got[0].Title != "Wonder Show S01E06 2160p NF WEB-DL" {
		t.Fatalf("parseSubtitleRows() title = %q", got[0].Title)
	}
	if got[0].DownCount != 181 {
		t.Fatalf("parseSubtitleRows() downCount = %d; want 181", got[0].DownCount)
	}
}

func TestParseSubtitleRowsSupportsCurrentDetailListLayout(t *testing.T) {
	html := `
<html><body>
  <div class="row pt-2 mb-2">
    <div class="col-12 col-lg-8">
      <div class="px-3 py-2">
        <div class="view-text">
          <a class="link-dark" href="/a/gXILlH">Interstellar.2014.1080p.BluRay.x264.DTS-RARBG</a>
        </div>
        <div class="pt-1 f11">
          <span class="p-1 text-secondary">ASS</span>
        </div>
      </div>
    </div>
    <div class="col-2 d-none d-lg-block">
      <div class="px-3 py-2 text-end text-secondary">30</div>
    </div>
  </div>
</body></html>`

	got, err := parseSubtitleRows(html, "https://subhd.me", true, 3)
	if err != nil {
		t.Fatalf("parseSubtitleRows() error = %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("parseSubtitleRows() len = %d; want 1", len(got))
	}
	if got[0].Url != "/a/gXILlH" {
		t.Fatalf("parseSubtitleRows() url = %q; want %q", got[0].Url, "/a/gXILlH")
	}
	if got[0].Title != "Interstellar.2014.1080p.BluRay.x264.DTS-RARBG" {
		t.Fatalf("parseSubtitleRows() title = %q", got[0].Title)
	}
	if got[0].DownCount != 30 {
		t.Fatalf("parseSubtitleRows() downCount = %d; want 30", got[0].DownCount)
	}
}

func TestProbeRootURLsAddsStableFallbacks(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()
	cfg.AdvancedSettings.SuppliersSettings.SubHD.RootUrl = "https://subhd.tv"

	supplier := &Supplier{log: logrus.New()}
	got := supplier.probeRootURLs()
	if len(got) < 2 {
		t.Fatalf("probeRootURLs() len = %d; want at least 2", len(got))
	}
	if got[0] != "https://subhd.me" {
		t.Fatalf("probeRootURLs()[0] = %q; want %q", got[0], "https://subhd.me")
	}
	if got[1] != "https://subhd.one" {
		t.Fatalf("probeRootURLs()[1] = %q; want %q", got[1], "https://subhd.one")
	}
}

func TestShouldKeepAliveOnProbeErrorTreatsTimeoutAsTransient(t *testing.T) {
	if shouldKeepAliveOnProbeError(errors.New("context deadline exceeded")) == false {
		t.Fatal("shouldKeepAliveOnProbeError() = false; want true for timeout")
	}
	if shouldKeepAliveOnProbeError(errors.New("connection reset by peer")) == false {
		t.Fatal("shouldKeepAliveOnProbeError() = false; want true for connection reset")
	}
	if shouldKeepAliveOnProbeError(errors.New("unexpected status")) == true {
		t.Fatal("shouldKeepAliveOnProbeError() = true; want false for non-transient error")
	}
}

func TestShouldFallbackToBrowserPageFetchTreats522AndTimeoutAsTransient(t *testing.T) {
	if shouldFallbackToBrowserPageFetch(errors.New("unexpected http status 522 for https://subhd.me/search/Interstellar")) == false {
		t.Fatal("shouldFallbackToBrowserPageFetch() = false; want true for 522")
	}
	if shouldFallbackToBrowserPageFetch(errors.New("Get \"https://subhd.me/search/tt0816692\": context deadline exceeded (Client.Timeout exceeded while awaiting headers)")) == false {
		t.Fatal("shouldFallbackToBrowserPageFetch() = false; want true for timeout")
	}
	if shouldFallbackToBrowserPageFetch(errors.New("unexpected http status 403 for https://subhd.me/search/Interstellar")) == true {
		t.Fatal("shouldFallbackToBrowserPageFetch() = true; want false for non-transient status")
	}
}

func TestResolveTesseractBinaryWithFallsBackToKnownCandidate(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "tesseract.exe")
	if err := os.WriteFile(binPath, []byte("stub"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	got, err := resolveTesseractBinaryWith(func(string) (string, error) {
		return "", errors.New("missing")
	}, []string{binPath})
	if err != nil {
		t.Fatalf("resolveTesseractBinaryWith() error = %v", err)
	}
	if got != binPath {
		t.Fatalf("resolveTesseractBinaryWith() = %q; want %q", got, binPath)
	}
}

func TestSameSubHDSIDMatchesAcrossADownPaths(t *testing.T) {
	if sameSubHDSID("https://subhd.me/a/Ks9kp5", "https://subhd.me/down/Ks9kp5") == false {
		t.Fatal("sameSubHDSID() = false; want true for same sid across /a and /down")
	}
	if sameSubHDSID("https://subhd.me/a/Ks9kp5", "https://subhd.me/down/gbhp4H") == true {
		t.Fatal("sameSubHDSID() = true; want false for different sid")
	}
}

func TestBuildTesseractArgsUsesLegacyCompatibleFlags(t *testing.T) {
	t.Setenv("TEMP", t.TempDir())
	t.Setenv("TMP", t.TempDir())

	got, outputBase, cleanup, err := buildTesseractArgs("captcha.png", 10)
	if err != nil {
		t.Fatalf("buildTesseractArgs() error = %v", err)
	}
	defer cleanup()

	wantPrefix := []string{
		"captcha.png",
		outputBase,
		"-l", "eng",
		"--psm", "10",
	}
	if len(got) != len(wantPrefix)+1 {
		t.Fatalf("buildTesseractArgs() len = %d; want %d", len(got), len(wantPrefix)+1)
	}
	for i := range wantPrefix {
		if got[i] != wantPrefix[i] {
			t.Fatalf("buildTesseractArgs()[%d] = %q; want %q", i, got[i], wantPrefix[i])
		}
	}
	if outputBase == "" {
		t.Fatal("buildTesseractArgs() outputBase is empty")
	}
	cfgBytes, err := os.ReadFile(got[6])
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if string(cfgBytes) != "tessedit_char_whitelist ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789\n" {
		t.Fatalf("unexpected tesseract config: %q", string(cfgBytes))
	}
}

func TestBuildEnhancedGlyphImageCreatesPaddedPNG(t *testing.T) {
	srcPath := filepath.Join(t.TempDir(), "glyph.png")
	img := image.NewRGBA(image.Rect(0, 0, 2, 3))
	img.Set(0, 0, color.Black)
	file, err := os.Create(srcPath)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if err := png.Encode(file, img); err != nil {
		_ = file.Close()
		t.Fatalf("png.Encode() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	gotPath, cleanup, err := buildEnhancedGlyphImage(srcPath, 8, 4)
	if err != nil {
		t.Fatalf("buildEnhancedGlyphImage() error = %v", err)
	}
	defer cleanup()

	gotFile, err := os.Open(gotPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer gotFile.Close()
	gotImg, _, err := image.Decode(gotFile)
	if err != nil {
		t.Fatalf("image.Decode() error = %v", err)
	}
	if gotImg.Bounds().Dx() != 24 || gotImg.Bounds().Dy() != 28 {
		t.Fatalf("buildEnhancedGlyphImage() bounds = %v; want 24x28", gotImg.Bounds())
	}
}

func TestRecognizeSingleCaptchaGlyphWithRunnerRetriesRawImageWithAlternatePSM(t *testing.T) {
	got, err := recognizeSingleCaptchaGlyphWithRunner(
		"glyph.png",
		"Z",
		func(string, int, int) (string, func(), error) {
			return "enhanced.png", func() {}, nil
		},
		func(path string, psm int) (string, error) {
			if path == "glyph.png" && psm == 6 {
				return "5", nil
			}
			return "%", nil
		},
	)
	if err != nil {
		t.Fatalf("recognizeSingleCaptchaGlyphWithRunner() error = %v", err)
	}
	if len(got) < 2 || got[0] != "Z" || got[1] != "5" {
		t.Fatalf("recognizeSingleCaptchaGlyphWithRunner() = %v; want raw psm retry to retain 5", got)
	}
}

func TestRecognizeSingleCaptchaGlyphWithRunnerPromotesStrongRawRetryAheadOfWeakLowercaseHead(t *testing.T) {
	got, err := recognizeSingleCaptchaGlyphWithRunner(
		"glyph.png",
		"k",
		func(string, int, int) (string, func(), error) {
			return "enhanced.png", func() {}, nil
		},
		func(path string, psm int) (string, error) {
			if path == "glyph.png" && psm == 6 {
				return "V", nil
			}
			if path == "glyph.png" && psm == 7 {
				return "X9", nil
			}
			return "%", nil
		},
	)
	if err != nil {
		t.Fatalf("recognizeSingleCaptchaGlyphWithRunner() error = %v", err)
	}
	if len(got) < 2 || got[0] != "V" || got[1] != "k" {
		t.Fatalf("recognizeSingleCaptchaGlyphWithRunner() = %v; want V promoted ahead of weak lowercase head", got)
	}
}

func TestRecognizeSingleCaptchaGlyphWithRunnerKeepsRawLowercaseHeadAheadOfEnhancedUppercaseGuess(t *testing.T) {
	got, err := recognizeSingleCaptchaGlyphWithRunner(
		"glyph.png",
		"%",
		func(string, int, int) (string, func(), error) {
			return "enhanced.png", func() {}, nil
		},
		func(path string, psm int) (string, error) {
			if path == "glyph.png" && psm == 6 {
				return "d", nil
			}
			if path == "enhanced.png" && psm == 6 {
				return "J", nil
			}
			return "%", nil
		},
	)
	if err != nil {
		t.Fatalf("recognizeSingleCaptchaGlyphWithRunner() error = %v", err)
	}
	if len(got) < 2 || got[0] != "d" || got[1] != "J" {
		t.Fatalf("recognizeSingleCaptchaGlyphWithRunner() = %v; want raw lowercase d kept ahead of enhanced J", got)
	}
}

func TestCookieNamesSortsAndDeduplicates(t *testing.T) {
	got := cookieNames([]*http.Cookie{
		{Name: "session"},
		{Name: "csrftoken"},
		{Name: "session"},
		{Name: "  "},
		nil,
	})
	if strings.Join(got, ",") != "csrftoken,session" {
		t.Fatalf("cookieNames() = %v; want [csrftoken session]", got)
	}
}

func TestWhichEpisodeNeedDownloadSubFiltersWrongSeriesTitle(t *testing.T) {
	supplier := &Supplier{log: logrus.New()}
	seriesInfo := &series.SeriesInfo{
		Name: "Lopez vs Lopez",
		NeedDlEpsKeyList: map[string]series.EpisodeInfo{
			pkg.GetEpisodeKeyName(1, 2): {
				Season:       1,
				Episode:      2,
				FileFullPath: filepath.Join("C:\\", "Media", "Lopez.vs.Lopez.S01E02.1080p.WEB-DL.mkv"),
			},
		},
	}
	mediaInfo := &models.MediaInfo{
		TitleCn:       "Lopez vs Lopez CN",
		TitleEn:       "Lopez vs Lopez",
		OriginalTitle: "Lopez vs Lopez",
	}
	allSubList := []HdListItem{
		{Title: "Spiderwick Chronicles S01E02 1080p", Url: "/wrong", DownCount: 99},
		{Title: "Lopez vs Lopez S01E02 1080p", Url: "/right", DownCount: 1},
	}

	got := supplier.whichEpisodeNeedDownloadSub(seriesInfo, mediaInfo, allSubList)
	if len(got) != 1 {
		t.Fatalf("whichEpisodeNeedDownloadSub() len = %d; want 1", len(got))
	}
	if got[0].Url != "/right" {
		t.Fatalf("whichEpisodeNeedDownloadSub() picked %q; want %q", got[0].Url, "/right")
	}
	if got[0].Season != 1 || got[0].Episode != 2 {
		t.Fatalf("whichEpisodeNeedDownloadSub() season/episode = S%02dE%02d; want S01E02", got[0].Season, got[0].Episode)
	}
}

func TestWhichEpisodeNeedDownloadSubAllowsMatchingSeasonPackOnce(t *testing.T) {
	supplier := &Supplier{log: logrus.New()}
	seriesInfo := &series.SeriesInfo{
		Name: "Lopez vs Lopez",
		NeedDlEpsKeyList: map[string]series.EpisodeInfo{
			pkg.GetEpisodeKeyName(1, 3): {
				Season:       1,
				Episode:      3,
				FileFullPath: filepath.Join("C:\\", "Media", "Lopez.vs.Lopez.S01E03.1080p.WEB-DL.mkv"),
			},
			pkg.GetEpisodeKeyName(1, 4): {
				Season:       1,
				Episode:      4,
				FileFullPath: filepath.Join("C:\\", "Media", "Lopez.vs.Lopez.S01E04.1080p.WEB-DL.mkv"),
			},
		},
	}
	mediaInfo := &models.MediaInfo{
		TitleCn:       "Lopez vs Lopez CN",
		TitleEn:       "Lopez vs Lopez",
		OriginalTitle: "Lopez vs Lopez",
	}
	allSubList := []HdListItem{
		{Title: "Wrong Show S01 Complete", Url: "/wrong-pack", DownCount: 99},
		{Title: "Lopez vs Lopez S01 Complete", Url: "/right-pack", DownCount: 10},
	}

	got := supplier.whichEpisodeNeedDownloadSub(seriesInfo, mediaInfo, allSubList)
	if len(got) != 1 {
		t.Fatalf("whichEpisodeNeedDownloadSub() len = %d; want 1 season pack", len(got))
	}
	if got[0].Url != "/right-pack" {
		t.Fatalf("whichEpisodeNeedDownloadSub() picked %q; want %q", got[0].Url, "/right-pack")
	}
	if got[0].Season != 1 || got[0].Episode != 0 {
		t.Fatalf("whichEpisodeNeedDownloadSub() season/episode = S%02dE%02d; want S01 pack", got[0].Season, got[0].Episode)
	}
}

func TestWhichEpisodeNeedDownloadSubLopezVsLopezPrefersExactEpisodes(t *testing.T) {
	supplier := &Supplier{log: logrus.New()}
	seriesInfo := &series.SeriesInfo{
		Name: "Lopez vs Lopez",
		NeedDlEpsKeyList: map[string]series.EpisodeInfo{
			pkg.GetEpisodeKeyName(1, 1): {
				Season:       1,
				Episode:      1,
				FileFullPath: filepath.Join("C:\\", "Media", "Lopez.vs.Lopez.S01E01.1080p.WEB-DL-GROUP.mkv"),
			},
			pkg.GetEpisodeKeyName(1, 2): {
				Season:       1,
				Episode:      2,
				FileFullPath: filepath.Join("C:\\", "Media", "Lopez.vs.Lopez.S01E02.1080p.WEB-DL-GROUP.mkv"),
			},
			pkg.GetEpisodeKeyName(1, 3): {
				Season:       1,
				Episode:      3,
				FileFullPath: filepath.Join("C:\\", "Media", "Lopez.vs.Lopez.S01E03.1080p.WEB-DL-GROUP.mkv"),
			},
			pkg.GetEpisodeKeyName(1, 4): {
				Season:       1,
				Episode:      4,
				FileFullPath: filepath.Join("C:\\", "Media", "Lopez.vs.Lopez.S01E04.1080p.WEB-DL-GROUP.mkv"),
			},
		},
	}
	mediaInfo := &models.MediaInfo{
		TitleCn:       "Lopez vs Lopez CN",
		TitleEn:       "Lopez vs Lopez",
		OriginalTitle: "Lopez vs Lopez",
	}
	allSubList := []HdListItem{
		{Title: "Spiderwick Chronicles S01E01 1080p WEB-DL-GROUP", Url: "/wrong-e01", DownCount: 99},
		{Title: "Lopez vs Lopez S01E01 1080p WEB-DL-GROUP", Url: "/right-e01", DownCount: 5},
		{Title: "Wrong Show S01E02 1080p WEB-DL-GROUP", Url: "/wrong-e02", DownCount: 100},
		{Title: "Lopez vs Lopez S01E02 1080p WEB-DL-GROUP", Url: "/right-e02", DownCount: 4},
		{Title: "Lopez vs Lopez S01 Complete 1080p WEB-DL-GROUP", Url: "/season-pack", DownCount: 2},
		{Title: "Wrong Show S01 Complete 1080p WEB-DL-GROUP", Url: "/wrong-pack", DownCount: 200},
	}

	got := supplier.whichEpisodeNeedDownloadSub(seriesInfo, mediaInfo, allSubList)
	gotByURL := make(map[string]HdListItem, len(got))
	for _, item := range got {
		gotByURL[item.Url] = item
	}
	if _, ok := gotByURL["/wrong-e01"]; ok {
		t.Fatal("whichEpisodeNeedDownloadSub() should reject wrong show episode for E01")
	}
	if _, ok := gotByURL["/wrong-e02"]; ok {
		t.Fatal("whichEpisodeNeedDownloadSub() should reject wrong show episode for E02")
	}
	if _, ok := gotByURL["/wrong-pack"]; ok {
		t.Fatal("whichEpisodeNeedDownloadSub() should reject wrong show season pack")
	}
	if _, ok := gotByURL["/right-e01"]; ok == false {
		t.Fatal("whichEpisodeNeedDownloadSub() should keep exact E01 match")
	}
	if _, ok := gotByURL["/right-e02"]; ok == false {
		t.Fatal("whichEpisodeNeedDownloadSub() should keep exact E02 match")
	}
	if pack, ok := gotByURL["/season-pack"]; ok == false {
		t.Fatal("whichEpisodeNeedDownloadSub() should keep season pack as fallback for missing episodes")
	} else if pack.Season != 1 || pack.Episode != 0 {
		t.Fatalf("whichEpisodeNeedDownloadSub() season pack = S%02dE%02d; want S01 pack", pack.Season, pack.Episode)
	}
}

func TestMatchSeriesTitleSupportsAliases(t *testing.T) {
	candidates := compactStrings("Lopez vs Lopez CN", "Lopez vs Lopez")
	if matchSeriesTitle("Lopez vs Lopez S01E01 1080p", candidates) == false {
		t.Fatal("matchSeriesTitle() should accept english title")
	}
	if matchSeriesTitle("Lopez vs Lopez CN S01E01", candidates) == false {
		t.Fatal("matchSeriesTitle() should accept alias title")
	}
	if matchSeriesTitle("Spiderwick Chronicles S01E01", candidates) == true {
		t.Fatal("matchSeriesTitle() should reject wrong series title")
	}
}

func TestSelectSearchResultURLsPrefersMatchingSeriesResults(t *testing.T) {
	searchResults := []searchResultItem{
		{Title: "Spiderwick Chronicles (2024)", URL: "/wrong"},
		{Title: "Lopez vs Lopez (2022)", URL: "/right"},
		{Title: "Lopez vs Lopez Season 1", URL: "/right-pack"},
	}

	got := selectSearchResultURLs(searchResults, []string{"Lopez vs Lopez"})
	if len(got) != 2 {
		t.Fatalf("selectSearchResultURLs() len = %d; want 2", len(got))
	}
	if got[0] != "/right" || got[1] != "/right-pack" {
		t.Fatalf("selectSearchResultURLs() = %v; want [/right /right-pack]", got)
	}
}

func TestSelectSearchResultURLsFallsBackWhenNoTitleMatches(t *testing.T) {
	searchResults := []searchResultItem{
		{Title: "Unknown Result A", URL: "/a"},
		{Title: "Unknown Result B", URL: "/b"},
	}

	got := selectSearchResultURLs(searchResults, []string{"Lopez vs Lopez"})
	if len(got) != 0 {
		t.Fatalf("selectSearchResultURLs() len = %d; want 0", len(got))
	}
}

func TestSelectSearchResultURLsKeepsAllResultsWithoutTitleCandidates(t *testing.T) {
	searchResults := []searchResultItem{
		{Title: "Unknown Result A", URL: "/a"},
		{Title: "Unknown Result B", URL: "/b"},
	}

	got := selectSearchResultURLs(searchResults, nil)
	if len(got) != 2 {
		t.Fatalf("selectSearchResultURLs() len = %d; want 2", len(got))
	}
	if got[0] != "/a" || got[1] != "/b" {
		t.Fatalf("selectSearchResultURLs() = %v; want [/a /b]", got)
	}
}

func TestSelectSearchResultURLsPrefersDirectSubtitlePages(t *testing.T) {
	searchResults := []searchResultItem{
		{Title: "Interstellar", URL: "/d/1889243"},
		{Title: "Interstellar 2160p", URL: "/a/Cm0tsS"},
		{Title: "Interstellar 1080p", URL: "/a/gbhp4H"},
	}

	got := selectSearchResultURLs(searchResults, nil)
	if len(got) != 3 {
		t.Fatalf("selectSearchResultURLs() len = %d; want 3", len(got))
	}
	if got[0] != "/a/Cm0tsS" || got[1] != "/a/gbhp4H" || got[2] != "/d/1889243" {
		t.Fatalf("selectSearchResultURLs() = %v; want [/a/Cm0tsS /a/gbhp4H /d/1889243]", got)
	}
}

func TestParseSearchResultsKeepsPageOrderAndDeduplicatesURL(t *testing.T) {
	html := `
<html><body>
  <h4>Lopez vs Lopez 闂傚倷鐒﹂惇褰掑礉瀹€鈧埀顒佸嚬閸欏啴鐛崘銊ф殾闁搞儺鐓堝鐔兼⒑闂堟稓绠為柛銊嚙閳绘棃濮€閵堝棛鍘?<span>闂?2 闂?闂佽崵鍠愮划搴㈡櫠濡ゅ懎绠伴柛娑橈攻濞呯娀鏌ｅΟ鍏兼毄缂?1 婵?/span></h4>
  <a href="/detail-b"><img class="rounded-start" src="b.jpg" />Lopez vs Lopez (2022)</a>
  <div><a href="/detail-a"><img class="rounded-start" src="a.jpg" />Lopez vs Lopez Season 1</a></div>
  <a href="/detail-b"><img class="rounded-start" src="b2.jpg" />Lopez vs Lopez Duplicate</a>
</body></html>`

	got, count, err := parseSearchResults(html)
	if err != nil {
		t.Fatalf("parseSearchResults() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("parseSearchResults() count = %d; want 2", count)
	}
	if len(got) != 2 {
		t.Fatalf("parseSearchResults() len = %d; want 2", len(got))
	}
	if got[0].URL != "/detail-b" || got[1].URL != "/detail-a" {
		t.Fatalf("parseSearchResults() urls = %v; want [/detail-b /detail-a]", []string{got[0].URL, got[1].URL})
	}
}

func TestParseSearchResultsTreatsZeroCountAsNoResults(t *testing.T) {
	html := `
<html><body>
  <h4>Lopez vs Lopez 闂傚倷鐒﹂惇褰掑礉瀹€鈧埀顒佸嚬閸欏啴鐛崘銊ф殾闁搞儺鐓堝鐔兼⒑闂堟稓绠為柛銊嚙閳绘棃濮€閵堝棛鍘?<span>闂?0 闂?闂佽崵鍠愮划搴㈡櫠濡ゅ懎绠伴柛娑橈攻濞呯娀鏌ｅΟ鍏兼毄缂?1 婵?/span></h4>
  <a class="link-dark" target="_blank" href="/d/123">闂傚倷鑳剁划顖炲春閸儱鐭楅柛鎰╁壆濞戙埄鏁嗛柛鏇ㄥ亞妤犲洭姊洪幐搴ｂ槈閻庢凹鍘剧划?/a>
</body></html>`

	got, count, err := parseSearchResults(html)
	if err != nil {
		t.Fatalf("parseSearchResults() error = %v", err)
	}
	if count != 0 {
		t.Fatalf("parseSearchResults() count = %d; want 0", count)
	}
	if len(got) != 0 {
		t.Fatalf("parseSearchResults() len = %d; want 0", len(got))
	}
}

func TestParseSearchResultsSupportsAnchorResultsWithoutImage(t *testing.T) {
	html := `
<html><body>
  <h4>tt35522483 闂傚倷鐒﹂惇褰掑礉瀹€鈧埀顒佸嚬閸欏啴鐛崘銊ф殾闁搞儺鐓堝鐔兼⒑闂堟稓绠為柛銊嚙閳绘棃濮€閵堝棛鍘?<span>闂?2 闂?闂佽崵鍠愮划搴㈡櫠濡ゅ懎绠伴柛娑橈攻濞呯娀鏌ｅΟ鍏兼毄缂?1 婵?/span></h4>
  <a class="link-dark align-middle" href="/a/right-a">闂備浇顕х换鎰崲濡ゅ懎纾婚柟鐗堟緲缁犵儤绻濋棃娑卞剱闁绘帊绮欓弻銊モ攽閸℃浼€濡炪倖鏌ㄥú顓烆潖婵犳艾绀冩い蹇撳閻忎礁顪冮妶搴′簻闁硅櫕锕㈠顐㈩吋閸滀焦鍍甸柡澶婄墑閸斿秴袙閸曨垱鍋℃繝濠傚椤庢粌霉濠婂骸澧柡?/a>
  <a class="link-dark align-middle" href="/a/right-b">闂備浇顕х换鎰崲濡ゅ懎纾婚柟鐗堟緲缁犵儤绻濋棃娑卞剱闁绘帊绮欓弻銊モ攽閸℃浼€濡炪倖鏌ㄥú顓烆潖婵犳艾绀冩い蹇撳閻忎礁顪冮妶搴′簻闁硅櫕锕㈠顐㈩吋閸滀焦鍍甸柡澶婄墑閸斿秴袙閸曨垱鍋℃繝濠傚椤庢粌霉濠婂骸澧柡?缂傚倸鍊烽悞锕傘€冭箛娑樼婵炴垶淇烘慨鎶芥煃閸濆嫭鍣圭紒?/a>
</body></html>`

	got, count, err := parseSearchResults(html)
	if err != nil {
		t.Fatalf("parseSearchResults() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("parseSearchResults() count = %d; want 2", count)
	}
	if len(got) != 2 {
		t.Fatalf("parseSearchResults() len = %d; want 2", len(got))
	}
	if got[0].URL != "/a/right-a" || got[1].URL != "/a/right-b" {
		t.Fatalf("parseSearchResults() urls = %v; want [/a/right-a /a/right-b]", []string{got[0].URL, got[1].URL})
	}
}

func TestParseSearchResultsDoesNotPanicWhenResultCountIsMissing(t *testing.T) {
	html := `
<html><body>
  <div class="cf-challenge">temporary challenge page</div>
</body></html>`

	got, count, err := parseSearchResults(html)
	if err != nil {
		t.Fatalf("parseSearchResults() error = %v; want nil for count-missing page", err)
	}
	if count != 0 {
		t.Fatalf("parseSearchResults() count = %d; want 0", count)
	}
	if len(got) != 0 {
		t.Fatalf("parseSearchResults() len = %d; want 0", len(got))
	}
}

func TestNextRepeatedDeadGateLoopCountTracksSameSID500ExpiredPattern(t *testing.T) {
	first := nextRepeatedDeadGateLoopCount(
		0,
		errors.New("probe_failed: unexpected gate status 500"),
		errors.New("download_gate_changed: expired gate"),
		"https://subhd.me/down/Ks9kp5",
		"https://subhd.me/down/Ks9kp5",
	)
	if first != 1 {
		t.Fatalf("nextRepeatedDeadGateLoopCount() first = %d; want 1", first)
	}
	second := nextRepeatedDeadGateLoopCount(
		first,
		errors.New("probe_failed: unexpected gate status 500"),
		errors.New("download_gate_changed: expired gate"),
		"https://subhd.me/down/Ks9kp5",
		"https://subhd.me/a/Ks9kp5",
	)
	if second != 2 {
		t.Fatalf("nextRepeatedDeadGateLoopCount() second = %d; want 2", second)
	}
}

func TestNextRepeatedDeadGateLoopCountResetsWhenPatternBreaks(t *testing.T) {
	got := nextRepeatedDeadGateLoopCount(
		1,
		errors.New("probe_failed: unexpected gate status 500"),
		errors.New("captcha_ocr_failed: subhd captcha rejected"),
		"https://subhd.me/down/Ks9kp5",
		"https://subhd.me/down/Ks9kp5",
	)
	if got != 0 {
		t.Fatalf("nextRepeatedDeadGateLoopCount() = %d; want 0 when expired pattern breaks", got)
	}
}

func TestIsSubHDDeadGateLoopErrorMatchesSentinel(t *testing.T) {
	if isSubHDDeadGateLoopError(errors.New("download_gate_changed: subhd dead gate loop detected sid Ks9kp5: expired gate")) == false {
		t.Fatal("expected dead gate loop sentinel to match")
	}
	if isSubHDDeadGateLoopError(errors.New("download_gate_changed: expired gate")) == true {
		t.Fatal("unexpected dead gate loop match for plain expired gate")
	}
}

func TestShouldStopAfterRepeatedDeadGateRowsUsesExpandedThreshold(t *testing.T) {
	if shouldStopAfterRepeatedDeadGateRows(maxConsecutiveDeadGateRowsBeforeStop - 1) {
		t.Fatalf("shouldStopAfterRepeatedDeadGateRows() = true before threshold %d", maxConsecutiveDeadGateRowsBeforeStop)
	}
	if shouldStopAfterRepeatedDeadGateRows(maxConsecutiveDeadGateRowsBeforeStop) == false {
		t.Fatalf("shouldStopAfterRepeatedDeadGateRows() = false at threshold %d", maxConsecutiveDeadGateRowsBeforeStop)
	}
}
