package service

func speechBatches(all []Line, size, overlap int) []Batch {
	if size <= 0 {
		return nil
	}
	lines := getSpokenLines(all)
	step := size - getSharedLines(size, overlap)
	var out []Batch
	for start := 0; start < len(lines); start += step {
		end := min(start+size, len(lines))
		out = append(out, Batch{Number: len(out), Lines: lines[start:end:end], IsJoinable: true, Context: ""})
		if end == len(lines) {
			break
		}
	}
	return out
}

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
		// is added only when it extends beyond the previous one.
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
			Context:    "",
		})
	}
	return out
}

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

func getSharedLines(size, overlap int) int {
	if overlap < 0 {
		return 0
	}
	return min(overlap, size-1)
}
