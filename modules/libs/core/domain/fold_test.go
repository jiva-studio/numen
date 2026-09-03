package domain_test

import (
	"testing"

	"golang.org/x/text/unicode/norm"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// One name each, composed and decomposed. The two forms are computed rather
// than typed out, so an editor that composed this file on the way in leaves the
// tests testing what they say they test.
var (
	hangulComposed   = norm.NFC.String("한글")
	hangulDecomposed = norm.NFD.String("한글")
	kanaComposed     = norm.NFC.String("がき")
	kanaDecomposed   = norm.NFD.String("がき")
	cafeComposed     = norm.NFC.String("Café")
	cafeDecomposed   = norm.NFD.String("Café")
)

// Two names are one name under the fold, across the scripts a vault is written
// in. The case rows and the composition rows are both here: a script that has
// no case is carried by the normal form alone.
func TestNamesThatAreOneName(t *testing.T) {
	for _, one := range []struct {
		what string
		a, b string
	}{
		{"Cyrillic", "ЭНТРОПИЯ", "энтропия"},
		{"an acute", "CAFÉ", "café"},
		{"Greek", "ΣΙΓΜΑ", "σιγμα"},
		{"a Greek final sigma", "ὈΔΥΣΣΕΎΣ", "ὀδυσσεύς"},
		{"a German sharp s", "STRASSE", "straße"},
		{"a Korean syllable against its jamo", hangulComposed, hangulDecomposed},
		{"a Japanese voiced mark", kanaComposed, kanaDecomposed},
		{"a decomposed acute", cafeComposed, cafeDecomposed},
		{"case and composition together", "CAFÉ", cafeDecomposed},
	} {
		if one.a == one.b {
			t.Fatalf("%s: the two sides are the same bytes, so nothing is tested", one.what)
		}
		if domain.FoldName(one.a) != domain.FoldName(one.b) {
			t.Errorf("%s: %q and %q are two names, folded %q and %q",
				one.what, one.a, one.b, domain.FoldName(one.a), domain.FoldName(one.b))
		}
	}
}

// Names the fold keeps apart, each for its own reason.
func TestNamesThatAreTwoNames(t *testing.T) {
	for _, two := range []struct {
		what string
		a, b string
	}{
		// The fold is the one every reader gets, and a dotted capital I is a
		// letter of its own outside Turkish.
		{"a Turkish dotted I", "İSTANBUL", "istanbul"},
		// Two filenames on a disk, so two names here.
		{"a full-width word", "Ｎｏｔｅ", "Note"},
		{"two words", "Энтропия", "Энергия"},
	} {
		if domain.FoldName(two.a) == domain.FoldName(two.b) {
			t.Errorf("%s: %q and %q became one name, %q",
				two.what, two.a, two.b, domain.FoldName(two.a))
		}
	}
}

// Folding decomposes some letters, so the key is composed after it and folding
// a key again gives the key back.
func TestTheKeyIsComposed(t *testing.T) {
	for _, name := range []string{
		"İstanbul", "ЭНТРОПИЯ", cafeDecomposed, hangulDecomposed, kanaDecomposed,
	} {
		key := domain.FoldName(name)
		if !norm.NFC.IsNormalString(key) {
			t.Errorf("the key for %q is %q, which is not composed", name, key)
		}
		if again := domain.FoldName(key); again != key {
			t.Errorf("folding the key for %q again gave %q", name, again)
		}
	}
}
