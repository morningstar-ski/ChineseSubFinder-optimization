package common

import "testing"

func TestDefaultSubSiteSequence(t *testing.T) {
	got := DefaultSubSiteSequence()
	want := []string{
		SubSiteSubtitleBest,
		SubSiteOpenSubtitles,
		SubSiteTVSubtitles,
		SubSiteMovieSubtitles,
		SubSiteAssrt,
		SubSiteSubDL,
		SubSiteSubHd,
		SubSiteShooter,
		SubSiteXunLei,
	}

	if len(got) != len(want) {
		t.Fatalf("DefaultSubSiteSequence len = %d; want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("DefaultSubSiteSequence[%d] = %q; want %q", i, got[i], want[i])
		}
	}
}

func TestOrderSubSiteNames(t *testing.T) {
	got := OrderSubSiteNames(
		[]string{SubSiteShooter, "custom_site", SubSiteAssrt, SubSiteOpenSubtitles, SubSiteSubDL, SubSiteSubHd, SubSiteShooter},
		DefaultSubSiteSequence(),
	)
	want := []string{
		SubSiteOpenSubtitles,
		SubSiteAssrt,
		SubSiteSubDL,
		SubSiteSubHd,
		SubSiteShooter,
		"custom_site",
	}

	if len(got) != len(want) {
		t.Fatalf("OrderSubSiteNames len = %d; want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("OrderSubSiteNames[%d] = %q; want %q", i, got[i], want[i])
		}
	}
}
