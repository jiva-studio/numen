// Package proofread asks about the lines a model read or heard in one batch,
// and decides whether what came back is an answer to that question.
//
// It is pure: no network, no clock, no model. What was asked and what came back
// are text, and every rule about them is here, so the adapter carries a request
// and nothing else.
package proofread

import (
	"strconv"
	"strings"
)

// A Line is one run of words the recogniser read in one go: the number it is
// known by, and what it says.
//
// A correction may cover a run of lines, and Last is the last of them. A line
// standing on its own has Last equal to Number, or left at zero where nothing
// put a run together.
type Line struct {
	Number int
	Last   int
	Text   string
}

// Joins says whether this correction puts more than one line together.
func (l Line) Joins() bool { return l.Last > l.Number }

// A Batch is the lines one reply is accepted or refused as a whole: the number
// it is known by, and its lines in the order they are read.
type Batch struct {
	Number int
	Lines  []Line

	// Joinable is whether a reply may answer for a run of these lines as one.
	// Where it does not, a run refuses the batch.
	Joinable bool

	// Context is what the whole text holds, said in its own words. It stands
	// before the lines in the question and is answered for by nothing.
	Context string
}

// Last is the number of the final line a batch carries, and nothing below the
// first line number for a batch carrying none.
func (b Batch) Last() int {
	if len(b.Lines) == 0 {
		return -1
	}
	return b.Lines[len(b.Lines)-1].Number
}

// A line's number is written between marks no recogniser can produce, so a mark
// in a reply is a mark of ours coming back.
const (
	Opens  = "⟦"
	Closes = "⟧"
)

// ScanInstruction is what the proofreader is told it is doing over a scan. It
// is part of what produced a correction, as the model's name is.
const ScanInstruction = `You are proofreading text a machine read off the pages of a printed book.

The text is one page as prose, with every printed line numbered: ` + Opens + `12` + Closes + ` opens the
line numbered 12, and that line runs to the next mark.

Answer with the lines you would put right, one to a line:

12|the line, put right

- Put right what the machine misread: letters, diacritics, words run together
  or broken apart, marks that are not words.
- Every word stays in the line it is in. Nothing moves from one line to
  another, and nothing is added that the page does not print.
- A line is answered for once.
- Do not write ` + Opens + ` or ` + Closes + ` in your answer.
- Do not translate, rephrase, repunctuate or improve a line that was read
  correctly.
- A line you would leave alone is a line you do not answer with. A page you
  would leave alone is an empty answer.`

// SpeechInstruction is what the proofreader is told it is doing over a
// transcript. It is part of what produced a correction, as the model's name is.
const SpeechInstruction = `You are proofreading text a machine heard in a recording.

The text is speech as prose, with every stretch of it numbered: ` + Opens + `12` + Closes + ` opens the
line numbered 12, and that line runs to the next mark.

What the recording holds stands before the first mark: how the speech opens,
and the words that recur through it as the machine heard them. It is there to
be read and is answered for by nothing.

Answer with the lines you would put right, one to a line:

12|the line, put right

A sentence broken across a run of lines is answered for as one line, written
as the first and the last of them:

12-14|the whole sentence, put right

- Put right what the machine misheard: a word for one that sounds like it,
  words run together or split apart, punctuation and capitalisation that are
  missing or wrong, numbers and names.
- Put a sentence broken across lines back together as one line. The run is
  every line it covers, from first to last, with none left out.
- A line where one sentence ends and the next begins stands in the run of
  both, and that run is answered with every sentence it covers.
- A line is answered for once: two runs never share a line, and a run says
  everything its lines say.
- A word listed as recurring is one the recording keeps coming back to. Put it
  right or leave it as it is, and do the same with it every time it is said.
- Nothing is added that was not said, and no word moves to a line outside the
  run it is answered in.
- Do not write ` + Opens + ` or ` + Closes + ` in your answer.
- Do not translate, rephrase, summarise or improve speech that was heard
  correctly. Every hesitation, repetition and false start that was spoken
  stays.
- A line you would leave alone is a line you do not answer with. Speech you
  would leave alone is an empty answer.`

// Ask is one batch as the proofreader is given it: its prose, with each line
// opened by its number.
//
// The lines are run together as prose: a line ending mid-word is finished by
// the next one. What the text is about stands before the first of them.
func Ask(batch Batch) string {
	var out strings.Builder
	if batch.Context != "" {
		out.WriteString(batch.Context)
		out.WriteString("\n\n")
	}
	for i, line := range batch.Lines {
		if i > 0 {
			out.WriteString(" ")
		}
		out.WriteString(Opens)
		out.WriteString(strconv.Itoa(line.Number))
		out.WriteString(Closes)
		out.WriteString(line.Text)
	}
	return out.String()
}
