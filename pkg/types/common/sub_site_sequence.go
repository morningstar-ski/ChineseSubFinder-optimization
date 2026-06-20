package common

import "sort"

// DefaultPrimarySubSiteSequence defines the default native-Chinese supplier priority.
func DefaultPrimarySubSiteSequence() []string {
	return []string{
		SubSiteAssrt,
		SubSiteSubHd,
		SubSiteShooter,
		SubSiteXunLei,
		SubSiteOpenSubtitles,
	}
}

// DefaultEnglishFallbackSubSiteSequence defines the default English fallback supplier priority.
func DefaultEnglishFallbackSubSiteSequence() []string {
	return []string{
		SubSiteOpenSubtitles,
		SubSiteSubDL,
		SubSiteSubtitleCat,
		SubSiteMovieSubtitles,
	}
}

// DefaultTranslatedFallbackSubSiteSequence defines the default translated-Chinese fallback priority.
func DefaultTranslatedFallbackSubSiteSequence() []string {
	return []string{
		SubSiteSubtitleCatTrans,
	}
}

// DefaultSubSiteSequence 定义默认的字幕站点优先级。
func DefaultSubSiteSequence() []string {
	sequence := append([]string{}, DefaultPrimarySubSiteSequence()...)
	sequence = append(sequence,
		SubSiteSubDL,
		SubSiteSubtitleCat,
		SubSiteTVSubtitles,
		SubSiteMovieSubtitles,
	)
	sequence = append(sequence, DefaultTranslatedFallbackSubSiteSequence()...)
	return sequence
}

// OrderSubSiteNames 按给定优先级排序站点名，未命中的站点按字典序追加在后面。
func OrderSubSiteNames(siteNames []string, preferred []string) []string {
	if len(siteNames) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(siteNames))
	for _, siteName := range siteNames {
		if siteName == "" {
			continue
		}
		seen[siteName] = struct{}{}
	}

	out := make([]string, 0, len(seen))
	for _, siteName := range preferred {
		if _, ok := seen[siteName]; ok {
			out = append(out, siteName)
			delete(seen, siteName)
		}
	}

	remaining := make([]string, 0, len(seen))
	for siteName := range seen {
		remaining = append(remaining, siteName)
	}
	sort.Strings(remaining)

	return append(out, remaining...)
}
