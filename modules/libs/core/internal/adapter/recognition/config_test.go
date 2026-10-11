package recognition

import "testing"

func TestRecogniserTakesFourThreadsWhenNoneAreSet(t *testing.T) {
	if got := Defaults().Recognise.threads(); got != 4 {
		t.Errorf("the default recogniser takes %d threads, want 4", got)
	}
	if got := (RecogniserModel{Threads: -1}).threads(); got != 4 {
		t.Errorf("a negative count takes %d threads, want 4", got)
	}
}

func TestRecogniserTakesTheThreadsItIsGiven(t *testing.T) {
	if got := (RecogniserModel{Threads: 2}).threads(); got != 2 {
		t.Errorf("the recogniser takes %d threads, want 2", got)
	}
}
