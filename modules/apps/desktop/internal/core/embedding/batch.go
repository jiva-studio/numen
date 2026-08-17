package embedding

import "unicode/utf8"

// DefaultBatchCharacters is the budget used when configuration names none.
const DefaultBatchCharacters = 32000

// Batches groups texts into requests, bounded by the characters of the whole
// group. A model bounds a request by everything in it, and a count of texts says
// nothing about their size: the same number of windows carries several times the
// tokens in transliterated Sanskrit that it does in English.
//
// The groups are subslices, in order, so their answers concatenate into an
// answer for texts. A text longer than the budget forms a group of one;
// shortening it is the chunker's decision.
func Batches(texts []string, maxChars int) [][]string {
	if maxChars <= 0 {
		maxChars = DefaultBatchCharacters
	}
	var batches [][]string
	start, chars := 0, 0
	for i, text := range texts {
		n := utf8.RuneCountInString(text)
		if i > start && chars+n > maxChars {
			batches = append(batches, texts[start:i])
			start, chars = i, 0
		}
		chars += n
	}
	if start < len(texts) {
		batches = append(batches, texts[start:])
	}
	return batches
}
