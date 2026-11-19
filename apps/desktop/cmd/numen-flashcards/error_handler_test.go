package main

import (
	"errors"
	"strings"
	"testing"
)

// TestCoreErrorHandler_ReportError. The core reports what went wrong in work
// it does not stop for, and a configuration naming no destination makes it a
// failure that is never reported.
func TestCoreErrorHandler_ReportError(t *testing.T) {
	var response strings.Builder
	services := makeServices(&response)
	if services.Resources.ErrorHandler == nil {
		t.Fatal("the core has nowhere to say what it carried on past")
	}

	services.Resources.ErrorHandler(errors.New("the days are counted again at every launch"))
	if !strings.Contains(response.String(), "numen-flashcards: the days are counted again at every launch") {
		t.Errorf("what the core said came out as %q", response.String())
	}
}
