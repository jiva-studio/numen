package plugin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
)

// executablePathManifest is a plugin manifest with an executable path setting; %t is whether the
// setting takes several paths.
const executablePathManifest = `{
	"id": %q,
	"name": %q,
	"version": "1.0.0",
	"targets": ["desktop.editor"],
	"settings": [{"key": "executablePath", "type": "file", "multiple": %t, "title": "Executable Path", "default": ""}]
}`

const (
	acpPluginID   = "md.numen.agents.acp"
	otherPluginID = "md.numen.agents.other"
)

// newHostWithGemini returns a host of the ACP plugin, whose setting takes several paths, and another
// agent plugin of one path, on a PATH that holds an executable named gemini, and that executable.
func newHostWithGemini(t *testing.T) (*Host, string) {
	t.Helper()
	bin := t.TempDir()
	gemini := filepath.Join(bin, "gemini")
	if err := os.WriteFile(gemini, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)

	root := t.TempDir()
	for _, one := range []struct {
		dir, id, name string
		multiple      bool
	}{
		{"acp", acpPluginID, "ACP Agents", true},
		{"other", otherPluginID, "Other Agent", false},
	} {
		dir := filepath.Join(root, one.dir)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		manifest := []byte(fmt.Sprintf(executablePathManifest, one.id, one.name, one.multiple))
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), manifest, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	host := NewHost(testsupport.GetMachine(), "desktop.editor", root)
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
	if got := getExecutablePathDefault(t, host, otherPluginID); got != `""` {
		t.Errorf("other agent default = %s, want the empty path of its manifest", got)
	}
}

func TestACPAgentsFoundOnMachineAreSentToTheACPPluginOnly(t *testing.T) {
	host, gemini := newHostWithGemini(t)

	if got := host.GetPluginSettings(acpPluginID)[keyExecutablePath]; got != gemini {
		t.Errorf("ACP executablePath = %v, want %s", got, gemini)
	}
	if got := host.GetPluginSettings(otherPluginID)[keyExecutablePath]; got != "" {
		t.Errorf("other agent executablePath = %v, want the empty path of its manifest", got)
	}
}

// countLookups empties the ACP agents the host found, makes it search PATH through a counter, and
// returns the counter.
func countLookups(host *Host) *atomic.Int32 {
	host.forgetDiscoveredACPOptions()
	count := &atomic.Int32{}
	host.lookPath = func(name string) (string, error) {
		count.Add(1)
		return exec.LookPath(name)
	}
	return count
}

func TestACPAgentsAreSearchedOnceForEveryReadOfTheSettings(t *testing.T) {
	host, gemini := newHostWithGemini(t)
	lookups := countLookups(host)

	for range 3 {
		if got, want := getExecutablePathDefault(t, host, acpPluginID), fmt.Sprintf("%q", gemini); got != want {
			t.Fatalf("ACP default = %s, want %s", got, want)
		}
		if got := host.GetPluginSettings(acpPluginID)[keyExecutablePath]; got != gemini {
			t.Fatalf("ACP executablePath = %v, want %s", got, gemini)
		}
	}

	if got, want := lookups.Load(), int32(len(getAcpExecutableCandidates())); got != want {
		t.Errorf("searched PATH %d times, want %d: once per candidate", got, want)
	}
}

func TestACPAgentsAreSearchedAgainAfterASettingChanges(t *testing.T) {
	host, gemini := newHostWithGemini(t)
	lookups := countLookups(host)
	getExecutablePathDefault(t, host, acpPluginID)
	codex := filepath.Join(filepath.Dir(gemini), "codex-acp")
	if err := os.WriteFile(codex, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}

	if err := host.UpdatePluginSetting(t.Context(), otherPluginID, "theme", "dark"); err != nil {
		t.Fatal(err)
	}

	if got, want := host.GetPluginSettings(acpPluginID)[keyExecutablePath], codex+"\n"+gemini; got != want {
		t.Errorf("ACP executablePath = %v, want %s", got, want)
	}
	if got, want := lookups.Load(), int32(2*len(getAcpExecutableCandidates())); got != want {
		t.Errorf("searched PATH %d times, want %d: once before and once after the change", got, want)
	}
}
