package service

import (
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// MaxEditDistance is how far a correction may stand from the line as read.
const MaxEditDistance = 0.30

// GetFixedLines is the lines a reply puts right, whether any of its runs ran on
// past the last line the batch carries, and whether the reply answers the
// question that was asked.
//
// A sentence carried on past the end of a batch is one the cut after that batch
// broke, and past is how a caller learns of it.
//
// A mark of ours coming back refuses the batch, as does a reply row that is not
// a number the batch carries and, after it, the line, and as does an answer
// with no letters over a line that had them. A run of lines answered for as one
// refuses a batch that does not put lines together, and a line two rows both
// answer for refuses the batch.
//
// A correction saying what the line already says is dropped, as is one that
// only puts something wordless in front of it, and as is one standing further
// than maxDistance from the line as read. A maxDistance at or below zero sets
// no limit.
func GetFixedLines(batch Batch, reply string, maxDistance float64) (put []Line, past, ok bool) {
	if strings.Contains(reply, Opens) || strings.Contains(reply, Closes) {
		return nil, false, false
	}
	return buildFixedLines(batch, reply, maxDistance, indexBatchLines(batch))
}

// indexBatchLines maps each line number in the batch to its text.
func indexBatchLines(batch Batch) map[int]string {
	read := make(map[int]string, len(batch.Lines))
	for _, line := range batch.Lines {
		read[line.Number] = line.Text
	}
	return read
}

// buildFixedLines parses reply rows, verifying each against the batch and collecting
// the accepted corrections.
func buildFixedLines(batch Batch, reply string, maxDistance float64, read map[int]string) ([]Line, bool, bool) {
	var out []Line
	past := false
	replyVal := make(map[int]bool, len(batch.Lines))
	for _, row := range splitRows(reply) {
		line, keep, reaches, valid := processRow(row, batch, read, replyVal, maxDistance)
		if !valid {
			return nil, false, false
		}
		if reaches {
			past = true
			continue
		}
		if keep {
			out = append(out, line)
		}
	}
	return out, past, true
}

// processRow parses and checks a single reply row against the batch lines and edit distance.
func processRow(row string, batch Batch, read map[int]string, answered map[int]bool, maxDistance float64) (line Line, keep, reaches, ok bool) {
	at, through, text, barred, numbers := readRow(row)
	if !numbers {
		return Line{}, false, false, false
	}
	if through > at && !batch.IsJoinable {
		return Line{}, false, false, false
	}
	was, reaches, matchOk := matchLines(read, at, through, batch.Last())
	if !matchOk {
		return Line{}, false, false, false
	}
	if reaches {
		return Line{}, false, true, true
	}
	if !claimLines(answered, at, through) {
		return Line{}, false, false, false
	}
	if !isValidRow(row, was, text, barred) {
		return Line{}, false, false, false
	}
	if !canKeepCorrection(was, text, at, through, maxDistance) {
		return Line{}, false, false, true
	}
	return Line{Number: at, Last: through, Text: text}, true, false, true
}

// matchLines gathers the lines from at through to, verifying each exists in the batch.
// A run reaching beyond the batch reports reaches true.
func matchLines(read map[int]string, at, through, last int) (joined string, reaches, ok bool) {
	was, named := read[at]
	if !named {
		return "", false, false
	}
	for line := at + 1; line <= through; line++ {
		next, target := read[line]
		if !target {
			if line > last {
				return "", true, true
			}
			return "", false, false
		}
		was += " " + next
	}
	return was, false, true
}

// claimLines marks the given line range as answered, refusing lines answered twice.
func claimLines(answered map[int]bool, at, through int) bool {
	for line := at; line <= through; line++ {
		if answered[line] {
			return false
		}
		answered[line] = true
	}
	return true
}

// isValidRow says whether the reply row has words and its digits separate from the line.
func isValidRow(row, was, text string, barred bool) bool {
	if !barred && strings.HasPrefix(strings.TrimSpace(was), readLeadingDigits(row)) {
		return false
	}
	if len(extractLetters(text)) == 0 && len(extractLetters(was)) > 0 {
		return false
	}
	return true
}

// canKeepCorrection says whether a correction changes the line and stands within
// the allowed edit distance. A run of lines is always kept.
func canKeepCorrection(was, text string, at, through int, maxDistance float64) bool {
	if through > at {
		return true
	}
	if text == strings.TrimSpace(was) {
		return false
	}
	if isPrefixedBySymbols(was, text) {
		return false
	}
	if maxDistance > 0 && EditDistance(was, text) > maxDistance {
		return false
	}
	return true
}

// readRow is the line a reply row is about, what that line now says, and
// whether a bar stood between the two.
//
// A row opens with the number, and a bar, spaces, or both stand between the
// number and the line.
func readRow(row string) (at, through int, text string, barred, ok bool) {
	digits := readLeadingDigits(row)
	if digits == "" || len(digits) == len(row) {
		return 0, 0, "", false, false
	}
	at, err := strconv.Atoi(digits)
	if err != nil {
		return 0, 0, "", false, false
	}
	rest := row[len(digits):]
	through = at

	// A run of lines put together is written as the first and the last of them.
	if rest[0] == '-' {
		var ok bool
		if through, rest, ok = readRowRange(rest, at); !ok {
			return 0, 0, "", false, false
		}
	}

	if rest[0] != '|' && rest[0] != ' ' && rest[0] != '\t' {
		return 0, 0, "", false, false
	}
	rest = strings.TrimLeft(rest, " \t")
	barred = strings.HasPrefix(rest, "|")
	return at, through, strings.TrimSpace(strings.TrimPrefix(rest, "|")), barred, true
}

// readRowRange parses the ending line number in a hyphenated line range.
func readRowRange(rest string, at int) (through int, remainder string, ok bool) {
	last := readLeadingDigits(rest[1:])
	if last == "" {
		return 0, "", false
	}
	through, err := strconv.Atoi(last)
	if err != nil || through <= at {
		return 0, "", false
	}
	remainder = rest[1+len(last):]
	if remainder == "" {
		return 0, "", false
	}
	return through, remainder, true
}

// readLeadingDigits is the run of digits a row opens with, and nothing for a row readLeadingDigits
// with anything else.
func readLeadingDigits(row string) string {
	digits := 0
	for digits < len(row) && row[digits] >= '0' && row[digits] <= '9' {
		digits++
	}
	return row[:digits]
}

// isPrefixedBySymbols is a correction that says what the line says with something
// wordless put in front of it.
//
// A separator this does not know — a dash, an arrow, a colon — stands where the
// line begins, and the letters either side of it are the same, so nothing that
// counts letters sees it.
func isPrefixedBySymbols(was, text string) bool {
	was = strings.TrimSpace(was)
	if was == "" || !strings.HasSuffix(text, was) {
		return false
	}
	for _, r := range text[:len(text)-len(was)] {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// stripFence is a reply with the code fence a model wrapped it in taken off.
func stripFence(reply string) string {
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

// EditDistance is the Levenshtein distance between two lines' letters, as a
// share of the longer of them. Two lines of no letters stand nowhere apart.
func EditDistance(a, b string) float64 {
	x, y := extractLetters(a), extractLetters(b)
	long := len(x)
	if len(y) > long {
		long = len(y)
	}
	if long == 0 {
		return 0
	}
	return float64(countEdits(x, y)) / float64(long)
}

// extractLetters returns what a line says with its spaces, punctuation, symbols, diacritics
// and case taken off it.
func extractLetters(line string) []rune {
	var out []rune
	for _, r := range norm.NFD.String(line) {
		if unicode.Is(unicode.Mn, r) || unicode.IsSpace(r) || unicode.IsPunct(r) || unicode.IsSymbol(r) {
			continue
		}
		out = append(out, unicode.ToLower(r))
	}
	return out
}

// countEdits returns the Levenshtein distance between two runs of letters, carrying one
// row of the table at a time.
func countEdits(x, y []rune) int {
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
