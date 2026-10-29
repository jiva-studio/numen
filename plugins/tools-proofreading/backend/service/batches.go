package service

// speechBatches is the lines of a transcript cut into batches of size lines,
// each batch opening on the last overlap lines of the one before it. Speech runs
// on past the cut, so the lines a batch shares with its neighbour are seen whole
// by one of the two.
//
// A batch is known by its place in the run. A line saying nothing is left out.
// A run of lines is answered for inside one batch, so a sentence broken over a
// cut is put back together where the overlap carries it whole.
func speechBatches(all []Line, size, overlap int) []Batch {
	if size <= 0 {
		return nil
	}
	lines := getSpokenLines(all)
	step := size - getSharedLines(size, overlap)
	var out []Batch
	for start := 0; start < len(lines); start += step {
		end := min(start+size, len(lines))
		out = append(out, Batch{Number: len(out), Lines: lines[start:end:end], IsJoinable: true})
		if end == len(lines) {
			break
		}
	}
	return out
}

// cutBatches is a batch for each of the cuts, holding the size lines around that
// cut, half of them before it and half after, and numbered on from the batches
// speechBatches gives back. A cut is named by the batch it comes after, and the
// cuts come in the order they stand in the transcript.
//
// A run of lines is answered for inside one batch, and a sentence reaching
// across a cut further than the overlap carries is in no batch speechBatches
// makes. The batch over that cut holds it whole.
//
// A transcript of one batch has no cut, and a batch of one line has no room for
// a line on either side of a cut. The last batch is followed by no cut.
func cutBatches(all []Line, size, overlap int, cuts []int) []Batch {
	batches := speechBatches(all, size, overlap)
	if size < 2 || len(batches) < 2 {
		return nil
	}

	lines := getSpokenLines(all)
	step := size - getSharedLines(size, overlap)
	var out []Batch
	reach := -1
	for _, cutIndex := range cuts {
		if cutIndex < 0 || cutIndex >= len(batches)-1 {
			continue
		}
		// Cuts near the end of the transcript can map to the same window. A batch
		// is added only when it reaches further than the previous one.
		end := min(max((cutIndex+1)*step-size/2, 0)+size, len(lines))
		if end <= reach {
			continue
		}
		reach = end
		start := max(end-size, 0)
		out = append(out, Batch{
			Number:     len(batches) + len(out),
			Lines:      lines[start:end:end],
			IsJoinable: true,
		})
	}
	return out
}

// getSpokenLines returns the lines that say something.
func getSpokenLines(all []Line) []Line {
	var out []Line
	for _, line := range all {
		if line.Text == "" {
			continue
		}
		out = append(out, line)
	}
	return out
}

// getSharedLines returns the number of lines a batch shares with the previous
// batch: always fewer than size, so each batch reaches further into the
// transcript than its predecessor.
func getSharedLines(size, overlap int) int {
	if overlap < 0 {
		return 0
	}
	return min(overlap, size-1)
}
