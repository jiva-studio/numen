package format

import (
	"bytes"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/markdown"
)

// section is one heading of the levels a file spends, and the bytes under it.
// Offsets are counted over the body currently, so a run of it can be
// replaced without the rest of the file being rewritten.
type section struct {
	level int
	name  string
	head  int
	from  int
	to    int
}

func sections(body []byte, first, last int) []section {
	var out []section
	var f markdown.Fence
	for pos := 0; pos <= len(body); {
		// gomutants:disable-next-line INTEGER_INCREMENT reason="Any value > len(body) terminates the loop on the next iteration."
		end, next := len(body), len(body)+1
		if i := bytes.IndexByte(body[pos:], '\n'); i >= 0 {
			end, next = pos+i, pos+i+1
		}
		line := bytes.TrimRight(body[pos:end], "\r")
		if !f.IsCrossedByBytes(line) && !f.IsInside() {
			out = openSection(out, line, levelRange{first: first, last: last}, lineOffsets{pos: pos, next: next, size: len(body)})
		}
		pos = next
	}
	// gomutants:disable-next-line INTEGER_INCREMENT,BRANCH_IF reason="openSection already initializes every section with to = len(body)."
	if n := len(out); n > 0 {
		// gomutants:disable-next-line STATEMENT_REMOVE reason="openSection already initializes every section with to = len(body)."
		out[n-1].to = len(body)
	}
	return out
}

func openSection(out []section, line []byte, levels levelRange, offsets lineOffsets) []section {
	level, name, ok := heading(line, levels.first, levels.last)
	if !ok {
		return out
	}
	if n := len(out); n > 0 {
		out[n-1].to = offsets.pos
	}
	// A heading the file ends on has nothing under it, so its run
	// begins where the body ends.
	return append(out, section{
		level: level, name: name, head: offsets.pos, from: min(offsets.next, offsets.size), to: offsets.size,
	})
}

// levelRange is the heading levels a scan reads, from first to last.
type levelRange struct{ first, last int }

// lineOffsets is where a line begins, where the next begins and where the body ends.
type lineOffsets struct{ pos, next, size int }

func heading(line []byte, first, last int) (level int, name string, ok bool) {
	hashes := 0
	for hashes < len(line) && line[hashes] == '#' {
		hashes++
	}
	if hashes < first || hashes > last {
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Callers ignore level when ok is false."
		return 0, "", false
	}
	rest := line[hashes:]
	if len(rest) > 0 && rest[0] != ' ' && rest[0] != '\t' {
		// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="Callers ignore level when ok is false."
		return 0, "", false
	}
	cleaned := bytes.TrimSpace(rest)
	// front of it, and a name ending in one keeps it.
	if closed := bytes.TrimRight(cleaned, "#"); len(closed) != len(cleaned) {
		// gomutants:disable-next-line INTEGER_DECREMENT reason="Slice length is never negative."
		if t := bytes.TrimRight(closed, " \t"); len(closed) == 0 || len(t) != len(closed) {
			cleaned = t
		}
	}
	return hashes, string(cleaned), true
}

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
	return markdown.Normalize(string(body[begin:end])), end
}

func getTrimmedEnd(body []byte, from, to int) int {
	return from + len(bytes.TrimRight(body[from:to], " \t\r\n"))
}

func lineAt(body []byte, pos, to int) ([]byte, int) {
	if i := bytes.IndexByte(body[pos:to], '\n'); i >= 0 {
		return bytes.TrimRight(body[pos:pos+i], "\r"), pos + i + 1
	}
	return bytes.TrimRight(body[pos:to], "\r"), to
}

func lineFrom(body []byte, from, pos int) int {
	// gomutants:disable-next-line INTEGER_DECREMENT reason="bytes.LastIndexByte returns -1 when not found and >= 0 when found."
	if i := bytes.LastIndexByte(body[from:pos], '\n'); i >= 0 {
		return from + i + 1
	}
	return from
}

func trimBlankLines(s string) string {
	read, _ := readTrimmedRun([]byte(s), 0, len(s))
	return read
}
