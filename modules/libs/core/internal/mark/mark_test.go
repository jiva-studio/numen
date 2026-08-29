package mark_test

import (
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/mark"
)

func TestAMarkIsTenCharactersOfTheAlphabet(t *testing.T) {
	const alphabet = "0123456789abcdefghjkmnpqrstvwxyz"

	minted, err := mark.New()
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if len(minted) != mark.Length {
		t.Errorf("mark = %q, %d characters, want %d", minted, len(minted), mark.Length)
	}
	for _, c := range minted {
		if !strings.ContainsRune(alphabet, c) {
			t.Errorf("mark = %q, and %q is outside the alphabet", minted, c)
		}
	}
	if !mark.Valid(minted) {
		t.Errorf("a minted mark is not a mark: %q", minted)
	}
}

// A mark says which card this is, so two of them are two cards.
func TestTwoMarksDiffer(t *testing.T) {
	seen := map[string]bool{}
	for range 100 {
		minted, err := mark.New()
		if err != nil {
			t.Fatalf("mint: %v", err)
		}
		if seen[minted] {
			t.Fatalf("%q was minted twice", minted)
		}
		seen[minted] = true
	}
}

// A mark is read as one only at that length and in that alphabet. Anything else
// is not a mark, and the card carrying it is given one.
func TestWhatIsNotAMark(t *testing.T) {
	for name, s := range map[string]string{
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
			if mark.Valid(s) {
				t.Errorf("%q reads as a mark", s)
			}
		})
	}
}
