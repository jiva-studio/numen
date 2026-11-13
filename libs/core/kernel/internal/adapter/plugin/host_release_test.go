package plugin_test

import (
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	catalogmanifest "github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const atlasPlugin = "org.example.atlas"

const releaseGrace = 50 * time.Millisecond

func newAtlasHost(t *testing.T, clock *testsupport.ManualClock, idleStopSeconds int) *container.PluginRoles {
	t.Helper()
	return newEventHostWithClock(t, clock,
		map[string]catalogmanifest.Manifest{atlasPlugin: {Backend: catalogmanifest.Backend{IdleStopSeconds: idleStopSeconds}}},
		map[string]int{atlasPlugin: servePlugin(t, nil)},
	)
}

func startAtlas(t *testing.T, host *container.PluginRoles) {
	t.Helper()
	if _, err := host.Activation.EnsureActive(t.Context(), atlasPlugin, getTestHolder()); err != nil {
		t.Fatalf("EnsureActive failed: %v", err)
	}
}

func TestAReleasedPluginWithoutALeaseStopsAtOnce(t *testing.T) {
	host := newAtlasHost(t, testsupport.NewClock(), 0)
	startAtlas(t, host)
	host.Activation.Release(atlasPlugin, getTestHolder())

	host.Teardown.ReleaseProcess(t.Context(), atlasPlugin)

	if state := supervisorOf(host, atlasPlugin).GetState(); state != lifecycle.StateDormant {
		t.Errorf("state after the release = %v, want dormant", state)
	}
}

func TestAReleasedPluginWithALeaseStopsWhenTheLeaseIsReturned(t *testing.T) {
	host := newAtlasHost(t, testsupport.NewClock(), 0)
	startAtlas(t, host)

	host.Teardown.ReleaseProcess(t.Context(), atlasPlugin)
	if state := supervisorOf(host, atlasPlugin).GetState(); state != lifecycle.StateActive {
		t.Fatalf("state while the call is open = %v, want active", state)
	}
	host.Activation.Release(atlasPlugin, getTestHolder())

	wiretest.WaitUntil(t, "the released plugin to stop", func() bool {
		return supervisorOf(host, atlasPlugin).GetState() == lifecycle.StateDormant
	})
}

func TestAUseAfterTheReleaseKeepsThePluginForTheIdleLimit(t *testing.T) {
	host := newAtlasHost(t, testsupport.NewClock(), 0)
	startAtlas(t, host)
	host.Teardown.ReleaseProcess(t.Context(), atlasPlugin)

	startAtlas(t, host)
	host.Activation.Release(atlasPlugin, getTestHolder())
	host.Activation.Release(atlasPlugin, getTestHolder())

	if evicted := host.Teardown.EvictIdle(t.Context(), supervision.IdleThreshold, releaseGrace); len(evicted) != 0 {
		t.Errorf("evicted %v, want none before the idle limit", evicted)
	}
}

func TestAPluginStopsAfterTheIdleLimitOfItsManifest(t *testing.T) {
	clock := testsupport.NewClock()
	host := newAtlasHost(t, clock, 600)
	startAtlas(t, host)
	host.Activation.Release(atlasPlugin, getTestHolder())

	clock.Advance(supervision.IdleThreshold + time.Second)
	if evicted := host.Teardown.EvictIdle(t.Context(), supervision.IdleThreshold, releaseGrace); len(evicted) != 0 {
		t.Fatalf("evicted %v after the default limit, want none before 600 seconds", evicted)
	}

	clock.Advance(600*time.Second - supervision.IdleThreshold)
	if evicted := host.Teardown.EvictIdle(t.Context(), supervision.IdleThreshold, releaseGrace); !slices.Equal(evicted, []string{atlasPlugin}) {
		t.Errorf("evicted %v after 600 seconds, want [%s]", evicted, atlasPlugin)
	}
}
