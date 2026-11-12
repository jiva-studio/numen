package settings

import (
	"testing"
	"time"
)

func TestExecutablePathIsReadFromTheCamelCaseKey(t *testing.T) {
	var s Settings
	s.Apply(map[string]string{"executablePath": "  /opt/bin/sample-agent  "})

	if got := s.Get().ExecutablePath; got != "/opt/bin/sample-agent" {
		t.Errorf("executable path = %q, want /opt/bin/sample-agent", got)
	}
}

func TestExecutablePathIsNotReadFromTheSnakeCaseKey(t *testing.T) {
	var s Settings
	s.Apply(map[string]string{"executable_path": "/opt/bin/sample-agent"})

	if got := s.Get().ExecutablePath; got != "" {
		t.Errorf("executable path = %q, want none", got)
	}
}

func TestHooksAndSkillsAreReadFromTheCamelCaseKey(t *testing.T) {
	var s Settings
	s.Apply(map[string]string{"readsHooksAndSkills": "true"})

	if !s.Get().MayReadHooks {
		t.Error("the program does not read hooks and skills while readsHooksAndSkills is true")
	}
}

func TestHooksAndSkillsAreNotReadFromTheSnakeCaseKey(t *testing.T) {
	var s Settings
	s.Apply(map[string]string{"reads_hooks_and_skills": "true"})

	if s.Get().MayReadHooks {
		t.Error("the program reads hooks and skills while only reads_hooks_and_skills is true")
	}
}

func TestApply_EmptyModelKeepsModel(t *testing.T) {
	var s Settings
	s.Apply(map[string]string{"model": "m1"})
	s.Apply(map[string]string{"model": ""})

	if got := s.Get().DefaultModel; got != "m1" {
		t.Errorf("default model = %q, want m1 kept when an empty model is configured", got)
	}
}

func TestRequestTimeoutIsReadInSecondsAndInvalidValuesLeaveNone(t *testing.T) {
	var s Settings
	s.Apply(map[string]string{"requestTimeout": "45"})
	if got := s.Get().RequestTimeout; got != 45*time.Second {
		t.Errorf("request timeout = %v, want 45s", got)
	}
	for _, value := range []string{"", "soon", "0", "-3", "NaN", "Inf"} {
		s.Apply(map[string]string{"requestTimeout": value})
		if got := s.Get().RequestTimeout; got != 0 {
			t.Errorf("value %q: request timeout = %v, want none", value, got)
		}
	}
}
