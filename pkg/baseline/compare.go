package baseline

import (
	"encoding/csv"
	"fmt"
	"io"
	"sort"
	"strings"
)

type SampleDiffStatus string

const (
	SampleDiffUnchanged SampleDiffStatus = "unchanged"
	SampleDiffChanged   SampleDiffStatus = "changed"
	SampleDiffImproved  SampleDiffStatus = "improved"
	SampleDiffRegressed SampleDiffStatus = "regressed"
)

type Comparison struct {
	Summary   ComparisonSummary    `json:"summary"`
	Samples   []SampleComparison   `json:"samples"`
	Providers []ProviderComparison `json:"providers"`
}

type ComparisonSummary struct {
	TotalSamples                 int                     `json:"total_samples"`
	BeforeDownloadedSamples      int                     `json:"before_downloaded_samples"`
	AfterDownloadedSamples       int                     `json:"after_downloaded_samples"`
	DownloadedDelta              int                     `json:"downloaded_delta"`
	BeforeHitSamples             int                     `json:"before_hit_samples"`
	AfterHitSamples              int                     `json:"after_hit_samples"`
	HitDelta                     int                     `json:"hit_delta"`
	ImprovedSamples              int                     `json:"improved_samples"`
	RegressedSamples             int                     `json:"regressed_samples"`
	ChangedSamples               int                     `json:"changed_samples"`
	UnchangedSamples             int                     `json:"unchanged_samples"`
	BeforeAverageAttemptCount    float64                 `json:"before_average_attempt_count"`
	AfterAverageAttemptCount     float64                 `json:"after_average_attempt_count"`
	AverageAttemptDelta          float64                 `json:"average_attempt_delta"`
	BeforeSubHDAttemptSamples    int                     `json:"before_subhd_attempt_samples"`
	AfterSubHDAttemptSamples     int                     `json:"after_subhd_attempt_samples"`
	SubHDAttemptDelta            int                     `json:"subhd_attempt_delta"`
	BeforeSubHDHitSamples        int                     `json:"before_subhd_hit_samples"`
	AfterSubHDHitSamples         int                     `json:"after_subhd_hit_samples"`
	SubHDHitDelta                int                     `json:"subhd_hit_delta"`
	BeforeSubHDDownloadSamples   int                     `json:"before_subhd_download_samples"`
	AfterSubHDDownloadSamples    int                     `json:"after_subhd_download_samples"`
	SubHDDownloadDelta           int                     `json:"subhd_download_delta"`
	BeforeSubHDCaptchaOCRSamples int                     `json:"before_subhd_captcha_ocr_samples"`
	AfterSubHDCaptchaOCRSamples  int                     `json:"after_subhd_captcha_ocr_samples"`
	SubHDCaptchaOCRDelta         int                     `json:"subhd_captcha_ocr_delta"`
	BeforePrimaryFailureCounts   map[FailureCategory]int `json:"before_primary_failure_counts"`
	AfterPrimaryFailureCounts    map[FailureCategory]int `json:"after_primary_failure_counts"`
}

type SampleComparison struct {
	Sample                    Sample           `json:"sample"`
	Status                    SampleDiffStatus `json:"status"`
	BeforePrimaryFailure      FailureCategory  `json:"before_primary_failure,omitempty"`
	AfterPrimaryFailure       FailureCategory  `json:"after_primary_failure,omitempty"`
	BeforeHitProviders        []string         `json:"before_hit_providers,omitempty"`
	AfterHitProviders         []string         `json:"after_hit_providers,omitempty"`
	BeforeDownloadedProviders []string         `json:"before_downloaded_providers,omitempty"`
	AfterDownloadedProviders  []string         `json:"after_downloaded_providers,omitempty"`
}

type ProviderComparison struct {
	Provider        string `json:"provider"`
	BeforeHits      int    `json:"before_hits"`
	AfterHits       int    `json:"after_hits"`
	HitDelta        int    `json:"hit_delta"`
	BeforeDownloads int    `json:"before_downloads"`
	AfterDownloads  int    `json:"after_downloads"`
	DownloadDelta   int    `json:"download_delta"`
}

func CompareResults(before []SampleResult, after []SampleResult) (Comparison, error) {
	if err := ValidateResults(before); err != nil {
		return Comparison{}, fmt.Errorf("before results: %w", err)
	}
	if err := ValidateResults(after); err != nil {
		return Comparison{}, fmt.Errorf("after results: %w", err)
	}

	beforeByID := make(map[string]SampleResult, len(before))
	for _, result := range before {
		beforeByID[result.Sample.ID] = result
	}

	afterByID := make(map[string]SampleResult, len(after))
	for _, result := range after {
		afterByID[result.Sample.ID] = result
	}

	if len(beforeByID) != len(afterByID) {
		return Comparison{}, fmt.Errorf("result count mismatch: before=%d after=%d", len(beforeByID), len(afterByID))
	}

	sampleIDs := make([]string, 0, len(beforeByID))
	for sampleID, beforeResult := range beforeByID {
		afterResult, ok := afterByID[sampleID]
		if ok == false {
			return Comparison{}, fmt.Errorf("after results missing sample %q", sampleID)
		}
		if beforeResult.Sample != afterResult.Sample {
			return Comparison{}, fmt.Errorf("sample %q differs between before and after results", sampleID)
		}
		sampleIDs = append(sampleIDs, sampleID)
	}
	for sampleID := range afterByID {
		if _, ok := beforeByID[sampleID]; ok == false {
			return Comparison{}, fmt.Errorf("before results missing sample %q", sampleID)
		}
	}
	sort.Strings(sampleIDs)

	comparison := Comparison{
		Summary: ComparisonSummary{
			TotalSamples:               len(sampleIDs),
			BeforePrimaryFailureCounts: make(map[FailureCategory]int),
			AfterPrimaryFailureCounts:  make(map[FailureCategory]int),
		},
		Samples:   make([]SampleComparison, 0, len(sampleIDs)),
		Providers: make([]ProviderComparison, 0),
	}

	providerStats := make(map[string]*ProviderComparison)
	for _, sampleID := range sampleIDs {
		beforeResult := beforeByID[sampleID]
		afterResult := afterByID[sampleID]

		sampleComparison := buildSampleComparison(beforeResult, afterResult)
		comparison.Samples = append(comparison.Samples, sampleComparison)

		accumulateSummary(&comparison.Summary, beforeResult, afterResult, sampleComparison.Status)
		accumulateProviderStats(providerStats, beforeResult, true)
		accumulateProviderStats(providerStats, afterResult, false)
	}

	providerNames := make([]string, 0, len(providerStats))
	for providerName := range providerStats {
		providerNames = append(providerNames, providerName)
	}
	sort.Strings(providerNames)
	for _, providerName := range providerNames {
		stats := providerStats[providerName]
		stats.HitDelta = stats.AfterHits - stats.BeforeHits
		stats.DownloadDelta = stats.AfterDownloads - stats.BeforeDownloads
		comparison.Providers = append(comparison.Providers, *stats)
	}

	comparison.Summary.DownloadedDelta = comparison.Summary.AfterDownloadedSamples - comparison.Summary.BeforeDownloadedSamples
	comparison.Summary.HitDelta = comparison.Summary.AfterHitSamples - comparison.Summary.BeforeHitSamples
	comparison.Summary.AverageAttemptDelta = comparison.Summary.AfterAverageAttemptCount - comparison.Summary.BeforeAverageAttemptCount
	comparison.Summary.SubHDAttemptDelta = comparison.Summary.AfterSubHDAttemptSamples - comparison.Summary.BeforeSubHDAttemptSamples
	comparison.Summary.SubHDHitDelta = comparison.Summary.AfterSubHDHitSamples - comparison.Summary.BeforeSubHDHitSamples
	comparison.Summary.SubHDDownloadDelta = comparison.Summary.AfterSubHDDownloadSamples - comparison.Summary.BeforeSubHDDownloadSamples
	comparison.Summary.SubHDCaptchaOCRDelta = comparison.Summary.AfterSubHDCaptchaOCRSamples - comparison.Summary.BeforeSubHDCaptchaOCRSamples

	if comparison.Summary.TotalSamples > 0 {
		total := float64(comparison.Summary.TotalSamples)
		comparison.Summary.BeforeAverageAttemptCount = comparison.Summary.BeforeAverageAttemptCount / total
		comparison.Summary.AfterAverageAttemptCount = comparison.Summary.AfterAverageAttemptCount / total
		comparison.Summary.AverageAttemptDelta = comparison.Summary.AfterAverageAttemptCount - comparison.Summary.BeforeAverageAttemptCount
	}

	return comparison, nil
}

func WriteComparisonCSV(w io.Writer, comparison Comparison) error {
	if err := writeUTF8BOM(w); err != nil {
		return err
	}
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{
		"sample_id",
		"video_path",
		"sample_kind",
		"season",
		"episode",
		"status",
		"before_primary_failure",
		"after_primary_failure",
		"before_hit_providers",
		"after_hit_providers",
		"before_downloaded_providers",
		"after_downloaded_providers",
	}); err != nil {
		return err
	}

	for _, sampleComparison := range comparison.Samples {
		row := []string{
			sampleComparison.Sample.ID,
			sampleComparison.Sample.VideoPath,
			string(sampleComparison.Sample.Kind),
			fmt.Sprintf("%d", sampleComparison.Sample.Season),
			fmt.Sprintf("%d", sampleComparison.Sample.Episode),
			string(sampleComparison.Status),
			string(sampleComparison.BeforePrimaryFailure),
			string(sampleComparison.AfterPrimaryFailure),
			strings.Join(sampleComparison.BeforeHitProviders, ";"),
			strings.Join(sampleComparison.AfterHitProviders, ";"),
			strings.Join(sampleComparison.BeforeDownloadedProviders, ";"),
			strings.Join(sampleComparison.AfterDownloadedProviders, ";"),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	writer.Flush()
	return writer.Error()
}

func buildSampleComparison(before SampleResult, after SampleResult) SampleComparison {
	beforeHitProviders := collectAttemptProviders(before.Attempts, func(attempt ProviderAttempt) bool {
		return attempt.Hit
	})
	afterHitProviders := collectAttemptProviders(after.Attempts, func(attempt ProviderAttempt) bool {
		return attempt.Hit
	})
	beforeDownloadedProviders := collectAttemptProviders(before.Attempts, func(attempt ProviderAttempt) bool {
		return attempt.Downloaded
	})
	afterDownloadedProviders := collectAttemptProviders(after.Attempts, func(attempt ProviderAttempt) bool {
		return attempt.Downloaded
	})

	status := compareSampleStatus(
		before,
		after,
		beforeHitProviders,
		afterHitProviders,
		beforeDownloadedProviders,
		afterDownloadedProviders,
	)

	return SampleComparison{
		Sample:                    before.Sample,
		Status:                    status,
		BeforePrimaryFailure:      before.PrimaryFailure,
		AfterPrimaryFailure:       after.PrimaryFailure,
		BeforeHitProviders:        beforeHitProviders,
		AfterHitProviders:         afterHitProviders,
		BeforeDownloadedProviders: beforeDownloadedProviders,
		AfterDownloadedProviders:  afterDownloadedProviders,
	}
}

func compareSampleStatus(before SampleResult, after SampleResult, beforeHitProviders []string, afterHitProviders []string, beforeDownloadedProviders []string, afterDownloadedProviders []string) SampleDiffStatus {
	beforeRank := sampleOutcomeRank(before)
	afterRank := sampleOutcomeRank(after)
	if afterRank > beforeRank {
		return SampleDiffImproved
	}
	if afterRank < beforeRank {
		return SampleDiffRegressed
	}

	if before.PrimaryFailure != after.PrimaryFailure {
		return SampleDiffChanged
	}
	if stringSlicesEqual(beforeHitProviders, afterHitProviders) == false {
		return SampleDiffChanged
	}
	if stringSlicesEqual(beforeDownloadedProviders, afterDownloadedProviders) == false {
		return SampleDiffChanged
	}

	return SampleDiffUnchanged
}

func sampleOutcomeRank(result SampleResult) int {
	downloaded := false
	hit := false
	for _, attempt := range result.Attempts {
		if attempt.Downloaded {
			downloaded = true
			break
		}
		if attempt.Hit {
			hit = true
		}
	}

	switch {
	case downloaded:
		return 2
	case hit:
		return 1
	default:
		return 0
	}
}

func collectAttemptProviders(attempts []ProviderAttempt, predicate func(ProviderAttempt) bool) []string {
	out := make([]string, 0)
	for _, attempt := range attempts {
		if predicate(attempt) == false {
			continue
		}
		out = append(out, attempt.Provider)
	}
	sort.Strings(out)
	return out
}

func accumulateSummary(summary *ComparisonSummary, before SampleResult, after SampleResult, status SampleDiffStatus) {
	if summary == nil {
		return
	}

	if sampleOutcomeRank(before) >= 1 {
		summary.BeforeHitSamples++
	}
	if sampleOutcomeRank(after) >= 1 {
		summary.AfterHitSamples++
	}
	if sampleOutcomeRank(before) == 2 {
		summary.BeforeDownloadedSamples++
	}
	if sampleOutcomeRank(after) == 2 {
		summary.AfterDownloadedSamples++
	}
	summary.BeforeAverageAttemptCount += float64(len(before.Attempts))
	summary.AfterAverageAttemptCount += float64(len(after.Attempts))

	if resultIncludesProvider(before, "subhd") {
		summary.BeforeSubHDAttemptSamples++
	}
	if resultIncludesProvider(after, "subhd") {
		summary.AfterSubHDAttemptSamples++
	}
	if resultHasProviderHit(before, "subhd") {
		summary.BeforeSubHDHitSamples++
	}
	if resultHasProviderHit(after, "subhd") {
		summary.AfterSubHDHitSamples++
	}
	if resultHasProviderDownload(before, "subhd") {
		summary.BeforeSubHDDownloadSamples++
	}
	if resultHasProviderDownload(after, "subhd") {
		summary.AfterSubHDDownloadSamples++
	}
	if resultHasProviderFailureCategory(before, "subhd", FailureCaptchaOCR) {
		summary.BeforeSubHDCaptchaOCRSamples++
	}
	if resultHasProviderFailureCategory(after, "subhd", FailureCaptchaOCR) {
		summary.AfterSubHDCaptchaOCRSamples++
	}

	summary.BeforePrimaryFailureCounts[before.PrimaryFailure]++
	summary.AfterPrimaryFailureCounts[after.PrimaryFailure]++

	switch status {
	case SampleDiffImproved:
		summary.ImprovedSamples++
	case SampleDiffRegressed:
		summary.RegressedSamples++
	case SampleDiffChanged:
		summary.ChangedSamples++
	default:
		summary.UnchangedSamples++
	}
}

func resultIncludesProvider(result SampleResult, provider string) bool {
	for _, attempt := range result.Attempts {
		if attempt.Provider == provider {
			return true
		}
	}
	return false
}

func resultHasProviderHit(result SampleResult, provider string) bool {
	for _, attempt := range result.Attempts {
		if attempt.Provider == provider && attempt.Hit {
			return true
		}
	}
	return false
}

func resultHasProviderDownload(result SampleResult, provider string) bool {
	for _, attempt := range result.Attempts {
		if attempt.Provider == provider && attempt.Downloaded {
			return true
		}
	}
	return false
}

func resultHasProviderFailureCategory(result SampleResult, provider string, category FailureCategory) bool {
	for _, attempt := range result.Attempts {
		if attempt.Provider == provider && attempt.FailureCategory == category {
			return true
		}
	}
	return false
}

func accumulateProviderStats(providerStats map[string]*ProviderComparison, result SampleResult, isBefore bool) {
	for _, attempt := range result.Attempts {
		stats, ok := providerStats[attempt.Provider]
		if ok == false {
			stats = &ProviderComparison{Provider: attempt.Provider}
			providerStats[attempt.Provider] = stats
		}

		if isBefore {
			if attempt.Hit {
				stats.BeforeHits++
			}
			if attempt.Downloaded {
				stats.BeforeDownloads++
			}
			continue
		}

		if attempt.Hit {
			stats.AfterHits++
		}
		if attempt.Downloaded {
			stats.AfterDownloads++
		}
	}
}

func stringSlicesEqual(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
