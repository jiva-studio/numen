package format

import (
	"strings"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host/markdown"
)

// formatSection returns the text that follows a heading: a blank line, the
// text, and a blank line after it. The last section of a file ends with the
// single line break every file ends with.
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

// insert returns a block to be written into the body at one byte offset, with
// the blank line before it that markdown requires and a blank line after it
// when content follows.
func insert(body []byte, pos int, block string) string {
	content := getSeparator(body, pos) + block + "\n"
	if pos < len(body) {
		content += "\n"
	}
	return content
}

// getSeparator returns the line break and blank line a new block needs,
// given the content that precedes it.
func getSeparator(body []byte, pos int) string {
	above := markdown.Normalise(string(body[:pos]))
	switch {
	// Only blank lines before the position means the top of the body, and a block
	// written there needs no separator.
	case strings.TrimSpace(above) == "" || strings.HasSuffix(above, "\n\n"):
		return ""
	case strings.HasSuffix(above, "\n"):
		return "\n"
	}
	return "\n\n"
}
