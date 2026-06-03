package common

import "testing"

func TestDefaultSubSiteSequence(t *testing.T) {
	got := DefaultSubSiteSequence()
	want := []string{
		SubSiteSubtitleBest,
		SubSiteAssrt,
		SubSiteSubDL,
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
	for _, site := range got {
		if site == SubSiteA4K {
			t.Fatalf("DefaultSubSiteSequence should not contain %q", SubSiteA4K)
		}
	}
}

func TestOrderSubSiteNames(t *testing.T) {
	got := OrderSubSiteNames(
		[]string{SubSiteShooter, "custom_site", SubSiteAssrt, SubSiteSubDL, SubSiteShooter},
		DefaultSubSiteSequence(),
	)
	want := []string{
		SubSiteAssrt,
		SubSiteSubDL,
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
