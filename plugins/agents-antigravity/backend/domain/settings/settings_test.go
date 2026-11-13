package settings

import (
	"testing"
	"time"
)

func TestExecutablePathIsReadFromTheCamelCaseKey(t *testing.T) {
	var s Settings
	s.Apply(map[string]string{"executablePath": "  /opt/bin/sample-agent  "})

	if got := s.ExecutablePath(); got != "/opt/bin/sample-agent" {
		t.Errorf("executable path = %q, want /opt/bin/sample-agent", got)
	}
}

func TestExecutablePathIsNotReadFromTheSnakeCaseKey(t *testing.T) {
	var s Settings
	s.Apply(map[string]string{"executable_path": "/opt/bin/sample-agent"})

	if got := s.ExecutablePath(); got != "" {
		t.Errorf("executable path = %q, want none", got)
	}
}

func TestUnrelatedValuesKeepTheExecutablePath(t *testing.T) {
	var s Settings
	s.Apply(map[string]string{"executablePath": "/opt/bin/sample-agent"})
	s.Apply(map[string]string{"unrelated": "val"})

	if got := s.ExecutablePath(); got != "/opt/bin/sample-agent" {
		t.Errorf("executable path = %q, want /opt/bin/sample-agent", got)
	}
}

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
