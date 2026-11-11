// Package prompt builds the text and the instructions the model is asked about.
package prompt

import (
	"strconv"
	"strings"

	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/numbered"
)

func BuildPrompt(batch numbered.Batch) string {
	var out strings.Builder
	if batch.Context != "" {
		out.WriteString(batch.Context)
		out.WriteString("\n\n")
	}
	for i, line := range batch.Lines {
		if i > 0 {
			out.WriteString(" ")
		}
		out.WriteString(numbered.Opens)
		out.WriteString(strconv.Itoa(line.Number))
		out.WriteString(numbered.Closes)
		out.WriteString(line.Text)
	}
	return out.String()
}
