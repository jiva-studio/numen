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
