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
	var response strings.Builder
	cfg := makeConfig(&response)
	if cfg.ErrorHandler == nil {
		t.Fatal("the core has nowhere to say what it carried on past")
	}

	cfg.ErrorHandler(errors.New("the days are counted again at every launch"))
	if !strings.Contains(response.String(), "numen-flashcards: the days are counted again at every launch") {
		t.Errorf("what the core said came out as %q", response.String())
	}
}
