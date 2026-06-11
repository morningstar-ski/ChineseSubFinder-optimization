package ranking

import (
	"path/filepath"
	"testing"
)

func TestTargetMatcherBestScore(t *testing.T) {
	matcher := NewTargetMatcher(filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false)

	better := matcher.BestScore([]string{"My.Show.S01E03.1080p.WEB-DL-GROUP"}, StandardReleaseMatchWeights)
	worse := matcher.BestScore([]string{"My.Show.S01E03.720p.HDTV-OTHER"}, StandardReleaseMatchWeights)
	if better <= worse {
		t.Fatalf("expected better score > worse score, got better=%d worse=%d", better, worse)
	}
}

func TestTargetMatcherHandlesEpisodeMismatch(t *testing.T) {
	matcher := NewTargetMatcher(filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false)

	exact := matcher.BestScore([]string{"My.Show.S01E03.1080p.WEB-DL-GROUP"}, StandardReleaseMatchWeights)
	mismatch := matcher.BestScore([]string{"My.Show.S01E04.1080p.WEB-DL-GROUP"}, StandardReleaseMatchWeights)
	if exact <= mismatch {
		t.Fatalf("expected exact episode score > mismatch score, got exact=%d mismatch=%d", exact, mismatch)
	}
}

func TestTargetMatcherUsesParsedTargetTitle(t *testing.T) {
	matcher := NewTargetMatcher(filepath.Join("C:\\", "Media", "Nirvana.The.Band.The.Show.The.Movie.2026.1080p.WEB-DL.mkv"), true)

	correct := matcher.BestScore([]string{"Nirvana.The.Band.The.Show.The.Movie.2026.1080p.WEB-DL.srt"}, StandardReleaseMatchWeights)
	wrong := matcher.BestScore([]string{"The.Muppet.Show.2026.1080p.WEB-DL.srt"}, StandardReleaseMatchWeights)
	if correct <= wrong {
		t.Fatalf("expected correct title score > wrong title score, got correct=%d wrong=%d", correct, wrong)
	}
}

func TestScoreEpisodeMatch(t *testing.T) {
	weights := EpisodeMatchWeights{
		ExactMatch:   120,
		SeasonPack:   20,
		WrongEpisode: -120,
	}
	if got := ScoreEpisodeMatch(1, 3, 1, 3, weights); got != 120 {
		t.Fatalf("exact match score = %d", got)
	}
	if got := ScoreEpisodeMatch(1, 0, 1, 3, weights); got != 20 {
		t.Fatalf("season pack score = %d", got)
	}
	if got := ScoreEpisodeMatch(1, 4, 1, 3, weights); got != -120 {
		t.Fatalf("wrong episode score = %d", got)
	}
}

func TestScoreSubtitleExt(t *testing.T) {
	if got := ScoreSubtitleExt(".srt", 1); got != 8 {
		t.Fatalf("expected srt preference, got %d", got)
	}
	if got := ScoreSubtitleExt(".ass", 2); got != 8 {
		t.Fatalf("expected ass preference, got %d", got)
	}
	if got := ScoreSubtitleExt(".ssa", 1); got != 0 {
		t.Fatalf("expected no score, got %d", got)
	}
}

func TestScoreBilingualSubtype(t *testing.T) {
	if got := ScoreBilingualSubtype("bilingual subtitle"); got != 5 {
		t.Fatalf("expected bilingual bonus, got %d", got)
	}
	if got := ScoreBilingualSubtype("single language"); got != 0 {
		t.Fatalf("expected zero bonus, got %d", got)
	}
}

func TestBaseScore(t *testing.T) {
	matcher := NewTargetMatcher(filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false)

	got := BaseScore(matcher, BaseScoreOptions{
		IsMovie:          false,
		CandidateSeason:  1,
		CandidateEpisode: 3,
		TargetSeason:     1,
		TargetEpisode:    3,
		EpisodeMatchWeights: &EpisodeMatchWeights{
			ExactMatch:   120,
			SeasonPack:   20,
			WrongEpisode: -120,
		},
		SubtitleExt:         ".srt",
		SubTypePriority:     1,
		Subtype:             "bilingual subtitle",
		HasHI:               true,
		HIPenalty:           -5,
		AuthorityScore:      7,
		ReleaseNames:        []string{"My.Show.S01E03.1080p.WEB-DL-GROUP"},
		ReleaseMatchWeights: StandardReleaseMatchWeights,
	})

	want := 120 + 8 + 5 - 5 + 7 + matcher.BestScore([]string{"My.Show.S01E03.1080p.WEB-DL-GROUP"}, StandardReleaseMatchWeights)
	if got != want {
		t.Fatalf("base score = %d, want %d", got, want)
	}
}

func TestScoreCandidate(t *testing.T) {
	matcher := NewTargetMatcher(filepath.Join("C:\\", "Media", "My.Show.S01E03.1080p.WEB-DL-GROUP.mkv"), false)
	metadata := CandidateMetadata{
		Name:           "My Show subtitle",
		ReleaseNames:   []string{"My.Show.S01E03.1080p.WEB-DL-GROUP"},
		Season:         1,
		Episode:        3,
		SubtitleExt:    ".srt",
		Subtype:        "bilingual subtitle",
		HasHI:          true,
		AuthorityScore: 9,
	}
	spec := CandidateScoreSpec{
		IsMovie:       false,
		TargetSeason:  1,
		TargetEpisode: 3,
		EpisodeMatchWeights: &EpisodeMatchWeights{
			ExactMatch:   120,
			SeasonPack:   20,
			WrongEpisode: -120,
		},
		SubTypePriority:     1,
		HIPenalty:           -5,
		ReleaseMatchWeights: StandardReleaseMatchWeights,
	}

	got := ScoreCandidate(matcher, metadata, spec)
	want := BaseScore(matcher, BaseScoreOptions{
		IsMovie:             false,
		CandidateSeason:     1,
		CandidateEpisode:    3,
		TargetSeason:        1,
		TargetEpisode:       3,
		EpisodeMatchWeights: spec.EpisodeMatchWeights,
		SubtitleExt:         ".srt",
		SubTypePriority:     1,
		Subtype:             "bilingual subtitle",
		HasHI:               true,
		HIPenalty:           -5,
		AuthorityScore:      9,
		ReleaseNames:        []string{"My Show subtitle", "My.Show.S01E03.1080p.WEB-DL-GROUP"},
		ReleaseMatchWeights: StandardReleaseMatchWeights,
	})
	if got != want {
		t.Fatalf("score candidate = %d, want %d", got, want)
	}
}

func TestCandidateMetadataReleaseNamesWithName(t *testing.T) {
	metadata := CandidateMetadata{
		Name:         "primary title",
		ReleaseNames: []string{"release-1", "", "release-2"},
	}

	got := metadata.ReleaseNamesWithName()
	want := []string{"primary title", "release-1", "release-2"}
	if len(got) != len(want) {
		t.Fatalf("release names len = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("release names[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
