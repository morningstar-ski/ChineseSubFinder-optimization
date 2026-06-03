package baseline

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/decode"
	seriesHelper "github.com/ChineseSubFinder/ChineseSubFinder/pkg/logic/series_helper"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/search"
	"github.com/sirupsen/logrus"
)

func BuildManifest(log *logrus.Logger, movieRoots []string, seriesRoots []string, movieLimit int, episodeLimit int) (Manifest, error) {
	if log == nil {
		return Manifest{}, fmt.Errorf("manifest builder requires logger")
	}

	movieRoots = normalizeRoots(movieRoots)
	seriesRoots = normalizeRoots(seriesRoots)
	manifest := Manifest{
		Samples: make([]Sample, 0, movieLimit+episodeLimit),
	}

	moviePaths, err := collectMoviePaths(log, movieRoots, movieLimit)
	if err != nil {
		return Manifest{}, err
	}
	for index, videoPath := range moviePaths {
		manifest.Samples = append(manifest.Samples, Sample{
			ID:        fmt.Sprintf("movie-%03d", index+1),
			VideoPath: videoPath,
			Kind:      SampleMovie,
		})
	}

	episodeSamples, err := collectEpisodeSamples(log, seriesRoots, episodeLimit)
	if err != nil {
		return Manifest{}, err
	}
	for index, sample := range episodeSamples {
		sample.ID = fmt.Sprintf("episode-%03d", index+1)
		manifest.Samples = append(manifest.Samples, sample)
	}

	return manifest, nil
}

func normalizeRoots(roots []string) []string {
	out := make([]string, 0, len(roots))
	seen := make(map[string]struct{}, len(roots))
	for _, root := range roots {
		if root == "" {
			continue
		}
		cleaned := filepath.Clean(root)
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		out = append(out, cleaned)
	}
	sort.Strings(out)
	return out
}

func collectMoviePaths(log *logrus.Logger, roots []string, limit int) ([]string, error) {
	return collectVideoPaths(log, roots, limit)
}

func collectEpisodeSamples(log *logrus.Logger, roots []string, limit int) ([]Sample, error) {
	if limit == 0 {
		return nil, nil
	}

	out := make([]Sample, 0, limit)
	seen := make(map[string]struct{})
	for _, root := range roots {
		seriesDirs, err := seriesHelper.GetSeriesList(log, root)
		if err != nil {
			return nil, err
		}
		sort.Strings(seriesDirs)

		for _, seriesDir := range seriesDirs {
			videoPaths, err := search.MatchedVideoFile(log, seriesDir)
			if err != nil {
				return nil, err
			}
			sort.Strings(videoPaths)

			for _, videoPath := range videoPaths {
				if _, ok := seen[videoPath]; ok {
					continue
				}
				info, err := decode.GetVideoInfoFromFileName(filepath.Base(videoPath))
				if err != nil || info == nil || info.Season == 0 || info.Episode == 0 {
					continue
				}

				seen[videoPath] = struct{}{}
				out = append(out, Sample{
					VideoPath: videoPath,
					Kind:      SampleEpisode,
					Season:    info.Season,
					Episode:   info.Episode,
				})
				if len(out) >= limit {
					return out, nil
				}
			}
		}
	}

	return out, nil
}

func collectVideoPaths(log *logrus.Logger, roots []string, limit int) ([]string, error) {
	if limit == 0 {
		return nil, nil
	}

	out := make([]string, 0, limit)
	seen := make(map[string]struct{})
	for _, root := range roots {
		videoPaths, err := search.MatchedVideoFile(log, root)
		if err != nil {
			return nil, err
		}
		sort.Strings(videoPaths)

		for _, videoPath := range videoPaths {
			if _, ok := seen[videoPath]; ok {
				continue
			}
			seen[videoPath] = struct{}{}
			out = append(out, videoPath)
			if len(out) >= limit {
				return out, nil
			}
		}
	}

	return out, nil
}
