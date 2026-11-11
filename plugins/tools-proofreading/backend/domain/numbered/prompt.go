// Package prompt builds the text and the instructions the model is asked about.
package numbered

import (
	"strconv"
	"strings"
)

func BuildPrompt(batch Batch) string {
	var out strings.Builder
	if batch.Context != "" {
		out.WriteString(batch.Context)
		out.WriteString("\n\n")
	}
	for i, line := range batch.Lines {
		if i > 0 {
			out.WriteString(" ")
		}
		out.WriteString(Opens)
		out.WriteString(strconv.Itoa(line.Number))
		out.WriteString(Closes)
		out.WriteString(line.Text)
	}
	return out.String()
}
