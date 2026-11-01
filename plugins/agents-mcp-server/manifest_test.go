package agentmcpserver

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/plugintest"
)

func TestManifestValid(t *testing.T) {
	t.Parallel()

	manifest := plugintest.ReadManifest(t, Manifest)

	if manifest.ID != "md.numen.agents.mcp.server" {
		t.Errorf("got ID %q, want %q", manifest.ID, "md.numen.agents.mcp.server")
	}

	// The plugin serves no PluginToolService: the core requests the tools of every plugin with the tools capability
	// when an agent run starts.
	if slices.Contains(manifest.Capabilities, "tools") {
		t.Errorf("manifest declares the tools capability: %v", manifest.Capabilities)
	}

	if !slices.Equal(manifest.Services, []string{"md.numen.agents.mcp.server"}) {
		t.Errorf("got services %v, want %q", manifest.Services, "md.numen.agents.mcp.server")
	}
}
