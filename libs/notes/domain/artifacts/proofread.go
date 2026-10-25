package artifacts

import (
	"cmp"
	"slices"
)

// A ProofreadLine is one numbered run of words a proofreader is asked about:
// the number the line is known by, and what it says.
type ProofreadLine struct {
	Number int
	Text   string
}

// A ProofreadPage is the lines of one printed page, answered for as a whole.
type ProofreadPage struct {
	Number int
	Lines  []ProofreadLine
}

// A ProofreadCorrection puts the line Number right. A correction of several
// lines puts them together as one, and Last is the final line of the run;
// otherwise Last is zero or Number.
type ProofreadCorrection struct {
	Number int
	Last   int
	Text   string
}

// IsJoined says whether the correction puts more than one line together.
func (c ProofreadCorrection) IsJoined() bool { return c.Last > c.Number }

// FindUnaskedCorrection returns the first correction whose line is none of the
// lines asked about, or whose run reaches past the last of them, and false
// where every correction stands within the lines asked about.
func FindUnaskedCorrection(lines []ProofreadLine, made []ProofreadCorrection) (ProofreadCorrection, bool) {
	numbers := make(map[int]bool, len(lines))
	last := -1
	for _, line := range lines {
		numbers[line.Number] = true
		last = max(last, line.Number)
	}
	for _, one := range made {
		if !numbers[one.Number] || one.Last > last {
			return one, true
		}
	}
	return ProofreadCorrection{}, false
}

// A ScanProofreading is what a proofreader put right over a run of pages.
type ScanProofreading struct {
	Corrections []ProofreadCorrection
	// Uncorrected is how many pages were replied to with something that is no
	// correction. Those pages stand as they were read.
	Uncorrected int
}

// A SpeechWindow is the part of a transcript a proofreader answers in one call,
// and how it cuts that part into requests to the model.
type SpeechWindow struct {
	// Size is how many lines one request holds, and Overlap how many of them the
	// request before it also held.
	Size, Overlap int
	// From is the number of the first line answered.
	From int
	// Limit is the most lines answered. Zero answers every line from From.
	Limit int
}

// GetLines returns the lines of a transcript the window answers for: those
// numbered From or later, no more than Limit of them. The lines stand in the
// order of their numbers.
func (w SpeechWindow) GetLines(lines []ProofreadLine) []ProofreadLine {
	idx, _ := slices.BinarySearchFunc(lines, w.From, func(line ProofreadLine, from int) int {
		return cmp.Compare(line.Number, from)
	})
	end := len(lines)
	if w.Limit > 0 {
		end = min(end, idx+w.Limit)
	}
	return lines[idx:end]
}

// A SpeechProofreading is what a proofreader put right over a window of a
// transcript.
type SpeechProofreading struct {
	Corrections []ProofreadCorrection
	// Uncorrected is how many requests to the model were replied to with
	// something that is no correction. Those lines stand as they were heard.
	Uncorrected int
	// Next is the number of the line to continue from, and IsDone says that no
	// line is left after the window.
	Next   int
	IsDone bool
}
