package hunks

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jusanchez/gitwatch/internal/patch"
)

var ErrUnsupportedPartialPatch = fmt.Errorf("partial patch is unsupported for binary, rename, or copy changes")

// BuildPatch creates a Git-applyable partial patch. Context is retained for
// every selected hunk while only selected additions/removals are emitted.
func (s Selection) BuildPatch(files []patch.File) ([]byte, error) {
	if !s.Valid(files) {
		return nil, fmt.Errorf("selection is stale")
	}
	var output strings.Builder
	for fileIndex, file := range files {
		if hasUnsupportedMetadata(file) && selectionTouchesFile(s, fileIndex) {
			return nil, ErrUnsupportedPartialPatch
		}
		var hunks []string
		for hunkIndex, hunk := range file.Hunks {
			var lines []patch.Line
			selectedChange := false
			oldCount, newCount := 0, 0
			for lineIndex, line := range hunk.Lines {
				include := line.Kind == patch.Context || s.Selected[ID{File: fileIndex, Hunk: hunkIndex, Line: lineIndex}]
				if !include {
					continue
				}
				if line.Kind == patch.Added || line.Kind == patch.Removed {
					selectedChange = true
				}
				lines = append(lines, line)
				if line.Kind != patch.Added {
					oldCount++
				}
				if line.Kind != patch.Removed {
					newCount++
				}
			}
			if !selectedChange {
				continue
			}
			oldStart, newStart := hunk.OldStart, hunk.NewStart
			var rendered strings.Builder
			for _, line := range lines {
				rendered.WriteByte(byte(line.Kind))
				rendered.WriteString(line.Text)
				rendered.WriteByte('\n')
				if line.NoNewline {
					rendered.WriteString("\\ No newline at end of file\n")
				}
			}
			header := fmt.Sprintf("@@ -%s +%s @@", rangeText(oldStart, oldCount), rangeText(newStart, newCount))
			hunks = append(hunks, header+"\n"+rendered.String())
		}
		if len(hunks) == 0 {
			continue
		}
		output.WriteString("diff --git a/")
		output.WriteString(file.OldPath)
		output.WriteString(" b/")
		output.WriteString(file.NewPath)
		output.WriteByte('\n')
		output.WriteString("--- a/")
		output.WriteString(file.OldPath)
		output.WriteByte('\n')
		output.WriteString("+++ b/")
		output.WriteString(file.NewPath)
		output.WriteByte('\n')
		output.WriteString(strings.Join(hunks, ""))
	}
	return []byte(output.String()), nil
}

func hasUnsupportedMetadata(file patch.File) bool {
	return file.Binary || file.RenameFrom != "" || file.RenameTo != "" || file.CopyFrom != "" || file.CopyTo != ""
}

func selectionTouchesFile(s Selection, fileIndex int) bool {
	for id := range s.Selected {
		if id.File == fileIndex {
			return true
		}
	}
	return false
}

func rangeText(start, count int) string {
	if count == 1 {
		return strconv.Itoa(start)
	}
	return strconv.Itoa(start) + "," + strconv.Itoa(count)
}
