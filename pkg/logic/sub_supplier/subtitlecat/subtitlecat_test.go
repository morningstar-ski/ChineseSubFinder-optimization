package subtitlecat

import (
	"os"
	"path/filepath"
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

func TestBuildSearchKeywordsUsesMovieNfoTitlesWithoutMediaInfo(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())

	rootDir := t.TempDir()
	videoPath := filepath.Join(rootDir, "movie.2002.1080p.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}
	nfoPath := filepath.Join(rootDir, "movie.2002.1080p.nfo")
	content := `<?xml version="1.0" encoding="utf-8"?>
<movie>
  <title>Movie CN</title>
  <originaltitle>The Hours</originaltitle>
</movie>`
	if err := os.WriteFile(nfoPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write nfo: %v", err)
	}

	keywords := buildSearchKeywords(nil, videoPath, true, 0, 0)
	for _, want := range []string{"The Hours", "Movie CN"} {
		if containsKeyword(keywords, want) == false {
			t.Fatalf("keywords = %#v; want %s", keywords, want)
		}
	}
}

func TestBuildSearchKeywordsUsesSeriesNfoTitlesWithoutMediaInfo(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())

	rootDir := t.TempDir()
	seriesDir := filepath.Join(rootDir, "Series")
	seasonDir := filepath.Join(seriesDir, "Season 1")
	if err := os.MkdirAll(seasonDir, 0o755); err != nil {
		t.Fatalf("mkdir season dir: %v", err)
	}
	videoPath := filepath.Join(seasonDir, "Series - S01E01 - Episode 1.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}
	nfoPath := filepath.Join(seriesDir, "tvshow.nfo")
	content := `<?xml version="1.0" encoding="utf-8"?>
<tvshow>
  <title>Localized Show</title>
  <originaltitle>George Lopez</originaltitle>
</tvshow>`
	if err := os.WriteFile(nfoPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write tvshow nfo: %v", err)
	}

	keywords := buildSearchKeywords(nil, videoPath, false, 1, 1)
	for _, want := range []string{"George Lopez", "Localized Show", "George Lopez S01E01", "George Lopez 1x1"} {
		if containsKeyword(keywords, want) == false {
			t.Fatalf("keywords = %#v; want %s", keywords, want)
		}
	}
}

func TestBuildSearchKeywordsUsesSeriesOriginalTitleEpisodeVariants(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())

	rootDir := t.TempDir()
	seriesDir := filepath.Join(rootDir, "The Boys")
	seasonDir := filepath.Join(seriesDir, "Season 1")
	if err := os.MkdirAll(seasonDir, 0o755); err != nil {
		t.Fatalf("mkdir season dir: %v", err)
	}
	videoPath := filepath.Join(seasonDir, "Localized Name - S01E02 - Episode 2.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}
	nfoPath := filepath.Join(seriesDir, "tvshow.nfo")
	content := `<?xml version="1.0" encoding="utf-8"?>
<tvshow>
  <title>黑袍纠察队</title>
  <originaltitle>The Boys</originaltitle>
</tvshow>`
	if err := os.WriteFile(nfoPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write tvshow nfo: %v", err)
	}

	keywords := buildSearchKeywords(nil, videoPath, false, 1, 2)
	for _, want := range []string{"The Boys", "The Boys S01E02", "The Boys 1x2"} {
		if containsKeyword(keywords, want) == false {
			t.Fatalf("keywords = %#v; want %s", keywords, want)
		}
	}
}

func containsKeyword(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestFilterLowConfidenceCandidatesDropsWrongMovieRelease(t *testing.T) {
	videoPath := filepath.Join("C:\\", "Media", "The.Hours.2002.1080p.BluRay.x264-GROUP.mkv")
	candidates := []subtitleCandidate{
		{
			name:        "Tehran.S01E08.Five.Hours.Until.the.Bombing.Run.1080p.ATVP.WEB-DL.DDP5.1.H264-NTb.English",
			detailURL:   "https://www.subtitlecat.com/subs/1393/Tehran.S01E08.Five.Hours.Until.the.Bombing.Run.1080p.ATVP.WEB-DL.DDP5.1.H264-NTb.English.html",
			downloadURL: "https://www.subtitlecat.com/subs/1393/Tehran.S01E08.Five.Hours.Until.the.Bombing.Run.1080p.ATVP.WEB-DL.DDP5.1.H264-NTb-orig.srt",
		},
		{
			name:        "The.Hours.2002.1080p.BluRay.x264-GROUP.English",
			detailURL:   "https://www.subtitlecat.com/subs/2001/The.Hours.2002.1080p.BluRay.x264-GROUP.English.html",
			downloadURL: "https://www.subtitlecat.com/subs/2001/The.Hours.2002.1080p.BluRay.x264-GROUP-orig.srt",
		},
	}

	filtered := filterLowConfidenceCandidates(candidates, nil, videoPath, true, 0, 0)
	if len(filtered) != 1 {
		t.Fatalf("len(filtered) = %d, want 1", len(filtered))
	}
	if filtered[0].name != "The.Hours.2002.1080p.BluRay.x264-GROUP.English" {
		t.Fatalf("filtered[0].name = %q", filtered[0].name)
	}
}

func TestFilterLowConfidenceCandidatesDropsWrongMovieForChinesePathWithNFO(t *testing.T) {
	rootDir := t.TempDir()
	videoPath := filepath.Join(rootDir, "movie.2002.1080p.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}
	nfoPath := filepath.Join(rootDir, "movie.2002.1080p.nfo")
	content := `<?xml version="1.0" encoding="utf-8"?>
<movie>
  <title>Movie CN</title>
  <originaltitle>The Hours</originaltitle>
</movie>`
	if err := os.WriteFile(nfoPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write nfo: %v", err)
	}

	candidates := []subtitleCandidate{
		{
			name:        "Tehran.S01E08.Five.Hours.Until.the.Bombing.Run.1080p.ATVP.WEB-DL.DDP5.1.H264-NTb.English",
			detailURL:   "https://www.subtitlecat.com/subs/1393/Tehran.S01E08.Five.Hours.Until.the.Bombing.Run.1080p.ATVP.WEB-DL.DDP5.1.H264-NTb.English.html",
			downloadURL: "https://www.subtitlecat.com/subs/1393/Tehran.S01E08.Five.Hours.Until.the.Bombing.Run.1080p.ATVP.WEB-DL.DDP5.1.H264-NTb-orig.srt",
		},
		{
			name:        "The.Hours.2002.1080p.BluRay.x264-GROUP.English",
			detailURL:   "https://www.subtitlecat.com/subs/2001/The.Hours.2002.1080p.BluRay.x264-GROUP.English.html",
			downloadURL: "https://www.subtitlecat.com/subs/2001/The.Hours.2002.1080p.BluRay.x264-GROUP-orig.srt",
		},
	}

	filtered := filterLowConfidenceCandidates(candidates, nil, videoPath, true, 0, 0)
	if len(filtered) != 1 {
		t.Fatalf("len(filtered) = %d, want 1", len(filtered))
	}
	if filtered[0].name != "The.Hours.2002.1080p.BluRay.x264-GROUP.English" {
		t.Fatalf("filtered[0].name = %q", filtered[0].name)
	}
}

func TestFilterLowConfidenceCandidatesDropsWrongSeriesTitleFromNFO(t *testing.T) {
	rootDir := t.TempDir()
	seriesDir := filepath.Join(rootDir, "Localized Show (2002)")
	seasonDir := filepath.Join(seriesDir, "Season 1")
	if err := os.MkdirAll(seasonDir, 0o755); err != nil {
		t.Fatalf("mkdir season dir: %v", err)
	}

	videoPath := filepath.Join(seasonDir, "Localized Show - S01E01 - Episode 1.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}

	nfoPath := filepath.Join(seriesDir, "tvshow.nfo")
	content := `<?xml version="1.0" encoding="utf-8"?>
<tvshow>
  <title>Localized Show</title>
  <originaltitle>George Lopez</originaltitle>
</tvshow>`
	if err := os.WriteFile(nfoPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write tvshow nfo: %v", err)
	}

	candidates := []subtitleCandidate{
		{
			name:        "The.Spy.Next.Door.2010.1080p.BluRay.x264-GROUP.English",
			detailURL:   "https://www.subtitlecat.com/subs/171/The.Spy.Next.Door.2010.1080p.BluRay.x264-GROUP.English.html",
			downloadURL: "https://www.subtitlecat.com/subs/171/The.Spy.Next.Door.2010.1080p.BluRay.x264-GROUP-orig.srt",
		},
		{
			name:        "George.Lopez.S01E01.480p.WEB-DL.English",
			detailURL:   "https://www.subtitlecat.com/subs/999/George.Lopez.S01E01.480p.WEB-DL.English.html",
			downloadURL: "https://www.subtitlecat.com/subs/999/George.Lopez.S01E01.480p.WEB-DL-orig.srt",
		},
	}

	filtered := filterLowConfidenceCandidates(candidates, nil, videoPath, false, 1, 1)
	if len(filtered) != 1 {
		t.Fatalf("len(filtered) = %d, want 1", len(filtered))
	}
	if filtered[0].name != "George.Lopez.S01E01.480p.WEB-DL.English" {
		t.Fatalf("filtered[0].name = %q", filtered[0].name)
	}
}

func TestFilterLowConfidenceCandidatesDropsSeriesCandidateWithoutEpisodeMatch(t *testing.T) {
	rootDir := t.TempDir()
	seriesDir := filepath.Join(rootDir, "Localized Show (2002)")
	seasonDir := filepath.Join(seriesDir, "Season 1")
	if err := os.MkdirAll(seasonDir, 0o755); err != nil {
		t.Fatalf("mkdir season dir: %v", err)
	}

	videoPath := filepath.Join(seasonDir, "Localized Show - S01E01 - Episode 1.mkv")
	if err := os.WriteFile(videoPath, []byte("video"), 0o644); err != nil {
		t.Fatalf("write video: %v", err)
	}

	nfoPath := filepath.Join(seriesDir, "tvshow.nfo")
	content := `<?xml version="1.0" encoding="utf-8"?>
<tvshow>
  <title>Localized Show</title>
  <originaltitle>George Lopez</originaltitle>
</tvshow>`
	if err := os.WriteFile(nfoPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write tvshow nfo: %v", err)
	}

	candidates := []subtitleCandidate{
		{
			name:        "George.Lopez.Americas.Mexican.2007.1080p.WEBRip.x264.AAC-YTS.English",
			detailURL:   "https://www.subtitlecat.com/subs/1024/George.Lopez.Americas.Mexican.2007.1080p.WEBRip.x264.AAC-YTS.English.html",
			downloadURL: "https://www.subtitlecat.com/subs/1024/George.Lopez.Americas.Mexican.2007.1080p.WEBRip.x264.AAC-YTS-orig.srt",
		},
	}

	filtered := filterLowConfidenceCandidates(candidates, nil, videoPath, false, 1, 1)
	if len(filtered) != 0 {
		t.Fatalf("len(filtered) = %d, want 0", len(filtered))
	}
}
