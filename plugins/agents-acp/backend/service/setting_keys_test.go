package service

import (
	"slices"
	"testing"
)

func TestExecutablePathsAreReadFromTheCamelCaseKey(t *testing.T) {
	s := &agentServer{}
	s.applyConfig(map[string]string{"executablePath": "/opt/bin/first-acp\n/opt/bin/second-acp"})

	if want := []string{"/opt/bin/first-acp", "/opt/bin/second-acp"}; !slices.Equal(s.executablePaths, want) {
		t.Errorf("executable paths = %v, want %v", s.executablePaths, want)
	}
}

func TestExecutablePathsAreNotReadFromTheSnakeCaseKey(t *testing.T) {
	s := &agentServer{}
	s.applyConfig(map[string]string{"executable_path": "/opt/bin/first-acp"})

	if len(s.executablePaths) != 0 {
		t.Errorf("executable paths = %v, want none", s.executablePaths)
	}
}
