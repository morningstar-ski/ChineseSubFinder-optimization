package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/baseline"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/cache_center"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/pre_download_process"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/random_auth_key"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/tmdb_api"
	"github.com/sirupsen/logrus"
)

func main() {
	inputPath := flag.String("in", "", "path to input JSON manifest file")
	outputPath := flag.String("out", "", "path to output JSON results file")
	csvPath := flag.String("csv", "", "optional path to output CSV file")
	configRoot := flag.String("config-root", "", "optional path to ChineseSubFinder config root")
	checkSuppliers := flag.Bool("check-suppliers", false, "check supplier availability before replay")
	flag.Parse()

	if err := run(*inputPath, *outputPath, *csvPath, *configRoot, *checkSuppliers); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(inputPath string, outputPath string, csvPath string, configRoot string, checkSuppliers bool) error {
	evaluator, cleanup, err := buildLiveEvaluator(configRoot, checkSuppliers)
	if err != nil {
		return err
	}
	defer cleanup()

	return runWithEvaluator(inputPath, outputPath, csvPath, evaluator)
}

func runWithEvaluator(inputPath string, outputPath string, csvPath string, evaluator baseline.Evaluator) error {
	if inputPath == "" || outputPath == "" {
		return fmt.Errorf("usage: baselinelive -in baseline-samples.json -out results.json [-csv baseline.csv] [-config-root /config] [-check-suppliers]")
	}

	manifest, err := baseline.LoadManifest(inputPath)
	if err != nil {
		return err
	}
	if err := manifest.Validate(); err != nil {
		return err
	}

	results, err := baseline.Runner{Evaluator: evaluator}.Run(context.Background(), manifest)
	if err != nil {
		return err
	}
	if err := baseline.SaveResults(outputPath, results); err != nil {
		return err
	}
	if csvPath != "" {
		return writeCSV(csvPath, results)
	}

	return nil
}

func buildLiveEvaluator(configRoot string, checkSuppliers bool) (baseline.Evaluator, func(), error) {
	resolvedConfigRoot := resolveConfigRoot(configRoot)
	pkg.SetLiteMode(true)
	pkg.SetLinuxConfigPathInSelfPath(resolvedConfigRoot)
	settings.SetConfigRootPath(resolvedConfigRoot)
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	authKey := loadAuthKey(logger)
	cacheName := "baseline_live_" + time.Now().Format("20060102150405.000000000")
	cacheCenter := cache_center.NewCacheCenter(cacheName, logger)
	fileDownloaderInstance := file_downloader.NewFileDownloader(cacheCenter, authKey)
	tmdbHelper, err := buildTmdbHelperFromSettings(logger)
	if err != nil {
		cacheCenter.Close()
		cache_center.DelDb(cacheName)
		return nil, nil, err
	}
	fileDownloaderInstance.MediaInfoDealers.SetTmdbHelperInstance(tmdbHelper)

	process := pre_download_process.NewPreDownloadProcess(fileDownloaderInstance).Init()
	if checkSuppliers {
		process = process.Check()
	}
	if err := process.Wait(); err != nil {
		cacheCenter.Close()
		cache_center.DelDb(cacheName)
		return nil, nil, err
	}

	cleanup := func() {
		cacheCenter.Close()
		cache_center.DelDb(cacheName)
	}

	return baseline.NewSupplierEvaluator(logger, process.SubSupplierHub.Suppliers...), cleanup, nil
}

func resolveConfigRoot(configRoot string) string {
	if configRoot != "" {
		return configRoot
	}

	return pkg.ConfigRootDirFPath()
}

func loadAuthKey(logger *logrus.Logger) random_auth_key.AuthKey {
	if pkg.ReadCustomAuthFile(logger) == false {
		pkg.SetBaseKey(random_auth_key.BaseKey)
		pkg.SetAESKey16(random_auth_key.AESKey16)
		pkg.SetAESIv16(random_auth_key.AESIv16)
	}

	return random_auth_key.AuthKey{
		BaseKey:  pkg.BaseKey(),
		AESKey16: pkg.AESKey16(),
		AESIv16:  pkg.AESIv16(),
	}
}

func buildTmdbHelperFromSettings(logger *logrus.Logger) (*tmdb_api.TmdbApi, error) {
	tmdbSettings := settings.Get().AdvancedSettings.TmdbApiSettings
	if tmdbSettings.Enable == false || tmdbSettings.ApiKey == "" {
		return nil, nil
	}

	return tmdb_api.NewTmdbHelper(logger, tmdbSettings.ApiKey, tmdbSettings.UseAlternateBaseURL)
}

func writeCSV(outputPath string, results []baseline.SampleResult) error {
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	return baseline.WriteCSV(outputFile, results)
}
