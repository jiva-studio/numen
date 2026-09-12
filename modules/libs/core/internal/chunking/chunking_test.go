package chunking

import (
	"sort"
	"strings"
	"testing"
	"unicode/utf8"
)

// line builds n words on one line, so that a test about words is not a test
// about lines.
func line(n int) string {
	rotation := []string{"hedge", "gate", "shed", "compost", "marrow"}
	out := make([]string, n)
	for i := range out {
		out[i] = rotation[i%len(rotation)]
	}
	return strings.Join(out, " ")
}

const (
	latin    = "The hedge is thick, tangled, overgrown and very old."
	cyrillic = "Забор старый, кривой, дырявый и очень длинный, но крепкий."
	sanskrit = "udyāna pathaḥ dvāram bījāni śākhāḥ jalaṁ kṣetram"
	noise    = "ει; ·, .· ;, ·. ;· ., ·· ;, .· ει ;· ., ·, .· ;,"
)

func TestCut(t *testing.T) {
	tests := []struct {
		name      string
		text      string
		parts     []PartStart
		sizes     Sizes
		wantLarge int
		wantSmall int
		locations []string
	}{
		{
			name:      "no parts at all",
			text:      line(120),
			wantLarge: 1,
			wantSmall: 3,
			locations: []string{""},
		},
		{
			name:      "a division smaller than one chunk",
			text:      "Only three words.",
			wantLarge: 1,
			wantSmall: 1,
			locations: []string{""},
		},
		{
			name:      "a part at offset zero",
			text:      "Beginning\n" + line(60),
			parts:     []PartStart{{Title: "Beginning", Offset: 0}},
			wantLarge: 1,
			wantSmall: 2,
			locations: []string{"Beginning"},
		},
		{
			name:      "text before the first part",
			text:      line(20) + "\nSecond\n" + line(20),
			parts:     []PartStart{{Title: "Second", Offset: len(line(20)) + 1}},
			wantLarge: 2,
			wantSmall: 2,
			locations: []string{"", "Second"},
		},
		{
			name: "parts out of order",
			text: line(20) + "\n" + line(20) + "\n" + line(20),
			parts: []PartStart{
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
			name:      "the whole text is the large chunk",
			text:      "Note\n" + line(30) + "\nHeading\n" + line(30),
			parts:     []PartStart{{Title: "Note", Offset: 0}, {Title: "Heading", Offset: len("Note\n") + len(line(30)) + 1}},
			sizes:     Sizes{Large: Whole},
			wantLarge: 1,
			wantSmall: 2,
			locations: []string{""},
		},
		{
			name:      "a division of noise is not produced",
			text:      latin + "\n" + noise,
			parts:     []PartStart{{Title: "Prose", Offset: 0}, {Title: "Epigraph", Offset: len(latin) + 1}},
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
			out := Cut(test.text, test.parts, test.sizes, Legibility{})

			if len(out) != test.wantLarge {
				t.Errorf("large chunks: got %d, want %d", len(out), test.wantLarge)
			}
			if small := countSmall(out); small != test.wantSmall {
				t.Errorf("small chunks: got %d, want %d", small, test.wantSmall)
			}
			if got := locations(out); !equal(got, test.locations) {
				t.Errorf("locations: got %v, want %v", got, test.locations)
			}
			assertCutRules(t, test.text, test.parts, test.sizes, out)
		})
	}
}

// TestCutOneLongLine is the rule that a chunk is bounded in words: a book on
// one line is cut like a book on many.
func TestCutOneLongLine(t *testing.T) {
	text := line(200000)
	out := Cut(text, nil, Sizes{}, Legibility{})

	if len(out) != 1250 {
		t.Errorf("large chunks: got %d, want 1250", len(out))
	}
	for _, c := range out {
		if words := len(strings.Fields(c.Slice(text))); words > DefaultLarge {
			t.Fatalf("large chunk of %d words, at %d", words, c.Start)
		}
	}
	assertCutRules(t, text, nil, Sizes{}, out)
}

// TestCutOverlaps is the rule that two consecutive chunks share words and that
// both of them slice back to the words they share.
func TestCutOverlaps(t *testing.T) {
	text := "Opening\n" + line(70) + "\nSecond part\n" + latin + " " + sanskrit + "\n" + line(90)
	parts := []PartStart{
		{Title: "Opening", Offset: 0},
		{Title: "Second part", Offset: len("Opening\n") + len(line(70)) + 1},
	}
	const size, overlap = 6, 3
	sizes := Sizes{Large: 25, LargeOverlap: 10, Small: size, SmallOverlap: overlap}
	out := Cut(text, parts, sizes, Legibility{})

	small := flatten(out)
	if len(small) < 2 {
		t.Fatalf("small chunks: got %d, want several", len(small))
	}
	shared := 0
	for i := 0; i+1 < len(small); i++ {
		before, after := strings.Fields(small[i].Slice(text)), strings.Fields(small[i+1].Slice(text))
		if small[i].Location != small[i+1].Location || len(before) <= size-overlap {
			continue
		}
		want := before[size-overlap:]
		if len(after) < len(want) || !equal(after[:len(want)], want) {
			t.Errorf("chunks at %d and %d share %v and %v", small[i].Start, small[i+1].Start, want, after)
		}
		shared++
	}
	if shared == 0 {
		t.Fatal("no two chunks shared words")
	}
	assertCutRules(t, text, parts, sizes, out)
}

// TestCutKeepsSmallChunksUnderTheLimit is the rule that a small chunk stays
// inside the model's input: the caller gives the limit in characters, and a
// chunk over it is cut further at a word.
func TestCutKeepsSmallChunksUnderTheLimit(t *testing.T) {
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
			out := Cut(test.text, nil, sizes, Legibility{})
			seen := 0
			for _, large := range out {
				for _, small := range large.Small {
					if n := utf8.RuneCountInString(small.Slice(test.text)); n > test.limit {
						t.Errorf("small chunk of %d characters, limit %d", n, test.limit)
					}
					seen++
				}
			}
			if seen == 0 {
				t.Fatal("no small chunks produced")
			}
			assertCutRules(t, test.text, nil, sizes, out)
		})
	}
}

// TestCutIsPure is what lets a chunk keep an offset and not the text: the same
// text and the same parts give the same offsets, and Cut leaves the parts as it
// found them.
func TestCutIsPure(t *testing.T) {
	text := line(40) + "\nSecond\n" + line(40) + "\nThird\n" + line(40)
	parts := []PartStart{
		{Title: "Third", Offset: 2*len(line(40)) + len("\nSecond\n") + 1},
		{Title: "Second", Offset: len(line(40)) + 1},
	}
	given := append([]PartStart(nil), parts...)
	sizes := Sizes{Large: 12, Small: 5}

	first := Cut(text, given, sizes, Legibility{})
	second := Cut(text, given, sizes, Legibility{})

	if !same(first, second) {
		t.Error("two cuts of the same text disagree")
	}
	for i := range given {
		if given[i] != parts[i] {
			t.Errorf("Cut changed the parts it was given: %v, want %v", given[i], parts[i])
		}
	}
}

// TestCutNothing covers the texts that carry no words.
func TestCutNothing(t *testing.T) {
	for _, text := range []string{"", "   \n\t\n  ", noise} {
		if out := Cut(text, nil, Sizes{}, Legibility{}); out != nil {
			t.Errorf("Cut(%q) produced %d chunks", text, len(out))
		}
		if out := Cut(text, nil, Sizes{Large: Whole}, Legibility{}); out != nil {
			t.Errorf("Cut(%q) as one chunk produced %d chunks", text, len(out))
		}
	}
}

// assertCutRules asserts what holds of every cut, whatever the sizes: a chunk names its
// own words, sits under the chunk enclosing it, carries the name of the part it
// is in, and never crosses one.
func assertCutRules(t *testing.T, text string, parts []PartStart, sizes Sizes, out []Chunk) {
	t.Helper()
	s := sizes.Resolve()
	bounds := boundaries(text, parts)

	ascending := -1
	for _, large := range out {
		if large.Start <= ascending {
			t.Errorf("large chunk at %d does not follow the one before it", large.Start)
		}
		ascending = large.Start
		wordBounded(t, text, large)
		if s.Large != Whole {
			inside(t, bounds, large)
			assertPartName(t, parts, large)
			if words := len(strings.Fields(large.Slice(text))); words > s.Large {
				t.Errorf("large chunk of %d words, bound %d", words, s.Large)
			}
		}
		for _, small := range large.Small {
			wordBounded(t, text, small)
			inside(t, bounds, small)
			assertPartName(t, parts, small)
			if words := len(strings.Fields(small.Slice(text))); words > s.Small {
				t.Errorf("small chunk of %d words, bound %d", words, s.Small)
			}
			if n := utf8.RuneCountInString(small.Slice(text)); n > s.Limit {
				t.Errorf("small chunk of %d characters, limit %d", n, s.Limit)
			}
			if middle := small.Start + small.Length/2; middle < large.Start || middle >= large.Start+large.Length {
				t.Errorf("small chunk at %d is centred outside its large chunk at %d", small.Start, large.Start)
			}
		}
	}
}

func wordBounded(t *testing.T, text string, c Chunk) {
	t.Helper()
	if c.Start < 0 || c.Length <= 0 || c.Start+c.Length > len(text) {
		t.Fatalf("chunk %d+%d is not inside %d bytes of text", c.Start, c.Length, len(text))
	}
	if body := c.Slice(text); body != strings.TrimSpace(body) {
		t.Errorf("chunk at %d carries space at an end: %q", c.Start, body)
	}
}

// inside asserts that no chunk runs across a part: a boundary strictly inside a
// chunk is structure the cut ignored.
func inside(t *testing.T, bounds []int, c Chunk) {
	t.Helper()
	for _, at := range bounds {
		if c.Start < at && at < c.Start+c.Length {
			t.Errorf("chunk %d+%d runs across the part at %d", c.Start, c.Length, at)
		}
	}
}

// assertPartName asserts a chunk carries the name of the last part at or before it.
func assertPartName(t *testing.T, parts []PartStart, c Chunk) {
	t.Helper()
	ordered := append([]PartStart(nil), parts...)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Offset < ordered[j].Offset })
	want := ""
	for _, p := range ordered {
		if p.Offset <= c.Start {
			want = p.Title
		}
	}
	if c.Location != want {
		t.Errorf("chunk at %d says it is in %q, want %q", c.Start, c.Location, want)
	}
}

// boundaries are the offsets that separate one division from the next.
func boundaries(text string, parts []PartStart) []int {
	var out []int
	for _, p := range parts {
		if p.Offset > 0 && p.Offset < len(text) {
			out = append(out, p.Offset)
		}
	}
	sort.Ints(out)
	return out
}

func countSmall(out []Chunk) int {
	n := 0
	for _, c := range out {
		n += len(c.Small)
	}
	return n
}

func locations(out []Chunk) []string {
	var seen []string
	for _, c := range out {
		if len(seen) == 0 || seen[len(seen)-1] != c.Location {
			seen = append(seen, c.Location)
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

func same(a, b []Chunk) bool {
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

// flatten is every small chunk, in the order the text carries them.
func flatten(out []Chunk) []Chunk {
	var small []Chunk
	for _, large := range out {
		small = append(small, large.Small...)
	}
	return small
}

// A chunk is cut in words and bounded in characters, while a model truncates in
// tokens. The bound is a floor over every script: transliterated Sanskrit takes
// several tokens a word, and a chunk the model silently truncates is a chunk
// indexed for text it does not contain.
func TestAChunkIsBoundedUnderTheModelsLimit(t *testing.T) {
	if got := Under(0); got != DefaultLimit {
		t.Errorf("a model that said nothing gives %d", got)
	}
	if got := Under(256); got >= 256*4 {
		t.Errorf("256 tokens allow %d characters, which is not a floor", got)
	}
	if Under(256) <= 0 {
		t.Error("a model with a limit allows nothing")
	}
	// More tokens allow more characters, and the two move together.
	if Under(512) <= Under(256) {
		t.Error("twice the tokens do not allow more characters")
	}
}
