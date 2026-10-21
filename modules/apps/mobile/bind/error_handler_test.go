package bind

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
	cfg := makeConfig(t.TempDir(), &response)
	if cfg.ErrorHandler == nil {
		t.Fatal("the core has nowhere to say what it carried on past")
	}

	cfg.ErrorHandler(errors.New("the schedules are worked out at every launch"))
	if !strings.Contains(response.String(), "numen: the schedules are worked out at every launch") {
		t.Errorf("what the core said came out as %q", response.String())
	}
}
