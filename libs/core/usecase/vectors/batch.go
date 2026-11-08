package vectors

import "unicode/utf8"

// defaultBatchCharacters is the budget used when configuration names none.
const defaultBatchCharacters = 32000

func splitBatches(texts []string, maxChars int) [][]string {
	if maxChars <= 0 {
		maxChars = defaultBatchCharacters
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
