package settings

import (
	"testing"
	"time"
)

func TestRequestTimeoutIsReadInSecondsAndInvalidValuesLeaveNone(t *testing.T) {
	var s Settings
	s.Apply(map[string]string{"requestTimeout": "45"})
	if got := s.RequestTimeout(); got != 45*time.Second {
		t.Errorf("request timeout = %v, want 45s", got)
	}
	for _, value := range []string{"", "soon", "0", "-3", "NaN", "Inf"} {
		s.Apply(map[string]string{"requestTimeout": value})
		if got := s.RequestTimeout(); got != 0 {
			t.Errorf("value %q: request timeout = %v, want none", value, got)
		}
	}
}
