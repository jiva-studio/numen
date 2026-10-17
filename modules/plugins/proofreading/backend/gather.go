package main

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

// readAnswer returns the answer a reply makes to a batch, with at most
// maxSuggestions corrections where that is above zero. A reply the gates refuse
// is no answer.
func readAnswer(batch Batch, reply string, maxDistance float64, maxSuggestions int) answer {
	fixed, past, ok := GetFixedLines(batch, reply, maxDistance)
	if !ok {
		return answer{batch: batch}
	}
	if maxSuggestions > 0 && len(fixed) > maxSuggestions {
		fixed = fixed[:maxSuggestions]
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
		if at, through, _, _, ok := readRow(row); ok && through > batch.Last() {
			starts = append(starts, at)
		}
	}
	return starts
}

// gatherAnswers is what the answers put right, keyed by the line, and the
// numbers of the batches whose reply ran on past the last line they carry, in
// the order they were asked.
//
// Where two batches answer about one line, the correction is the one from the
// batch in which the line stands further from the end; where they stand equally
// far, it is the one from the later batch. A line no accepted reply covers is
// not in the result.
func gatherAnswers(answers []answer) (map[int]Line, []int) {
	put := make(map[int]Line)
	best := make(map[int]int)
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
			if stood, seen := best[line.Number]; seen && after[line.Number] < stood {
				continue
			}
			put[line.Number] = line
			best[line.Number] = after[line.Number]
		}
	}
	return put, past
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
