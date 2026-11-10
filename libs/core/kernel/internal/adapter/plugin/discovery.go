package plugin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"slices"
	"sync"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

var (
	errNotForTarget      = errors.New("the plugin does not target this host")
	errBuiltInRegistered = errors.New("a built-in plugin of this id is registered")
)

// Discovery finds the plugins in the folders it is given, registers them with a supervisor each,
// and retires the ones that left the disk.
type Discovery struct {
	t      *table
	clock  machineport.Clock
	errs   *errorSink
	dirs   []string
	target string

	teardown     *Teardown
	listeners    *Listeners
	storage      *Storage
	config       *ConfigPush
	dropEndpoint func(pluginID string)

	mu         sync.RWMutex
	installed  []manifest.Discovered
	hasScanned bool
}

func newDiscovery(t *table, clock machineport.Clock, errs *errorSink, target string, dirs []string) *Discovery {
	return &Discovery{t: t, clock: clock, errs: errs, target: target, dirs: dirs}
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
	d.t.discovering.Lock()
	defer d.t.discovering.Unlock()

	all, problems := manifestjson.Discover(d.dirs)
	discovered := manifest.FilterByTarget(all, d.target)

	d.t.mu.Lock()
	if d.t.isClosed {
		d.t.mu.Unlock()
		return errHostClosed
	}
	if d.t.safeMode {
		d.t.mu.Unlock()
		return nil
	}
	d.t.problems = problems
	d.mu.Lock()
	d.installed = all
	d.hasScanned = true
	d.mu.Unlock()
	stalePlugins := d.registerDiscoveredLocked(discovered)
	d.t.mu.Unlock()

	for _, problem := range problems {
		d.errs.report(problem.Err)
	}
	for _, id := range stalePlugins {
		d.retirePlugin(id)
		d.t.mu.RLock()
		isInstalled := d.t.onDisk[id]
		d.t.mu.RUnlock()
		if !isInstalled {
			d.storage.removeCacheDir(id)
		}
	}
	d.pruneSecretPermissions()
	return nil
}

func (d *Discovery) registerDiscoveredLocked(discovered []manifest.Discovered) []string {
	present := make(map[string]bool, len(discovered))
	var stale []string
	for _, p := range discovered {
		id := p.Manifest.ID
		present[id] = true
		if recognized, ok := d.t.plugins[id]; ok && !reflect.DeepEqual(recognized, p) {
			stale = append(stale, id)
		}
		d.t.plugins[id] = p
		d.t.onDisk[id] = true
		d.registerSupervisorLocked(p)
	}
	for id := range d.t.onDisk {
		if !present[id] && !d.isCoveredByProblemLocked(id) {
			d.removeDiskPluginLocked(id)
			stale = append(stale, id)
		}
	}
	return stale
}

func (d *Discovery) removeDiskPluginLocked(pluginID string) {
	delete(d.t.onDisk, pluginID)
	if builtIn, ok := d.t.getBuiltInLocked(pluginID); ok {
		d.t.plugins[pluginID] = builtIn.plugin
		d.registerSupervisorLocked(builtIn.plugin)
		return
	}
	delete(d.t.plugins, pluginID)
}

func (d *Discovery) isCoveredByProblemLocked(pluginID string) bool {
	dir := d.t.plugins[pluginID].Dir
	for _, problem := range d.t.problems {
		if problem.Dir == dir || problem.Dir == filepath.Dir(dir) {
			return true
		}
	}
	return false
}

func (d *Discovery) registerSupervisorLocked(p manifest.Discovered) {
	id := p.Manifest.ID
	sup, exists := d.t.supervisors[id]
	if !exists {
		sup = lifecycle.New(d.clock)
		sup.SetOnExit(func() { d.dropEndpoint(id) })
		sup.SetOnError(d.errs.report)
		d.t.supervisors[id] = sup
	}
	sup.SetPermissions(p.Manifest.Permissions)
	sup.SetActivationPolicy(d.t.activation)
}

func (d *Discovery) retirePlugin(pluginID string) {
	if err := d.teardown.StopPlugin(context.Background(), pluginID); err != nil {
		d.errs.report(err)
	}
	d.t.mu.Lock()
	defer d.t.mu.Unlock()
	if _, registered := d.t.plugins[pluginID]; !registered {
		delete(d.t.supervisors, pluginID)
	}
}

// RegisterPlugin adds a plugin with the supervisor that tracks it, as discovery does for a plugin found on disk.
func (d *Discovery) RegisterPlugin(p manifest.Discovered, sup *lifecycle.Supervisor) {
	d.t.mu.Lock()
	defer d.t.mu.Unlock()
	d.t.plugins[p.Manifest.ID] = p
	d.t.supervisors[p.Manifest.ID] = sup
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
	p := manifest.Discovered{Dir: "", Manifest: parsed}

	d.t.mu.Lock()
	defer d.t.mu.Unlock()
	if _, exists := d.t.builtIns[parsed.ID]; exists {
		return fmt.Errorf("register built-in plugin %s: %w", parsed.ID, errBuiltInRegistered)
	}
	d.t.builtIns[parsed.ID] = builtInPlugin{plugin: p, serve: serve}
	if _, registered := d.t.plugins[parsed.ID]; !registered {
		d.t.plugins[parsed.ID] = p
		d.registerSupervisorLocked(p)
	}
	return nil
}

func (d *Discovery) pruneSecretPermissions() {
	prune := d.config.getSecretSource().Prune
	d.t.mu.RLock()
	d.mu.RLock()
	skip := !d.hasScanned || len(d.t.problems) > 0
	declared := d.listInstalledDeclarationsLocked()
	d.mu.RUnlock()
	d.t.mu.RUnlock()
	if prune == nil || skip {
		return
	}
	if err := prune(declared); err != nil {
		d.errs.report(fmt.Errorf("drop the permissions of removed declarations: %w", err))
	}
}

func (d *Discovery) listInstalledDeclarationsLocked() []plugindomain.DeclarationKey {
	var declarations []plugindomain.DeclarationKey
	installed := make(map[string]bool, len(d.installed))
	for _, p := range d.installed {
		installed[p.Manifest.ID] = true
		declarations = appendDeclarations(declarations, p)
	}
	for _, p := range d.t.getPluginsInOrderLocked() {
		if !installed[p.Manifest.ID] {
			declarations = appendDeclarations(declarations, p)
		}
	}
	return declarations
}
