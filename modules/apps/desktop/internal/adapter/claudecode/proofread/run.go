package proofread

import (
	"context"
	"slices"
	"sort"
	"sync"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// Model returns the model's reply to a prompt under an instruction.
type Model func(ctx context.Context, instruction, prompt string) (string, error)

// Scan asks about each page, all of them at once, and returns the corrections the
// replies make in the order of the lines they put right. A page whose reply is
// no answer is counted and left as it was read.
func Scan(ctx context.Context, ask Model, pages []domain.ProofreadGroup, maxDistance float64) (domain.ProofreadAnswer, error) {
	batches := make([]Batch, len(pages))
	for at, page := range pages {
		batches[at] = Batch{Number: page.Number, Lines: getLines(page.Lines)}
	}
	if maxDistance <= 0 {
		maxDistance = MaxEditDistance
	}
	replies, err := askBatches(ctx, ask, ScanInstruction, batches)
	if err != nil {
		return domain.ProofreadAnswer{}, err
	}
	answer := domain.ProofreadAnswer{IsDone: true}
	for _, batch := range batches {
		lines, _, ok := GetFixedLines(batch, replies[batch.Number], maxDistance)
		if !ok {
			answer.Uncorrected++
			continue
		}
		answer.Corrections = append(answer.Corrections, getCorrections(lines)...)
	}
	sortCorrections(answer.Corrections)
	return answer, nil
}

// Speech asks about the batches of a transcript from the one the request begins
// at, as many as its line limit holds, and returns the corrections the replies
// make. A sentence reaching across a cut between two batches is asked about
// whole in a batch over that cut. The next line is where the batch after the
// last one asked begins.
func Speech(ctx context.Context, ask Model, request domain.ProofreadRequest) (domain.ProofreadAnswer, error) {
	cues := getCues(request.Groups)
	about := Describe(cues)
	batches := setContext(GetSpeechBatches(cues, request.Size, request.Overlap), about)
	start := getFirstBatch(batches, request.FromLine)
	if start >= len(batches) {
		return domain.ProofreadAnswer{NextLine: request.FromLine, IsDone: true}, nil
	}
	end := len(batches)
	if request.LineLimit > 0 {
		end = min(end, start+max(request.LineLimit/request.Size, 1))
	}
	group := batches[start:end]

	replies, err := askBatches(ctx, ask, SpeechInstruction, group)
	if err != nil {
		return domain.ProofreadAnswer{}, err
	}
	put, past := GetGatheredLines(group, replies, 0)
	answer := domain.ProofreadAnswer{Uncorrected: countUncorrected(group, replies)}

	across := setContext(CutIntoBatches(cues, request.Size, request.Overlap, past), about)
	if len(across) > 0 {
		more, err := askBatches(ctx, ask, SpeechInstruction, across)
		if err != nil {
			return domain.ProofreadAnswer{}, err
		}
		again, _ := GetGatheredLines(across, more, 0)
		for number, line := range again {
			put[number] = line
		}
		answer.Uncorrected += countUncorrected(across, more)
	}
	for _, line := range put {
		answer.Corrections = append(answer.Corrections, getCorrections([]Line{line})...)
	}
	sortCorrections(answer.Corrections)

	if end == len(batches) {
		answer.NextLine, answer.IsDone = batches[end-1].Last()+1, true
		return answer, nil
	}
	answer.NextLine = batches[end].Lines[0].Number
	return answer, nil
}

// getFirstBatch is the place of the batch a request from a line begins at: the
// batch opening on that line, else the first batch reaching it.
func getFirstBatch(batches []Batch, from int) int {
	reaching := len(batches)
	for at, batch := range batches {
		switch {
		case batch.Lines[0].Number == from:
			return at
		case batch.Last() >= from && reaching == len(batches):
			reaching = at
		}
	}
	return reaching
}

// countUncorrected is how many of the batches were answered with what is no
// correction.
func countUncorrected(batches []Batch, replies map[int]string) int {
	out := 0
	for _, batch := range batches {
		reply, answered := replies[batch.Number]
		if !answered {
			continue
		}
		if _, _, ok := GetFixedLines(batch, reply, 0); !ok {
			out++
		}
	}
	return out
}

// askBatches asks about every batch at once, and returns the replies by the number
// of the batch. The first failure ends the run.
func askBatches(ctx context.Context, ask Model, instruction string, batches []Batch) (map[int]string, error) {
	ctx, stop := context.WithCancel(ctx)
	defer stop()

	var (
		mu      sync.Mutex
		wg      sync.WaitGroup
		failed  error
		replies = make(map[int]string, len(batches))
	)
	for _, batch := range batches {
		wg.Go(func() {
			reply, err := ask(ctx, instruction, Ask(batch))
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if failed == nil {
					failed = err
					stop()
				}
				return
			}
			replies[batch.Number] = reply
		})
	}
	wg.Wait()
	if failed != nil {
		return nil, failed
	}
	return replies, nil
}

// getLines returns the lines of a group as this package knows them.
func getLines(lines []domain.ProofreadLine) []Line {
	out := make([]Line, len(lines))
	for at, line := range lines {
		out[at] = Line{Number: line.Number, Text: line.Text}
	}
	return out
}

// getCues returns the lines of a request as the cues of a transcript, each at
// the place its number gives it and the places between them saying nothing.
func getCues(groups []domain.ProofreadGroup) []domain.TranscriptCue {
	var cues []domain.TranscriptCue
	for _, group := range groups {
		for _, line := range group.Lines {
			if line.Number >= len(cues) {
				cues = slices.Grow(cues, line.Number+1-len(cues))[:line.Number+1]
			}
			cues[line.Number].Text = line.Text
		}
	}
	return cues
}

// getCorrections returns the lines put right as the core states them.
func getCorrections(lines []Line) []domain.ProofreadCorrection {
	out := make([]domain.ProofreadCorrection, len(lines))
	for at, line := range lines {
		out[at] = domain.ProofreadCorrection{Number: line.Number, Last: line.Last, Text: line.Text}
	}
	return out
}

// sortCorrections puts the corrections in the order of the lines they put right.
func sortCorrections(corrections []domain.ProofreadCorrection) {
	sort.SliceStable(corrections, func(i, j int) bool { return corrections[i].Number < corrections[j].Number })
}

// setContext is the batches with what the recording holds on each of them.
func setContext(batches []Batch, about string) []Batch {
	for at := range batches {
		batches[at].Context = about
	}
	return batches
}
