package plugin_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func newDiscoveredHost(t *testing.T) *plugin.Host {
	t.Helper()
	tmp := t.TempDir()
	wiretest.CreateTestPluginTree(t, tmp)
	host := plugin.NewHost(testsupport.GetMachine(), "desktop.editor", tmp)
	if err := host.Discovery.Discover(); err != nil {
		t.Fatalf("Discover failed: %v", err)
	}
	return host
}

// TestHelperProcessIdles is the body of the child process the launcher starts; it does nothing in a normal run.
func TestHelperProcessIdles(t *testing.T) { wiretest.IdleChild(t) }
