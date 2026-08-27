package cards

import (
	"bytes"
	"strings"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/markdown"
)

// section is one heading of the two levels the format spends, and the bytes
// under it. Offsets are counted over the body as it stands, so a run of it can
// be replaced without the rest of the file being rewritten.
type section struct {
	level int
	name  string
	// head is the byte the heading's own line begins at, from is the byte after
	// that line, and to is where the next heading of either level begins.
	head int
	from int
	to   int
}

// sections walks the body a line at a time and returns its second and
// third-level headings in document order. A heading inside a code fence is an
// example of one, and a heading of any other level is text under the section it
// falls in.
func sections(body []byte) []section {
	var out []section
	fenced := false
	for at := 0; at <= len(body); {
		end, next := len(body), len(body)+1
		if i := bytes.IndexByte(body[at:], '\n'); i >= 0 {
			end, next = at+i, at+i+1
		}
		line := strings.TrimRight(string(body[at:end]), "\r")
		switch {
		case isFence(line):
			fenced = !fenced
		case !fenced:
			if level, name, ok := heading(line); ok {
				if n := len(out); n > 0 {
					out[n-1].to = at
				}
				out = append(out, section{level: level, name: name, head: at, from: next, to: len(body)})
			}
		}
		at = next
	}
	if n := len(out); n > 0 {
		out[n-1].to = len(body)
	}
	return out
}

// heading reads a second or third-level heading. The name may be empty: `##`
// on its own opens a card, and a card with no name is reported as one.
func heading(line string) (level int, name string, ok bool) {
	hashes := 0
	for hashes < len(line) && line[hashes] == '#' {
		hashes++
	}
	if hashes < 2 || hashes > 3 {
		return 0, "", false
	}
	rest := line[hashes:]
	if rest != "" && rest[0] != ' ' && rest[0] != '\t' {
		return 0, "", false
	}
	name = strings.TrimSpace(rest)
	name = strings.TrimSpace(strings.TrimRight(name, "#"))
	return hashes, name, true
}

func isFence(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "```") || strings.HasPrefix(t, "~~~")
}

// text is a run of the body as a person reads it: one kind of line break, and
// the blank lines at either end dropped.
func text(body []byte, from, to int) string {
	read, _ := run(body, from, to)
	return read
}

// run is text, and the byte the reading of it stopped at. The blank lines it
// stopped short of separate one part of a deck from the next and belong to
// neither, which is what leaves a tail at the end of a file.
func run(body []byte, from, to int) (string, int) {
	begin, end := from, to
	for begin < end {
		line, next := lineAt(body, begin, end)
		if strings.TrimSpace(line) != "" {
			break
		}
		begin = next
	}
	end = trimmedEnd(body, begin, end)
	return markdown.Normalised(string(body[begin:end])), end
}

// trimmedEnd is the byte a run's own text stops at, the whitespace that
// follows it counted as nobody's.
func trimmedEnd(body []byte, from, to int) int {
	return from + len(bytes.TrimRight(body[from:to], " \t\r\n"))
}

// lineAt is one line without its ending, and the byte the next one begins at.
func lineAt(body []byte, at, to int) (string, int) {
	if i := bytes.IndexByte(body[at:to], '\n'); i >= 0 {
		return strings.TrimRight(string(body[at:at+i]), "\r"), at + i + 1
	}
	return strings.TrimRight(string(body[at:to]), "\r"), to
}

// trimBlankLines drops the lines at either end that hold nothing but
// whitespace. What a line begins with is the person's, so an indented block
// arrives indented.
func trimBlankLines(s string) string {
	read, _ := run([]byte(s), 0, len(s))
	return read
}
