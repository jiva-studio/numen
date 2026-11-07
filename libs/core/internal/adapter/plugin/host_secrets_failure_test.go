package plugin

import (
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/plugintest"
	pluginport "github.com/jiva-studio/numen/libs/core/port/pluginport"
)

// collectFailures records the failures the host passes on.
type collectFailures struct {
	mu       sync.Mutex
	failures []pluginport.PluginFailure
}

func (c *collectFailures) add(failure pluginport.PluginFailure) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failures = append(c.failures, failure)
}

func (c *collectFailures) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.failures)
}

// A plugin whose Configure fails does not fail the operation that changed its list. The plugin is
// marked as failed, once.
func TestAFailingConfigureMarksThePluginAndDoesNotFailTheOperation(t *testing.T) {
	host, probe := newSecretHost(t, newKeys(map[string]string{}))
	failed := &collectFailures{}
	host.SetOnPluginFailed(failed.add)
	probe.mu.Lock()
	probe.unimplemented = true
	probe.mu.Unlock()

	for range 2 {
		err := host.ApplySecretChange(t.Context(), func() ([]string, error) { return []string{secretPluginID}, nil })
		if err != nil {
			t.Errorf("the operation returned %v, want nil", err)
		}
	}
	plugintest.WaitUntil(t, "the failure is passed on", func() bool { return failed.count() > 0 })
	plugintest.WaitUntil(t, "both calls were made", func() bool {
		probe.mu.Lock()
		defer probe.mu.Unlock()
		return len(probe.calls) >= 1
	})
	if failed.count() != 1 {
		t.Errorf("%d failures passed on, want 1", failed.count())
	}
}

// A plugin whose Configure failed is configured again by the next refresh, whatever its list.
func TestAFailedConfigureIsRepeatedByTheNextRefresh(t *testing.T) {
	host, probe := newSecretHost(t, newKeys(map[string]string{"secret/" + secretID: fromKeychain}))
	host.SetOnPluginFailed(func(pluginport.PluginFailure) {})
	probe.mu.Lock()
	probe.unimplemented = true
	probe.mu.Unlock()

	for range 3 {
		if err := host.RefreshSecrets(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	probe.mu.Lock()
	calls := len(probe.calls)
	probe.mu.Unlock()
	if calls != 3 {
		t.Errorf("%d Configure calls over three refreshes of a failing plugin, want 3", calls)
	}
}
