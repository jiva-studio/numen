package domain_test

import (
	"testing"

	"golang.org/x/text/unicode/norm"

	"github.com/jiva-studio/numen/libs/note/domain"
)

// One name each, composed and decomposed. The two forms are computed rather
// than typed out, so an editor that composed this file on the way in leaves the
// tests testing what they say they test.
const (
	hangul = "한글"
	kana   = "がき"
	cafe   = "Café"
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
		// COLLATE NOCASE only touches ASCII, and Å is outside it: it would leave
		// this pair two names.
		{"a ring above", "ÅNGSTRÖM", "ångström"},
		{"Greek", "ΣΙΓΜΑ", "σιγμα"},
		{"a Greek final sigma", "ὈΔΥΣΣΕΎΣ", "ὀδυσσεύς"},
		{"a German sharp s", "STRASSE", "straße"},
		{"a Korean syllable against its jamo", norm.NFC.String(hangul), norm.NFD.String(hangul)},
		{"a Japanese voiced mark", norm.NFC.String(kana), norm.NFD.String(kana)},
		{"a decomposed acute", norm.NFC.String(cafe), norm.NFD.String(cafe)},
		{"case and composition together", "CAFÉ", norm.NFD.String(cafe)},
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
		"İstanbul", "ЭНТРОПИЯ", norm.NFD.String(cafe), norm.NFD.String(hangul), norm.NFD.String(kana),
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

func TestHasPathPrefixMatchesComposedAndDecomposedForms(t *testing.T) {
	composed := norm.NFC.String("Bücher")
	decomposed := norm.NFD.String("Bücher")
	for _, one := range []struct {
		path, prefix string
		want         bool
	}{
		{composed + "/Teebuch.md", composed + "/", true},
		{composed + "/Teebuch.md", decomposed + "/", true},
		{decomposed + "/Teebuch.md", composed + "/", true},
		{decomposed + "/Teebuch.md", decomposed + "/", true},
		{"library/Tea.md", "library/", true},
		{"library2/x", "library/", false},
		{"library/Tea.md", "", true},
		{composed + "/Teebuch.md", "library/", false},
	} {
		if got := domain.HasPathPrefix(one.path, one.prefix); got != one.want {
			t.Errorf("HasPathPrefix(%+q, %+q) = %v, want %v", one.path, one.prefix, got, one.want)
		}
	}
}
