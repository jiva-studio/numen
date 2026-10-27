package format

import (
	"bytes"

	"github.com/jiva-studio/numen/libs/notes/markdown"
)

// section is one heading of the levels a file spends, and the bytes under it.
// Offsets are counted over the body currently, so a run of it can be
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
// The levels are given because a deck and a card model spend different ones: a
// first-level heading opens a section of a deck, and the same line on a template is
// text the template lays out.
func sections(body []byte, first, last int) []section {
	var out []section
	var f markdown.Fence
	for pos := 0; pos <= len(body); {
		end, next := len(body), len(body)+1
		if i := bytes.IndexByte(body[pos:], '\n'); i >= 0 {
			end, next = pos+i, pos+i+1
		}
		line := bytes.TrimRight(body[pos:end], "\r")
		if !f.IsCrossedByBytes(line) && !f.IsInside() {
			if level, name, ok := heading(line, first, last); ok {
				if n := len(out); n > 0 {
					out[n-1].to = pos
				}
				// A heading the file ends on has nothing under it, so its run
				// begins where the body ends.
				out = append(out, section{
					level: level, name: name, head: pos, from: min(next, len(body)), to: len(body),
				})
			}
		}
		pos = next
	}
	if n := len(out); n > 0 {
		out[n-1].to = len(body)
	}
	return out
}

// heading reads a heading of one of the levels from first to last. The name may
// be empty: `##` on its own opens a card whose first field holds nothing.
func heading(line []byte, first, last int) (level int, name string, ok bool) {
	hashes := 0
	for hashes < len(line) && line[hashes] == '#' {
		hashes++
	}
	if hashes < first || hashes > last {
		return 0, "", false
	}
	rest := line[hashes:]
	if len(rest) > 0 && rest[0] != ' ' && rest[0] != '\t' {
		return 0, "", false
	}
	cleaned := bytes.TrimSpace(rest)
	// front of it, and a name ending in one keeps it.
	if closed := bytes.TrimRight(cleaned, "#"); len(closed) != len(cleaned) {
		if t := bytes.TrimRight(closed, " \t"); len(closed) == 0 || len(t) != len(closed) {
			cleaned = t
		}
	}
	return hashes, string(cleaned), true
}

// readTrimmedRun is one part of the body as a person reads it — one kind of line break,
// and the blank lines at either end dropped — and the byte the reading of it
// stopped at. The blank lines it stopped short of separate one part of a deck
// from the next and belong to neither, which is what leaves a tail at the end
// of a file.
func readTrimmedRun(body []byte, from, to int) (string, int) {
	begin, end := from, to
	for begin < end {
		line, next := lineAt(body, begin, end)
		if len(bytes.TrimSpace(line)) != 0 {
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
func lineAt(body []byte, pos, to int) ([]byte, int) {
	if i := bytes.IndexByte(body[pos:to], '\n'); i >= 0 {
		return bytes.TrimRight(body[pos:pos+i], "\r"), pos + i + 1
	}
	return bytes.TrimRight(body[pos:to], "\r"), to
}

// lineFrom is the byte the line holding at begins at, looked for no further
// back than from.
func lineFrom(body []byte, from, pos int) int {
	if i := bytes.LastIndexByte(body[from:pos], '\n'); i >= 0 {
		return from + i + 1
	}
	return from
}

// trimBlankLines drops the lines at either end that hold nothing but
// whitespace. What a line begins with is the person's, so an indented block
// arrives indented.
func trimBlankLines(s string) string {
	read, _ := readTrimmedRun([]byte(s), 0, len(s))
	return read
}
