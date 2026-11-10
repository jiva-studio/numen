package service

import "testing"

func TestExecutablePathIsReadFromTheCamelCaseKey(t *testing.T) {
	s := newAgentServer(false)
	s.applyConfig(map[string]string{"executablePath": "  /opt/bin/sample-agent  "})

	if got := s.getExecutablePath(); got != "/opt/bin/sample-agent" {
		t.Errorf("executable path = %q, want /opt/bin/sample-agent", got)
	}
}

func TestExecutablePathIsNotReadFromTheSnakeCaseKey(t *testing.T) {
	s := newAgentServer(false)
	s.applyConfig(map[string]string{"executable_path": "/opt/bin/sample-agent"})

	if got := s.getExecutablePath(); got != "" {
		t.Errorf("executable path = %q, want none", got)
	}
}
