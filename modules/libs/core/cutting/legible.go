package cutting

import (
	"strings"
	"unicode"
)

// legible says whether a chunk reads as text. Recognition that went wrong reads
// as punctuation with letters in it, and it is caught by two fractions: how much
// of the chunk is letters, and how many of its words carry a mark inside them.
//
// Both thresholds are configuration, because where they sit depends on the
// scripts a corpus is written in.
func legible(chunk string, s Sizes) bool {
	letters, characters := 0, 0
	for _, r := range chunk {
		if unicode.IsSpace(r) {
			continue
		}
		characters++
		if unicode.IsLetter(r) || unicode.IsMark(r) {
			letters++
		}
	}
	if characters == 0 {
		return false
	}
	if float64(letters)/float64(characters) < s.Alphabetic {
		return false
	}

	if s.Dirty < 0 {
		return true
	}
	words, dirty := 0, 0
	for _, token := range strings.Fields(chunk) {
		words++
		if !spelled(token) {
			dirty++
		}
	}
	return float64(dirty)/float64(words) <= s.Dirty
}

// spelled says whether one word is written the way words are: letters, the marks
// that belong to them, digits, and the few characters that join a word to
// itself, with anything else only at its ends.
func spelled(token string) bool {
	inside := strings.TrimFunc(token, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for _, r := range inside {
		switch {
		case unicode.IsLetter(r), unicode.IsMark(r), unicode.IsDigit(r):
		case r == '\'', r == '’', r == '-', r == '‑', r == '.':
		default:
			return false
		}
	}
	return true
}
