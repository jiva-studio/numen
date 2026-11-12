package markdown

import (
	"strings"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// Wikilink is one `[[…]]` at its current position in a line.
type Wikilink struct {
	At, To int
	Inside string
	Target domain.Address
}

// WikilinksIn is every wikilink in one line, in the order they are written.
// Brackets holding nothing point nowhere, so they are the text they are
// written as. Brackets inside inline code are an example of a link, not one.
func WikilinksIn(line string) []Wikilink {
	var out []Wikilink
	spans := getCodeSpans(line)
	pos := 0
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="at pos == len(line) strings.Index on the empty slice returns -1 and breaks"
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
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="at lineStart == len(text) the trailing empty slice contains no wikilinks"
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

func getCodeSpans(line string) [][2]int {
	var out [][2]int
	// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="at pos == len(line) strings.IndexByte on the empty slice returns -1 and breaks"
	for pos := 0; pos < len(line); {
		open := strings.IndexByte(line[pos:], '`')
		if open < 0 {
			break
		}
		open += pos
		size := getBacktickRun(line, open)
		closeAt := -1
		// gomutants:disable-next-line CONDITIONALS_BOUNDARY reason="at p == len(line) strings.IndexByte on the empty slice returns -1 and breaks"
		for p := open + size; p < len(line); {
			next := strings.IndexByte(line[p:], '`')
			// gomutants:disable-next-line CONDITIONALS_BOUNDARY,INTEGER_INCREMENT reason="p is always immediately past a full backtick run so line[p] is never a backtick and next is never 0"
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

func getBacktickRun(line string, start int) int {
	end := start
	for end < len(line) && line[end] == '`' {
		end++
	}
	return end - start
}

func findCodeSpan(spans [][2]int, pos int) ([2]int, bool) {
	for _, span := range spans {
		if pos >= span[0] && pos < span[1] {
			return span, true
		}
	}
	return [2]int{}, false
}

func hasCodeSpanIn(spans [][2]int, from, to int) bool {
	for _, span := range spans {
		if span[0] > from && span[0] < to {
			return true
		}
	}
	return false
}
