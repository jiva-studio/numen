// Package proofread asks about the lines a model read on one page, and decides
// whether what came back is an answer to that question.
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
type Line struct {
	At   int
	Text string
}

// A Page is the lines of one page, in the order they are read.
type Page struct {
	At    int
	Lines []Line
}

// A line's number is written between marks no recogniser can produce, so a mark
// in a reply is a mark of ours coming back.
const (
	Opens  = "⟦"
	Closes = "⟧"
)

// Instruction is what the proofreader is told it is doing. It is part of what
// produced a correction, as the model's name is.
const Instruction = `You are proofreading text a machine read off the pages of a printed book.

The text is one page as prose, with every printed line numbered: ` + Opens + `12` + Closes + ` opens the
line numbered 12, and that line runs to the next mark.

Answer with the lines you would put right, one to a line:

12|the line, put right

- Put right what the machine misread: letters, diacritics, words run together
  or broken apart, marks that are not words.
- Every word stays in the line it is in. Nothing moves from one line to
  another, and nothing is added that the page does not print.
- Do not write ` + Opens + ` or ` + Closes + ` in your answer.
- Do not translate, rephrase, repunctuate or improve a line that was read
  correctly.
- A line you would leave alone is a line you do not answer with. A page you
  would leave alone is an empty answer.`

// Ask is one page as the proofreader is given it: its prose, with each printed
// line opened by its number.
//
// The lines are run together as prose: a line ending mid-word is finished by
// the next one.
func Ask(page Page) string {
	var out strings.Builder
	for i, line := range page.Lines {
		if i > 0 {
			out.WriteString(" ")
		}
		out.WriteString(Opens)
		out.WriteString(strconv.Itoa(line.At))
		out.WriteString(Closes)
		out.WriteString(line.Text)
	}
	return out.String()
}
