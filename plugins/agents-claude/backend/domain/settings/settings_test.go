package settings

import "testing"

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
