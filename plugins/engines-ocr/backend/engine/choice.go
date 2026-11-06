package engine

import (
	"cmp"
	"errors"
	"image"
	"slices"
	"unicode"

	read "github.com/getcharzp/go-ocr"
)

// readPart reads one part of a page with every recognition model and keeps, per row of text, the
// reading of the most confident one. A model that fails is left out; the part fails only when every
// model fails.
func (r *Recogniser) readPart(crop image.Image) ([]read.RecResult, error) {
	results := make([][]read.RecResult, 0, len(r.read))
	var failures []error
	for _, readPart := range r.read {
		found, err := readPart(crop)
		if err != nil {
			failures = append(failures, err)
			continue
		}
		results = append(results, found)
	}
	if len(results) == 0 {
		return nil, errors.Join(failures...)
	}
	return chooseLines(results), nil
}

// chooseLines keeps for each row of text the reading with the highest confidence, the earlier
// reading on a tie. Readings that do not box the same lines cannot be paired, so the first is kept
// whole.
func chooseLines(results [][]read.RecResult) []read.RecResult {
	first := results[0]
	for _, other := range results[1:] {
		if !hasSameBoxes(first, other) {
			return first
		}
	}
	out := slices.Clone(first)
	for _, row := range getRows(first) {
		best := measureConfidence(first, row)
		for _, other := range results[1:] {
			if confidence := measureConfidence(other, row); confidence > best {
				best = confidence
				for _, i := range row {
					out[i] = other[i]
				}
			}
		}
	}
	return out
}

// measureConfidence sums the confidence of every letter the reading gives the row. The score of a
// line is the mean over the letters a model emits, and a model whose alphabet lacks a letter skips
// it; the sum counts a skipped letter as nothing.
func measureConfidence(lines []read.RecResult, row []int) float64 {
	total := 0.0
	for _, i := range row {
		count := 0
		for _, letter := range lines[i].Text {
			if !unicode.IsSpace(letter) {
				count++
			}
		}
		total += float64(lines[i].Score) * float64(count)
	}
	return total
}

// getRows groups the lines into rows of text, as Assemble does, by their positions in found.
func getRows(found []read.RecResult) [][]int {
	order := make([]int, len(found))
	for i := range order {
		order[i] = i
	}
	slices.SortStableFunc(order, func(a, b int) int { return cmp.Compare(found[a].Box[1], found[b].Box[1]) })
	var rows [][]int
	var row []Line
	for _, i := range order {
		box := found[i].Box
		line := Line{Box: image.Rect(box[0], box[1], box[2], box[3]), Text: "", Score: 0}
		if len(row) > 0 && measureVerticalOverlap(row, line) >= 0.5 {
			row = append(row, line)
			rows[len(rows)-1] = append(rows[len(rows)-1], i)
			continue
		}
		row = []Line{line}
		rows = append(rows, []int{i})
	}
	return rows
}

func hasSameBoxes(a, b []read.RecResult) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Box != b[i].Box {
			return false
		}
	}
	return true
}
