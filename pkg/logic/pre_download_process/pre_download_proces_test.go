package pre_download_process

import (
	"testing"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/file_downloader"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/settings"
	"github.com/sirupsen/logrus"
)

func TestBuildEnabledSuppliersOrdersProvidersByDefaultSequence(t *testing.T) {
	settings.SetConfigRootPath(t.TempDir())
	cfg := settings.Get()

	cfg.SubtitleSources.AssrtSettings.Enabled = true
	cfg.SubtitleSources.AssrtSettings.Token = "assrt-token"
	cfg.SubtitleSources.SubDLSettings.Enabled = true
	cfg.SubtitleSources.SubDLSettings.Key = "subdl-key"
	cfg.SubtitleSources.SubtitleBestSettings.Enabled = true
	cfg.SubtitleSources.SubtitleBestSettings.ApiKey = "subtitle-best-key"
	cfg.SubtitleSources.OpenSubtitlesSettings.Enabled = true
	cfg.SubtitleSources.OpenSubtitlesSettings.ApiKey = "open-key"
	cfg.SubtitleSources.OpenSubtitlesSettings.Username = "user"
	cfg.SubtitleSources.OpenSubtitlesSettings.Password = "pass"
	cfg.SubtitleSources.TVsubtitlesSettings.Enabled = true
	cfg.SubtitleSources.MoviesubtitlesSettings.Enabled = true
	cfg.SubtitleSources.SubHDSettings.Enabled = true

	process := &PreDownloadProcess{
		fileDownloader: &file_downloader.FileDownloader{
			Log: logrus.New(),
		},
	}
	gotSuppliers := process.buildEnabledSuppliers()
	got := make([]string, 0, len(gotSuppliers))
	for _, item := range gotSuppliers {
		got = append(got, item.GetSupplierName())
	}

	want := []string{
		"subtitle_best",
		"opensubtitles",
		"tvsubtitles",
		"moviesubtitles",
		"assrt",
		"subdl",
		"subhd",
		"shooter",
		"xunlei",
	}

	if len(got) != len(want) {
		t.Fatalf("buildEnabledSuppliers() len = %d; want %d; got %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("buildEnabledSuppliers()[%d] = %q; want %q; full = %v", i, got[i], want[i], got)
		}
	}
}
