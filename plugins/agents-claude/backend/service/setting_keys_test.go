package service

import "testing"

func TestExecutablePathIsReadFromTheCamelCaseKey(t *testing.T) {
	s := &agentServer{}
	s.applyConfig(map[string]string{"executablePath": "  /opt/bin/sample-agent  "})

	if s.executablePath != "/opt/bin/sample-agent" {
		t.Errorf("executable path = %q, want /opt/bin/sample-agent", s.executablePath)
	}
}

func TestExecutablePathIsNotReadFromTheSnakeCaseKey(t *testing.T) {
	s := &agentServer{}
	s.applyConfig(map[string]string{"executable_path": "/opt/bin/sample-agent"})

	if s.executablePath != "" {
		t.Errorf("executable path = %q, want none", s.executablePath)
	}
}

func TestHooksAndSkillsAreReadFromTheCamelCaseKey(t *testing.T) {
	s := &agentServer{}
	s.applyConfig(map[string]string{"readsHooksAndSkills": "true"})

	if !s.mayReadHooks {
		t.Error("the program does not read hooks and skills while readsHooksAndSkills is true")
	}
}

func TestHooksAndSkillsAreNotReadFromTheSnakeCaseKey(t *testing.T) {
	s := &agentServer{}
	s.applyConfig(map[string]string{"reads_hooks_and_skills": "true"})

	if s.mayReadHooks {
		t.Error("the program reads hooks and skills while only reads_hooks_and_skills is true")
	}
}
