package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/baseline"
)

func main() {
	beforePath := flag.String("before", "", "path to before JSON results file")
	afterPath := flag.String("after", "", "path to after JSON results file")
	outputPath := flag.String("out", "", "path to output JSON comparison file")
	csvPath := flag.String("csv", "", "optional path to output CSV diff file")
	flag.Parse()

	if err := run(*beforePath, *afterPath, *outputPath, *csvPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(beforePath string, afterPath string, outputPath string, csvPath string) error {
	if beforePath == "" || afterPath == "" || outputPath == "" {
		return fmt.Errorf("usage: baselinecompare -before before-results.json -after after-results.json -out comparison.json [-csv comparison.csv]")
	}

	beforeResults, err := baseline.LoadResults(beforePath)
	if err != nil {
		return err
	}
	afterResults, err := baseline.LoadResults(afterPath)
	if err != nil {
		return err
	}

	comparison, err := baseline.CompareResults(beforeResults, afterResults)
	if err != nil {
		return err
	}
	if err := baseline.SaveComparison(outputPath, comparison); err != nil {
		return err
	}
	if csvPath != "" {
		return writeComparisonCSV(csvPath, comparison)
	}

	return nil
}

func writeComparisonCSV(outputPath string, comparison baseline.Comparison) error {
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	return baseline.WriteComparisonCSV(outputFile, comparison)
}
