package main

import (
	"strconv"
	"strings"
)

// StrictEditDistance is how far a correction may stand from the line as read
// when strict proofreading is on.
const StrictEditDistance = 0.50

const strictRule = "\n\nEnforce grammar, spelling and punctuation strictly: put right every error you find, whatever kind it is."

type reviewOptions struct {
	strict         bool
	maxSuggestions int
}

// buildInstruction is the instruction the model is given. Strict proofreading
// adds the rule that every error is put right.
func buildInstruction(instruction string, strict bool) string {
	if strict {
		return instruction + strictRule
	}
	return instruction
}

// readPrompt returns the numbered lines of a prompt.
func readPrompt(prompt string) []Line {
	parts := strings.Split(prompt, Opens)
	var lines []Line
	for _, part := range parts[1:] {
		digits, text, found := strings.Cut(part, Closes)
		if !found {
			continue
		}
		number, err := strconv.Atoi(digits)
		if err != nil {
			continue
		}
		lines = append(lines, Line{Number: number, Text: strings.TrimSpace(text)})
	}
	return lines
}

// reviewReply returns the rows of a model reply that answer the prompt: the
// corrections the gates accept, at most maxSuggestions of them where that is
// above zero, and every run reaching past the last line of the prompt. A reply
// the gates refuse is empty.
func reviewReply(prompt, reply string, options reviewOptions) string {
	batch := Batch{Lines: readPrompt(prompt), IsJoinable: true}
	distance := MaxEditDistance
	if options.strict {
		distance = StrictEditDistance
	}
	fixed, past, ok := GetFixedLines(batch, reply, distance)
	if !ok {
		return ""
	}
	if options.maxSuggestions > 0 && len(fixed) > options.maxSuggestions {
		fixed = fixed[:options.maxSuggestions]
	}

	var rows []string
	for _, line := range fixed {
		rows = append(rows, formatRow(line))
	}
	if past {
		rows = append(rows, readPastRows(batch, reply)...)
	}
	return strings.Join(rows, "\n")
}

func formatRow(line Line) string {
	if line.IsJoined() {
		return strconv.Itoa(line.Number) + "-" + strconv.Itoa(line.Last) + "|" + line.Text
	}
	return strconv.Itoa(line.Number) + "|" + line.Text
}

// readPastRows returns the rows of a reply whose run ends after the last line
// of the batch.
func readPastRows(batch Batch, reply string) []string {
	var rows []string
	for _, row := range strings.Split(stripFence(reply), "\n") {
		row = strings.TrimSpace(row)
		if _, through, _, _, ok := readRow(row); ok && through > batch.Last() {
			rows = append(rows, row)
		}
	}
	return rows
}
