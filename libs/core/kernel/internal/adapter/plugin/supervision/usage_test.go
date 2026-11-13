package supervision

import (
	"slices"
	"testing"
	"time"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const (
	atlasID   = "org.example.atlas"
	mapsID    = "org.example.maps"
	dormantID = "org.example.dormant"
)

func getStartTime() time.Time { return time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC) }

func startInProcess(t *testing.T, sups *Supervisors, ids ...string) {
	t.Helper()
	for _, id := range ids {
		sup := sups.Register(manifest.Discovered{Manifest: manifest.Manifest{ID: id}})
		sup.SetActivating()
		endpoint := transport.Endpoint{Host: "127.0.0.1", Port: 4100, Token: id}
		if err := sups.AttachInProcess(id, sup, endpoint, getStartTime()); err != nil {
			t.Fatal(err)
		}
	}
}

func TestListProcessesReturnsTheRunningPluginsWithTheIdleStopTime(t *testing.T) {
	sups := NewSupervisors(func(error) {}, func(string) {})
	sups.Register(manifest.Discovered{Manifest: manifest.Manifest{ID: dormantID}})
	startInProcess(t, sups, mapsID, atlasID)
	sups.Release(mapsID, mapsID, getStartTime().Add(time.Minute))

	want := []plugindomain.Process{
		{PluginID: atlasID, StartTime: getStartTime(), LastUseTime: getStartTime(), LeaseCount: 1, IsInProcess: true},
		{
			PluginID: mapsID, StartTime: getStartTime(), LastUseTime: getStartTime().Add(time.Minute), IsInProcess: true,
			IdleStopTime: getStartTime().Add(time.Minute + IdleThreshold),
		},
	}
	if got := NewRunningProcesses(sups).ListProcesses(); !slices.Equal(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestAWatchIsSignaledWhenAPluginReturnsALease(t *testing.T) {
	sups := NewSupervisors(func(error) {}, func(string) {})
	startInProcess(t, sups, mapsID)
	watch := NewRunningProcesses(sups).WatchProcesses(t.Context())

	sups.Release(mapsID, mapsID, getStartTime().Add(time.Minute))

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

	sups.Release(mapsID, mapsID, getStartTime().Add(time.Minute))
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
	sup.SetActivating()
	if err := sups.AttachInProcess(atlasID, sup, transport.Endpoint{Host: "127.0.0.1", Port: 4100, Token: atlasID}, getStartTime()); err != nil {
		t.Fatal(err)
	}
	sups.Release(atlasID, atlasID, getStartTime().Add(time.Minute))

	got := NewRunningProcesses(sups).ListProcesses()

	if len(got) != 1 || !got[0].IdleStopTime.Equal(getStartTime().Add(time.Minute+600*time.Second)) {
		t.Errorf("got %+v, want the idle stop 600 seconds after the last use", got)
	}
}

func TestAReleasedPluginIsDueToStopAtItsLastUse(t *testing.T) {
	sups := NewSupervisors(func(error) {}, func(string) {})
	startInProcess(t, sups, mapsID)
	sup, _ := sups.Get(mapsID)
	sup.MarkReleased()
	sups.Release(mapsID, mapsID, getStartTime().Add(time.Minute))

	got := NewRunningProcesses(sups).ListProcesses()

	if len(got) != 1 || !got[0].IdleStopTime.Equal(getStartTime().Add(time.Minute)) {
		t.Errorf("got %+v, want the idle stop at the last use", got)
	}
}
