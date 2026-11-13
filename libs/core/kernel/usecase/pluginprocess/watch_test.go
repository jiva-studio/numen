package pluginprocess_test

import (
	"context"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/pluginprocess"
)

const (
	atlasID = "org.example.atlas"
	mapsID  = "org.example.maps"
)

func getStartTime() time.Time { return time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC) }

type processes struct {
	mu      sync.Mutex
	list    []plugin.Process
	changes chan struct{}
}

func newProcesses(list ...plugin.Process) *processes {
	return &processes{list: list, changes: make(chan struct{}, 1)}
}

func (p *processes) ListProcesses() []plugin.Process {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.list)
}

func (p *processes) WatchProcesses(context.Context) <-chan struct{} { return p.changes }

func (p *processes) set(list ...plugin.Process) {
	p.mu.Lock()
	p.list = list
	p.mu.Unlock()
	p.changes <- struct{}{}
}

type health struct{ items chan []plugin.HealthItem }

func newHealth(items ...plugin.HealthItem) *health {
	watch := make(chan []plugin.HealthItem, 1)
	watch <- items
	return &health{items: watch}
}

func (h *health) Watch(context.Context) <-chan []plugin.HealthItem { return h.items }

type plugins map[string]string

func (p plugins) GetPlugin(id string) (manifest.Discovered, bool) {
	name, ok := p[id]
	return manifest.Discovered{Manifest: manifest.Manifest{ID: id, Name: name}}, ok
}

func getNames() plugins { return plugins{atlasID: "Atlas", mapsID: "Maps"} }

type clock struct{}

func (clock) Now() time.Time { return getStartTime().Add(10 * time.Minute) }

func startWatch(t *testing.T, p *processes, h *health) <-chan plugin.ProcessList {
	t.Helper()
	watch := pluginprocess.NewWatchProcesses(p, h, getNames(), clock{})
	watch.Interval = 0
	return watch.Execute(t.Context())
}

func receive(t *testing.T, watch <-chan plugin.ProcessList) []plugin.ProcessStatus {
	t.Helper()
	select {
	case list := <-watch:
		return list.Processes
	case <-time.After(time.Second):
		t.Fatal("no list arrived")
		return nil
	}
}

func TestTheFirstListNamesEachPluginAndItsWorstHealth(t *testing.T) {
	maps := plugin.Process{PluginID: mapsID, StartTime: getStartTime(), LastUseTime: getStartTime(), LeaseCount: 1}
	idle := plugin.Process{PluginID: atlasID, StartTime: getStartTime(), LastUseTime: getStartTime(), IdleStopTime: getStartTime().Add(3 * time.Minute)}
	h := newHealth(
		plugin.HealthItem{PluginID: atlasID, Key: "tiles", Severity: plugin.HealthWarning, Text: "The tiles are old"},
		plugin.HealthItem{PluginID: atlasID, Key: "start", Severity: plugin.HealthError, Text: "The atlas did not start"},
	)

	got := receive(t, startWatch(t, newProcesses(idle, maps), h))

	want := []plugin.ProcessStatus{
		{Process: idle, PluginName: "Atlas", Health: plugin.HealthError, HealthText: "The atlas did not start"},
		{Process: maps, PluginName: "Maps", Health: plugin.HealthOK},
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestAPluginThatStopsLeavesTheList(t *testing.T) {
	maps := plugin.Process{PluginID: mapsID, StartTime: getStartTime(), LastUseTime: getStartTime(), LeaseCount: 1}
	atlas := plugin.Process{PluginID: atlasID, StartTime: getStartTime(), LastUseTime: getStartTime(), LeaseCount: 1}
	p := newProcesses(atlas, maps)
	watch := startWatch(t, p, newHealth())
	receive(t, watch)

	p.set(maps)

	if got := receive(t, watch); len(got) != 1 || got[0].PluginID != mapsID {
		t.Errorf("got %+v, want maps alone", got)
	}
}

func TestAHealthItemReportedLaterIsSent(t *testing.T) {
	maps := plugin.Process{PluginID: mapsID, StartTime: getStartTime(), LastUseTime: getStartTime(), LeaseCount: 1}
	h := newHealth()
	watch := startWatch(t, newProcesses(maps), h)
	receive(t, watch)

	h.items <- []plugin.HealthItem{{PluginID: mapsID, Key: "tiles", Severity: plugin.HealthWarning, Text: "The tiles are old"}}

	if got := receive(t, watch); len(got) != 1 || got[0].Health != plugin.HealthWarning || got[0].HealthText != "The tiles are old" {
		t.Errorf("got %+v, want the warning of maps", got)
	}
}

func TestAChangeThatLeavesTheListAsItWasSendsNothing(t *testing.T) {
	maps := plugin.Process{PluginID: mapsID, StartTime: getStartTime(), LastUseTime: getStartTime(), LeaseCount: 1}
	p := newProcesses(maps)
	watch := startWatch(t, p, newHealth())
	receive(t, watch)

	p.set(maps)
	p.set(maps, plugin.Process{PluginID: atlasID, StartTime: getStartTime(), LastUseTime: getStartTime(), LeaseCount: 1})

	if got := receive(t, watch); len(got) != 2 {
		t.Errorf("got %+v, want the list with the atlas, the unchanged one skipped", got)
	}
}

func TestTheListIsClosedWhenTheWatchEnds(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	watch := pluginprocess.NewWatchProcesses(newProcesses(), newHealth(), getNames(), clock{})
	watch.Interval = 0
	lists := watch.Execute(ctx)
	receive(t, lists)

	cancel()

	select {
	case _, isOpen := <-lists:
		if isOpen {
			t.Error("a list arrived after the end")
		}
	case <-time.After(time.Second):
		t.Error("the list was not closed")
	}
}

func TestEachListCarriesTheTimeItWasTaken(t *testing.T) {
	watch := startWatch(t, newProcesses(), newHealth())

	select {
	case list := <-watch:
		if want := getStartTime().Add(10 * time.Minute); !list.Time.Equal(want) {
			t.Errorf("got %v, want %v", list.Time, want)
		}
	case <-time.After(time.Second):
		t.Fatal("no list arrived")
	}
}
