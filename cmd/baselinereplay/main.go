package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/baseline"
)

func main() {
	inputPath := flag.String("in", "", "path to input JSON manifest file")
	fixturePath := flag.String("fixture", "", "path to fixture JSON file containing []baseline.SampleResult")
	outputPath := flag.String("out", "", "path to output JSON results file")
	flag.Parse()

	if err := run(*inputPath, *fixturePath, *outputPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(inputPath string, fixturePath string, outputPath string) error {
	if inputPath == "" || fixturePath == "" || outputPath == "" {
		return fmt.Errorf("usage: baselinereplay -in baseline-samples.json -fixture fixture-results.json -out results.json")
	}

	manifest, err := baseline.LoadManifest(inputPath)
	if err != nil {
		return err
	}
	if err := manifest.Validate(); err != nil {
		return err
	}

	fixtures, err := baseline.LoadResults(fixturePath)
	if err != nil {
		return err
	}

	evaluator, err := newFixtureEvaluator(fixtures)
	if err != nil {
		return err
	}

	results, err := baseline.Runner{Evaluator: evaluator}.Run(context.Background(), manifest)
	if err != nil {
		return err
	}

	return baseline.SaveResults(outputPath, results)
}

type fixtureEvaluator struct {
	results map[string]baseline.SampleResult
}

func newFixtureEvaluator(fixtures []baseline.SampleResult) (baseline.Evaluator, error) {
	if err := baseline.ValidateResults(fixtures); err != nil {
		return nil, err
	}

	results := make(map[string]baseline.SampleResult, len(fixtures))
	for _, fixture := range fixtures {
		results[fixture.Sample.ID] = fixture
	}

	return fixtureEvaluator{results: results}, nil
}

func (f fixtureEvaluator) Evaluate(ctx context.Context, sample baseline.Sample) (baseline.Evaluation, error) {
	result, ok := f.results[sample.ID]
	if !ok {
		return baseline.Evaluation{}, fmt.Errorf("fixture missing sample %q", sample.ID)
	}
	if result.Sample != sample {
		return baseline.Evaluation{}, fmt.Errorf("fixture sample %q does not match manifest", sample.ID)
	}

	return baseline.Evaluation{
		PrimaryFailure: result.PrimaryFailure,
		Attempts:       append([]baseline.ProviderAttempt(nil), result.Attempts...),
	}, nil
}
