package plugin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

var (
	errNotForTarget      = errors.New("the plugin does not target this host")
	errBuiltInRegistered = errors.New("a built-in plugin of this id is registered")
)

// scanRecorder keeps what a scan found, for the permissions of the declarations that are gone.
type scanRecorder interface {
	recordScan(installed []manifest.Discovered)
	pruneSecretPermissions()
}

// cacheRemover removes the files a plugin cached.
type cacheRemover interface {
	RemoveCacheDir(pluginID string)
}

// discoveredPlugins is what a discovery reads and writes of the set of plugins.
type discoveredPlugins interface {
	refuseConflicts(discovered []manifest.Discovered, problems []manifestjson.DiscoveryProblem) ([]manifest.Discovered, []manifestjson.DiscoveryProblem)
	replaceDiscovered(discovered []manifest.Discovered, problems []manifestjson.DiscoveryProblem) []string
	isOnDisk(pluginID string) bool
	isRegistered(pluginID string) bool
}

// pluginRegistrar adds a plugin to the set.
type pluginRegistrar interface {
	put(p manifest.Discovered, sup *lifecycle.Supervisor)
	addBuiltIn(p manifest.Discovered, serve func(vars map[string]string) (http.Handler, error)) error
}

// scanHold holds the set of plugins against the start of a plugin while a scan of the folders runs.
type scanHold interface {
	holdScan()
	releaseScan()
}

// Discovery finds the plugins in the folders it is given, registers them, and retires the ones that left the disk.
type Discovery struct {
	set       discoveredPlugins
	registrar pluginRegistrar
	scan      scanHold
	mode      hostState
	sups      *supervision.Supervisors
	stopper   pluginStopper
	listeners listenerControl
	storage   cacheRemover
	scans     scanRecorder
	report    func(error)
	dirs      []string
	target    string
}

// Discover scans the configured directories and synchronizes the registered plugins, and the ports bound
// for their listeners, with what is on disk. A plugin whose directory could not be read stays
// registered.
func (d *Discovery) Discover() error {
	if err := d.discoverPlugins(); err != nil {
		return err
	}
	d.listeners.syncAll(context.Background())
	return nil
}

func (d *Discovery) discoverPlugins() error {
	d.scan.holdScan()
	defer d.scan.releaseScan()

	all, problems := manifestjson.Discover(d.dirs)
	discovered := manifest.FilterByTarget(all, d.target)
	discovered, problems = d.set.refuseConflicts(discovered, problems)

	if d.mode.IsClosed() {
		return errHostClosed
	}
	if d.mode.IsSafeMode() {
		return nil
	}
	d.scans.recordScan(all)
	stalePlugins := d.set.replaceDiscovered(discovered, problems)

	for _, problem := range problems {
		d.report(problem.Err)
	}
	for _, id := range stalePlugins {
		d.retirePlugin(id)
		if !d.set.isOnDisk(id) {
			d.storage.RemoveCacheDir(id)
		}
	}
	d.scans.pruneSecretPermissions()
	return nil
}

func (d *Discovery) retirePlugin(pluginID string) {
	if err := d.stopper.StopPlugin(context.Background(), pluginID); err != nil {
		d.report(err)
	}
	if !d.set.isRegistered(pluginID) {
		d.sups.Forget(pluginID)
	}
}

// RegisterPlugin adds a plugin with the supervisor that tracks it, as discovery does for a plugin found on disk.
func (d *Discovery) RegisterPlugin(p manifest.Discovered, sup *lifecycle.Supervisor) {
	d.registrar.put(p, sup)
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
		return fmt.Errorf("register built-in plugin %s: %w: %s", parsed.ID, errNotForTarget, d.target)
	}
	if err := d.registrar.addBuiltIn(manifest.Discovered{Dir: "", Manifest: parsed}, serve); err != nil {
		return fmt.Errorf("register built-in plugin %s: %w", parsed.ID, err)
	}
	return nil
}
