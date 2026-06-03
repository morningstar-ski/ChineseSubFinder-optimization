package baseline

import (
	"context"
	"fmt"
)

type Evaluation struct {
	PrimaryFailure FailureCategory
	Attempts       []ProviderAttempt
}

type Evaluator interface {
	Evaluate(ctx context.Context, sample Sample) (Evaluation, error)
}

type EvaluatorFunc func(ctx context.Context, sample Sample) (Evaluation, error)

func (f EvaluatorFunc) Evaluate(ctx context.Context, sample Sample) (Evaluation, error) {
	return f(ctx, sample)
}

type Runner struct {
	Evaluator Evaluator
}

func (r Runner) Run(ctx context.Context, manifest Manifest) ([]SampleResult, error) {
	if r.Evaluator == nil {
		return nil, fmt.Errorf("runner requires evaluator")
	}
	if err := manifest.Validate(); err != nil {
		return nil, err
	}

	results := make([]SampleResult, 0, len(manifest.Samples))
	for _, sample := range manifest.Samples {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		evaluation, err := r.Evaluator.Evaluate(ctx, sample)
		if err != nil {
			return nil, fmt.Errorf("sample %q: %w", sample.ID, err)
		}

		result := SampleResult{
			Sample:         sample,
			PrimaryFailure: evaluation.PrimaryFailure,
			Attempts:       append([]ProviderAttempt(nil), evaluation.Attempts...),
		}
		if err := result.Validate(); err != nil {
			return nil, err
		}

		results = append(results, result)
	}

	return results, nil
}

func ValidateResults(results []SampleResult) error {
	seen := make(map[string]struct{}, len(results))
	for _, result := range results {
		if err := result.Validate(); err != nil {
			return err
		}
		if _, ok := seen[result.Sample.ID]; ok {
			return fmt.Errorf("duplicate sample result %q", result.Sample.ID)
		}
		seen[result.Sample.ID] = struct{}{}
	}

	return nil
}
