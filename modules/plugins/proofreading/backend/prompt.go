package main

import (
	"strconv"
	"strings"
)

// buildPrompt is one batch as the model is given it: its prose, with each line
// opened by its number. A line ending mid-word is finished by the next one, and
// what the text is about stands before the first of them.
func buildPrompt(batch Batch) string {
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
