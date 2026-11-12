package settings

import "testing"

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
