package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/baseline"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/sirupsen/logrus"
)

func main() {
	movieRootsArg := flag.String("movie-roots", "", "movie root directories separated by ';'")
	seriesRootsArg := flag.String("series-roots", "", "series root directories separated by ';'")
	movieLimit := flag.Int("movies", 20, "maximum number of movie samples")
	episodeLimit := flag.Int("episodes", 50, "maximum number of episode samples")
	outputPath := flag.String("out", "", "path to output manifest JSON file")
	flag.Parse()

	if err := run(*movieRootsArg, *seriesRootsArg, *movieLimit, *episodeLimit, *outputPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(movieRootsArg string, seriesRootsArg string, movieLimit int, episodeLimit int, outputPath string) error {
	if outputPath == "" {
		return fmt.Errorf("usage: baselineseed -movie-roots C:\\Movies -series-roots C:\\TV -movies 20 -episodes 50 -out baseline-samples.json")
	}

	settings.SetConfigRootPath(pkg.ConfigRootDirFPath())
	manifest, err := baseline.BuildManifest(
		logrus.New(),
		splitRootsArg(movieRootsArg),
		splitRootsArg(seriesRootsArg),
		movieLimit,
		episodeLimit,
	)
	if err != nil {
		return err
	}

	return baseline.SaveManifest(outputPath, manifest)
}

func splitRootsArg(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ';' || r == ','
	})

	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}

	return out
}
