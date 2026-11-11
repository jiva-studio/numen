// Package discovery finds the plugins in the folders it is given, registers them, and retires the ones that left the disk.
package discovery

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/hostmode"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/pluginset"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

// ErrNotForTarget is what the registration of a built-in plugin fails with when its manifest does not name the target of the host.
var ErrNotForTarget = errors.New("the plugin does not target this host")

// modeState is what discovery reads of the mode of the host.
type modeState interface {
	IsClosed() bool
	IsSafeMode() bool
}

// scanRecorder keeps what a scan found, for the permissions of the declarations that are gone.
type scanRecorder interface {
	RecordScan(installed []manifest.Discovered)
	PruneSecretPermissions()
}

// pluginStopper stops the process of a plugin.
type pluginStopper interface {
	StopPlugin(ctx context.Context, pluginID string) error
}

// listenerSync syncs the ports bound for the listeners of the plugins with the host.
type listenerSync interface {
	SyncAll(ctx context.Context)
}

// cacheRemover removes the files a plugin cached.
type cacheRemover interface {
	RemoveCacheDir(pluginID string)
}

// Discovery finds the plugins in the folders it is given, registers them, and retires the ones that
// left the disk.
type Discovery struct {
	set       *pluginset.Set
	mode      modeState
	sups      *supervision.Supervisors
	stopper   pluginStopper
	listeners listenerSync
	storage   cacheRemover
	scans     scanRecorder
	report    func(error)
	dirs      []string
	target    string
}

// Deps are what a discovery of the plugins in Dirs that target Target reads and informs: the set of plugins, the mode of the
// host, the supervisors, the stopper, the listeners, the storage and the record of scans. Report receives the errors of a scan.
type Deps struct {
	Set       *pluginset.Set
	Mode      modeState
	Sups      *supervision.Supervisors
	Stopper   pluginStopper
	Listeners listenerSync
	Storage   cacheRemover
	Scans     scanRecorder
	Report    func(error)
	Dirs      []string
	Target    string
}

// New returns the discovery over deps.
func New(deps Deps) *Discovery {
	return &Discovery{
		set: deps.Set, mode: deps.Mode, sups: deps.Sups, stopper: deps.Stopper, listeners: deps.Listeners,
		storage: deps.Storage, scans: deps.Scans, report: deps.Report, dirs: deps.Dirs, target: deps.Target,
	}
}

// Discover scans the configured directories and synchronizes the registered plugins, and the ports bound
// for their listeners, with what is on disk. A plugin whose directory could not be read stays
// registered.
func (d *Discovery) Discover() error {
	if err := d.discoverPlugins(); err != nil {
		return err
	}
	d.listeners.SyncAll(context.Background())
	return nil
}

func (d *Discovery) discoverPlugins() error {
	scan := d.set.Scan()
	scan.BeginScan()
	defer scan.EndScan()

	all, problems := manifestjson.Discover(d.dirs)
	discovered := manifest.FilterByTarget(all, d.target)
	discovered, problems = d.set.RefuseConflicts(discovered, problems)

	if d.mode.IsClosed() {
		return hostmode.ErrHostClosed
	}
	if d.mode.IsSafeMode() {
		return nil
	}
	d.scans.RecordScan(all)
	stalePlugins := d.set.ReplaceDiscovered(discovered, problems)

	for _, problem := range problems {
		d.report(problem.Err)
	}
	for _, id := range stalePlugins {
		d.retirePlugin(id)
		if !d.set.IsOnDisk(id) {
			d.storage.RemoveCacheDir(id)
		}
	}
	d.scans.PruneSecretPermissions()
	return nil
}

func (d *Discovery) retirePlugin(pluginID string) {
	if err := d.stopper.StopPlugin(context.Background(), pluginID); err != nil {
		d.report(err)
	}
	if !d.set.IsRegistered(pluginID) {
		d.sups.Forget(pluginID)
	}
}

// RegisterPlugin adds a plugin with the supervisor that tracks it, as discovery does for a plugin found on disk.
func (d *Discovery) RegisterPlugin(p manifest.Discovered, sup *lifecycle.Supervisor) {
	d.set.Put(p, sup)
}

// RegisterBuiltIn registers a plugin linked into this binary from its manifest.json and what
// makes the handler of its backend. serve is called at each activation with the variables a
// launched process of the plugin is given: the host variables a process is launched with and the
// launch variables. A plugin of the same id found on disk is registered
// over it, and the built-in takes its place again when that one leaves the disk.
func (d *Discovery) RegisterBuiltIn(manifestJSON []byte, serve func(vars map[string]string) (http.Handler, error)) error {
	parsed, err := manifestjson.Read(bytes.NewReader(manifestJSON))
	if err != nil {
		return fmt.Errorf("register built-in plugin: %w", err)
	}
	if !slices.Contains(parsed.Targets, d.target) {
		return fmt.Errorf("register built-in plugin %s: %w: %s", parsed.ID, ErrNotForTarget, d.target)
	}
	if err := d.set.AddBuiltIn(manifest.Discovered{Dir: "", Manifest: parsed}, serve); err != nil {
		return fmt.Errorf("register built-in plugin %s: %w", parsed.ID, err)
	}
	return nil
}
