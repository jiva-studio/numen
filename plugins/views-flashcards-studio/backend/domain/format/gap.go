package format

import (
	"strings"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
)

func formatSection(value string, last bool) string {
	text := trimBlankLines(markdown.Normalize(value))
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

func insert(body []byte, pos int, block string) string {
	content := getSeparator(body, pos) + block + "\n"
	if pos < len(body) {
		content += "\n"
	}
	return content
}

func getSeparator(body []byte, pos int) string {
	above := markdown.Normalize(string(body[:pos]))
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
