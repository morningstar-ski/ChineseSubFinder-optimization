package baseline

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompareResults(t *testing.T) {
	before := []SampleResult{
		{
			Sample: Sample{
				ID:        "episode-001",
				VideoPath: "C:\\Media\\Show\\S01E01.mkv",
				Kind:      SampleEpisode,
				Season:    1,
				Episode:   1,
			},
			PrimaryFailure: FailureNoProviderHit,
			Attempts: []ProviderAttempt{
				{Provider: "assrt", Hit: false, Downloaded: false, FailureCategory: FailureNoProviderHit},
				{Provider: "subdl", Hit: false, Downloaded: false, FailureCategory: FailureNoProviderHit},
			},
		},
		{
			Sample: Sample{
				ID:        "movie-001",
				VideoPath: "C:\\Media\\Movie.mkv",
				Kind:      SampleMovie,
			},
			Attempts: []ProviderAttempt{
				{Provider: "subdl", Hit: true, Downloaded: true},
			},
		},
	}
	after := []SampleResult{
		{
			Sample: Sample{
				ID:        "episode-001",
				VideoPath: "C:\\Media\\Show\\S01E01.mkv",
				Kind:      SampleEpisode,
				Season:    1,
				Episode:   1,
			},
			Attempts: []ProviderAttempt{
				{Provider: "assrt", Hit: true, Downloaded: true},
				{Provider: "subdl", Hit: false, Downloaded: false, FailureCategory: FailureNoProviderHit},
			},
		},
		{
			Sample: Sample{
				ID:        "movie-001",
				VideoPath: "C:\\Media\\Movie.mkv",
				Kind:      SampleMovie,
			},
			PrimaryFailure: FailureBadArchive,
			Attempts: []ProviderAttempt{
				{Provider: "subdl", Hit: true, Downloaded: false, FailureCategory: FailureBadArchive},
			},
		},
	}

	comparison, err := CompareResults(before, after)
	if err != nil {
		t.Fatalf("CompareResults returned error: %v", err)
	}

	if comparison.Summary.TotalSamples != 2 {
		t.Fatalf("expected 2 samples, got %d", comparison.Summary.TotalSamples)
	}
	if comparison.Summary.ImprovedSamples != 1 {
		t.Fatalf("expected 1 improved sample, got %d", comparison.Summary.ImprovedSamples)
	}
	if comparison.Summary.RegressedSamples != 1 {
		t.Fatalf("expected 1 regressed sample, got %d", comparison.Summary.RegressedSamples)
	}
	if comparison.Summary.BeforeDownloadedSamples != 1 || comparison.Summary.AfterDownloadedSamples != 1 {
		t.Fatalf("unexpected downloaded summary %#v", comparison.Summary)
	}
	if comparison.Summary.BeforeHitSamples != 1 || comparison.Summary.AfterHitSamples != 2 {
		t.Fatalf("unexpected hit summary %#v", comparison.Summary)
	}
	if comparison.Summary.BeforeAverageAttemptCount != 1.5 || comparison.Summary.AfterAverageAttemptCount != 1.5 {
		t.Fatalf("unexpected attempt summary %#v", comparison.Summary)
	}
	if comparison.Summary.BeforeSubHDAttemptSamples != 0 || comparison.Summary.AfterSubHDAttemptSamples != 0 {
		t.Fatalf("unexpected subhd attempt summary %#v", comparison.Summary)
	}
	if comparison.Samples[0].Sample.ID != "episode-001" || comparison.Samples[0].Status != SampleDiffImproved {
		t.Fatalf("unexpected first sample comparison %#v", comparison.Samples[0])
	}
	if comparison.Samples[1].Sample.ID != "movie-001" || comparison.Samples[1].Status != SampleDiffRegressed {
		t.Fatalf("unexpected second sample comparison %#v", comparison.Samples[1])
	}
	if len(comparison.Providers) != 2 {
		t.Fatalf("expected 2 provider summaries, got %#v", comparison.Providers)
	}
	if comparison.Providers[0].Provider != "assrt" || comparison.Providers[0].DownloadDelta != 1 {
		t.Fatalf("unexpected assrt summary %#v", comparison.Providers[0])
	}
	if comparison.Providers[1].Provider != "subdl" || comparison.Providers[1].DownloadDelta != -1 {
		t.Fatalf("unexpected subdl summary %#v", comparison.Providers[1])
	}
}

func TestCompareResultsTracksSubHDFallbackPressure(t *testing.T) {
	before := []SampleResult{
		{
			Sample: Sample{
				ID:        "movie-001",
				VideoPath: "C:\\Media\\Movie.mkv",
				Kind:      SampleMovie,
			},
			PrimaryFailure: FailureCaptchaOCR,
			Attempts: []ProviderAttempt{
				{Provider: "opensubtitles", Hit: false, Downloaded: false, FailureCategory: FailureNoProviderHit},
				{Provider: "subhd", Hit: false, Downloaded: false, FailureCategory: FailureCaptchaOCR},
			},
		},
	}
	after := []SampleResult{
		{
			Sample: Sample{
				ID:        "movie-001",
				VideoPath: "C:\\Media\\Movie.mkv",
				Kind:      SampleMovie,
			},
			Attempts: []ProviderAttempt{
				{Provider: "opensubtitles", Hit: true, Downloaded: true},
			},
		},
	}

	comparison, err := CompareResults(before, after)
	if err != nil {
		t.Fatalf("CompareResults returned error: %v", err)
	}

	if comparison.Summary.BeforeSubHDAttemptSamples != 1 || comparison.Summary.AfterSubHDAttemptSamples != 0 {
		t.Fatalf("unexpected subhd attempt samples %#v", comparison.Summary)
	}
	if comparison.Summary.BeforeSubHDHitSamples != 0 || comparison.Summary.AfterSubHDHitSamples != 0 {
		t.Fatalf("unexpected subhd hit samples %#v", comparison.Summary)
	}
	if comparison.Summary.BeforeSubHDDownloadSamples != 0 || comparison.Summary.AfterSubHDDownloadSamples != 0 {
		t.Fatalf("unexpected subhd download samples %#v", comparison.Summary)
	}
	if comparison.Summary.BeforeSubHDCaptchaOCRSamples != 1 || comparison.Summary.AfterSubHDCaptchaOCRSamples != 0 {
		t.Fatalf("unexpected subhd captcha ocr samples %#v", comparison.Summary)
	}
	if comparison.Summary.SubHDCaptchaOCRDelta != -1 {
		t.Fatalf("unexpected subhd captcha ocr delta %#v", comparison.Summary)
	}
	if comparison.Summary.BeforeAverageAttemptCount != 2 || comparison.Summary.AfterAverageAttemptCount != 1 {
		t.Fatalf("unexpected average attempt counts %#v", comparison.Summary)
	}
	if comparison.Summary.AverageAttemptDelta != -1 {
		t.Fatalf("unexpected average attempt delta %#v", comparison.Summary)
	}
}

func TestCompareResultsRejectsMismatchedSamples(t *testing.T) {
	before := []SampleResult{
		{
			Sample: Sample{
				ID:        "movie-001",
				VideoPath: "C:\\Media\\Movie.mkv",
				Kind:      SampleMovie,
			},
			Attempts: []ProviderAttempt{
				{Provider: "subdl", Hit: true, Downloaded: true},
			},
		},
	}
	after := []SampleResult{
		{
			Sample: Sample{
				ID:        "movie-002",
				VideoPath: "C:\\Media\\Movie.mkv",
				Kind:      SampleMovie,
			},
			Attempts: []ProviderAttempt{
				{Provider: "subdl", Hit: true, Downloaded: true},
			},
		},
	}

	if _, err := CompareResults(before, after); err == nil {
		t.Fatal("expected mismatched samples to fail")
	}
}

func TestWriteComparisonCSV(t *testing.T) {
	comparison := Comparison{
		Samples: []SampleComparison{
			{
				Sample: Sample{
					ID:        "episode-001",
					VideoPath: "C:\\Media\\Show\\S01E01.mkv",
					Kind:      SampleEpisode,
					Season:    1,
					Episode:   1,
				},
				Status:                    SampleDiffImproved,
				BeforePrimaryFailure:      FailureNoProviderHit,
				AfterPrimaryFailure:       FailureNone,
				BeforeHitProviders:        []string{},
				AfterHitProviders:         []string{"assrt"},
				BeforeDownloadedProviders: []string{},
				AfterDownloadedProviders:  []string{"assrt"},
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteComparisonCSV(&buf, comparison); err != nil {
		t.Fatalf("WriteComparisonCSV returned error: %v", err)
	}

	output := buf.String()
	if strings.HasPrefix(output, utf8BOM) == false {
		t.Fatalf("csv output missing utf-8 bom: %q", output)
	}
	for _, want := range []string{
		"sample_id,video_path,sample_kind,season,episode,status,before_primary_failure,after_primary_failure,before_hit_providers,after_hit_providers,before_downloaded_providers,after_downloaded_providers",
		"episode-001,C:\\Media\\Show\\S01E01.mkv,episode,1,1,improved,no_provider_hit,,,assrt,,assrt",
	} {
		if strings.Contains(output, want) == false {
			t.Fatalf("csv output missing %q\n%s", want, output)
		}
	}
}
