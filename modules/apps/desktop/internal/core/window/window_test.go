package window

import (
	"sort"
	"strings"
	"testing"
	"unicode/utf8"
)

// line builds n words on one line, so that a test about words is not a test
// about lines.
func line(n int) string {
	rotation := []string{"dharma", "artha", "kama", "moksha", "satya"}
	out := make([]string, n)
	for i := range out {
		out[i] = rotation[i%len(rotation)]
	}
	return strings.Join(out, " ")
}

const (
	latin    = "The mind is restless, turbulent, obstinate and very strong."
	cyrillic = "Ум беспокоен, упрям, необуздан и очень силён, о Кришна."
	sanskrit = "cañcalaṁ hi manaḥ kṛṣṇa pramāthi balavad dṛḍham"
	noise    = "ει; ·, .· ;, ·. ;· ., ·· ;, .· ει ;· ., ·, .· ;,"
)

func TestCut(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		places    []Place
		sizes     Sizes
		wantLarge int
		wantSmall int
		locations []string
	}{
		{
			name:      "no places at all",
			text:      line(120),
			wantLarge: 1,
			wantSmall: 3,
			locations: []string{""},
		},
		{
			name:      "a span smaller than one window",
			text:      "Only three words.",
			wantLarge: 1,
			wantSmall: 1,
			locations: []string{""},
		},
		{
			name:      "a place at offset zero",
			text:      "Beginning\n" + line(60),
			places:    []Place{{Title: "Beginning", Offset: 0}},
			wantLarge: 1,
			wantSmall: 2,
			locations: []string{"Beginning"},
		},
		{
			name:      "text before the first place",
			text:      line(20) + "\nSecond\n" + line(20),
			places:    []Place{{Title: "Second", Offset: len(line(20)) + 1}},
			wantLarge: 2,
			wantSmall: 2,
			locations: []string{"", "Second"},
		},
		{
			name: "places out of order",
			text: line(20) + "\n" + line(20) + "\n" + line(20),
			places: []Place{
				{Title: "Third", Offset: 2*len(line(20)) + 2},
				{Title: "Second", Offset: len(line(20)) + 1},
			},
			wantLarge: 3,
			wantSmall: 3,
			locations: []string{"", "Second", "Third"},
		},
		{
			name:      "sizes given by configuration",
			text:      line(30),
			sizes:     Sizes{Large: 10, LargeOverlap: 2, Small: 4, SmallOverlap: 1},
			wantLarge: 4,
			wantSmall: 10,
			locations: []string{""},
		},
		{
			name:      "the whole text is the large window",
			text:      "Note\n" + line(30) + "\nHeading\n" + line(30),
			places:    []Place{{Title: "Note", Offset: 0}, {Title: "Heading", Offset: len("Note\n") + len(line(30)) + 1}},
			sizes:     Sizes{Large: Whole},
			wantLarge: 1,
			wantSmall: 2,
			locations: []string{""},
		},
		{
			name:      "a span of noise is not produced",
			text:      latin + "\n" + noise,
			places:    []Place{{Title: "Prose", Offset: 0}, {Title: "Epigraph", Offset: len(latin) + 1}},
			wantLarge: 1,
			wantSmall: 1,
			locations: []string{"Prose"},
		},
		{
			name:      "prose in three scripts is kept",
			text:      latin + " " + cyrillic + " " + sanskrit,
			wantLarge: 1,
			wantSmall: 1,
			locations: []string{""},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			out := Cut(test.text, test.places, test.sizes)

			if len(out) != test.wantLarge {
				t.Errorf("large windows: got %d, want %d", len(out), test.wantLarge)
			}
			if small := counted(out); small != test.wantSmall {
				t.Errorf("small windows: got %d, want %d", small, test.wantSmall)
			}
			if got := locations(out); !equal(got, test.locations) {
				t.Errorf("locations: got %v, want %v", got, test.locations)
			}
			obeyed(t, test.text, test.places, test.sizes, out)
		})
	}
}

// TestCutOneLongLine is the rule that a window is bounded in words: a book on
// one line is cut like a book on many.
func TestCutOneLongLine(t *testing.T) {
	text := line(200000)
	out := Cut(text, nil, Sizes{})

	if len(out) != 1250 {
		t.Errorf("large windows: got %d, want 1250", len(out))
	}
	for _, w := range out {
		if words := len(strings.Fields(w.Slice(text))); words > DefaultLarge {
			t.Fatalf("large window of %d words, at %d", words, w.Start)
		}
	}
	obeyed(t, text, nil, Sizes{}, out)
}

// TestCutOverlaps is the rule that two consecutive windows share words and that
// both of them slice back to the words they share.
func TestCutOverlaps(t *testing.T) {
	text := "Opening\n" + line(70) + "\nSecond place\n" + latin + " " + sanskrit + "\n" + line(90)
	places := []Place{
		{Title: "Opening", Offset: 0},
		{Title: "Second place", Offset: len("Opening\n") + len(line(70)) + 1},
	}
	const size, overlap = 6, 3
	sizes := Sizes{Large: 25, LargeOverlap: 10, Small: size, SmallOverlap: overlap}
	out := Cut(text, places, sizes)

	small := flatten(out)
	if len(small) < 2 {
		t.Fatalf("small windows: got %d, want several", len(small))
	}
	shared := 0
	for i := 0; i+1 < len(small); i++ {
		before, after := strings.Fields(small[i].Slice(text)), strings.Fields(small[i+1].Slice(text))
		if small[i].Location != small[i+1].Location || len(before) <= size-overlap {
			continue
		}
		want := before[size-overlap:]
		if len(after) < len(want) || !equal(after[:len(want)], want) {
			t.Errorf("windows at %d and %d share %v and %v", small[i].Start, small[i+1].Start, want, after)
		}
		shared++
	}
	if shared == 0 {
		t.Fatal("no two windows shared words")
	}
	obeyed(t, text, places, sizes, out)
}

// TestCutKeepsSmallWindowsUnderTheLimit is the rule that a small window stays
// inside the model's input: the caller gives the limit in characters, and a
// window over it is cut further at a word.
func TestCutKeepsSmallWindowsUnderTheLimit(t *testing.T) {
	tests := []struct {
		name  string
		text  string
		limit int
	}{
		{name: "latin", text: strings.Repeat(latin+" ", 20), limit: 40},
		{name: "sanskrit", text: strings.Repeat(sanskrit+" ", 20), limit: 30},
		{name: "cyrillic", text: strings.Repeat(cyrillic+" ", 20), limit: 60},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sizes := Sizes{Small: 50, Limit: test.limit}
			out := Cut(test.text, nil, sizes)
			seen := 0
			for _, large := range out {
				for _, small := range large.Small {
					if n := utf8.RuneCountInString(small.Slice(test.text)); n > test.limit {
						t.Errorf("small window of %d characters, limit %d", n, test.limit)
					}
					seen++
				}
			}
			if seen == 0 {
				t.Fatal("no small windows produced")
			}
			obeyed(t, test.text, nil, sizes, out)
		})
	}
}

// TestCutIsPure is what lets a chunk keep an offset and not the text: the same
// text and the same places give the same offsets, and Cut leaves the places as it
// found them.
func TestCutIsPure(t *testing.T) {
	text := line(40) + "\nSecond\n" + line(40) + "\nThird\n" + line(40)
	places := []Place{
		{Title: "Third", Offset: 2*len(line(40)) + len("\nSecond\n") + 1},
		{Title: "Second", Offset: len(line(40)) + 1},
	}
	given := append([]Place(nil), places...)
	sizes := Sizes{Large: 12, Small: 5}

	first := Cut(text, given, sizes)
	second := Cut(text, given, sizes)

	if !same(first, second) {
		t.Error("two cuts of the same text disagree")
	}
	for i := range given {
		if given[i] != places[i] {
			t.Errorf("Cut changed the places it was given: %v, want %v", given[i], places[i])
		}
	}
}

// TestCutNothing covers the texts that carry no words.
func TestCutNothing(t *testing.T) {
	for _, text := range []string{"", "   \n\t\n  ", noise} {
		if out := Cut(text, nil, Sizes{}); out != nil {
			t.Errorf("Cut(%q) produced %d windows", text, len(out))
		}
		if out := Cut(text, nil, Sizes{Large: Whole}); out != nil {
			t.Errorf("Cut(%q) as one window produced %d windows", text, len(out))
		}
	}
}

// obeyed asserts what holds of every cut, whatever the sizes: a window names its
// own words, sits under the window enclosing it, carries the name of the place it
// is in, and never crosses one.
func obeyed(t *testing.T, text string, places []Place, sizes Sizes, out []Window) {
	t.Helper()
	s := sizes.resolve()
	bounds := boundaries(text, places)

	ascending := -1
	for _, large := range out {
		if large.Start <= ascending {
			t.Errorf("large window at %d does not follow the one before it", large.Start)
		}
		ascending = large.Start
		wordBounded(t, text, large)
		if s.Large != Whole {
			inside(t, bounds, large)
			named(t, places, large)
			if words := len(strings.Fields(large.Slice(text))); words > s.Large {
				t.Errorf("large window of %d words, bound %d", words, s.Large)
			}
		}
		for _, small := range large.Small {
			wordBounded(t, text, small)
			inside(t, bounds, small)
			named(t, places, small)
			if words := len(strings.Fields(small.Slice(text))); words > s.Small {
				t.Errorf("small window of %d words, bound %d", words, s.Small)
			}
			if n := utf8.RuneCountInString(small.Slice(text)); n > s.Limit {
				t.Errorf("small window of %d characters, limit %d", n, s.Limit)
			}
			if middle := small.Start + small.Length/2; middle < large.Start || middle >= large.Start+large.Length {
				t.Errorf("small window at %d is centred outside its large window at %d", small.Start, large.Start)
			}
		}
	}
}

func wordBounded(t *testing.T, text string, w Window) {
	t.Helper()
	if w.Start < 0 || w.Length <= 0 || w.Start+w.Length > len(text) {
		t.Fatalf("window %d+%d is not inside %d bytes of text", w.Start, w.Length, len(text))
	}
	if body := w.Slice(text); body != strings.TrimSpace(body) {
		t.Errorf("window at %d carries space at an end: %q", w.Start, body)
	}
}

// inside asserts that no window runs across a place: a boundary strictly inside a
// window is structure the cut ignored.
func inside(t *testing.T, bounds []int, w Window) {
	t.Helper()
	for _, at := range bounds {
		if w.Start < at && at < w.Start+w.Length {
			t.Errorf("window %d+%d runs across the place at %d", w.Start, w.Length, at)
		}
	}
}

// named asserts a window carries the name of the last place at or before it.
func named(t *testing.T, places []Place, w Window) {
	t.Helper()
	ordered := append([]Place(nil), places...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Offset < ordered[j].Offset })
	want := ""
	for _, p := range ordered {
		if p.Offset <= w.Start {
			want = p.Title
		}
	}
	if w.Location != want {
		t.Errorf("window at %d says it is in %q, want %q", w.Start, w.Location, want)
	}
}

// boundaries are the offsets that separate one span from the next.
func boundaries(text string, places []Place) []int {
	var out []int
	for _, p := range places {
		if p.Offset > 0 && p.Offset < len(text) {
			out = append(out, p.Offset)
		}
	}
	sort.Ints(out)
	return out
}

func counted(out []Window) int {
	n := 0
	for _, w := range out {
		n += len(w.Small)
	}
	return n
}

func locations(out []Window) []string {
	var seen []string
	for _, w := range out {
		if len(seen) == 0 || seen[len(seen)-1] != w.Location {
			seen = append(seen, w.Location)
		}
	}
	return seen
}

func equal(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}
	return true
}

func same(a, b []Window) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Start != b[i].Start || a[i].Length != b[i].Length || a[i].Location != b[i].Location {
			return false
		}
		if !same(a[i].Small, b[i].Small) {
			return false
		}
	}
	return true
}

// flatten is every small window, in the order the text carries them.
func flatten(out []Window) []Window {
	var small []Window
	for _, large := range out {
		small = append(small, large.Small...)
	}
	return small
}
