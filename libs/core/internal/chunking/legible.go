package chunking

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

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
	for _, word := range strings.FieldsFunc(chunk, unicode.IsSpace) {
		words++
		if !isSpelled(word) {
			dirty++
		}
	}
	return words, dirty
}

func isSpelled(token string) bool {
	if isASCIIToken(token) {
		return isASCIISpelled(token)
	}
	return isUnicodeSpelled(token)
}

func isASCIIToken(s string) bool {
	for i := range len(s) {
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
	for i := range len(token) {
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
	case isCJKPunctuation(r):
		return true
	default:
		return false
	}
}

func isCJKPunctuation(r rune) bool {
	return (r >= 0x3000 && r <= 0x303F) || (r >= 0xFF00 && r <= 0xFFEF)
}
