package markdown

import (
	"strings"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Wikilink is one `[[…]]` at its current position in a line.
type Wikilink struct {
	// At is the byte the brackets open at, and To the byte after they close.
	At, To int
	Inside string
	// Target is where it points.
	Target domain.Address
}

// WikilinksIn is every wikilink in one line, in the order they are written.
// Brackets holding nothing point nowhere, so they are the text they are
// written as. Brackets inside inline code are an example of a link, not one.
func WikilinksIn(line string) []Wikilink {
	var out []Wikilink
	spans := getCodeSpans(line)
	pos := 0
	for pos < len(line) {
		start := strings.Index(line[pos:], "[[")
		if start < 0 {
			break
		}
		start += pos
		if span, inCode := findCodeSpan(spans, start); inCode {
			pos = span[1]
			continue
		}
		closeIdx := strings.Index(line[start+2:], "]]")
		if closeIdx < 0 {
			break
		}
		end := start + 2 + closeIdx
		inside := line[start+2 : end]
		if strings.ContainsAny(inside, "[]") || hasCodeSpanIn(spans, start, end+2) {
			pos = start + 1
			continue
		}
		if len(inside) == 0 {
			pos = end + 2
			continue
		}
		target := domain.ParseAddress(inside)
		if target.Value != "" {
			out = append(out, Wikilink{At: start, To: end + 2, Inside: inside, Target: target})
		}
		pos = end + 2
	}
	return out
}

// WikilinksInText is every wikilink in a text, in the order they are written,
// with At and To counted from the start of the text. Lines inside a code fence
// hold none.
func WikilinksInText(text string) []Wikilink {
	var out []Wikilink
	var f Fence
	for lineStart := 0; lineStart <= len(text); {
		lineEnd := len(text)
		if next := strings.IndexByte(text[lineStart:], '\n'); next >= 0 {
			lineEnd = lineStart + next
		}
		line := strings.TrimRight(text[lineStart:lineEnd], "\r")
		if !f.IsCrossedBy(line) && !f.IsInside() {
			for _, found := range WikilinksIn(line) {
				found.At += lineStart
				found.To += lineStart
				out = append(out, found)
			}
		}
		lineStart = lineEnd + 1
	}
	return out
}

// getCodeSpans is where each inline code span of one line begins and the byte
// after it ends. A run of backticks opens a span that the next run of the same
// length closes; a run that nothing closes is text.
func getCodeSpans(line string) [][2]int {
	var out [][2]int
	for pos := 0; pos < len(line); {
		open := strings.IndexByte(line[pos:], '`')
		if open < 0 {
			break
		}
		open += pos
		size := getBacktickRun(line, open)
		closeAt := -1
		for p := open + size; p < len(line); {
			next := strings.IndexByte(line[p:], '`')
			if next < 0 {
				break
			}
			next += p
			run := getBacktickRun(line, next)
			if run == size {
				closeAt = next
				break
			}
			p = next + run
		}
		if closeAt < 0 {
			pos = open + size
			continue
		}
		out = append(out, [2]int{open, closeAt + size})
		pos = closeAt + size
	}
	return out
}

// getBacktickRun is how many backticks stand in a row from start.
func getBacktickRun(line string, start int) int {
	end := start
	for end < len(line) && line[end] == '`' {
		end++
	}
	return end - start
}

// findCodeSpan is the code span holding the byte at pos, if one does.
func findCodeSpan(spans [][2]int, pos int) ([2]int, bool) {
	for _, span := range spans {
		if pos >= span[0] && pos < span[1] {
			return span, true
		}
	}
	return [2]int{}, false
}

// hasCodeSpanIn reports whether a code span begins between from and to.
func hasCodeSpanIn(spans [][2]int, from, to int) bool {
	for _, span := range spans {
		if span[0] > from && span[0] < to {
			return true
		}
	}
	return false
}
