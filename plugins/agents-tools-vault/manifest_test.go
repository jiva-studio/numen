package vaulttools_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/plugintest"
	vaulttools "github.com/jiva-studio/numen/plugins/agents-tools-vault"
)

func TestManifest(t *testing.T) {
	m := plugintest.ReadManifest(t, vaulttools.Manifest)

	if m.ID != "md.numen.agents.tools.vault" {
		t.Errorf("manifest id = %q, want %q", m.ID, "md.numen.agents.tools.vault")
	}
	if m.Backend.Entrypoint != "bin/agents-tools-vault" {
		t.Errorf("manifest entrypoint = %q, want %q", m.Backend.Entrypoint, "bin/agents-tools-vault")
	}
	if len(m.Capabilities) == 0 || m.Capabilities[0] != "tools" {
		t.Errorf("manifest capabilities = %v, want [tools]", m.Capabilities)
	}
}
