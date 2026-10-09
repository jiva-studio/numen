package chunking

import (
	"unicode"
	"unicode/utf8"
)

// isLegible says whether a chunk reads as text. Recognition that went wrong reads
// as punctuation with letters in it, and it is caught by two fractions: how much
// of the chunk is letters, and how many of its words carry a mark inside them.
//
// Both thresholds are configuration, because where they sit depends on the
// scripts a corpus is written in.
func isLegible(chunk string, reads Legibility) bool {
	if !isAlphabetic(chunk, reads.Alphabetic) {
		return false
	}
	if reads.Dirty < 0 {
		return true
	}
	return isClean(chunk, reads.Dirty)
}

func isAlphabetic(chunk string, threshold float64) bool {
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
	return float64(letters)/float64(characters) >= threshold
}

func isClean(chunk string, dirtyLimit float64) bool {
	words, dirty := countWords(chunk)
	if words == 0 {
		return true
	}
	return float64(dirty)/float64(words) <= dirtyLimit
}

func countWords(chunk string) (words, dirty int) {
	start := -1
	for i, r := range chunk {
		if unicode.IsSpace(r) {
			if start >= 0 {
				words++
				if !isSpelled(chunk[start:i]) {
					dirty++
				}
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		words++
		if !isSpelled(chunk[start:]) {
			dirty++
		}
	}
	return words, dirty
}

// isSpelled says whether one word is written the way words are: letters, the marks
// that belong to them, digits, and the few characters that join a word to
// itself, with anything else only at its ends.
func isSpelled(token string) bool {
	if isASCIIToken(token) {
		return isASCIISpelled(token)
	}
	return isUnicodeSpelled(token)
}

func isASCIIToken(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= utf8.RuneSelf {
			return false
		}
	}
	return true
}

func isASCIISpelled(token string) bool {
	first := -1
	last := -1
	hasSpecial := false
	for i := 0; i < len(token); i++ {
		b := token[i]
		if isASCIILetter(b) || isASCIIDigit(b) {
			if first < 0 {
				first = i
			}
			last = i + 1
		} else {
			hasSpecial = true
		}
	}
	if first < 0 || !hasSpecial {
		return true
	}
	for i := first; i < last; i++ {
		if !isASCIIAllowedInside(token[i]) {
			return false
		}
	}
	return true
}

func isUnicodeSpelled(token string) bool {
	first := -1
	for i, r := range token {
		if isLetterOrDigit(r) {
			first = i
			break
		}
	}
	if first < 0 {
		return true
	}
	last := len(token)
	for last > first {
		r, size := utf8.DecodeLastRuneInString(token[:last])
		if isLetterOrDigit(r) {
			break
		}
		last -= size
	}
	for _, r := range token[first:last] {
		if !isAllowedInside(r) {
			return false
		}
	}
	return true
}

func isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isASCIIDigit(b byte) bool {
	return b >= '0' && b <= '9'
}

func isASCIIAllowedInside(b byte) bool {
	return isASCIILetter(b) || isASCIIDigit(b) || b == '\'' || b == '-' || b == '.'
}

func isLetterOrDigit(r rune) bool {
	if r < utf8.RuneSelf {
		b := byte(r)
		return isASCIILetter(b) || isASCIIDigit(b)
	}
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func isAllowedInside(r rune) bool {
	switch {
	case unicode.IsLetter(r), unicode.IsMark(r), unicode.IsDigit(r):
		return true
	case r == '\'', r == '’', r == '-', r == '‑', r == '.':
		return true
	default:
		return false
	}
}
