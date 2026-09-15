package transcription

import "testing"

// buildScores is a run of windows, sure where the ones named are and sure of nothing
// elsewhere.
func buildScores(count int, sure ...int) []float32 {
	out := make([]float32, count)
	for _, at := range sure {
		out[at] = 1
	}
	return out
}

// A pause inside a sentence is shorter than the quiet that closes a span,
// and does not cut the sentence in half.
func TestAPauseInsideASentenceDoesNotCloseIt(t *testing.T) {
	scores := buildScores(12, 2, 3, 6, 7)
	got := runs(scores, 0.5, 3, 0, 100, 1)
	if len(got) != 1 || got[0].From != 2 || got[0].To != 8 {
		t.Errorf("a sentence with a pause in it came out as %v", got)
	}
}

// Quiet enough closes a span, and what comes after it is another.
func TestQuietClosesASpan(t *testing.T) {
	scores := buildScores(16, 1, 2, 10, 11)
	got := runs(scores, 0.5, 3, 0, 100, 1)
	if len(got) != 2 || got[0].To != 3 || got[1].From != 10 {
		t.Errorf("two sentences came out as %v", got)
	}
}

// A span is widened at both ends, and two that then meet are one.
func TestWideningJoinsTwoSpansThatMeet(t *testing.T) {
	scores := buildScores(20, 4, 10)
	got := runs(scores, 0.5, 2, 3, 100, 1)
	if len(got) != 1 || got[0].From != 1 || got[0].To != 14 {
		t.Errorf("two spans widened into %v", got)
	}
}

// A span shorter than the shortest is not a span.
func TestASpanTooShortIsNotOne(t *testing.T) {
	if got := runs(buildScores(10, 4), 0.5, 2, 0, 100, 3); len(got) != 0 {
		t.Errorf("one window came out as %v", got)
	}
}

// Speech that runs on longer than one span may is cut, and the cut falls
// where the speaker was quietest.
func TestSpeechThatRunsOnIsCutWhereItIsQuietest(t *testing.T) {
	scores := make([]float32, 16)
	for i := range scores {
		scores[i] = 1
	}
	scores[7] = 0.6

	got := runs(scores, 0.5, 2, 0, 10, 1)
	if len(got) != 2 || got[0].To != 7 || got[1].From != 7 {
		t.Errorf("a span of sixteen windows came out as %v", got)
	}
	if got[1].To != 16 {
		t.Errorf("the second span ends at %d", got[1].To)
	}
}

// A line of a transcript is read, so a span too short to be one is put
// together with what follows it.
func TestAShortSpanJoinsTheNextOne(t *testing.T) {
	// least 10 windows, longest 100.
	got := joinShortSpans([]span{
		{From: 0, To: 3},   // "So"
		{From: 5, To: 8},   // "The"
		{From: 10, To: 40}, // a sentence
		{From: 50, To: 90}, // another
	}, 10, 100)

	want := []span{{From: 0, To: 40}, {From: 50, To: 90}}
	if len(got) != len(want) {
		t.Fatalf("joined into %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("span %d is %v, want %v", i, got[i], want[i])
		}
	}
}

// Joining stops at the longest a span may run to, however short the pieces.
func TestJoiningStopsAtTheLongest(t *testing.T) {
	got := joinShortSpans([]span{
		{From: 0, To: 5},
		{From: 6, To: 11},
		{From: 12, To: 17},
	}, 100, 12)

	if len(got) != 2 || got[0] != (span{From: 0, To: 11}) {
		t.Errorf("joined into %v", got)
	}
}
