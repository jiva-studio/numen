package vector

import "unicode/utf8"

// DefaultBatchCharacters is the budget used when configuration names none.
const DefaultBatchCharacters = 32000

// SplitBatches groups texts in order into batches of at most maxChars characters, never splitting a text.
func SplitBatches(texts []string, maxChars int) [][]string {
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
