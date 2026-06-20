package common

import "testing"

func TestDefaultSubSiteSequence(t *testing.T) {
	got := DefaultSubSiteSequence()
	want := []string{
		SubSiteAssrt,
		SubSiteSubHd,
		SubSiteShooter,
		SubSiteXunLei,
		SubSiteOpenSubtitles,
		SubSiteSubDL,
		SubSiteSubtitleCat,
		SubSiteTVSubtitles,
		SubSiteMovieSubtitles,
		SubSiteSubtitleCatTrans,
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

func TestDefaultPrimarySubSiteSequence(t *testing.T) {
	got := DefaultPrimarySubSiteSequence()
	want := []string{
		SubSiteAssrt,
		SubSiteSubHd,
		SubSiteShooter,
		SubSiteXunLei,
		SubSiteOpenSubtitles,
	}

	if len(got) != len(want) {
		t.Fatalf("DefaultPrimarySubSiteSequence len = %d; want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("DefaultPrimarySubSiteSequence[%d] = %q; want %q", i, got[i], want[i])
		}
	}
}

func TestDefaultEnglishFallbackSubSiteSequence(t *testing.T) {
	got := DefaultEnglishFallbackSubSiteSequence()
	want := []string{
		SubSiteOpenSubtitles,
		SubSiteSubDL,
		SubSiteSubtitleCat,
		SubSiteMovieSubtitles,
	}

	if len(got) != len(want) {
		t.Fatalf("DefaultEnglishFallbackSubSiteSequence len = %d; want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("DefaultEnglishFallbackSubSiteSequence[%d] = %q; want %q", i, got[i], want[i])
		}
	}
}

func TestDefaultTranslatedFallbackSubSiteSequence(t *testing.T) {
	got := DefaultTranslatedFallbackSubSiteSequence()
	want := []string{SubSiteSubtitleCatTrans}

	if len(got) != len(want) {
		t.Fatalf("DefaultTranslatedFallbackSubSiteSequence len = %d; want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("DefaultTranslatedFallbackSubSiteSequence[%d] = %q; want %q", i, got[i], want[i])
		}
	}
}

func TestOrderSubSiteNames(t *testing.T) {
	got := OrderSubSiteNames(
		[]string{SubSiteShooter, "custom_site", SubSiteAssrt, SubSiteOpenSubtitles, SubSiteSubDL, SubSiteSubHd, SubSiteSubtitleBest, SubSiteShooter},
		DefaultSubSiteSequence(),
	)
	want := []string{
		SubSiteAssrt,
		SubSiteSubHd,
		SubSiteShooter,
		SubSiteOpenSubtitles,
		SubSiteSubDL,
		"custom_site",
		SubSiteSubtitleBest,
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
