package sessions_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/usecase/sessions"
)

func TestGetRunReturnsErrNoRunForAnUnopenedRun(t *testing.T) {
	r := sessions.NewRegistry()
	if _, err := r.GetRun("v1", "r1"); !errors.Is(err, sessions.ErrNoRun) {
		t.Fatalf("expected ErrNoRun, got %v", err)
	}
}
