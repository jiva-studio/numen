package service

import (
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// MaxEditDistance is the maximum normalized edit distance allowed for a correction.
const MaxEditDistance = 0.30

// GetFixedLines parses and validates correction lines from a model reply against a batch.
// Returns validated lines, hasOverflow if a line range extends beyond the batch, and ok indicating overall validity.
func GetFixedLines(batch Batch, reply string, maxDistance float64) (lines []Line, hasOverflow, ok bool) {
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

// buildFixedLines parses reply rows, validates each against the batch, and collects accepted corrections.
func buildFixedLines(batch Batch, reply string, maxDistance float64, read map[int]string) ([]Line, bool, bool) {
	var out []Line
	hasOverflow := false
	claimedLines := make(map[int]bool, len(batch.Lines))
	for _, row := range splitRows(reply) {
		line, keep, reaches, valid := processRow(row, batch, read, claimedLines, maxDistance)
		if !valid {
			return nil, false, false
		}
		if reaches {
			hasOverflow = true
			continue
		}
		if keep {
			out = append(out, line)
		}
	}
	return out, hasOverflow, true
}

// processRow parses and checks a single reply row against the batch lines and edit distance.
func processRow(row string, batch Batch, read map[int]string, claimedLines map[int]bool, maxDistance float64) (line Line, keep, reaches, ok bool) {
	startLine, endLine, text, barred, numbers := readRow(row)
	if !numbers {
		return Line{}, false, false, false
	}
	if endLine > startLine && !batch.IsJoinable {
		return Line{}, false, false, false
	}
	was, reaches, matchOk := matchLines(read, startLine, endLine, batch.Last())
	if !matchOk {
		return Line{}, false, false, false
	}
	if reaches {
		return Line{}, false, true, true
	}
	if !claimLines(claimedLines, startLine, endLine) {
		return Line{}, false, false, false
	}
	if !isValidRow(row, was, text, barred) {
		return Line{}, false, false, false
	}
	if !canKeepCorrection(was, text, startLine, endLine, maxDistance) {
		return Line{}, false, false, true
	}
	return Line{Number: startLine, Last: endLine, Text: text}, true, false, true
}

// matchLines concatenates lines from startLine to endLine and checks whether each exists in the batch.
// If a line range extends beyond the last batch line, reaches is true.
func matchLines(read map[int]string, startLine, endLine, last int) (joined string, reaches, ok bool) {
	was, named := read[startLine]
	if !named {
		return "", false, false
	}
	for line := startLine + 1; line <= endLine; line++ {
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

// claimLines marks the given line range as answered, rejecting duplicate lines.
func claimLines(claimedLines map[int]bool, startLine, endLine int) bool {
	for line := startLine; line <= endLine; line++ {
		if claimedLines[line] {
			return false
		}
		claimedLines[line] = true
	}
	return true
}

// isValidRow validates that the reply row format and content are well-formed.
func isValidRow(row, was, text string, barred bool) bool {
	if !barred && strings.HasPrefix(strings.TrimSpace(was), readLeadingDigits(row)) {
		return false
	}
	if len(extractLetters(text)) == 0 && len(extractLetters(was)) > 0 {
		return false
	}
	return true
}

// canKeepCorrection reports whether a correction modifies the line within the maximum allowed edit distance.
func canKeepCorrection(was, text string, startLine, endLine int, maxDistance float64) bool {
	if endLine > startLine {
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

// readRow parses a reply row into line bounds, text, and delimiter flags.
func readRow(row string) (startLine, endLine int, text string, barred, ok bool) {
	digits := readLeadingDigits(row)
	if digits == "" || len(digits) == len(row) {
		return 0, 0, "", false, false
	}
	startLine, err := strconv.Atoi(digits)
	if err != nil {
		return 0, 0, "", false, false
	}
	rest := row[len(digits):]
	endLine = startLine

	// Hyphenated range indicates multiple lines combined into one.
	if rest[0] == '-' {
		var ok bool
		if endLine, rest, ok = readRowRange(rest, startLine); !ok {
			return 0, 0, "", false, false
		}
	}

	if rest[0] != '|' && rest[0] != ' ' && rest[0] != '\t' {
		return 0, 0, "", false, false
	}
	rest = strings.TrimLeft(rest, " \t")
	barred = strings.HasPrefix(rest, "|")
	return startLine, endLine, strings.TrimSpace(strings.TrimPrefix(rest, "|")), barred, true
}

// readRowRange parses the ending line number in a hyphenated line range.
func readRowRange(rest string, startLine int) (endLine int, remainder string, ok bool) {
	last := readLeadingDigits(rest[1:])
	if last == "" {
		return 0, "", false
	}
	endLine, err := strconv.Atoi(last)
	if err != nil || endLine <= startLine {
		return 0, "", false
	}
	remainder = rest[1+len(last):]
	if remainder == "" {
		return 0, "", false
	}
	return endLine, remainder, true
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
