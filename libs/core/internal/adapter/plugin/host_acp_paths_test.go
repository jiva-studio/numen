package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
)

const executablePathManifest = `{
	"id": %q,
	"name": %q,
	"version": "1.0.0",
	"targets": ["desktop.editor"],
	"settings": [{"key": "executable_path", "type": "file", "title": "Executable Path", "default": ""}]
}`

// newHostWithGemini returns a host of the ACP plugin and another agent plugin, on a PATH that holds
// an executable named gemini, and that executable.
func newHostWithGemini(t *testing.T) (*Host, string) {
	t.Helper()
	bin := t.TempDir()
	gemini := filepath.Join(bin, "gemini")
	if err := os.WriteFile(gemini, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)

	root := t.TempDir()
	for _, one := range []struct{ dir, id, name string }{
		{"acp", acpPluginID, "ACP Agents"},
		{"other", "md.numen.agents.other", "Other Agent"},
	} {
		dir := filepath.Join(root, one.dir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		manifest := []byte(fmt.Sprintf(executablePathManifest, one.id, one.name))
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), manifest, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	host := NewHost(testsupport.Machine, "desktop.editor", root)
	if err := host.Discover(); err != nil {
		t.Fatal(err)
	}
	return host, gemini
}

func getExecutablePathDefault(t *testing.T, host *Host, pluginID string) string {
	t.Helper()
	for _, info := range host.GetPluginInfos() {
		if info.ID != pluginID {
			continue
		}
		for _, s := range info.Settings {
			if s.Key == keyExecutablePath {
				return s.DefaultJSON
			}
		}
	}
	t.Fatalf("plugin %s has no executable path setting", pluginID)
	return ""
}

func TestACPAgentsFoundOnMachineAreTheDefaultOfTheACPPluginOnly(t *testing.T) {
	host, gemini := newHostWithGemini(t)

	if got, want := getExecutablePathDefault(t, host, acpPluginID), fmt.Sprintf("%q", gemini); got != want {
		t.Errorf("ACP default = %s, want %s", got, want)
	}
	if got := getExecutablePathDefault(t, host, "md.numen.agents.other"); got != `""` {
		t.Errorf("other agent default = %s, want the empty path of its manifest", got)
	}
}

func TestACPAgentsFoundOnMachineAreSentToTheACPPluginOnly(t *testing.T) {
	host, gemini := newHostWithGemini(t)

	if got := host.GetPluginSettings(acpPluginID)[keyExecutablePath]; got != gemini {
		t.Errorf("ACP executable_path = %v, want %s", got, gemini)
	}
	if got := host.GetPluginSettings("md.numen.agents.other")[keyExecutablePath]; got != "" {
		t.Errorf("other agent executable_path = %v, want the empty path of its manifest", got)
	}
}
