package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/testsupport"
)

const camelCaseExecutablePathManifest = `{
	"id": "md.numen.agents.acp",
	"name": "ACP Agents",
	"version": "1.0.0",
	"targets": ["desktop.editor"],
	"settings": [{"key": "executablePath", "type": "file", "multiple": true, "title": "Executable Paths", "default": ""}]
}`

func newHostWithGeminiAndCamelCaseManifest(t *testing.T) (*Host, string) {
	t.Helper()
	bin := t.TempDir()
	gemini := filepath.Join(bin, "gemini")
	if err := os.WriteFile(gemini, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)

	root := t.TempDir()
	dir := filepath.Join(root, "acp")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(camelCaseExecutablePathManifest), 0o600); err != nil {
		t.Fatal(err)
	}
	host := NewHost(testsupport.GetMachine(), "desktop.editor", root)
	if err := host.Discover(); err != nil {
		t.Fatal(err)
	}
	return host, gemini
}

func TestACPExecutablePathIsFilledFromAgentsOnMachineAsCamelCaseKey(t *testing.T) {
	host, gemini := newHostWithGeminiAndCamelCaseManifest(t)

	settings := host.GetPluginSettings("md.numen.agents.acp")
	if got := settings["executablePath"]; got != gemini {
		t.Errorf("ACP executablePath = %v, want %s", got, gemini)
	}
	if got, found := settings["executable_path"]; found {
		t.Errorf("ACP settings carry executable_path = %v, want only executablePath", got)
	}
}

func TestACPPathsSettingIsRecognisedByTheCamelCaseKey(t *testing.T) {
	host, gemini := newHostWithGeminiAndCamelCaseManifest(t)

	found := false
	for _, info := range host.GetPluginInfos() {
		for _, s := range info.Settings {
			if s.Key != "executablePath" {
				continue
			}
			found = true
			if s.DefaultJSON != fmt.Sprintf("%q", gemini) {
				t.Errorf("ACP default = %s, want %q", s.DefaultJSON, gemini)
			}
		}
	}
	if !found {
		t.Error("the ACP plugin declares no executablePath setting")
	}
}
