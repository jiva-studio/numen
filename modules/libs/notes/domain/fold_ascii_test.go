package domain

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTheASCIIFoldIsTheGeneralFoldForEveryASCIIByte(t *testing.T) {
	for c := 0; c < utf8.RuneSelf; c++ {
		name := string([]byte{byte(c)})
		if got, want := FoldName(name), foldGeneral(name); got != want {
			t.Errorf("FoldName(%q) = %q, want %q", name, got, want)
		}
		wrapped := "a" + name + "Z" + name
		if got, want := FoldName(wrapped), foldGeneral(wrapped); got != want {
			t.Errorf("FoldName(%q) = %q, want %q", wrapped, got, want)
		}
	}
}

func TestTheASCIIFoldIsTheGeneralFoldForEveryASCIIPair(t *testing.T) {
	for a := 0; a < utf8.RuneSelf; a++ {
		for b := 0; b < utf8.RuneSelf; b++ {
			name := string([]byte{byte(a), byte(b)})
			if got, want := FoldName(name), foldGeneral(name); got != want {
				t.Fatalf("FoldName(%q) = %q, want %q", name, got, want)
			}
		}
	}
}

func TestTheFoldIsTheGeneralFoldForNamesBeyondASCII(t *testing.T) {
	names := []string{
		"",
		"a",
		"A",
		"ß",
		"İ",
		"İstanbul",
		"Σ",
		"ΣΑΣ",
		"K",
		"Kelvin",
		"Kelvin K",
		"Noté",
		"noté Ab",
		"é",
		"é",
		"É",
		"Ｎｏｔｅ",
		"Śrīmad Bhāgavatam",
		"Ab\u0000Cd",
		"abc\xff",
		"ABC\xc3",
		strings.Repeat("Ab", 100) + "ß",
	}
	for _, name := range names {
		if got, want := FoldName(name), foldGeneral(name); got != want {
			t.Errorf("FoldName(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestAnASCIINameAlreadyFoldedIsReturnedWithoutAllocating(t *testing.T) {
	name := strings.Repeat("folded name-1_", 4)
	allocs := testing.AllocsPerRun(100, func() { _ = FoldName(name) })
	if allocs != 0 {
		t.Errorf("FoldName allocated %v times for a folded ASCII name", allocs)
	}
	if got := FoldName(name); got != name {
		t.Errorf("FoldName(%q) = %q", name, got)
	}
}
