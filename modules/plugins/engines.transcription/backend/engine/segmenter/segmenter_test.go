package segmenter

import (
	"errors"
	"slices"
	"testing"

	ort "github.com/getcharzp/onnxruntime_purego"
)

// buildScores is a run of windows, sure where the ones named are and sure of nothing
// elsewhere.
func buildScores(count int, sure ...int) []float32 {
	out := make([]float32, count)
	for _, item := range sure {
		out[item] = 1
	}
	return out
}

// A pause inside a sentence is shorter than the quiet that closes a span,
// and does not cut the sentence in half.
func TestAPauseInsideASentenceDoesNotCloseIt(t *testing.T) {
	scores := buildScores(12, 2, 3, 6, 7)
	got := getSpans(scores, 0.5, 3, 0, 100, 1)
	if len(got) != 1 || got[0].From != 2 || got[0].To != 8 {
		t.Errorf("a sentence with a pause in it came out as %v", got)
	}
}

// Quiet enough closes a span, and what comes after it is another.
func TestQuietClosesASpan(t *testing.T) {
	scores := buildScores(16, 1, 2, 10, 11)
	got := getSpans(scores, 0.5, 3, 0, 100, 1)
	if len(got) != 2 || got[0].To != 3 || got[1].From != 10 {
		t.Errorf("two sentences came out as %v", got)
	}
}

// A span is widened at both ends, and two that then meet are one.
func TestWideningJoinsTwoSpansThatMeet(t *testing.T) {
	scores := buildScores(20, 4, 10)
	got := getSpans(scores, 0.5, 2, 3, 100, 1)
	if len(got) != 1 || got[0].From != 1 || got[0].To != 14 {
		t.Errorf("two spans widened into %v", got)
	}
}

// A span shorter than the shortest is not a span.
func TestASpanTooShortIsNotOne(t *testing.T) {
	if got := getSpans(buildScores(10, 4), 0.5, 2, 0, 100, 3); len(got) != 0 {
		t.Errorf("one window came out as %v", got)
	}
}

// Speech that runs on longer than one span may is cut, and the cut falls
// where the speaker was quietest.
func TestSpeechThatRunsOnIsSplitWhereItIsQuietest(t *testing.T) {
	scores := make([]float32, 16)
	for i := range scores {
		scores[i] = 1
	}
	scores[7] = 0.6

	got := getSpans(scores, 0.5, 2, 0, 10, 1)
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

// Empty input returns no spans.
func TestJoinShortSpansEmpty(t *testing.T) {
	if got := joinShortSpans(nil, 10, 100); len(got) != 0 {
		t.Errorf("got %v, want empty", got)
	}
}

// Short spans with non-zero offsets are joined up to longest.
func TestJoinShortSpansNonZeroOffset(t *testing.T) {
	got := joinShortSpans([]span{
		{From: 20, To: 23},
		{From: 25, To: 28},
	}, 10, 20)
	want := []span{{From: 20, To: 28}}
	if len(got) != 1 || got[0] != want[0] {
		t.Errorf("got %v, want %v", got, want)
	}
}

// A span exactly equal to the shortest bound is preserved.
func TestASpanExactlyShortestIsCached(t *testing.T) {
	scores := buildScores(10, 2, 3, 4)
	got := getSpans(scores, 0.5, 2, 0, 100, 3)
	if len(got) != 1 || got[0].From != 2 || got[0].To != 5 {
		t.Errorf("span of exact shortest length came out as %v", got)
	}
}

// A recording of silence produces no speech spans.
func TestSilenceProducesNoSpans(t *testing.T) {
	scores := buildScores(10)
	got := getSpans(scores, 0.5, 2, 0, 100, 1)
	if len(got) != 0 {
		t.Errorf("silence produced spans %v", got)
	}
}

func TestSpeechPassLoad(t *testing.T) {
	if pass, err := newSpeechPass(make([]float32, 1024)); err == nil {
		defer pass.destroy()
		if len(pass.window) != 512 {
			t.Errorf("newSpeechPass window len = %d, want 512", len(pass.window))
		}
		if len(pass.state) != 256 {
			t.Errorf("newSpeechPass state len = %d, want 256", len(pass.state))
		}
	}

	sound := make([]float32, 1000)
	for i := range sound {
		sound[i] = float32(i + 1)
	}
	pass := &speechPass{
		sound:  sound,
		window: make([]float32, speechWindow),
		state:  make([]float32, speechMemory),
		count:  (len(sound) + speechWindow - 1) / speechWindow,
	}
	if len(pass.window) != 512 {
		t.Errorf("window len = %d, want 512", len(pass.window))
	}
	if len(pass.state) != 256 {
		t.Errorf("state len = %d, want 256", len(pass.state))
	}
	if pass.count != 2 {
		t.Errorf("count = %d, want 2", pass.count)
	}
	pass.load(0)
	if pass.window[0] != 1 || pass.window[511] != 512 {
		t.Errorf("window[0] = %f, window[511] = %f", pass.window[0], pass.window[511])
	}
	pass.load(1)
	if pass.window[0] != 513 || pass.window[487] != 1000 {
		t.Errorf("window[0] = %f, window[487] = %f", pass.window[0], pass.window[487])
	}
	if pass.window[488] != 0 || pass.window[511] != 0 {
		t.Errorf("window[488] = %f, window[511] = %f, want 0", pass.window[488], pass.window[511])
	}
}

// A recording ending before silence elapses preserves the open span.
func TestRecordingEndingBeforeSilenceKeepsSpan(t *testing.T) {
	scores := buildScores(10, 2, 3)
	got := getSpans(scores, 0.5, 10, 0, 100, 1)
	if len(got) != 1 || got[0].From != 2 || got[0].To != 4 {
		t.Errorf("got %v, want span [2, 4]", got)
	}
}

// A span longer than longest is split when longest is two.
func TestSplitLongSpanWhenLongestIsTwo(t *testing.T) {
	scores := []float32{1, 0, 1}
	got := splitLongSpan(span{From: 0, To: 3}, scores, 2)
	want := []span{{From: 0, To: 1}, {From: 1, To: 3}}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("got %v, want %v", got, want)
	}
}

// A recording's window count covers every sample, rounding up to the next full window.
func TestNewSpeechPassCount(t *testing.T) {
	orig := newTensor
	newTensor = func(shape []int64, data any) (*ort.Value, error) {
		return &ort.Value{}, nil
	}
	t.Cleanup(func() { newTensor = orig })

	for _, tc := range []struct {
		samples int
		want    int
	}{
		{0, 0},
		{1, 1},
		{511, 1},
		{512, 1},
		{513, 2},
		{1024, 2},
		{1025, 3},
	} {
		pass, err := newSpeechPass(make([]float32, tc.samples))
		if err != nil {
			t.Fatalf("samples %d: unexpected error: %v", tc.samples, err)
		}
		if pass.count != tc.want {
			t.Errorf("samples %d: count = %d, want %d", tc.samples, pass.count, tc.want)
		}
		if len(pass.window) != 512 {
			t.Errorf("samples %d: window len = %d, want 512", tc.samples, len(pass.window))
		}
		if len(pass.state) != 256 {
			t.Errorf("samples %d: state len = %d, want 256", tc.samples, len(pass.state))
		}
		pass.destroy()
	}
}

// Tensor construction prepares input, memory, and sampling tensors with expected shapes.
func TestNewSpeechPassTensorShapes(t *testing.T) {
	orig := newTensor
	var shapes [][]int64
	newTensor = func(shape []int64, data any) (*ort.Value, error) {
		shapes = append(shapes, shape)
		return &ort.Value{}, nil
	}
	t.Cleanup(func() { newTensor = orig })

	pass, err := newSpeechPass(make([]float32, 513))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer pass.destroy()

	if len(shapes) != 3 {
		t.Fatalf("tensors created = %d, want 3", len(shapes))
	}
	if !slices.Equal(shapes[0], []int64{1, 512}) {
		t.Errorf("input shape = %v, want [1, 512]", shapes[0])
	}
	if !slices.Equal(shapes[1], []int64{2, 1, 128}) {
		t.Errorf("memory shape = %v, want [2, 1, 128]", shapes[1])
	}
	if shapes[2] != nil {
		t.Errorf("sampling shape = %v, want nil", shapes[2])
	}
}

// Tensor construction fails when no runtime engine is initialized.
func TestNewSpeechPassWithoutEngineFails(t *testing.T) {
	pass, err := newSpeechPass(make([]float32, 513))
	if err == nil {
		defer pass.destroy()
		t.Fatal("expected error when engine is uninitialized, got nil")
	}
	if pass != nil {
		t.Errorf("pass = %v, want nil on error", pass)
	}
}

// Construction failure at any tensor allocation cleans up and returns the error.
func TestNewSpeechPassConstructorErrors(t *testing.T) {
	orig := newTensor
	t.Cleanup(func() { newTensor = orig })

	errBoom := errors.New("boom")
	for _, failStep := range []int{0, 1, 2} {
		step := 0
		newTensor = func(shape []int64, data any) (*ort.Value, error) {
			if step == failStep {
				step++
				return nil, errBoom
			}
			step++
			return &ort.Value{}, nil
		}

		pass, err := newSpeechPass(make([]float32, 513))
		if !errors.Is(err, errBoom) {
			t.Errorf("step %d: err = %v, want %v", failStep, err, errBoom)
		}
		if pass != nil {
			t.Errorf("step %d: pass = %v, want nil", failStep, pass)
		}
	}
}
