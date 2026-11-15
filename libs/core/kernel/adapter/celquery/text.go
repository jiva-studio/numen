package celquery

import (
	"unicode"

	"cel.dev/cel-go/common"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

type queryText struct {
	runes      []rune
	lineStarts []int
}

func newQueryText(text string) queryText {
	runes := []rune(text)
	starts := []int{0}
	for i, r := range runes {
		if r == '\n' {
			starts = append(starts, i+1)
		}
	}
	return queryText{runes: runes, lineStarts: starts}
}

func (t queryText) getOffset(location common.Location) int {
	line := min(max(location.Line(), 1), len(t.lineStarts))
	return min(t.lineStarts[line-1]+max(location.Column(), 0), len(t.runes))
}

func (t queryText) getPosition(offset int) searchquery.Position {
	line := 1
	for line < len(t.lineStarts) && t.lineStarts[line] <= offset {
		line++
	}
	return searchquery.Position{Line: line, Column: offset - t.lineStarts[line-1] + 1}
}

func (t queryText) getRune(offset int) rune {
	if offset < 0 || offset >= len(t.runes) {
		return 0
	}
	return t.runes[offset]
}

func (t queryText) isAtEnd(offset int) bool {
	for _, r := range t.runes[min(offset, len(t.runes)):] {
		if !unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func (t queryText) getLastRuneOffset() int {
	i := len(t.runes) - 1
	for i >= 0 && unicode.IsSpace(t.runes[i]) {
		i--
	}
	return i
}

func (t queryText) findNameStart(offset int) int {
	if t.getRune(offset) != '(' {
		return offset
	}
	i := offset - 1
	for i >= 0 && unicode.IsSpace(t.runes[i]) {
		i--
	}
	end := i
	for i >= 0 && isNameRune(t.runes[i]) {
		i--
	}
	if i == end {
		return offset
	}
	return i + 1
}

func isNameRune(r rune) bool {
	return r == '_' || r < unicode.MaxASCII && (unicode.IsLetter(r) || unicode.IsDigit(r))
}
