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
	if len(got) != 1 || got[0].from != 2 || got[0].to != 8 {
		t.Errorf("a sentence with a pause in it came out as %v", got)
	}
}

// Quiet enough closes a stretch, and what comes after it is another.
func TestQuietClosesAStretch(t *testing.T) {
	scores := scored(16, 1, 2, 10, 11)
	got := runs(scores, 0.5, 3, 0, 100, 1)
	if len(got) != 2 || got[0].to != 3 || got[1].from != 10 {
		t.Errorf("two sentences came out as %v", got)
	}
}

// A stretch is widened at both ends, and two that then meet are one.
func TestWideningJoinsTwoStretchesThatMeet(t *testing.T) {
	scores := scored(20, 4, 10)
	got := runs(scores, 0.5, 2, 3, 100, 1)
	if len(got) != 1 || got[0].from != 1 || got[0].to != 14 {
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
	if len(got) != 2 || got[0].to != 7 || got[1].from != 7 {
		t.Errorf("a stretch of sixteen windows came out as %v", got)
	}
	if got[1].to != 16 {
		t.Errorf("the second stretch ends at %d", got[1].to)
	}
}

// A line of a transcript is read, so a stretch too short to be one is put
// together with what follows it.
func TestAShortStretchJoinsTheNextOne(t *testing.T) {
	// least 10 windows, longest 100.
	got := joined([]run{
		{from: 0, to: 3},   // "So"
		{from: 5, to: 8},   // "The"
		{from: 10, to: 40}, // a sentence
		{from: 50, to: 90}, // another
	}, 10, 100)

	want := []run{{from: 0, to: 40}, {from: 50, to: 90}}
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
	got := joined([]run{
		{from: 0, to: 5},
		{from: 6, to: 11},
		{from: 12, to: 17},
	}, 100, 12)

	if len(got) != 2 || got[0] != (run{from: 0, to: 11}) {
		t.Errorf("joined into %v", got)
	}
}
