package baseline

import (
	"bytes"
	"strings"
	"testing"
)

func TestSampleResultValidateRejectsMissingPrimaryFailure(t *testing.T) {
	result := SampleResult{
		Sample: Sample{
			ID:        "tv-001",
			VideoPath: "C:\\Media\\Show\\S01E01.mkv",
			Kind:      SampleEpisode,
			Season:    1,
			Episode:   1,
		},
		Attempts: []ProviderAttempt{
			{Provider: "assrt", Hit: false, Downloaded: false, FailureCategory: FailureKeywordMiss},
		},
	}

	if err := result.Validate(); err == nil {
		t.Fatal("expected missing primary failure to be rejected")
	}
}

func TestSampleResultValidateAcceptsSuccessfulDownload(t *testing.T) {
	result := SampleResult{
		Sample: Sample{
			ID:        "movie-001",
			VideoPath: "C:\\Media\\Movie.mkv",
			Kind:      SampleMovie,
		},
		Attempts: []ProviderAttempt{
			{Provider: "subdl", Hit: true, Downloaded: true},
		},
	}

	if err := result.Validate(); err != nil {
		t.Fatalf("expected valid successful result, got %v", err)
	}
}

func TestWriteCSV(t *testing.T) {
	results := []SampleResult{
		{
			Sample: Sample{
				ID:        "tv-001",
				VideoPath: "C:\\Media\\Show\\S01E01.mkv",
				Kind:      SampleEpisode,
				Season:    1,
				Episode:   1,
			},
			PrimaryFailure: FailureKeywordMiss,
			Attempts: []ProviderAttempt{
				{Provider: "assrt", Hit: false, Downloaded: false, FailureCategory: FailureKeywordMiss, Note: "fallback exhausted"},
				{Provider: "subdl", Hit: false, Downloaded: false, FailureCategory: FailureNoProviderHit, Note: "no candidates"},
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteCSV(&buf, results); err != nil {
		t.Fatalf("WriteCSV returned error: %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		"sample_id,video_path,sample_kind,season,episode,provider,hit,downloaded,primary_failure,provider_failure,note",
		"tv-001,C:\\Media\\Show\\S01E01.mkv,episode,1,1,assrt,false,false,keyword_search_miss,keyword_search_miss,fallback exhausted",
		"tv-001,C:\\Media\\Show\\S01E01.mkv,episode,1,1,subdl,false,false,keyword_search_miss,no_provider_hit,no candidates",
	} {
		if strings.Contains(output, want) == false {
			t.Fatalf("csv output missing %q\n%s", want, output)
		}
	}
}
