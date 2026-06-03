package baseline

import "fmt"

type Manifest struct {
	Samples []Sample `json:"samples"`
}

func (m Manifest) Validate() error {
	if len(m.Samples) == 0 {
		return fmt.Errorf("manifest requires at least one sample")
	}

	seen := make(map[string]struct{}, len(m.Samples))
	for _, sample := range m.Samples {
		if _, ok := seen[sample.ID]; ok {
			return fmt.Errorf("duplicate sample id %q", sample.ID)
		}
		seen[sample.ID] = struct{}{}

		if err := validateSample(sample); err != nil {
			return fmt.Errorf("sample %q is invalid: %w", sample.ID, err)
		}
	}

	return nil
}

func (m Manifest) CountByKind() (movies int, episodes int) {
	for _, sample := range m.Samples {
		switch sample.Kind {
		case SampleMovie:
			movies++
		case SampleEpisode:
			episodes++
		}
	}
	return movies, episodes
}
