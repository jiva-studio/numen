package proofread

import (
	"context"
	"sort"

	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/numbered"
)

// textResult is what proofreading a request came to.
type textResult struct {
	corrections []numbered.Line
	uncorrected int
	next        int
	isDone      bool
}

func getScanDistance(maxDistance float64) float64 {
	if maxDistance <= 0 {
		return numbered.MaxEditDistance
	}
	return maxDistance
}

func proofreadScan(ctx context.Context, ask AskModel, instruction string, pages []numbered.Batch, maxDistance float64) (textResult, error) {
	answers, err := askBatches(ctx, ask, numbered.ScanInstruction, pages, getScanDistance(maxDistance))
	if err != nil {
		return textResult{}, err
	}
	var out textResult
	var lines []numbered.Line
	for _, one := range answers {
		if !one.IsAnswer {
			out.uncorrected++
		}
		lines = append(lines, one.Lines...)
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].Number < lines[j].Number })
	out.corrections = lines
	out.isDone = true
	return out, nil
}

// batchShape is how the lines are cut into batches: the lines in a batch and those it shares with the next.
type batchShape struct {
	size    int
	overlap int
}

func proofreadSpeech(
	ctx context.Context, ask AskModel, instruction string, lines []numbered.Line, shape batchShape, from, limit int,
) (textResult, error) {
	size, overlap := shape.size, shape.overlap
	cues := numbered.GetSpokenLines(lines)
	pos := sort.Search(len(cues), func(i int) bool { return cues[i].Number >= from })
	if pos >= len(cues) {
		return textResult{corrections: nil, uncorrected: 0, next: from, isDone: true}, nil
	}
	end := len(cues)
	if limit > 0 {
		end = min(end, pos+limit)
	}
	window := cues[pos:end:end]

	about := numbered.DescribeSpeech(cues)

	main := setContext(numbered.SpeechBatches(window, size, overlap), about)
	answers, err := askBatches(ctx, ask, numbered.SpeechInstruction, main, 0)
	if err != nil {
		return textResult{}, err
	}
	_, past := numbered.GatherAnswers(answers)
	batchedContext := setContext(numbered.CutBatches(window, size, overlap, past), about)
	more, err := askBatches(ctx, ask, numbered.SpeechInstruction, batchedContext, 0)
	if err != nil {
		return textResult{}, err
	}
	answers = append(answers, more...)

	placed, _ := numbered.GatherAnswers(answers)
	out := textResult{corrections: numbered.SortLines(placed), uncorrected: 0, next: 0, isDone: false}
	for _, one := range answers {
		if !one.IsAnswer {
			out.uncorrected++
		}
	}

	if end == len(cues) {
		out.next, out.isDone = cues[end-1].Number+1, true
		return out, nil
	}
	resume := max(end-max(overlap, 0), pos+1)
	for _, one := range answers {
		if one.Batch.Last() != window[len(window)-1].Number {
			continue
		}
		for _, start := range one.Reaching {
			resume = min(resume, max(getLineIndex(cues, start), pos+1))
		}
	}
	out.next = cues[resume].Number
	return out, nil
}

func getLineIndex(lines []numbered.Line, number int) int {
	return sort.Search(len(lines), func(i int) bool { return lines[i].Number >= number })
}

func askBatches(
	ctx context.Context, ask AskModel, instruction string, batches []numbered.Batch, maxDistance float64,
) ([]numbered.Answer, error) {
	out := make([]numbered.Answer, 0, len(batches))
	for _, batch := range batches {
		text, err := ask(ctx, instruction, numbered.BuildPrompt(batch))
		if err != nil {
			return nil, err
		}
		out = append(out, numbered.ReadAnswer(batch, text, maxDistance))
	}
	return out, nil
}

func setContext(batches []numbered.Batch, about string) []numbered.Batch {
	for item := range batches {
		batches[item].Context = about
	}
	return batches
}
