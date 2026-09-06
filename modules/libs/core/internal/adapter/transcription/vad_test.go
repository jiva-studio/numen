package transcription

import "testing"

// scored is a run of windows, sure where the ones named are and sure of nothing
// elsewhere.
func scored(count int, sure ...int) []float32 {
	out := make([]float32, count)
	for _, at := range sure {
		out[at] = 1
	}
	return out
}

// A pause inside a sentence is shorter than the quiet that closes a stretch,
// and does not cut the sentence in half.
func TestAPauseInsideASentenceDoesNotCloseIt(t *testing.T) {
	scores := scored(12, 2, 3, 6, 7)
	got := runs(scores, 0.5, 3, 0, 100, 1)
	if len(got) != 1 || got[0].start != 2 || got[0].end != 8 {
		t.Errorf("a sentence with a pause in it came out as %v", got)
	}
}

// Quiet enough closes a stretch, and what comes after it is another.
func TestQuietClosesAStretch(t *testing.T) {
	scores := scored(16, 1, 2, 10, 11)
	got := runs(scores, 0.5, 3, 0, 100, 1)
	if len(got) != 2 || got[0].end != 3 || got[1].start != 10 {
		t.Errorf("two sentences came out as %v", got)
	}
}

// A stretch is widened at both ends, and two that then meet are one.
func TestWideningJoinsTwoStretchesThatMeet(t *testing.T) {
	scores := scored(20, 4, 10)
	got := runs(scores, 0.5, 2, 3, 100, 1)
	if len(got) != 1 || got[0].start != 1 || got[0].end != 14 {
		t.Errorf("two stretches widened into %v", got)
	}
}

// A stretch shorter than the shortest is not a stretch.
func TestAStretchTooShortIsNotOne(t *testing.T) {
	if got := runs(scored(10, 4), 0.5, 2, 0, 100, 3); len(got) != 0 {
		t.Errorf("one window came out as %v", got)
	}
}

// Speech that runs on longer than one stretch may is cut, and the cut falls
// where the speaker was quietest.
func TestSpeechThatRunsOnIsCutWhereItIsQuietest(t *testing.T) {
	scores := make([]float32, 16)
	for i := range scores {
		scores[i] = 1
	}
	scores[7] = 0.6

	got := runs(scores, 0.5, 2, 0, 10, 1)
	if len(got) != 2 || got[0].end != 7 || got[1].start != 7 {
		t.Errorf("a stretch of sixteen windows came out as %v", got)
	}
	if got[1].end != 16 {
		t.Errorf("the second stretch ends at %d", got[1].end)
	}
}

// A line of a transcript is read, so a stretch too short to be one is put
// together with what follows it.
func TestAShortStretchJoinsTheNextOne(t *testing.T) {
	// least 10 windows, longest 100.
	got := joined([]stretch{
		{start: 0, end: 3},   // "So"
		{start: 5, end: 8},   // "The"
		{start: 10, end: 40}, // a sentence
		{start: 50, end: 90}, // another
	}, 10, 100)

	want := []stretch{{start: 0, end: 40}, {start: 50, end: 90}}
	if len(got) != len(want) {
		t.Fatalf("joined into %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("stretch %d is %v, want %v", i, got[i], want[i])
		}
	}
}

// Joining stops at the longest a stretch may run to, however short the pieces.
func TestJoiningStopsAtTheLongest(t *testing.T) {
	got := joined([]stretch{
		{start: 0, end: 5},
		{start: 6, end: 11},
		{start: 12, end: 17},
	}, 100, 12)

	if len(got) != 2 || got[0] != (stretch{start: 0, end: 11}) {
		t.Errorf("joined into %v", got)
	}
}
