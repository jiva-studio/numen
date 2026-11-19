package app

import (
	"errors"
	"strings"
	"testing"
)

// TestCoreErrorHandler_ReportError. The core reports errors from work it does
// not abort, and a configuration without a handler leaves such errors
// unreported.
func TestCoreErrorHandler_ReportError(t *testing.T) {
	var response strings.Builder
	services := makeServices(&response)
	if services.Resources.ErrorHandler == nil {
		t.Fatal("the core has nowhere to say what it carried on past")
	}

	services.Resources.ErrorHandler(errors.New("the schedules are worked out at every launch"))
	if !strings.Contains(response.String(), "numen: the schedules are worked out at every launch") {
		t.Errorf("what the core said came out as %q", response.String())
	}
}
