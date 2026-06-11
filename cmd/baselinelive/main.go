package main

import (
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"os"
	"path/filepath"

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
	cacheName := flag.String("cache-name", "", "optional stable cache namespace for download reuse")
	freshCache := flag.Bool("fresh-cache", false, "clear the selected baselinelive cache before replay")
	liteMode := flag.Bool("lite-mode", true, "run in lite mode and skip browser-only suppliers such as subhd")
	flag.Parse()

	if err := run(*inputPath, *outputPath, *csvPath, *configRoot, *checkSuppliers, *cacheName, *freshCache, *liteMode); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(inputPath string, outputPath string, csvPath string, configRoot string, checkSuppliers bool, cacheName string, freshCache bool, liteMode bool) error {
	evaluator, cleanup, err := buildLiveEvaluator(configRoot, checkSuppliers, cacheName, freshCache, liteMode)
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

func buildLiveEvaluator(configRoot string, checkSuppliers bool, cacheNameOverride string, freshCache bool, liteMode bool) (baseline.Evaluator, func(), error) {
	resolvedConfigRoot := resolveConfigRoot(configRoot)
	restoreWorkRoot, err := enterWorkRoot(resolvedConfigRoot)
	if err != nil {
		return nil, nil, err
	}
	pkg.SetLiteMode(liteMode)
	pkg.SetLinuxConfigPathInSelfPath(resolvedConfigRoot)
	settings.SetConfigRootPath(resolvedConfigRoot)
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	authKey := loadAuthKey(logger)
	cacheName := resolveCacheName(resolvedConfigRoot, cacheNameOverride)
	if freshCache {
		cache_center.DelDb(cacheName)
	}
	cacheCenter := cache_center.NewCacheCenter(cacheName, logger)
	fileDownloaderInstance := file_downloader.NewFileDownloader(cacheCenter, authKey)
	tmdbHelper, err := buildTmdbHelperFromSettings(logger)
	if err != nil {
		cacheCenter.Close()
		cache_center.DelDb(cacheName)
		restoreWorkRoot()
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
		restoreWorkRoot()
		return nil, nil, err
	}

	cleanup := func() {
		cacheCenter.Close()
		restoreWorkRoot()
	}

	return baseline.NewSupplierEvaluator(logger, process.SubSupplierHub.Suppliers...), cleanup, nil
}

func resolveConfigRoot(configRoot string) string {
	if configRoot != "" {
		return configRoot
	}

	return pkg.ConfigRootDirFPath()
}

func resolveCacheName(configRoot string, cacheNameOverride string) string {
	if cacheNameOverride != "" {
		return cacheNameOverride
	}

	absConfigRoot, err := filepath.Abs(configRoot)
	if err != nil {
		absConfigRoot = configRoot
	}

	cacheKey := fmt.Sprintf("%x", sha256.Sum256([]byte(absConfigRoot)))
	return "baseline_live_" + cacheKey[:16]
}

func enterWorkRoot(configRoot string) (func(), error) {
	absConfigRoot, err := filepath.Abs(configRoot)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(absConfigRoot, os.ModePerm); err != nil {
		return nil, err
	}

	originalWorkingDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	if err := os.Chdir(absConfigRoot); err != nil {
		return nil, err
	}

	return func() {
		_ = os.Chdir(originalWorkingDir)
	}, nil
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
