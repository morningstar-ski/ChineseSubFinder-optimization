package baseline

import (
	"context"
	"errors"
	"testing"
)

func TestRunnerRun(t *testing.T) {
	manifest := Manifest{
		Samples: []Sample{
			{ID: "movie-001", VideoPath: "C:\\Media\\Movie.mkv", Kind: SampleMovie},
			{ID: "tv-001", VideoPath: "C:\\Media\\Show\\S01E01.mkv", Kind: SampleEpisode, Season: 1, Episode: 1},
		},
	}

	runner := Runner{
		Evaluator: EvaluatorFunc(func(ctx context.Context, sample Sample) (Evaluation, error) {
			switch sample.ID {
			case "movie-001":
				return Evaluation{
					Attempts: []ProviderAttempt{
						{Provider: "subdl", Hit: true, Downloaded: true},
					},
				}, nil
			case "tv-001":
				return Evaluation{
					PrimaryFailure: FailureKeywordMiss,
					Attempts: []ProviderAttempt{
						{Provider: "assrt", Hit: false, Downloaded: false, FailureCategory: FailureKeywordMiss},
					},
				}, nil
			default:
				return Evaluation{}, errors.New("unexpected sample")
			}
		}),
	}

	results, err := runner.Run(context.Background(), manifest)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if results[0].Sample != manifest.Samples[0] {
		t.Fatalf("unexpected first result sample %#v", results[0].Sample)
	}
	if results[1].PrimaryFailure != FailureKeywordMiss {
		t.Fatalf("unexpected second primary failure %q", results[1].PrimaryFailure)
	}
}

func TestValidateResultsRejectsDuplicateSampleID(t *testing.T) {
	results := []SampleResult{
		{
			Sample: Sample{ID: "movie-001", VideoPath: "C:\\Media\\Movie1.mkv", Kind: SampleMovie},
			Attempts: []ProviderAttempt{
				{Provider: "subdl", Hit: true, Downloaded: true},
			},
		},
		{
			Sample: Sample{ID: "movie-001", VideoPath: "C:\\Media\\Movie2.mkv", Kind: SampleMovie},
			Attempts: []ProviderAttempt{
				{Provider: "assrt", Hit: true, Downloaded: true},
			},
		},
	}

	if err := ValidateResults(results); err == nil {
		t.Fatal("expected duplicate sample id to be rejected")
	}
}
