package format

import (
	"bytes"
	"strings"

	"github.com/jiva-studio/numen/modules/libs/core/markdown"
)

// section is one heading of the levels a file spends, and the bytes under it.
// Offsets are counted over the body as it stands, so a run of it can be
// replaced without the rest of the file being rewritten.
type section struct {
	level int
	name  string
	// head is the byte the heading's own line begins at, from is the byte after
	// that line, and to is where the next heading of either level begins.
	head int
	from int
	to   int
}

// sections walks the body a line at a time and returns its headings of the
// levels from first to last, in document order. A heading inside a code fence
// is an example of one, and a heading of any other level is text under the
// section it falls in.
//
// The levels are given because a deck and a stencil spend different ones: a
// first-level heading opens a section of a deck, and the same line on a face is
// text the face lays out.
func sections(body []byte, first, last int) []section {
	var out []section
	var f markdown.Fence
	for at := 0; at <= len(body); {
		end, next := len(body), len(body)+1
		if i := bytes.IndexByte(body[at:], '\n'); i >= 0 {
			end, next = at+i, at+i+1
		}
		line := strings.TrimRight(string(body[at:end]), "\r")
		if !f.Cross(line) && !f.IsInside() {
			if level, name, ok := heading(line, first, last); ok {
				if n := len(out); n > 0 {
					out[n-1].to = at
				}
				// A heading the file ends on has nothing under it, so its run
				// begins where the body ends.
				out = append(out, section{
					level: level, name: name, head: at, from: min(next, len(body)), to: len(body),
				})
			}
		}
		at = next
	}
	if n := len(out); n > 0 {
		out[n-1].to = len(body)
	}
	return out
}

// heading reads a heading of one of the levels from first to last. The name may
// be empty: `##` on its own opens a card whose first field holds nothing.
func heading(line string, first, last int) (level int, name string, ok bool) {
	hashes := 0
	for hashes < len(line) && line[hashes] == '#' {
		hashes++
	}
	if hashes < first || hashes > last {
		return 0, "", false
	}
	rest := line[hashes:]
	if rest != "" && rest[0] != ' ' && rest[0] != '\t' {
		return 0, "", false
	}
	name = strings.TrimSpace(rest)
	// A run of hashes at the end closes the heading where whitespace stands in
	// front of it, and a name ending in one keeps it.
	if closed := strings.TrimRight(name, "#"); closed != name {
		if trimmed := strings.TrimRight(closed, " \t"); closed == "" || trimmed != closed {
			name = trimmed
		}
	}
	return hashes, name, true
}

// run is one part of the body as a person reads it — one kind of line break,
// and the blank lines at either end dropped — and the byte the reading of it
// stopped at. The blank lines it stopped short of separate one part of a deck
// from the next and belong to neither, which is what leaves a tail at the end
// of a file.
func run(body []byte, from, to int) (string, int) {
	begin, end := from, to
	for begin < end {
		line, next := lineAt(body, begin, end)
		if strings.TrimSpace(line) != "" {
			break
		}
		begin = next
	}
	end = getTrimmedEnd(body, begin, end)
	return markdown.Normalise(string(body[begin:end])), end
}

// getTrimmedEnd is the byte a run's own text stops at, the whitespace that
// follows it counted as nobody's.
func getTrimmedEnd(body []byte, from, to int) int {
	return from + len(bytes.TrimRight(body[from:to], " \t\r\n"))
}

// lineAt is one line without its ending, and the byte the next one begins at.
func lineAt(body []byte, at, to int) (string, int) {
	if i := bytes.IndexByte(body[at:to], '\n'); i >= 0 {
		return strings.TrimRight(string(body[at:at+i]), "\r"), at + i + 1
	}
	return strings.TrimRight(string(body[at:to]), "\r"), to
}

// lineFrom is the byte the line holding at begins at, looked for no further
// back than from.
func lineFrom(body []byte, from, at int) int {
	if i := bytes.LastIndexByte(body[from:at], '\n'); i >= 0 {
		return from + i + 1
	}
	return from
}

// trimBlankLines drops the lines at either end that hold nothing but
// whitespace. What a line begins with is the person's, so an indented block
// arrives indented.
func trimBlankLines(s string) string {
	read, _ := run([]byte(s), 0, len(s))
	return read
}
