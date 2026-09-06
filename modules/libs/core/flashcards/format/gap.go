package format

import (
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/markdown"
)

// under is what stands beneath a heading: a blank line, the text, and a blank
// line after it. The last section of a file ends with the one break every file
// ends with.
func under(value string, last bool) string {
	text := trimBlankLines(markdown.Normalised(value))
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
func insert(body []byte, at int, block string) string {
	written := gap(body, at) + block + "\n"
	if at < len(body) {
		written += "\n"
	}
	return written
}

// gap is the break and the blank line a new block needs, counting what already
// stands above it.
func gap(body []byte, at int) string {
	above := markdown.Normalised(string(body[:at]))
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
