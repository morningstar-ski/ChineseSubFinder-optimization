package baseline

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/ifaces"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/sub_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/sirupsen/logrus"
)

type SupplierEvaluator struct {
	Log       *logrus.Logger
	Suppliers []ifaces.ISupplier
}

func NewSupplierEvaluator(log *logrus.Logger, suppliers ...ifaces.ISupplier) *SupplierEvaluator {
	return &SupplierEvaluator{
		Log:       log,
		Suppliers: append([]ifaces.ISupplier(nil), suppliers...),
	}
}

func (e *SupplierEvaluator) Evaluate(ctx context.Context, sample Sample) (Evaluation, error) {
	if e == nil || e.Log == nil {
		return Evaluation{}, fmt.Errorf("supplier evaluator requires logger")
	}
	if len(e.Suppliers) == 0 {
		return Evaluation{}, fmt.Errorf("supplier evaluator requires at least one supplier")
	}
	if err := validateSample(sample); err != nil {
		return Evaluation{}, err
	}

	attempts := make([]ProviderAttempt, 0, len(e.Suppliers))
	for _, oneSupplier := range e.Suppliers {
		select {
		case <-ctx.Done():
			return Evaluation{}, ctx.Err()
		default:
		}

		attempts = append(attempts, e.evaluateSupplier(sample, oneSupplier))
	}

	return Evaluation{
		PrimaryFailure: summarizePrimaryFailure(attempts),
		Attempts:       attempts,
	}, nil
}

func (e *SupplierEvaluator) evaluateSupplier(sample Sample, oneSupplier ifaces.ISupplier) ProviderAttempt {
	attempt := ProviderAttempt{
		Provider: oneSupplier.GetSupplierName(),
	}

	subInfos, err := collectSupplierSubInfos(sample, oneSupplier)
	if err != nil {
		attempt.FailureCategory = classifyAttemptError(oneSupplier, err)
		attempt.Note = err.Error()
		return attempt
	}
	if len(subInfos) == 0 {
		attempt.FailureCategory = FailureNoProviderHit
		attempt.Note = "no subtitle candidates"
		return attempt
	}

	attempt.Hit = true
	organizedFiles, note := e.organizeAndSelect(sample, oneSupplier.GetSupplierName(), subInfos)
	attempt.Note = note
	if len(organizedFiles) > 0 {
		attempt.Downloaded = true
		return attempt
	}

	attempt.FailureCategory = FailureBadArchive
	return attempt
}

func collectSupplierSubInfos(sample Sample, oneSupplier ifaces.ISupplier) ([]supplier.SubInfo, error) {
	if sample.Kind == SampleMovie {
		return oneSupplier.GetSubListFromFile4Movie(sample.VideoPath)
	}

	return oneSupplier.GetSubListFromFile4Series(buildSampleSeriesInfo(sample))
}

func buildSampleSeriesInfo(sample Sample) *series.SeriesInfo {
	epsKey := pkg.GetEpisodeKeyName(sample.Season, sample.Episode)
	episodeInfo := series.EpisodeInfo{
		Title:        strings.TrimSuffix(filepath.Base(sample.VideoPath), filepath.Ext(sample.VideoPath)),
		Season:       sample.Season,
		Episode:      sample.Episode,
		Dir:          filepath.Dir(sample.VideoPath),
		FileFullPath: sample.VideoPath,
	}

	return &series.SeriesInfo{
		Name:             filepath.Base(filepath.Dir(sample.VideoPath)),
		DirPath:          filepath.Dir(sample.VideoPath),
		SeasonDict:       map[int]int{sample.Season: sample.Season},
		NeedDlSeasonDict: map[int]int{sample.Season: sample.Season},
		NeedDlEpsKeyList: map[string]series.EpisodeInfo{epsKey: episodeInfo},
	}
}

func (e *SupplierEvaluator) organizeAndSelect(sample Sample, supplierName string, subInfos []supplier.SubInfo) ([]string, string) {
	tmpFolderName := sanitizeTmpFolderName(sample.ID + "_" + supplierName)
	_ = pkg.ClearTmpFolderByName(tmpFolderName)
	defer func() {
		_ = pkg.ClearTmpFolderByName(tmpFolderName)
	}()

	organized, err := sub_helper.OrganizeDlSubFiles(e.Log, tmpFolderName, subInfos, sample.Kind == SampleMovie)
	if err != nil {
		return nil, err.Error()
	}

	files := pickOrganizedFiles(sample, organized)
	if len(files) == 0 {
		return nil, fmt.Sprintf("downloaded %d candidates but organized 0 matching subtitle files", len(subInfos))
	}

	return files, fmt.Sprintf("downloaded %d candidates and organized %d subtitle files", len(subInfos), len(files))
}

func pickOrganizedFiles(sample Sample, organized map[string][]string) []string {
	if len(organized) == 0 {
		return nil
	}

	if sample.Kind == SampleMovie {
		out := make([]string, 0)
		for _, files := range organized {
			out = append(out, files...)
		}
		return out
	}

	return append([]string(nil), organized[pkg.GetEpisodeKeyName(sample.Season, sample.Episode)]...)
}

func sanitizeTmpFolderName(input string) string {
	replacer := strings.NewReplacer(
		"\\", "_",
		"/", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
		" ", "_",
	)
	return "baseline_" + replacer.Replace(input)
}

func classifyAttemptError(oneSupplier ifaces.ISupplier, err error) FailureCategory {
	if err == nil {
		return FailureNone
	}
	if oneSupplier != nil && oneSupplier.IsAlive() == false {
		return FailureDeadProvider
	}

	lowerMsg := strings.ToLower(err.Error())
	for _, marker := range []string{
		"connection refused",
		"no such host",
		"timeout",
		"deadline exceeded",
		"proxyconnect",
		"tls handshake",
		"connection reset",
		"eof",
	} {
		if strings.Contains(lowerMsg, marker) {
			return FailureDeadProvider
		}
	}

	return FailureDownloadError
}

func summarizePrimaryFailure(attempts []ProviderAttempt) FailureCategory {
	allNoProviderHit := len(attempts) > 0
	hasDeadProvider := false
	hasDownloadError := false
	hasBadArchive := false
	hasKeywordMiss := false

	for _, attempt := range attempts {
		if attempt.Downloaded {
			return FailureNone
		}
		if attempt.FailureCategory != FailureNoProviderHit {
			allNoProviderHit = false
		}
		switch attempt.FailureCategory {
		case FailureDeadProvider:
			hasDeadProvider = true
		case FailureDownloadError:
			hasDownloadError = true
		case FailureBadArchive:
			hasBadArchive = true
		case FailureKeywordMiss:
			hasKeywordMiss = true
		}
	}

	switch {
	case allNoProviderHit:
		return FailureNoProviderHit
	case hasBadArchive:
		return FailureBadArchive
	case hasDownloadError:
		return FailureDownloadError
	case hasDeadProvider:
		return FailureDeadProvider
	case hasKeywordMiss:
		return FailureKeywordMiss
	default:
		return FailureNoProviderHit
	}
}
