package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

const acpManifest = `{
	"id": "md.numen.agents.acp",
	"name": "ACP Agents",
	"version": "1.0.0",
	"targets": ["desktop.editor"],
	"settings": [{"key": "executablePath", "type": "file", "multiple": true, "title": "Executable Paths", "default": ""}]
}`

func TestExecutablePathHasNoDefaultOrOptionsFromThePath(t *testing.T) {
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "gemini"), []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin)
	root := t.TempDir()
	dir := filepath.Join(root, "acp")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.json"), []byte(acpManifest), 0o600); err != nil {
		t.Fatal(err)
	}
	host := NewHost(testsupport.GetMachine(), "desktop.editor", root)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatal(err)
	}

	for _, info := range host.Registry.GetPluginInfos() {
		for _, s := range info.Settings {
			if s.DefaultJSON != `""` || len(s.Options) != 0 {
				t.Errorf("%s: default %s, options %+v; want the empty default of the manifest and no options", s.Key, s.DefaultJSON, s.Options)
			}
		}
	}
	if got := host.Settings.GetPluginSettings("md.numen.agents.acp")["executablePath"]; got != "" {
		t.Errorf("executablePath = %v, want the empty path of the manifest", got)
	}
}
