package main

import (
	"context"
	"sort"
)

// askModel returns the model's reply to a prompt under an instruction.
type askModel func(ctx context.Context, instruction, prompt string) (string, error)

// textOptions are the settings a request is proofread under.
type textOptions struct {
	strict         bool
	maxSuggestions int
	maxDistance    float64
}

// textResult is what proofreading a request came to.
type textResult struct {
	corrections []Line
	uncorrected int
	// next is the line to continue from, and isDone says nothing is left after
	// the lines answered. Both are for speech.
	next   int
	isDone bool
}

// getScanDistance is how far a correction may stand from a line read off a page.
func getScanDistance(opts textOptions) float64 {
	distance := opts.maxDistance
	if distance <= 0 {
		distance = MaxEditDistance
	}
	if opts.strict {
		distance = max(distance, StrictEditDistance)
	}
	return distance
}

// proofreadScan asks the model about each page and returns the corrections the
// replies make, in the order of the lines they put right.
func proofreadScan(ctx context.Context, ask askModel, pages []Batch, opts textOptions) (textResult, error) {
	instruction := buildInstruction(scanInstruction, opts.strict)
	answers, err := askBatches(ctx, ask, instruction, pages, getScanDistance(opts), opts.maxSuggestions)
	if err != nil {
		return textResult{}, err
	}
	var out textResult
	var lines []Line
	for _, one := range answers {
		if !one.isAnswer {
			out.uncorrected++
		}
		lines = append(lines, one.lines...)
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].Number < lines[j].Number })
	out.corrections = lines
	out.isDone = true
	return out, nil
}

// proofreadSpeech asks the model about the lines of a recording from a line on,
// at most limit of them where that is above zero. The batches are cut by size
// and overlap, and a sentence reaching across a cut is asked about whole in a
// batch over that cut. The next line is where a sentence reaching past the last
// line answered begins, or the lines shared with the end of the window.
func proofreadSpeech(
	ctx context.Context, ask askModel, lines []Line, size, overlap, from, limit int, opts textOptions,
) (textResult, error) {
	spoken := getSpokenLines(lines)
	at := sort.Search(len(spoken), func(i int) bool { return spoken[i].Number >= from })
	if at >= len(spoken) {
		return textResult{next: from, isDone: true}, nil
	}
	end := len(spoken)
	if limit > 0 {
		end = min(end, at+limit)
	}
	window := spoken[at:end:end]

	about := describeSpeech(spoken)
	instruction := buildInstruction(speechInstruction, opts.strict)

	main := setContext(speechBatches(window, size, overlap), about)
	answers, err := askBatches(ctx, ask, instruction, main, 0, opts.maxSuggestions)
	if err != nil {
		return textResult{}, err
	}
	_, past := gatherAnswers(answers)
	across := setContext(cutBatches(window, size, overlap, past), about)
	more, err := askBatches(ctx, ask, instruction, across, 0, opts.maxSuggestions)
	if err != nil {
		return textResult{}, err
	}
	answers = append(answers, more...)

	put, _ := gatherAnswers(answers)
	out := textResult{corrections: sortLines(put)}
	for _, one := range answers {
		if !one.isAnswer {
			out.uncorrected++
		}
	}

	if end == len(spoken) {
		out.next, out.isDone = spoken[end-1].Number+1, true
		return out, nil
	}
	resume := max(end-max(overlap, 0), at+1)
	for _, one := range answers {
		if one.batch.Last() != window[len(window)-1].Number {
			continue
		}
		for _, start := range one.reaching {
			resume = min(resume, max(getLineIndex(spoken, start), at+1))
		}
	}
	out.next = spoken[resume].Number
	return out, nil
}

// getLineIndex is the place of the line numbered number among the lines, or the
// place of the first line after it.
func getLineIndex(lines []Line, number int) int {
	return sort.Search(len(lines), func(i int) bool { return lines[i].Number >= number })
}

// askBatches asks the model about each batch in turn.
func askBatches(
	ctx context.Context, ask askModel, instruction string, batches []Batch, maxDistance float64, maxSuggestions int,
) ([]answer, error) {
	out := make([]answer, 0, len(batches))
	for _, batch := range batches {
		reply, err := ask(ctx, instruction, buildPrompt(batch))
		if err != nil {
			return nil, err
		}
		out = append(out, readAnswer(batch, reply, maxDistance, maxSuggestions))
	}
	return out, nil
}

// setContext returns the batches with what the recording holds on each of them.
func setContext(batches []Batch, about string) []Batch {
	for at := range batches {
		batches[at].Context = about
	}
	return batches
}
