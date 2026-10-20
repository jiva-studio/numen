package agentrun

import (
	"errors"
	"testing"
)

func TestFailureMessageFallsBackToTheError(t *testing.T) {
	if got := FailureMessage(errors.New("exit 1"), "warming up\n\nboom: no key\n"); got != "boom: no key" {
		t.Errorf("got %q", got)
	}
	if got := FailureMessage(errors.New("exit 1"), "  \n"); got != "exit 1" {
		t.Errorf("got %q", got)
	}
}

// The error line of an output that goes on to list what was available is the message, not the list.
func TestFailureMessageIsTheErrorLineOfALongerOutput(t *testing.T) {
	output := "error: invalid model selection (--model \"m-old\"): not recognized\nAvailable models:\n  Model One\n  Model Two\n"
	want := "invalid model selection (--model \"m-old\"): not recognized"
	if got := FailureMessage(errors.New("exit 1"), output); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
