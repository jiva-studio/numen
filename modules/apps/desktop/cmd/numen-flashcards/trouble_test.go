package main

import (
	"errors"
	"strings"
	"testing"
)

// TestTheCoreSaysWhatItCarriedOnPast. The core reports what went wrong in work
// it does not stop for, and a configuration naming nowhere to say it is a
// failure nobody is ever told about.
func TestTheCoreSaysWhatItCarriedOnPast(t *testing.T) {
	var said strings.Builder
	cfg := configured(&said)
	if cfg.Trouble == nil {
		t.Fatal("the core has nowhere to say what it carried on past")
	}

	cfg.Trouble(errors.New("the days are counted again at every launch"))
	if !strings.Contains(said.String(), "numen-flashcards: the days are counted again at every launch") {
		t.Errorf("what the core said came out as %q", said.String())
	}
}
