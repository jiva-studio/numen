// Package wordform is how a word is matched in another form: the stem a search
// tries after the word itself, and the folding that drops case and diacritics.
// The index, the query and the highlight of a result all use it, so a result is
// marked where the index matched it.
package wordform

import (
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// minStemmed is the shortest word that has a stem. A shorter word is a whole
// word or a prefix only: removing a letter from it leaves a fragment that
// begins too many other words.
const minStemmed = 4

// longStemmed is the length from which the stem drops two letters and not one,
// because the endings of longer Russian, Latin and Sanskrit words (-ami, -asya)
// have two letters or more.
// -u, -ami, -asya) fall off whatever their length.
const longStemmed = 7

// GetStem returns a word without its ending: the word less its last letter, or
// its last two letters from seven letters on. A word shorter than four letters
// has no stem and the result is empty. A combining mark at the cut goes
// with the letter it belongs to.
func GetStem(word string) string {
	runes := []rune(word)
	if len(runes) < minStemmed {
		return ""
	}
	end := len(runes) - 1
	if len(runes) >= longStemmed {
		end--
	}
	for end > 0 && unicode.Is(unicode.Mn, runes[end]) {
		end--
	}
	return string(runes[:end])
}

// FoldRune is a letter without case and without its diacritics, as the index
// holds it: "Ṛ" and "r" are one letter, and so are "й" and "и". A rune with no
// base letter is returned in lower case.
func FoldRune(r rune) rune {
	r = unicode.ToLower(r)
	if r < 0x80 {
		return r
	}
	if base := []rune(norm.NFD.String(string(r)))[0]; !unicode.Is(unicode.Mn, base) {
		return base
	}
	return r
}
