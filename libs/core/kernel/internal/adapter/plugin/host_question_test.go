package plugin_test

import (
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire/wiretest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
)

func TestAPluginStartedForAQuestionStopsWhenTheQuestionEnds(t *testing.T) {
	host := newAtlasHost(t, testsupport.NewClock(), 0)
	if _, err := host.Activation.EnsureActiveForQuestion(t.Context(), atlasPlugin, getTestHolder()); err != nil {
		t.Fatal(err)
	}
	if state := supervisorOf(host, atlasPlugin).GetState(); state != lifecycle.StateActive {
		t.Fatalf("state while the question is open = %v, want active", state)
	}

	host.Activation.ReleaseQuestion(atlasPlugin, getTestHolder())

	wiretest.WaitUntil(t, "the plugin asked only a question to stop", func() bool {
		return supervisorOf(host, atlasPlugin).GetState() == lifecycle.StateDormant
	})
}

func TestAQuestionToAPluginInUseKeepsItsIdleCountdown(t *testing.T) {
	clock := testsupport.NewClock()
	host := newAtlasHost(t, clock, 0)
	startAtlas(t, host)
	host.Activation.Release(atlasPlugin, getTestHolder())

	clock.Advance(supervision.IdleThreshold - time.Second)
	if _, isRunning := host.Activation.AcquireRunningForQuestion(atlasPlugin, getTestHolder()); !isRunning {
		t.Fatal("the running plugin was not held for the question")
	}
	host.Activation.ReleaseQuestion(atlasPlugin, getTestHolder())

	clock.Advance(time.Second)
	if evicted := host.Teardown.EvictIdle(t.Context(), supervision.IdleThreshold, releaseGrace); !slices.Equal(evicted, []string{atlasPlugin}) {
		t.Errorf("evicted %v at the idle limit of the last use, want [%s]", evicted, atlasPlugin)
	}
}

func TestAUseDuringAQuestionKeepsThePluginForTheIdleLimit(t *testing.T) {
	host := newAtlasHost(t, testsupport.NewClock(), 0)
	if _, err := host.Activation.EnsureActiveForQuestion(t.Context(), atlasPlugin, getTestHolder()); err != nil {
		t.Fatal(err)
	}
	startAtlas(t, host)
	host.Activation.Release(atlasPlugin, getTestHolder())
	host.Activation.ReleaseQuestion(atlasPlugin, getTestHolder())

	if evicted := host.Teardown.EvictIdle(t.Context(), supervision.IdleThreshold, releaseGrace); len(evicted) != 0 {
		t.Errorf("evicted %v, want none before the idle limit", evicted)
	}
	if state := supervisorOf(host, atlasPlugin).GetState(); state != lifecycle.StateIdle {
		t.Errorf("state after the use = %v, want idle", state)
	}
}
