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

func readAnswer(batch Batch, reply string, maxDistance float64) answer {
	fixed, past, ok := GetFixedLines(batch, reply, maxDistance)
	if !ok {
		return answer{batch: batch, lines: nil, reaching: nil, isAnswer: false}
	}
	out := answer{batch: batch, lines: fixed, reaching: nil, isAnswer: true}
	if past {
		out.reaching = getReachingRuns(batch, reply)
	}
	return out
}

func getReachingRuns(batch Batch, reply string) []int {
	var starts []int
	for _, row := range splitRows(reply) {
		if pos, through, _, _, ok := readRow(row); ok && through > batch.Last() {
			starts = append(starts, pos)
		}
	}
	return starts
}

func gatherAnswers(answers []answer) (map[int]Line, []int) {
	collector := gathering{
		lines: make(map[int]Line),
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
			collector.keep(line, after)
		}
	}
	return collector.lines, past
}

// gathering is the corrections kept so far.
type gathering struct {
	lines map[int]Line
	owner map[int]int
	best  map[int]int
}

func (g *gathering) keep(line Line, after map[int]int) {
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
	g.lines[first] = line
	for index := first; index <= last; index++ {
		g.owner[index], g.best[index] = first, after[index]
	}
}

func (g *gathering) evict(first int) {
	line := g.lines[first]
	for pos := first; pos <= max(line.Last, line.Number); pos++ {
		delete(g.owner, pos)
		delete(g.best, pos)
	}
	delete(g.lines, first)
}

func countRemainingLines(lines []Line) map[int]int {
	remaining := make(map[int]int, len(lines))
	for i, line := range lines {
		remaining[line.Number] = len(lines) - 1 - i
	}
	return remaining
}

func sortLines(lines map[int]Line) []Line {
	out := make([]Line, 0, len(lines))
	for _, line := range lines {
		out = append(out, line)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Number < out[j].Number })
	return out
}

func splitRows(reply string) []string {
	var rows []string
	for _, row := range strings.Split(stripFence(reply), "\n") {
		if row = strings.TrimSpace(row); row != "" {
			rows = append(rows, row)
		}
	}
	return rows
}
