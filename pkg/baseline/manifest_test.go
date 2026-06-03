package baseline

import "testing"

func TestManifestValidateRejectsDuplicateID(t *testing.T) {
	manifest := Manifest{
		Samples: []Sample{
			{ID: "movie-001", VideoPath: "C:\\Media\\Movie1.mkv", Kind: SampleMovie},
			{ID: "movie-001", VideoPath: "C:\\Media\\Movie2.mkv", Kind: SampleMovie},
		},
	}

	if err := manifest.Validate(); err == nil {
		t.Fatal("expected duplicate sample id to be rejected")
	}
}

func TestManifestCountByKind(t *testing.T) {
	manifest := Manifest{
		Samples: []Sample{
			{ID: "movie-001", VideoPath: "C:\\Media\\Movie1.mkv", Kind: SampleMovie},
			{ID: "tv-001", VideoPath: "C:\\Media\\Show\\S01E01.mkv", Kind: SampleEpisode, Season: 1, Episode: 1},
			{ID: "tv-002", VideoPath: "C:\\Media\\Show\\S01E02.mkv", Kind: SampleEpisode, Season: 1, Episode: 2},
		},
	}

	movies, episodes := manifest.CountByKind()
	if movies != 1 || episodes != 2 {
		t.Fatalf("CountByKind = (%d, %d), want (1, 2)", movies, episodes)
	}
}
