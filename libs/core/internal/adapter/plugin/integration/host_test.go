package integration

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
)

func newDiscoveredHost(t *testing.T) *plugin.Host {
	t.Helper()
	tmp := t.TempDir()
	plugintest.CreateTestPluginTree(t, tmp)
	host := plugin.NewHost("desktop.editor", tmp)
	if err := host.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	return host
}

// TestHelperProcessIdles is the body of the child process the launcher starts; it does nothing in a normal run.
func TestHelperProcessIdles(t *testing.T) { plugintest.IdleChild(t) }
