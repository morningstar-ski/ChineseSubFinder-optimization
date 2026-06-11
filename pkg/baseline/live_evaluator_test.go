package baseline

import (
	"context"
	"errors"
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/ifaces"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/sub_supplier/subhd"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/language"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/series"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/supplier"
	"github.com/sirupsen/logrus"
)

func TestSupplierEvaluatorMovieSuccess(t *testing.T) {
	evaluator := NewSupplierEvaluator(logrus.New(), fakeSupplier{
		name: "subdl",
		movieSubInfos: []supplier.SubInfo{
			validSubInfo("subdl", "Movie.2024.1080p.srt", ".srt", simpleSRT),
		},
	})

	evaluation, err := evaluator.Evaluate(context.Background(), Sample{
		ID:        "movie-001",
		VideoPath: "C:\\Media\\Movie.2024.1080p.mkv",
		Kind:      SampleMovie,
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if evaluation.PrimaryFailure != FailureNone {
		t.Fatalf("unexpected primary failure %q", evaluation.PrimaryFailure)
	}
	if len(evaluation.Attempts) != 1 || evaluation.Attempts[0].Downloaded == false {
		t.Fatalf("unexpected attempts %#v", evaluation.Attempts)
	}
}

func TestSupplierEvaluatorStopsAfterMovieSuccess(t *testing.T) {
	evaluator := NewSupplierEvaluator(logrus.New(),
		fakeSupplier{
			name: "subdl",
			movieSubInfos: []supplier.SubInfo{
				validSubInfo("subdl", "Movie.2024.1080p.srt", ".srt", simpleSRT),
			},
		},
		fakeSupplier{
			name: "subhd",
			movieSubInfos: []supplier.SubInfo{
				validSubInfo("subhd", "Movie.2024.1080p.srt", ".srt", simpleSRT),
			},
		},
	)

	evaluation, err := evaluator.Evaluate(context.Background(), Sample{
		ID:        "movie-stop-001",
		VideoPath: "C:\\Media\\Movie.2024.1080p.mkv",
		Kind:      SampleMovie,
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if len(evaluation.Attempts) != 1 {
		t.Fatalf("expected early stop after first success, got %#v", evaluation.Attempts)
	}
	if evaluation.Attempts[0].Provider != "subdl" || evaluation.Attempts[0].Downloaded == false {
		t.Fatalf("unexpected first attempt %#v", evaluation.Attempts[0])
	}
}

func TestSupplierEvaluatorBadArchive(t *testing.T) {
	evaluator := NewSupplierEvaluator(logrus.New(), fakeSupplier{
		name: "assrt",
		movieSubInfos: []supplier.SubInfo{
			validSubInfo("assrt", "Movie.2024.1080p.zip", ".zip", []byte("not-a-zip")),
		},
	})

	evaluation, err := evaluator.Evaluate(context.Background(), Sample{
		ID:        "movie-002",
		VideoPath: "C:\\Media\\Movie.2024.1080p.mkv",
		Kind:      SampleMovie,
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if evaluation.PrimaryFailure != FailureBadArchive {
		t.Fatalf("unexpected primary failure %q", evaluation.PrimaryFailure)
	}
	if got := evaluation.Attempts[0].FailureCategory; got != FailureBadArchive {
		t.Fatalf("unexpected provider failure %q", got)
	}
}

func TestSupplierEvaluatorDeadProvider(t *testing.T) {
	evaluator := NewSupplierEvaluator(logrus.New(),
		fakeSupplier{name: "subdl"},
		fakeSupplier{name: "subtitle_best", err: errors.New("connection refused"), alive: false},
	)

	evaluation, err := evaluator.Evaluate(context.Background(), Sample{
		ID:        "movie-003",
		VideoPath: "C:\\Media\\Movie.2024.1080p.mkv",
		Kind:      SampleMovie,
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if evaluation.PrimaryFailure != FailureDeadProvider {
		t.Fatalf("unexpected primary failure %q", evaluation.PrimaryFailure)
	}
	if len(evaluation.Attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %d", len(evaluation.Attempts))
	}
}

func TestSupplierEvaluatorStopsAfterSeriesSuccess(t *testing.T) {
	evaluator := NewSupplierEvaluator(logrus.New(),
		fakeSupplier{
			name: "subtitle_best",
			seriesSubInfos: []supplier.SubInfo{
				validSeriesSubInfo("subtitle_best", "Show.S01E02.ass", ".ass", 1, 2, simpleASS),
			},
		},
		fakeSupplier{
			name: "subhd",
			seriesSubInfos: []supplier.SubInfo{
				validSeriesSubInfo("subhd", "Show.S01E02.srt", ".srt", 1, 2, simpleSRT),
			},
		},
	)

	evaluation, err := evaluator.Evaluate(context.Background(), Sample{
		ID:        "series-stop-001",
		VideoPath: "C:\\Media\\Show\\Season 1\\Show.S01E02.mkv",
		Kind:      SampleEpisode,
		Season:    1,
		Episode:   2,
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if len(evaluation.Attempts) != 1 {
		t.Fatalf("expected early stop after first series success, got %#v", evaluation.Attempts)
	}
	if evaluation.Attempts[0].Provider != "subtitle_best" || evaluation.Attempts[0].Downloaded == false {
		t.Fatalf("unexpected first attempt %#v", evaluation.Attempts[0])
	}
}

func TestSupplierEvaluatorClassifiesSubHDCaptchaOCRFailure(t *testing.T) {
	evaluator := NewSupplierEvaluator(logrus.New(), fakeSupplier{
		name:  "subhd",
		err:   errors.New(subhd.ReasonCaptchaOcrFailed + ": unexpected captcha OCR output"),
		alive: true,
	})

	evaluation, err := evaluator.Evaluate(context.Background(), Sample{
		ID:        "movie-ocr-001",
		VideoPath: "C:\\Media\\Movie.2024.1080p.mkv",
		Kind:      SampleMovie,
	})
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	if evaluation.PrimaryFailure != FailureCaptchaOCR {
		t.Fatalf("unexpected primary failure %q", evaluation.PrimaryFailure)
	}
	if len(evaluation.Attempts) != 1 || evaluation.Attempts[0].FailureCategory != FailureCaptchaOCR {
		t.Fatalf("unexpected attempts %#v", evaluation.Attempts)
	}
}

type fakeSupplier struct {
	name           string
	alive          bool
	err            error
	movieSubInfos  []supplier.SubInfo
	seriesSubInfos []supplier.SubInfo
}

var _ ifaces.ISupplier = fakeSupplier{}

func (f fakeSupplier) CheckAlive() (bool, int64) {
	return f.IsAlive(), 0
}

func (f fakeSupplier) IsAlive() bool {
	return f.alive || (f.alive == false && f.err == nil)
}

func (f fakeSupplier) GetSupplierName() string {
	return f.name
}

func (f fakeSupplier) OverDailyDownloadLimit() bool {
	return false
}

func (f fakeSupplier) GetLogger() *logrus.Logger {
	return logrus.New()
}

func (f fakeSupplier) GetSubListFromFile4Movie(filePath string) ([]supplier.SubInfo, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]supplier.SubInfo(nil), f.movieSubInfos...), nil
}

func (f fakeSupplier) GetSubListFromFile4Series(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]supplier.SubInfo(nil), f.seriesSubInfos...), nil
}

func (f fakeSupplier) GetSubListFromFile4Anime(seriesInfo *series.SeriesInfo) ([]supplier.SubInfo, error) {
	return f.GetSubListFromFile4Series(seriesInfo)
}

func validSubInfo(fromWhere string, name string, ext string, data []byte) supplier.SubInfo {
	return *supplier.NewSubInfo(fromWhere, 0, name, language.ChineseSimple, "https://example.com/"+name, 0, 0, ext, data)
}

func validSeriesSubInfo(fromWhere string, name string, ext string, season int, episode int, data []byte) supplier.SubInfo {
	info := supplier.NewSubInfo(fromWhere, 0, name, language.ChineseSimple, "https://example.com/"+name, 0, 0, ext, data)
	info.Season = season
	info.Episode = episode
	return *info
}

var simpleSRT = []byte("1\n00:00:01,000 --> 00:00:02,000\nhello\n")
var simpleASS = []byte("[Script Info]\nTitle: test\n\n[Events]\nFormat: Layer, Start, End, Style, Name, MarginL, MarginR, MarginV, Effect, Text\nDialogue: 0,0:00:01.00,0:00:02.00,Default,,0,0,0,,hello\n")
