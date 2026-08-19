package webui

import (
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

func TestWhereTheWordsTypedStandInAPassage(t *testing.T) {
	for _, c := range []struct {
		name  string
		text  string
		query string
		want  []domain.Span
	}{
		{name: "nothing typed", text: "no engine is more efficient", query: "  "},
		{name: "nothing to read", text: "", query: "engine"},
		{
			name:  "one word, wherever it stands",
			text:  "no engine beats a reversible engine",
			query: "engine",
			want:  []domain.Span{{From: 3, To: 9}, {From: 29, To: 35}},
		},
		{
			name:  "each word typed, in the order the runs stand",
			text:  "heat and work",
			query: "work heat",
			want:  []domain.Span{{From: 0, To: 4}, {From: 9, To: 13}},
		},
		{
			name:  "case is folded and nothing else is",
			text:  "Entropy is not entropy",
			query: "ENTROPY",
			want:  []domain.Span{{From: 0, To: 7}, {From: 15, To: 22}},
		},
		{
			// The index matched a word. A run of the same letters inside another
			// word is not that word, and a mark on it says a passage is here for
			// something nobody typed.
			name:  "letters standing inside another word are not marked",
			text:  "the party started early",
			query: "art",
		},
		{
			// The last word typed may still be growing, and the index reached it
			// by its opening. Every word before it is finished.
			name:  "the last word typed matches a word by its opening",
			text:  "a reversible engine",
			query: "revers",
			want:  []domain.Span{{From: 2, To: 8}},
		},
		{
			name:  "a word before the last one is matched whole",
			text:  "a reversible engine",
			query: "revers engine",
			want:  []domain.Span{{From: 13, To: 19}},
		},
		{
			name:  "two words naming the same characters come back as one run",
			text:  "entropy again",
			query: "ent entropy",
			want:  []domain.Span{{From: 0, To: 7}},
		},
		{
			name:  "the runs come back in the order they stand",
			text:  "engine and entropy",
			query: "entropy engine",
			want:  []domain.Span{{From: 0, To: 6}, {From: 11, To: 18}},
		},
		{
			name:  "a word the person did not type is not marked",
			text:  "no engine beats a reversible engine",
			query: "demon",
		},
		{
			// Counted the way a client counts text, so a character outside the
			// basic plane moves what follows it by two and not by one.
			name:  "past a character of two code units",
			text:  "👋 engine",
			query: "engine",
			want:  []domain.Span{{From: 3, To: 9}},
		},
		{
			// Folding a whole string can change how many characters it holds;
			// folded a rune at a time, the offsets still address the text.
			name:  "a letter whose lower case is longer",
			text:  "İstanbul and engine",
			query: "engine",
			want:  []domain.Span{{From: 13, To: 19}},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			got := marks(c.text, c.query)
			if len(got) != len(c.want) {
				t.Fatalf("marks(%q, %q) = %+v, want %+v", c.text, c.query, got, c.want)
			}
			for i := range got {
				if got[i] != c.want[i] {
					t.Fatalf("marks(%q, %q) = %+v, want %+v", c.text, c.query, got, c.want)
				}
			}
		})
	}
}

func TestAPassageIsCutToWhatCanBeReadAtAGlance(t *testing.T) {
	long := func(n int) string {
		out := make([]rune, n)
		for i := range out {
			out[i] = 'a'
		}
		return string(out)
	}

	t.Run("a passage short enough is left as it stands", func(t *testing.T) {
		text := "no engine beats a reversible engine"
		at := marks(text, "engine")

		cut, kept := around(text, at, 0)
		if cut != text {
			t.Errorf("cut %q, want it whole", cut)
		}
		if len(kept) != len(at) || kept[0] != at[0] {
			t.Errorf("kept %+v, want %+v", kept, at)
		}
	})

	t.Run("a long passage opens on the words about the first run", func(t *testing.T) {
		text := long(500) + " engine " + long(500)
		cut, kept := around(text, marks(text, "engine"), 0)

		if len([]rune(cut)) > glancing+2 {
			t.Errorf("cut is %d characters, want no more than %d and two marks",
				len([]rune(cut)), glancing)
		}
		if len(kept) != 1 {
			t.Fatalf("kept %+v runs, want the one that is inside the window", kept)
		}
		if got := string([]rune(cut)[kept[0].From:kept[0].To]); got != "engine" {
			t.Errorf("the run stands on %q, want it still on the word", got)
		}
		if !strings.HasPrefix(cut, "…") || !strings.HasSuffix(cut, "…") {
			t.Errorf("cut %q…%q, want it to say it was cut at both ends",
				cut[:6], cut[len(cut)-6:])
		}
	})

	t.Run("a long passage with nothing marked opens at its beginning", func(t *testing.T) {
		text := "The vault format is not settled. " + long(500)
		cut, kept := around(text, nil, 0)

		if !strings.HasPrefix(cut, "The vault format") {
			t.Errorf("cut opens on %q, want the beginning of the passage", cut[:20])
		}
		if strings.HasPrefix(cut, "…") {
			t.Errorf("cut %q says something stands before the beginning", cut[:6])
		}
		if len(kept) != 0 {
			t.Errorf("kept %+v, want nothing marked", kept)
		}
	})

	t.Run("a run left outside the window is dropped", func(t *testing.T) {
		text := "engine " + long(600) + " engine"
		at := marks(text, "engine")
		if len(at) != 2 {
			t.Fatalf("the passage holds %d runs, want 2", len(at))
		}

		_, kept := around(text, at, 0)
		if len(kept) != 1 {
			t.Errorf("kept %+v, want only the run the window holds", kept)
		}
	})

	t.Run("runs stay counted the way a client counts text", func(t *testing.T) {
		text := "👋 " + long(400) + " engine " + long(400)
		cut, kept := around(text, marks(text, "engine"), 0)

		if len(kept) != 1 {
			t.Fatalf("kept %+v runs, want 1", kept)
		}
		// Read back through UTF-16, which is what the client slices with.
		units := utf16.Encode([]rune(cut))
		if got := string(utf16.Decode(units[kept[0].From:kept[0].To])); got != "engine" {
			t.Errorf("the run reads %q through UTF-16, want %q", got, "engine")
		}
	})
}

func TestAPassageOpensOnTheHitWhenNoWordMatched(t *testing.T) {
	long := strings.Repeat("a", 500)
	text := long + " Bhishma was there " + long

	// Nothing matched a word — this is what a hit by meaning looks like — and
	// the chunk that did match begins where the sentence does.
	cut, kept := around(text, nil, 501)

	if !strings.Contains(cut, "Bhishma") {
		t.Errorf("cut %q, want it to hold the words the hit stands on", cut[:40])
	}
	if len(kept) != 0 {
		t.Errorf("kept %+v, want nothing marked: no word matched", kept)
	}
}
