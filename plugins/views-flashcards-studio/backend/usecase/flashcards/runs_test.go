package flashcards_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/flashcards"
)

func TestGetRunReturnsErrNoRunForAnUnopenedRun(t *testing.T) {
	r := flashcards.NewRunRegistry()
	if _, err := r.GetRun("v1", "r1"); !errors.Is(err, flashcards.ErrNoRun) {
		t.Fatalf("expected ErrNoRun, got %v", err)
	}
}
