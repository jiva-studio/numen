// Package pluginprocess lists the plugins that run, with the name and the worst health item of each,
// for a window to watch.
package pluginprocess

import (
	"context"
	"slices"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
)

// Processes is the plugins that run.
type Processes interface {
	// ListProcesses returns the plugins that run, in the order of their ids.
	ListProcesses() []plugin.Process
	// WatchProcesses signals each change of the plugins that run until the context ends. Signals not
	// received yet are merged into one.
	WatchProcesses(ctx context.Context) <-chan struct{}
}

// HealthItems keeps what the plugins and the core report about whether each plugin works.
type HealthItems interface {
	// Watch returns the items at once and on every change until the context ends, and is closed then.
	Watch(ctx context.Context) <-chan []plugin.HealthItem
}

// Plugins gives the manifest of a plugin, which names it.
type Plugins interface {
	GetPlugin(id string) (manifest.Discovered, bool)
}

// WatchProcesses lists the plugins that run each time they or their health change, with the time the
// list was taken. Interval is the shortest time between two lists; changes within it are sent together
// after it.
type WatchProcesses struct {
	processes Processes
	health    HealthItems
	plugins   Plugins
	clock     machineport.Clock
	Interval  time.Duration
}

// NewWatchProcesses returns the use case over the plugins that run, their health items, their manifests
// and the clock that stamps each list, sending at most one list a second.
func NewWatchProcesses(processes Processes, health HealthItems, plugins Plugins, clock machineport.Clock) *WatchProcesses {
	return &WatchProcesses{processes: processes, health: health, plugins: plugins, clock: clock, Interval: time.Second}
}

// Execute returns the plugins that run at once, and again after each change that alters them, until the
// context ends; the channel is closed then.
func (u *WatchProcesses) Execute(ctx context.Context) <-chan plugin.ProcessList {
	out := make(chan plugin.ProcessList, 1)
	run := &watchRun{use: u, out: out, changes: u.processes.WatchProcesses(ctx), health: u.health.Watch(ctx)}
	go run.sendLists(ctx)
	return out
}

// watchRun is one watch: the channels it reads, the health items it read last and the list it sent
// last.
type watchRun struct {
	use     *WatchProcesses
	out     chan<- plugin.ProcessList
	changes <-chan struct{}
	health  <-chan []plugin.HealthItem
	items   []plugin.HealthItem
	last    []plugin.ProcessStatus
}

func (r *watchRun) sendLists(ctx context.Context) {
	defer close(r.out)
	items, isOpen := <-r.health
	if !isOpen {
		return
	}
	r.items = items
	if !r.sendList(ctx, true) {
		return
	}
	for waitInterval(ctx, r.use.Interval) && r.awaitChange(ctx) {
		if !r.sendList(ctx, false) {
			return
		}
	}
}

func (r *watchRun) sendList(ctx context.Context, isFirst bool) bool {
	list := r.use.listStatuses(r.items)
	if !isFirst && slices.Equal(list, r.last) {
		return true
	}
	select {
	case r.out <- plugin.ProcessList{Time: r.use.clock.Now(), Processes: list}:
		r.last = list
		return true
	case <-ctx.Done():
		return false
	}
}

func (r *watchRun) awaitChange(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return false
	case <-r.changes:
		return true
	case items, isOpen := <-r.health:
		r.items = items
		return isOpen
	}
}

func (u *WatchProcesses) listStatuses(items []plugin.HealthItem) []plugin.ProcessStatus {
	processes := u.processes.ListProcesses()
	statuses := make([]plugin.ProcessStatus, 0, len(processes))
	for _, process := range processes {
		status := plugin.ProcessStatus{Process: process, Health: plugin.HealthOK}
		if found, ok := u.plugins.GetPlugin(process.PluginID); ok {
			status.PluginName = found.Manifest.Name
		}
		if worst, ok := plugin.GetWorstHealth(items, process.PluginID); ok {
			status.Health, status.HealthText = worst.Severity, worst.Text
		}
		statuses = append(statuses, status)
	}
	return statuses
}

func waitInterval(ctx context.Context, interval time.Duration) bool {
	if interval <= 0 {
		return ctx.Err() == nil
	}
	timer := time.NewTimer(interval)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}
