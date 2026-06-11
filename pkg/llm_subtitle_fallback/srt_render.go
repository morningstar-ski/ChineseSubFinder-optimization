package llm_subtitle_fallback

import (
	"fmt"
	"os"
	"strings"

	"github.com/ChineseSubFinder/ChineseSubFinder/pkg"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/common"
	"github.com/ChineseSubFinder/ChineseSubFinder/pkg/types/subparser"
)

func writeCandidateAsSRT(info subparser.FileInfo, destPath string) error {
	if strings.EqualFold(info.Ext, common.SubExtSRT) {
		if info.FileFullPath != "" {
			data, err := os.ReadFile(info.FileFullPath)
			if err == nil {
				return pkg.WriteFile(destPath, data)
			}
		}
		if len(info.Data) > 0 {
			return pkg.WriteFile(destPath, info.Data)
		}
	}

	if len(info.Dialogues) == 0 {
		return fmt.Errorf("subtitle %q has no dialogues to render", info.Name)
	}

	var builder strings.Builder
	for i, dialogue := range info.Dialogues {
		startTime := pkg.Time2SubTimeString(dialogue.GetStartTime(), common.TimeFormatPoint3)
		endTime := pkg.Time2SubTimeString(dialogue.GetEndTime(), common.TimeFormatPoint3)

		lines := make([]string, 0, len(dialogue.Lines))
		for _, line := range dialogue.Lines {
			line = strings.TrimSpace(strings.ReplaceAll(line, `\N`, "\n"))
			if line != "" {
				lines = append(lines, line)
			}
		}
		if len(lines) == 0 {
			continue
		}

		builder.WriteString(fmt.Sprintf("%d\n", i+1))
		builder.WriteString(startTime)
		builder.WriteString(" --> ")
		builder.WriteString(endTime)
		builder.WriteString("\n")
		builder.WriteString(strings.Join(lines, "\n"))
		builder.WriteString("\n\n")
	}

	if builder.Len() == 0 {
		return fmt.Errorf("subtitle %q rendered empty SRT", info.Name)
	}

	return pkg.WriteFile(destPath, []byte(builder.String()))
}
