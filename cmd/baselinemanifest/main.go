package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/baseline"
)

func main() {
	inputPath := flag.String("in", "", "path to input JSON manifest file")
	flag.Parse()

	if *inputPath == "" {
		fmt.Fprintln(os.Stderr, "usage: baselinemanifest -in baseline-samples.json")
		os.Exit(2)
	}

	manifest, err := loadManifest(*inputPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := manifest.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	movies, episodes := manifest.CountByKind()
	fmt.Printf("samples=%d movies=%d episodes=%d\n", len(manifest.Samples), movies, episodes)
}

func loadManifest(inputPath string) (baseline.Manifest, error) {
	return baseline.LoadManifest(inputPath)
}
