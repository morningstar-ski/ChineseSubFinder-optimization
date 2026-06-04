package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/baseline"
)

func main() {
	inputPath := flag.String("in", "", "path to input JSON file containing []baseline.SampleResult")
	outputPath := flag.String("out", "", "path to output CSV file")
	flag.Parse()

	if *inputPath == "" || *outputPath == "" {
		fmt.Fprintln(os.Stderr, "usage: baselinecsv -in results.json -out baseline.csv")
		os.Exit(2)
	}

	results, err := loadResults(*inputPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	outputFile, err := os.Create(*outputPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer outputFile.Close()

	if err := baseline.WriteCSV(outputFile, results); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func loadResults(inputPath string) ([]baseline.SampleResult, error) {
	return baseline.LoadResults(inputPath)
}
