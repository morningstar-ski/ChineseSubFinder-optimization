package subtitlecat

import (
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
)

func TestParseSearchResults(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())

	html := `
<table class="sub-table"><tbody>
  <tr>
    <td><a href="subs/1459/Interstellar_2014_Bluray_720p_AAC_HEVC_x265.English.html">Interstellar_2014_Bluray_720p_AAC_HEVC_x265.English</a> (translated from English)</td>
    <td></td>
    <td>182 KB</td>
    <td>6 downloads</td>
    <td>6 languages</td>
  </tr>
</tbody></table>`

	results, err := parseSearchResults(html)
	if err != nil {
		t.Fatalf("parseSearchResults() error = %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("len(results) = %d; want 1", len(results))
	}
	if results[0].translatedFrom != "English" {
		t.Fatalf("translatedFrom = %q; want %q", results[0].translatedFrom, "English")
	}
	if results[0].downloads != 6 {
		t.Fatalf("downloads = %d; want 6", results[0].downloads)
	}
	if results[0].languages != 6 {
		t.Fatalf("languages = %d; want 6", results[0].languages)
	}
	if results[0].detailURL != "https://www.subtitlecat.com/subs/1459/Interstellar_2014_Bluray_720p_AAC_HEVC_x265.English.html" {
		t.Fatalf("detailURL = %q", results[0].detailURL)
	}
}

func TestParseTranslatedDownloadURLPrefersSimplifiedChinese(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())

	html := `
<div class="sub-single">
  <span>Chinese (Traditional)</span>
  <span><a id="download_zh-TW" href="/subs/1414/interstellar-zh-TW.srt" class="green-link">Download</a></span>
</div>
<div class="sub-single">
  <span>Chinese (Simplified)</span>
  <span><a id="download_zh-CN" href="/subs/1414/interstellar-zh-CN.srt" class="green-link">Download</a></span>
</div>`

	got, found, err := parseTranslatedDownloadURL(html, "https://www.subtitlecat.com/subs/1397/Interstellar.eng.html")
	if err != nil {
		t.Fatalf("parseTranslatedDownloadURL() error = %v", err)
	}
	if found == false {
		t.Fatal("parseTranslatedDownloadURL() found = false; want true")
	}
	want := "https://www.subtitlecat.com/subs/1414/interstellar-zh-CN.srt"
	if got != want {
		t.Fatalf("downloadURL = %q; want %q", got, want)
	}
}

func TestDetailToOriginalDownloadURL(t *testing.T) {
	got := detailToOriginalDownloadURL("https://www.subtitlecat.com/subs/1397/Interstellar.2014.eng.html")
	want := "https://www.subtitlecat.com/subs/1397/Interstellar.2014.eng-orig.srt"
	if got != want {
		t.Fatalf("detailToOriginalDownloadURL() = %q; want %q", got, want)
	}
}
