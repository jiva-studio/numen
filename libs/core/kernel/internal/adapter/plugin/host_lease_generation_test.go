package plugin_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func getLeaseCount(t *testing.T, sup *lifecycle.Supervisor) int {
	t.Helper()
	usage, _ := sup.GetUsage()
	return usage.LeaseCount
}

func TestAReleaseTakenOnAnEarlierProcessKeepsTheLeaseOfTheNewOne(t *testing.T) {
	host := newAtlasHost(t, testsupport.NewClock(), 0)
	startAtlas(t, host)
	first := endpointOf(host, atlasPlugin).Token
	host.Activation.Release(atlasPlugin, first, getTestHolder())
	host.Teardown.ReleaseProcess(t.Context(), atlasPlugin)
	if state := supervisorOf(host, atlasPlugin).GetState(); state != lifecycle.StateDormant {
		t.Fatalf("state after the release = %v, want dormant", state)
	}

	startAtlas(t, host)
	host.Activation.Release(atlasPlugin, first, getTestHolder())

	if count := getLeaseCount(t, supervisorOf(host, atlasPlugin)); count != 1 {
		t.Errorf("the new process holds %d leases after a release for the earlier one, want 1", count)
	}
}

func TestAStartWhoseConfigureCannotBeSentReturnsItsLease(t *testing.T) {
	host := newAtlasHost(t, testsupport.NewClock(), 0)
	delivery := plugin.GetDelivery(host.Config)
	if err := delivery.Lock(t.Context()); err != nil {
		t.Fatal(err)
	}
	defer delivery.Unlock()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	result := make(chan error, 1)
	go func() {
		_, err := host.Activation.EnsureActive(ctx, atlasPlugin, getTestHolder())
		result <- err
	}()
	wiretest.WaitUntil(t, "the process to start", func() bool { return endpointOf(host, atlasPlugin).Token != "" })

	cancel()

	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("EnsureActive = %v, want the end of the context", err)
	}
	if count := getLeaseCount(t, supervisorOf(host, atlasPlugin)); count != 0 {
		t.Errorf("the process holds %d leases after a start the caller gave up, want none", count)
	}
}
