package baseline

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
)

type FailureCategory string

const (
	FailureNone          FailureCategory = ""
	FailureNoProviderHit FailureCategory = "no_provider_hit"
	FailureKeywordMiss   FailureCategory = "keyword_search_miss"
	FailureBadArchive    FailureCategory = "bad_archive"
	FailureDeadProvider  FailureCategory = "dead_provider"
	FailureDownloadError FailureCategory = "download_error"
)

type SampleKind string

const (
	SampleMovie   SampleKind = "movie"
	SampleEpisode SampleKind = "episode"
)

type Sample struct {
	ID        string     `json:"id"`
	VideoPath string     `json:"video_path"`
	Kind      SampleKind `json:"kind"`
	Season    int        `json:"season,omitempty"`
	Episode   int        `json:"episode,omitempty"`
}

type ProviderAttempt struct {
	Provider        string          `json:"provider"`
	Hit             bool            `json:"hit"`
	Downloaded      bool            `json:"downloaded"`
	FailureCategory FailureCategory `json:"failure_category,omitempty"`
	Note            string          `json:"note,omitempty"`
}

type SampleResult struct {
	Sample         Sample            `json:"sample"`
	PrimaryFailure FailureCategory   `json:"primary_failure,omitempty"`
	Attempts       []ProviderAttempt `json:"attempts"`
}

func (r SampleResult) Validate() error {
	if err := validateSample(r.Sample); err != nil {
		return err
	}
	if len(r.Attempts) == 0 {
		return fmt.Errorf("sample %q requires at least one provider attempt", r.Sample.ID)
	}
	if err := validateFailureCategory(r.PrimaryFailure); err != nil {
		return fmt.Errorf("sample %q primary failure: %w", r.Sample.ID, err)
	}

	downloaded := false
	for _, attempt := range r.Attempts {
		if attempt.Provider == "" {
			return fmt.Errorf("sample %q has attempt with empty provider", r.Sample.ID)
		}
		if err := validateFailureCategory(attempt.FailureCategory); err != nil {
			return fmt.Errorf("sample %q provider %q: %w", r.Sample.ID, attempt.Provider, err)
		}
		if attempt.Downloaded {
			downloaded = true
		}
		if attempt.Downloaded && attempt.FailureCategory != FailureNone {
			return fmt.Errorf("sample %q provider %q cannot be downloaded and failed at once", r.Sample.ID, attempt.Provider)
		}
	}

	if downloaded {
		if r.PrimaryFailure != FailureNone {
			return fmt.Errorf("sample %q has primary failure %q despite successful download", r.Sample.ID, r.PrimaryFailure)
		}
	} else if r.PrimaryFailure == FailureNone {
		return fmt.Errorf("sample %q failed overall but has no primary failure", r.Sample.ID)
	}

	return nil
}

func validateSample(sample Sample) error {
	if sample.ID == "" {
		return fmt.Errorf("sample id is required")
	}
	if sample.VideoPath == "" {
		return fmt.Errorf("sample video path is required")
	}
	if sample.Kind != SampleMovie && sample.Kind != SampleEpisode {
		return fmt.Errorf("sample kind %q is invalid", sample.Kind)
	}
	if sample.Kind == SampleEpisode {
		if sample.Season <= 0 || sample.Episode <= 0 {
			return fmt.Errorf("episode sample requires positive season and episode")
		}
	}
	return nil
}

func WriteCSV(w io.Writer, results []SampleResult) error {
	writer := csv.NewWriter(w)
	if err := writer.Write([]string{
		"sample_id",
		"video_path",
		"sample_kind",
		"season",
		"episode",
		"provider",
		"hit",
		"downloaded",
		"primary_failure",
		"provider_failure",
		"note",
	}); err != nil {
		return err
	}

	for _, result := range results {
		if err := result.Validate(); err != nil {
			return err
		}
		for _, attempt := range result.Attempts {
			row := []string{
				result.Sample.ID,
				result.Sample.VideoPath,
				string(result.Sample.Kind),
				strconv.Itoa(result.Sample.Season),
				strconv.Itoa(result.Sample.Episode),
				attempt.Provider,
				strconv.FormatBool(attempt.Hit),
				strconv.FormatBool(attempt.Downloaded),
				string(result.PrimaryFailure),
				string(attempt.FailureCategory),
				attempt.Note,
			}
			if err := writer.Write(row); err != nil {
				return err
			}
		}
	}

	writer.Flush()
	return writer.Error()
}

func validateFailureCategory(category FailureCategory) error {
	switch category {
	case FailureNone,
		FailureNoProviderHit,
		FailureKeywordMiss,
		FailureBadArchive,
		FailureDeadProvider,
		FailureDownloadError:
		return nil
	default:
		return fmt.Errorf("unknown failure category %q", category)
	}
}
