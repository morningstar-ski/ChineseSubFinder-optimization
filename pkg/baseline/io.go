package baseline

import (
	"encoding/json"
	"os"
)

func LoadManifest(inputPath string) (Manifest, error) {
	bytes, err := os.ReadFile(inputPath)
	if err != nil {
		return Manifest{}, err
	}

	var manifest Manifest
	if err := json.Unmarshal(bytes, &manifest); err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}

func SaveManifest(outputPath string, manifest Manifest) error {
	if err := manifest.Validate(); err != nil {
		return err
	}

	bytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	bytes = append(bytes, '\n')
	return os.WriteFile(outputPath, bytes, 0o644)
}

func LoadResults(inputPath string) ([]SampleResult, error) {
	bytes, err := os.ReadFile(inputPath)
	if err != nil {
		return nil, err
	}

	var results []SampleResult
	if err := json.Unmarshal(bytes, &results); err != nil {
		return nil, err
	}

	return results, nil
}

func SaveResults(outputPath string, results []SampleResult) error {
	if err := ValidateResults(results); err != nil {
		return err
	}

	bytes, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return err
	}

	bytes = append(bytes, '\n')
	return os.WriteFile(outputPath, bytes, 0o644)
}

func SaveComparison(outputPath string, comparison Comparison) error {
	bytes, err := json.MarshalIndent(comparison, "", "  ")
	if err != nil {
		return err
	}

	bytes = append(bytes, '\n')
	return os.WriteFile(outputPath, bytes, 0o644)
}
