package detection

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
)

func writeProgram(t *testing.T, folder, name string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the fixtures are shell scripts")
	}
	if err := os.WriteFile(filepath.Join(folder, name), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestTheAgentsOnThePathAreDetectedWithTheirPrograms(t *testing.T) {
	folder := t.TempDir()
	writeProgram(t, folder, "opencode")
	writeProgram(t, folder, "gemini")
	writeProgram(t, folder, "orchard-agent")
	t.Setenv("PATH", folder)

	found, err := New(exec.LookPath).DetectConnections(t.Context(), connection.Call{Type: "acp"})

	if err != nil || len(found) != 2 {
		t.Fatalf("DetectConnections = %+v, %v, want Gemini CLI and OpenCode", found, err)
	}
	if found[0].Name != "Gemini CLI" || found[0].Fields["program"] != "gemini" || found[1].Name != "OpenCode" || found[1].Fields["program"] != "opencode" {
		t.Errorf("found = %+v", found)
	}
}

func TestNothingIsDetectedWithoutAKnownAgentOnThePath(t *testing.T) {
	folder := t.TempDir()
	writeProgram(t, folder, "orchard-agent")
	t.Setenv("PATH", folder)

	for _, lookPath := range []func(string) (string, error){exec.LookPath, nil} {
		found, err := New(lookPath).DetectConnections(t.Context(), connection.Call{Type: "acp"})
		if err != nil || len(found) != 0 {
			t.Errorf("DetectConnections = %+v, %v, want none", found, err)
		}
	}
}
