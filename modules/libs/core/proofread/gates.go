package proofread

import (
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// MaxEditDistance is how far a correction may stand from the line as read.
const MaxEditDistance = 0.30

// Fixed is the lines a reply puts right, whether any of its runs ran on past
// the last line the batch carries, and whether the reply answers the question
// that was asked.
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
func Fixed(batch Batch, reply string, maxDistance float64) (put []Line, past, ok bool) {
	if strings.Contains(reply, Opens) || strings.Contains(reply, Closes) {
		return nil, false, false
	}

	read := make(map[int]string, len(batch.Lines))
	for _, line := range batch.Lines {
		read[line.Number] = line.Text
	}

	var out []Line
	answered := make(map[int]bool, len(batch.Lines))
	for _, row := range strings.Split(unfenced(reply), "\n") {
		row = strings.TrimSpace(row)
		if row == "" {
			continue
		}
		at, through, text, barred, numbers := numbered(row)
		if !numbers {
			return nil, false, false
		}
		was, named := read[at]
		if !named {
			return nil, false, false
		}
		if through > at && !batch.Joinable {
			return nil, false, false
		}
		// A sentence runs on past the last line a batch was given, and a model
		// reading it names the whole of it. Such a run is dropped and the rest
		// of the batch stands; a run skipping a line in the middle is an answer
		// against the wrong numbers, and refuses the batch.
		joined, reaches := was, false
		for line := at + 1; line <= through; line++ {
			next, held := read[line]
			if !held {
				if line > batch.Last() {
					reaches = true
					break
				}
				return nil, false, false
			}
			joined += " " + next
		}
		if reaches {
			past = true
			continue
		}
		// A line is answered for once, by one row of the reply.
		for line := at; line <= through; line++ {
			if answered[line] {
				return nil, false, false
			}
			answered[line] = true
		}
		was = joined
		// A line opening with the digits the row opens with, and no bar to tell
		// the two apart, is a row whose number was left out: the line's own
		// first word reads as the number, and the batch is refused.
		if !barred && strings.HasPrefix(strings.TrimSpace(was), opening(row)) {
			return nil, false, false
		}
		// An answer of nothing, or of marks that are not words, over a line that
		// said something empties it.
		if len(letters(text)) == 0 && len(letters(was)) > 0 {
			return nil, false, false
		}
		// A run answered with what its lines already say is still a change:
		// the lines are put together.
		if through == at && text == strings.TrimSpace(was) {
			continue
		}
		// What is dropped is a correction that changes nothing. A run changes
		// the lines whatever it says: they become one.
		if through == at {
			if fronted(was, text) {
				continue
			}
			if maxDistance > 0 && EditDistance(was, text) > maxDistance {
				continue
			}
		}
		out = append(out, Line{Number: at, Last: through, Text: text})
	}
	return out, past, true
}

// Gathered is what a run of batches put right, keyed by the line, and the
// batches whose reply ran on past the last line they carry, in the order they
// were asked. It is given the batches as they were asked and the replies keyed
// by the number each batch is known by.
//
// Where two batches answer about one line, the correction is the one from the
// batch in which the line stands further from the end; where they stand equally
// far, it is the one from the later batch. A reply the gates refuse puts
// nothing right, and a line no accepted reply covers is not in the result.
func Gathered(asked []Batch, replies map[int]string, maxDistance float64) (map[int]Line, []int) {
	put := make(map[int]Line)
	best := make(map[int]int)
	var past []int
	for _, batch := range asked {
		reply, answered := replies[batch.Number]
		if !answered {
			continue
		}
		lines, ran, ok := Fixed(batch, reply, maxDistance)
		if !ok {
			continue
		}
		if ran {
			past = append(past, batch.Number)
		}
		after := make(map[int]int, len(batch.Lines))
		for i, line := range batch.Lines {
			after[line.Number] = len(batch.Lines) - 1 - i
		}
		for _, line := range lines {
			if stood, seen := best[line.Number]; seen && after[line.Number] < stood {
				continue
			}
			put[line.Number] = line
			best[line.Number] = after[line.Number]
		}
	}
	return put, past
}

// numbered is the line a reply row is about, what that line now says, and
// whether a bar stood between the two.
//
// A row opens with the number, and a bar, spaces, or both stand between the
// number and the line.
func numbered(row string) (at, through int, text string, barred, ok bool) {
	digits := opening(row)
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
		last := opening(rest[1:])
		if last == "" {
			return 0, 0, "", false, false
		}
		if through, err = strconv.Atoi(last); err != nil || through <= at {
			return 0, 0, "", false, false
		}
		rest = rest[1+len(last):]
		if rest == "" {
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

// opening is the run of digits a row opens with, and nothing for a row opening
// with anything else.
func opening(row string) string {
	digits := 0
	for digits < len(row) && row[digits] >= '0' && row[digits] <= '9' {
		digits++
	}
	return row[:digits]
}

// fronted is a correction that says what the line says with something wordless
// put in front of it.
//
// A separator this does not know — a dash, an arrow, a colon — stands where the
// line begins, and the letters either side of it are the same, so nothing that
// counts letters sees it.
func fronted(was, text string) bool {
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

// EditDistance is the Levenshtein distance between two lines' letters, as a
// share of the longer of them. Two lines of no letters stand nowhere apart.
func EditDistance(a, b string) float64 {
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
