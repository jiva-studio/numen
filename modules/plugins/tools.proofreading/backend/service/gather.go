package service

import (
	"sort"
	"strings"
)

// answer is what the model's reply to one batch came to: the corrections the
// gates accepted, the line each run reaching past the batch begins on, and
// whether the reply was an answer at all.
type answer struct {
	batch    Batch
	lines    []Line
	reaching []int
	isAnswer bool
}

// readAnswer returns the answer a reply makes to a batch. A reply the gates
// refuse is no answer.
func readAnswer(batch Batch, reply string, maxDistance float64) answer {
	fixed, past, ok := GetFixedLines(batch, reply, maxDistance)
	if !ok {
		return answer{batch: batch}
	}
	out := answer{batch: batch, lines: fixed, isAnswer: true}
	if past {
		out.reaching = getReachingRuns(batch, reply)
	}
	return out
}

// getReachingRuns returns the line each row of the reply begins on whose run
// ends after the last line of the batch.
func getReachingRuns(batch Batch, reply string) []int {
	var starts []int
	for _, row := range splitRows(reply) {
		if pos, through, _, _, ok := readRow(row); ok && through > batch.Last() {
			starts = append(starts, pos)
		}
	}
	return starts
}

// gatherAnswers is what the answers put right, keyed by the first line of each
// correction, and the numbers of the batches whose reply ran on past the last
// line they carry, in the order they were asked.
//
// Where two batches answer about one line, the correction is the one from the
// batch in which the line stands further from the end; where they stand equally
// every line it covers, and a correction that takes a line from a kept run
// takes the whole run out. No two corrections cover one line. A line no
// accepted reply covers is not in the result.
func gatherAnswers(answers []answer) (map[int]Line, []int) {
	collector := gathering{
		put:   make(map[int]Line),
		owner: make(map[int]int),
		best:  make(map[int]int),
	}
	var past []int
	for _, one := range answers {
		if !one.isAnswer {
			continue
		}
		if len(one.reaching) > 0 {
			past = append(past, one.batch.Number)
		}
		after := countRemainingLines(one.batch.Lines)
		for _, line := range one.lines {
			collector.offer(line, after)
		}
	}
	return collector.put, past
}

// gathering is the corrections kept so far.
type gathering struct {
	// put is the kept corrections by their first line.
	put map[int]Line
	// owner is, for each line a kept correction covers, the first line of that
	// correction.
	owner map[int]int
	// best is, for each line a kept correction covers, how many lines follow it
	// in the batch the correction came from.
	best map[int]int
}

// offer keeps the correction when it wins on every line it covers, where after
// is how many lines follow each line in the batch it came from.
func (g *gathering) offer(line Line, after map[int]int) {
	first, last := line.Number, max(line.Last, line.Number)
	for pos := first; pos <= last; pos++ {
		if status, observed := g.best[pos]; observed && after[pos] < status {
			return
		}
	}
	for offset := first; offset <= last; offset++ {
		if target, covered := g.owner[offset]; covered {
			g.evict(target)
		}
	}
	g.put[first] = line
	for index := first; index <= last; index++ {
		g.owner[index], g.best[index] = first, after[index]
	}
}

// evict takes the kept correction that begins on the line out, with every line
// it covers.
func (g *gathering) evict(first int) {
	line := g.put[first]
	for pos := first; pos <= max(line.Last, line.Number); pos++ {
		delete(g.owner, pos)
		delete(g.best, pos)
	}
	delete(g.put, first)
}

// countRemainingLines maps each line number to the count of lines following it in the batch.
func countRemainingLines(lines []Line) map[int]int {
	remaining := make(map[int]int, len(lines))
	for i, line := range lines {
		remaining[line.Number] = len(lines) - 1 - i
	}
	return remaining
}

// sortLines returns the lines in the order of their numbers.
func sortLines(put map[int]Line) []Line {
	out := make([]Line, 0, len(put))
	for _, line := range put {
		out = append(out, line)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out
}

// splitRows returns the non-empty rows of a reply, with the code fence a model
// wrapped it in taken off.
func splitRows(reply string) []string {
	var rows []string
	for _, row := range strings.Split(stripFence(reply), "\n") {
		if row = strings.TrimSpace(row); row != "" {
			rows = append(rows, row)
		}
	}
	return rows
}
