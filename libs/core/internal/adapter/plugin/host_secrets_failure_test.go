package plugin

import (
	"sync"
	"testing"
)

// collectErrors records the errors the host reports.
type collectErrors struct {
	mu     sync.Mutex
	errors []error
}

func (c *collectErrors) add(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.errors = append(c.errors, err)
}

func (c *collectErrors) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.errors)
}

// A plugin whose Configure fails does not fail an operation that does not concern it: the failure
// is reported through the error handler.
func TestAFailingConfigureOfAnUnrelatedPluginDoesNotFailTheOperation(t *testing.T) {
	host, probe := newSecretHost(t, newKeys(map[string]string{}))
	reported := &collectErrors{}
	host.SetErrorHandler(reported.add)
	probe.mu.Lock()
	probe.unimplemented = true
	probe.mu.Unlock()

	err := host.ApplySecretChange(t.Context(), func() ([]string, error) { return nil, nil })
	if err != nil {
		t.Errorf("the operation returned %v, want nil", err)
	}
	if reported.count() != 1 {
		t.Errorf("%d failures reported, want 1", reported.count())
	}
}

// A plugin whose Configure failed with a list is not configured again until its list changes.
func TestAFailedConfigureIsNotRepeatedWhileTheListIsTheSame(t *testing.T) {
	host, probe := newSecretHost(t, newKeys(map[string]string{"secret/" + secretID: fromKeychain}))
	host.SetErrorHandler(func(error) {})
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
	if calls != 1 {
		t.Errorf("%d Configure calls over three refreshes of a failing plugin, want 1", calls)
	}

	host.SetSecretSource(allowOne(secretPluginID))
	if err := host.RefreshSecrets(t.Context()); err != nil {
		t.Fatal(err)
	}
	probe.mu.Lock()
	defer probe.mu.Unlock()
	if len(probe.calls) != 2 {
		t.Errorf("%d Configure calls after the list changed, want 2", len(probe.calls))
	}
}

// A plugin an operation names gets its failure reported, not returned.
func TestAFailingConfigureOfANamedPluginIsReportedNotReturned(t *testing.T) {
	host, probe := newSecretHost(t, newKeys(map[string]string{}))
	reported := &collectErrors{}
	host.SetErrorHandler(reported.add)
	probe.mu.Lock()
	probe.unimplemented = true
	probe.mu.Unlock()

	err := host.ApplySecretChange(t.Context(), func() ([]string, error) { return []string{secretPluginID}, nil })
	if err != nil {
		t.Errorf("the operation returned %v, want nil", err)
	}
	if reported.count() != 1 {
		t.Errorf("%d failures reported, want 1", reported.count())
	}
}
