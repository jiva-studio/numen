package proofread

import (
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// LettersApart is how far a correction's letters may stand from the line as
// read: the share measured over proofread pages.
const LettersApart = 0.30

// Fixed is the lines a reply puts right, and whether the reply answers the
// question that was asked.
//
// A mark of ours coming back refuses the page, as does a reply line that is not
// a number the page carries, a bar and text. A correction whose letters stand
// further than apart from the line as read is dropped, as is one saying what
// the line already says.
func Fixed(page Page, reply string, apart float64) ([]Line, bool) {
	if strings.Contains(reply, Opens) || strings.Contains(reply, Closes) {
		return nil, false
	}

	read := make(map[int]string, len(page.Lines))
	for _, line := range page.Lines {
		read[line.At] = line.Text
	}

	var out []Line
	for _, row := range strings.Split(unfenced(reply), "\n") {
		row = strings.TrimSpace(row)
		if row == "" {
			continue
		}
		bar := strings.IndexByte(row, '|')
		if bar < 0 {
			return nil, false
		}
		at, err := strconv.Atoi(strings.TrimSpace(row[:bar]))
		if err != nil {
			return nil, false
		}
		was, named := read[at]
		if !named {
			return nil, false
		}
		text := strings.TrimSpace(row[bar+1:])
		if text == strings.TrimSpace(was) {
			continue
		}
		if Apart(was, text) > apart {
			continue
		}
		out = append(out, Line{At: at, Text: text})
	}
	return out, true
}

// unfenced is a reply with the code fence a model wrapped it in taken off.
func unfenced(reply string) string {
	body := strings.TrimSpace(reply)
	if !strings.HasPrefix(body, "```") {
		return body
	}
	head := strings.IndexByte(body, '\n')
	if head < 0 {
		return ""
	}
	body = body[head+1:]
	if tail := strings.LastIndex(body, "```"); tail >= 0 {
		body = body[:tail]
	}
	return body
}

// Apart is how far two lines' letters stand from one another, as a share of the
// longer of them. Two lines of no letters stand nowhere apart.
func Apart(a, b string) float64 {
	x, y := letters(a), letters(b)
	long := len(x)
	if len(y) > long {
		long = len(y)
	}
	if long == 0 {
		return 0
	}
	return float64(edits(x, y)) / float64(long)
}

// letters is what a line says with its spaces, punctuation, symbols, diacritics
// and case taken off it.
func letters(line string) []rune {
	var out []rune
	for _, r := range norm.NFD.String(line) {
		if unicode.Is(unicode.Mn, r) || unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		out = append(out, unicode.ToLower(r))
	}
	return out
}

// edits is the Levenshtein distance between two runs of letters, carrying one
// row of the table at a time.
func edits(x, y []rune) int {
	row := make([]int, len(y)+1)
	for j := range row {
		row[j] = j
	}
	for i := 1; i <= len(x); i++ {
		last := row[0]
		row[0] = i
		for j := 1; j <= len(y); j++ {
			keep := row[j]
			cost := 1
			if x[i-1] == y[j-1] {
				cost = 0
			}
			row[j] = min(row[j]+1, row[j-1]+1, last+cost)
			last = keep
		}
	}
	return row[len(y)]
}
