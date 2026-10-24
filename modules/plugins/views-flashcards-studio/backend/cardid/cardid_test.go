package cardid_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/plugins/views-flashcards-studio/backend/cardid"
)

func TestAnIdentifierIsTenCharactersOfTheAlphabet(t *testing.T) {
	const alphabet = "0123456789abcdefghjkmnpqrstvwxyz"

	minted := cardid.New()
	if len(minted) != cardid.Length {
		t.Errorf("id = %q, %d characters, want %d", minted, len(minted), cardid.Length)
	}
	for _, c := range minted {
		if !strings.ContainsRune(alphabet, c) {
			t.Errorf("id = %q, and %q is outside the alphabet", minted, c)
		}
	}
	if !cardid.IsValid(minted) {
		t.Errorf("a minted identifier is not one: %q", minted)
	}
}

// An identifier says which card this is, so two of them are two cards.
func TestTwoIdentifiersDiffer(t *testing.T) {
	observed := map[cardid.ID]bool{}
	for range 100 {
		minted := cardid.New()
		if observed[minted] {
			t.Fatalf("%q was minted twice", minted)
		}
		observed[minted] = true
	}
}

// An identifier is read as one only at that length and in that alphabet.
// Anything else is not one, and the card carrying it is given one.
func TestWhatIsNotAnIdentifier(t *testing.T) {
	for name, s := range map[string]cardid.ID{
		"nothing":       "",
		"too short":     "k7m2xq9fz",
		"too long":      "k7m2xq9fzpp",
		"upper case":    "K7M2XQ9FZP",
		"the letter i":  "k7m2xqifzp",
		"the letter l":  "k7m2xqlfzp",
		"the letter o":  "k7m2xqofzp",
		"the letter u":  "k7m2xqufzp",
		"a caret":       "^k7m2xq9fz",
		"a space":       "k7m2xq9f p",
		"beyond latin":  "к7m2xq9fzp",
		"a hyphen":      "k7m2-xq9fz",
		"a full stop":   "k7m2xq9fz.",
		"an apostrophe": "k7m2xq9fz'",
	} {
		t.Run(name, func(t *testing.T) {
			if cardid.IsValid(s) {
				t.Errorf("%q reads as an identifier", s)
			}
		})
	}
}

func TestIsValidRejectsBytesOutsideTheAlphabet(t *testing.T) {
	for name, s := range map[string]string{
		"upper case": "012345678A",
		"letter i":   "012345678i",
		"letter u":   "012345678u",
		"non-ascii":  "01234567é",
		"high byte":  "012345678\xff",
		"nul byte":   "012345678\x00",
		"too long":   "01234567890",
		"too short":  "012345678",
	} {
		if cardid.IsValid(cardid.ID(s)) {
			t.Errorf("%s: %q validated", name, s)
		}
	}
}

// Every character of the alphabet can be minted, in every position.
func TestEveryCharacterOfTheAlphabetIsMinted(t *testing.T) {
	const alphabet = "0123456789abcdefghjkmnpqrstvwxyz"

	observed := map[rune]bool{}
	for range 3000 {
		minted := cardid.New()
		for _, c := range minted {
			observed[c] = true
		}
	}
	for _, c := range alphabet {
		if !observed[c] {
			t.Errorf("%q was never minted", c)
		}
	}
}
