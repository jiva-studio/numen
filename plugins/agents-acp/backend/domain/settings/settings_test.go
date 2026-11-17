package settings

import (
	"testing"
	"time"
)

func TestRequestTimeoutIsReadInSecondsAndInvalidValuesLeaveNone(t *testing.T) {
	var s Settings
	for value, want := range map[string]time.Duration{
		"45":  45 * time.Second,
		"0.5": 500 * time.Millisecond,
	} {
		s.Apply(map[string]string{"requestTimeout": value})
		if got := s.RequestTimeout(); got != want {
			t.Errorf("value %q: request timeout = %v, want %v", value, got, want)
		}
	}
	// 9223372035 s is the largest whole number of seconds accepted; float64 seconds hold no exact nanoseconds there.
	s.Apply(map[string]string{"requestTimeout": "9223372035"})
	if got, want := s.RequestTimeout(), 9223372035*time.Second; got-want > time.Microsecond || want-got > time.Microsecond {
		t.Errorf("value 9223372035: request timeout = %v, want about %v", got, want)
	}
	for _, value := range []string{"", "soon", "0", "-0", "-3", "NaN", "Inf", "9223372036"} {
		s.Apply(map[string]string{"requestTimeout": value})
		if got := s.RequestTimeout(); got != 0 {
			t.Errorf("value %q: request timeout = %v, want none", value, got)
		}
	}
}
