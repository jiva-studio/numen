package supervision

import (
	"slices"
	"testing"
	"time"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const (
	atlasID   = "org.example.atlas"
	mapsID    = "org.example.maps"
	dormantID = "org.example.dormant"
)

var (
	useLease      = lifecycle.Lease{Kind: lifecycle.UseLease, Holder: plugindomain.Holder{Who: plugindomain.WhoWindow, Why: "open view"}}
	questionLease = lifecycle.Lease{Kind: lifecycle.QuestionLease, Holder: plugindomain.Holder{Who: plugindomain.WhoSettings, Why: "list tools"}}
)

func getStartTime() time.Time { return time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC) }

func startInProcess(t *testing.T, sups *Supervisors, ids ...string) {
	t.Helper()
	for _, id := range ids {
		sup := sups.Register(manifest.Discovered{Manifest: manifest.Manifest{ID: id}})
		sup.SetActivating(plugindomain.Holder{})
		endpoint := transport.Endpoint{Host: "127.0.0.1", Port: 4100, Token: id}
		if err := sups.AttachInProcess(id, sup, endpoint, useLease, getStartTime()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestListProcessesReturnsTheRunningPluginsWithTheIdleStopTime(t *testing.T) {
	sups := NewSupervisors(func(error) {}, func(string) {})
	sups.Register(manifest.Discovered{Manifest: manifest.Manifest{ID: dormantID}})
	startInProcess(t, sups, mapsID, atlasID)
	sups.Release(mapsID, mapsID, useLease.Holder, getStartTime().Add(time.Minute))

	want := []plugindomain.Process{
		{PluginID: atlasID, StartTime: getStartTime(), LastUseTime: getStartTime(), LeaseCount: 1, IsInProcess: true, Holders: []plugindomain.Holder{useLease.Holder}},
		{
			PluginID: mapsID, StartTime: getStartTime(), LastUseTime: getStartTime().Add(time.Minute), IsInProcess: true,
			IdleStopTime: getStartTime().Add(time.Minute + IdleThreshold),
		},
	}
	if got := NewRunningProcesses(sups).ListProcesses(); !slices.EqualFunc(got, want, plugindomain.IsSameProcess) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestAWatchIsSignaledWhenAPluginReturnsALease(t *testing.T) {
	sups := NewSupervisors(func(error) {}, func(string) {})
	startInProcess(t, sups, mapsID)
	watch := NewRunningProcesses(sups).WatchProcesses(t.Context())

	sups.Release(mapsID, mapsID, useLease.Holder, getStartTime().Add(time.Minute))

	select {
	case <-watch:
	default:
		t.Fatal("the watch was not signaled")
	}
}

func TestSignalsTheWatchHasNotReceivedAreMergedIntoOne(t *testing.T) {
	sups := NewSupervisors(func(error) {}, func(string) {})
	startInProcess(t, sups, mapsID)
	watch := NewRunningProcesses(sups).WatchProcesses(t.Context())

	sups.Release(mapsID, mapsID, useLease.Holder, getStartTime().Add(time.Minute))
	startInProcess(t, sups, atlasID)
	<-watch

	select {
	case <-watch:
		t.Fatal("a second signal was queued")
	default:
	}
}

func TestTheIdleStopTimeFollowsTheIdleLimitOfTheManifest(t *testing.T) {
	sups := NewSupervisors(func(error) {}, func(string) {})
	sup := sups.Register(manifest.Discovered{Manifest: manifest.Manifest{ID: atlasID, Backend: manifest.Backend{Entrypoint: "bin/atlas", IdleStopSeconds: 600}}})
	sup.SetActivating(plugindomain.Holder{})
	if err := sups.AttachInProcess(atlasID, sup, transport.Endpoint{Host: "127.0.0.1", Port: 4100, Token: atlasID}, useLease, getStartTime()); err != nil {
		t.Fatal(err)
	}
	sups.Release(atlasID, atlasID, useLease.Holder, getStartTime().Add(time.Minute))

	got := NewRunningProcesses(sups).ListProcesses()

	if len(got) != 1 || !got[0].IdleStopTime.Equal(getStartTime().Add(time.Minute+600*time.Second)) {
		t.Errorf("got %+v, want the idle stop 600 seconds after the last use", got)
	}
}

func TestAReleasedPluginIsDueToStopAtItsLastUse(t *testing.T) {
	sups := NewSupervisors(func(error) {}, func(string) {})
	startInProcess(t, sups, mapsID)
	if targets := takeReleasedTarget(sups, mapsID, getStartTime()); len(targets) != 0 {
		t.Fatalf("a plugin that holds a lease was taken to stop: %+v", targets)
	}
	sups.Release(mapsID, mapsID, useLease.Holder, getStartTime().Add(time.Minute))

	got := NewRunningProcesses(sups).ListProcesses()

	if len(got) != 1 || !got[0].IdleStopTime.Equal(getStartTime().Add(time.Minute)) {
		t.Errorf("got %+v, want the idle stop at the last use", got)
	}
}

func TestAPluginOnlyAskedAQuestionIsDueToStopAtItsStart(t *testing.T) {
	sups := NewSupervisors(func(error) {}, func(string) {})
	sup := sups.Register(manifest.Discovered{Manifest: manifest.Manifest{ID: atlasID}})
	sup.SetActivating(plugindomain.Holder{})
	if err := sups.AttachInProcess(atlasID, sup, transport.Endpoint{Host: "127.0.0.1", Port: 4100, Token: atlasID}, questionLease, getStartTime()); err != nil {
		t.Fatal(err)
	}
	sups.ReturnHold(atlasID, sup, questionLease, getStartTime().Add(time.Minute))

	want := []plugindomain.Process{{
		PluginID: atlasID, StartTime: getStartTime(), LastUseTime: getStartTime(), IsInProcess: true, IdleStopTime: getStartTime(),
	}}
	if got := NewRunningProcesses(sups).ListProcesses(); !slices.EqualFunc(got, want, plugindomain.IsSameProcess) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestAQuestionDoesNotCancelTheReleaseOfAProcess(t *testing.T) {
	sups := NewSupervisors(func(error) {}, func(string) {})
	startInProcess(t, sups, mapsID)
	takeReleasedTarget(sups, mapsID, getStartTime())
	sup, _ := sups.Get(mapsID)

	if _, isRunning := sups.AcquireRunning(mapsID, sup, questionLease, getStartTime()); !isRunning {
		t.Fatal("the running plugin was not held for the question")
	}
	sups.ReturnHold(mapsID, sup, questionLease, getStartTime())
	sups.ReturnHold(mapsID, sup, useLease, getStartTime())

	if targets := takeIdleTargets(sups, IdleThreshold, getStartTime()); len(targets) != 1 || targets[0].ID != mapsID {
		t.Errorf("got %+v, want the released plugin to stop", targets)
	}
}

func TestListProcessesReturnsTheStarterAndTheHoldersOfEachPlugin(t *testing.T) {
	starter := lifecycle.Lease{Kind: lifecycle.UseLease, Holder: plugindomain.Holder{Who: plugindomain.WhoIndexing, Why: "embed notes"}}
	asker := lifecycle.Lease{Kind: lifecycle.UseLease, Holder: plugindomain.Holder{PluginID: mapsID, Why: "call tool"}}
	sups := NewSupervisors(func(error) {}, func(string) {})
	sup := sups.Register(manifest.Discovered{Manifest: manifest.Manifest{ID: atlasID}})
	if claim := sups.Begin(atlasID, sup, starter, getStartTime()); !claim.Started {
		t.Fatalf("the dormant plugin was not started: %+v", claim)
	}
	if err := sups.AttachInProcess(atlasID, sup, transport.Endpoint{Host: "127.0.0.1", Port: 4100, Token: atlasID}, starter, getStartTime()); err != nil {
		t.Fatal(err)
	}
	sups.AcquireRunning(atlasID, sup, asker, getStartTime())
	sups.AcquireRunning(atlasID, sup, questionLease, getStartTime())
	sups.ReturnHold(atlasID, sup, starter, getStartTime())

	got := NewRunningProcesses(sups).ListProcesses()

	if len(got) != 1 || got[0].StartedBy != starter.Holder {
		t.Fatalf("got %+v, want the plugin started by %+v", got, starter.Holder)
	}
	if want := []plugindomain.Holder{asker.Holder, questionLease.Holder}; !slices.Equal(got[0].Holders, want) {
		t.Errorf("holders %+v, want %+v", got[0].Holders, want)
	}
}
