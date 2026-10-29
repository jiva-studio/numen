package bind

import (
	"errors"
	"strings"
	"testing"
)

// TestCoreErrorHandler_ReportError checks that the core reports errors from
// work it does not stop for. A configuration with no error destination makes
// such failures invisible.
func TestCoreErrorHandler_ReportError(t *testing.T) {
	var response strings.Builder
	cfg := makeConfig(t.TempDir(), &response)
	if cfg.ErrorHandler == nil {
		t.Fatal("the core has nowhere to say what it carried on past")
	}

	cfg.ErrorHandler(errors.New("the schedules are worked out at every launch"))
	if !strings.Contains(response.String(), "numen: the schedules are worked out at every launch") {
		t.Errorf("what the core said came out as %q", response.String())
	}
}
