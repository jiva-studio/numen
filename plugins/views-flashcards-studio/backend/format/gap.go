package format

import (
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/markdown"
)

// formatSection is what stands beneath a heading: a blank line, the text, and a
// blank line after it. The last section of a file ends with the one break every
// file ends with.
func formatSection(value string, last bool) string {
	text := trimBlankLines(markdown.Normalise(value))
	switch {
	case text == "" && last:
		return ""
	case text == "":
		return "\n"
	case last:
		return "\n" + text + "\n"
	}
	return "\n" + text + "\n\n"
}

// insert is a block written into the body at one byte, with the blank line
// above it that markdown wants and the blank line below it when something
// follows.
func insert(body []byte, pos int, block string) string {
	content := getSeparator(body, pos) + block + "\n"
	if pos < len(body) {
		content += "\n"
	}
	return content
}

// getSeparator is the break and the blank line a new block needs, counting what already
// stands above it.
func getSeparator(body []byte, pos int) string {
	above := markdown.Normalise(string(body[:pos]))
	switch {
	// Nothing but blank lines above is the top of the body, and a block written
	// there stands where it is.
	case strings.TrimSpace(above) == "" || strings.HasSuffix(above, "\n\n"):
		return ""
	case strings.HasSuffix(above, "\n"):
		return "\n"
	}
	return "\n\n"
}
