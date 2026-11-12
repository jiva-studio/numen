package chunking

import "testing"

func TestAChunkIsLegibleOnceItsLetterFractionReachesTheThreshold(t *testing.T) {
	cases := []struct {
		name      string
		chunk     string
		threshold float64
		want      bool
	}{
		{"half letters at one half", "ab12", 0.5, true},
		{"half letters above one half", "ab12", 0.51, false},
		{"spaces are not counted", "ab 12", 0.5, true},
		{"newlines and tabs are not counted", "\n\tab\t 12\n", 0.5, true},
		{"all letters at one", "a b c", 1, true},
		{"one symbol below one", "a b #", 1, false},
		{"digits are not letters", "123", 0.1, false},
		{"marks count as letters", "\u0301\u0301", 1, true},
		{"a letter with its mark", "a\u0301", 1, true},
		{"a devanagari word", "हिन्दी", 1, true},
		{"a letter alone", "x", 1, true},
		{"punctuation alone", "?!", 0.1, false},
		{"a negative threshold asks for none", "?!", -1, true},
	}
	for _, c := range cases {
		reads := Legibility{Alphabetic: c.threshold, Dirty: -1}
		if got := isLegible(c.chunk, reads); got != c.want {
			t.Errorf("%s: isLegible(%q, alphabetic %v) = %v, want %v", c.name, c.chunk, c.threshold, got, c.want)
		}
	}
}

func TestAChunkWithNoCharactersIsNeverLegible(t *testing.T) {
	for _, chunk := range []string{"", " ", " \t\n ", "\u00a0\u3000"} {
		for _, reads := range []Legibility{{Alphabetic: -1, Dirty: -1}, {Alphabetic: 0, Dirty: 0}, {Alphabetic: 0.5, Dirty: 0.5}} {
			if isLegible(chunk, reads) {
				t.Errorf("isLegible(%q, %+v) = true", chunk, reads)
			}
		}
	}
}

func TestIsLegible_DirtyWordFractionThreshold(t *testing.T) {
	cases := []struct {
		name  string
		chunk string
		limit float64
		want  bool
	}{
		{"no dirty word at zero", "ab cd", 0, true},
		{"a dirty word above zero", "ab c#d", 0, false},
		{"half dirty at one half", "ab c#d", 0.5, true},
		{"half dirty below one half", "ab c#d", 0.49, false},
		{"one third dirty below the limit", "ab cd e#f", 0.3, false},
		{"one third dirty above the limit", "ab cd e#f", 0.34, true},
		{"one quarter dirty at one quarter", "ab cd ef g#h", 0.25, true},
		{"one quarter dirty below one quarter", "ab cd ef g#h", 0.24, false},
		{"a lone dirty word", "a#b", 0.5, false},
		{"a lone dirty word at one", "a#b", 1, true},
		{"a lone clean word", "ab", 0.5, true},
		{"leading dirty word", "g#h ab cd ef", 0.2, false},
		{"leading dirty word within", "g#h ab cd ef", 0.25, true},
		{"middle dirty word", "ab c#d ef gh", 0.2, false},
		{"last dirty word", "ab cd ef g#h", 0.2, false},
		{"last clean word counts", "c#d ab", 0.5, true},
		{"last clean word counts below", "c#d ab", 0.49, false},
		{"every word dirty", "a#b c#d e#f", 0.9, false},
		{"every word dirty at one", "a#b c#d e#f", 1, true},
		{"surrounding and repeated whitespace", "  ab \t\n  c#d  ", 0.5, true},
		{"surrounding and repeated whitespace below", "  ab \t\n  c#d  ", 0.4, false},
		{"a no-break space separates words", "ab\u00a0c#d", 0.5, true},
		{"a no-break space separates words below", "ab\u00a0c#d", 0.4, false},
		{"an ideographic space separates words", "ab\u3000cd\u3000e#f", 0.3, false},
		{"multi-byte words", "é#é ab", 0.4, false},
		{"multi-byte words within", "é#é ab", 0.5, true},
		{"a negative limit asks for none", "a#b c#d", -1, true},
	}
	for _, c := range cases {
		reads := Legibility{Alphabetic: -1, Dirty: c.limit}
		if got := isLegible(c.chunk, reads); got != c.want {
			t.Errorf("%s: isLegible(%q, dirty %v) = %v, want %v", c.name, c.chunk, c.limit, got, c.want)
		}
	}
}

func TestBothThresholdsMustHold(t *testing.T) {
	cases := []struct {
		name  string
		chunk string
		reads Legibility
		want  bool
	}{
		{"both hold", "ab cd", Legibility{Alphabetic: 1, Dirty: 0}, true},
		{"too few letters", "12 34", Legibility{Alphabetic: 0.5, Dirty: 1}, false},
		{"dirty words at their limit", "a#b c#d", Legibility{Alphabetic: 0.5, Dirty: 1}, true},
		{"too many dirty words and few letters", "a#b c#d", Legibility{Alphabetic: 0.9, Dirty: 0.1}, false},
		{"letters hold, dirty fails", "ab c#d", Legibility{Alphabetic: 0.5, Dirty: 0.1}, false},
	}
	for _, c := range cases {
		if got := isLegible(c.chunk, c.reads); got != c.want {
			t.Errorf("%s: isLegible(%q, %+v) = %v, want %v", c.name, c.chunk, c.reads, got, c.want)
		}
	}
}
