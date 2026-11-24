package plugin

import (
	"context"
	"fmt"
	"slices"
	"sync"

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

func newPersistentHolder() plugindomain.Holder {
	return plugindomain.Holder{Who: plugindomain.WhoPersistent, Why: "stay running"}
}

type heldStarter interface {
	ensureActiveDuringDiscovery(ctx context.Context, pluginID string, lease lifecycle.Lease) (transport.Endpoint, error)
}

// Keeper keeps the processes of the plugins whose manifest asks to stay running: it holds a lease on each,
// starts it, and starts it again once for each exit of a process that was up, also an exit that comes before
// the start that brought the process up has returned. A start has the activation limit; one that fails is not
// repeated by the keeper until the next exit, use or enabling.
type Keeper struct {
	set      pluginViews
	mode     hostState
	sups     *supervision.Supervisors
	starter  heldStarter
	requires *Requirements
	stopper  pluginStopper
	report   func(error)

	mu           sync.Mutex
	kept         map[string]struct{}
	starting     map[string]bool
	refreshing   bool
	refreshAgain bool
}

// KeeperDeps are what a Keeper reads the plugins from and starts them with.
type KeeperDeps struct {
	Set          pluginViews
	Mode         hostState
	Supervisors  *supervision.Supervisors
	Activation   *Activation
	Requirements *Requirements
	Stopper      pluginStopper
	Report       func(error)
}

// NewKeeper creates a keeper that keeps no plugin and takes the exits of the processes from the supervisors.
func NewKeeper(d KeeperDeps) *Keeper {
	k := &Keeper{
		set: d.Set, mode: d.Mode, sups: d.Supervisors, starter: d.Activation, requires: d.Requirements, stopper: d.Stopper, report: d.Report,
		kept: make(map[string]struct{}), starting: make(map[string]bool),
	}
	d.Supervisors.Exits.Set(k.restartAfterExit)
	return k
}

// KeepAll keeps every enabled plugin that stays running. It returns at once; the starts run on their own.
func (k *Keeper) KeepAll(ctx context.Context) {
	v, release := k.set.getView()
	var ids []string
	for _, p := range v.enabled() {
		if p.Manifest.Backend.IsPersistent() {
			ids = append(ids, p.Manifest.ID)
		}
	}
	release()
	for _, id := range ids {
		k.Keep(ctx, id)
	}
}

// Keep keeps the plugin if it stays running and is enabled, and starts its process. It returns at once.
func (k *Keeper) Keep(ctx context.Context, pluginID string) {
	if k.mode.IsClosed() || k.mode.IsSafeMode() || !k.isKeepable(pluginID) {
		return
	}
	k.mu.Lock()
	k.kept[pluginID] = struct{}{}
	k.mu.Unlock()
	go k.start(context.WithoutCancel(ctx), pluginID)
}

// Drop stops keeping the plugin. The caller stops its process.
func (k *Keeper) Drop(_ context.Context, pluginID string) {
	k.mu.Lock()
	delete(k.kept, pluginID)
	k.mu.Unlock()
}

func (k *Keeper) isKeepable(pluginID string) bool {
	v, release := k.set.getView()
	defer release()
	p, ok := v.get(pluginID)
	return ok && p.Manifest.Backend.IsPersistent() && v.isEnabled(pluginID) && k.requires.GetUnmet(context.Background(), p.Manifest.Backend) == ""
}

func (k *Keeper) isKept(pluginID string) bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	_, ok := k.kept[pluginID]
	return ok
}

func (k *Keeper) restartAfterExit(pluginID string, exited <-chan struct{}) {
	if !k.isKept(pluginID) {
		return
	}
	go func() {
		<-exited
		k.start(context.Background(), pluginID)
	}()
}

func (k *Keeper) start(ctx context.Context, pluginID string) {
	if !k.isKept(pluginID) || !k.beginStart(pluginID) {
		return
	}
	defer k.endStart(ctx, pluginID)
	if !k.isKeepable(pluginID) {
		k.Drop(ctx, pluginID)
		return
	}
	if k.isHolding(pluginID) {
		return
	}
	lease := lifecycle.Lease{Kind: lifecycle.OwnLease, Holder: newPersistentHolder()}
	if _, err := k.starter.ensureActiveDuringDiscovery(withoutProvisioning(ctx), pluginID, lease); err != nil {
		// The activation has recorded the failure in the state of the plugin. The keeper does not repeat the start
		// until the next exit, use or enabling.
		return
	}
	if !k.isKeepable(pluginID) {
		k.Drop(ctx, pluginID)
		k.stop(ctx, pluginID)
	}
}

func (k *Keeper) beginStart(pluginID string) bool {
	k.mu.Lock()
	defer k.mu.Unlock()
	if _, busy := k.starting[pluginID]; busy {
		k.starting[pluginID] = true
		return false
	}
	k.starting[pluginID] = false
	return true
}

func (k *Keeper) endStart(ctx context.Context, pluginID string) {
	k.mu.Lock()
	isAskedAgain := k.starting[pluginID]
	delete(k.starting, pluginID)
	k.mu.Unlock()
	if isAskedAgain {
		go k.start(ctx, pluginID)
	}
}

func (k *Keeper) isHolding(pluginID string) bool {
	sup, ok := k.sups.Get(pluginID)
	if !ok {
		return false
	}
	usage, isRunning := sup.GetUsage()
	if !isRunning {
		return false
	}
	return slices.Contains(usage.Holders, newPersistentHolder())
}

// Refresh matches the processes of the plugins that require a condition of the host to it: a plugin whose
// requirements hold is kept where it stays running, and one whose requirements do not hold is not kept and its process
// is stopped. It returns at once; calls that come while a refresh runs make it run once more.
func (k *Keeper) Refresh(ctx context.Context) {
	k.mu.Lock()
	if k.refreshing {
		k.refreshAgain = true
		k.mu.Unlock()
		return
	}
	k.refreshing = true
	k.mu.Unlock()
	go k.runRefresh(context.WithoutCancel(ctx))
}

func (k *Keeper) runRefresh(ctx context.Context) {
	for {
		k.refreshOnce(ctx)
		k.mu.Lock()
		if !k.refreshAgain {
			k.refreshing = false
			k.mu.Unlock()
			return
		}
		k.refreshAgain = false
		k.mu.Unlock()
	}
}

func (k *Keeper) refreshOnce(ctx context.Context) {
	v, release := k.set.getView()
	var demanding []manifest.Discovered
	for _, p := range v.enabled() {
		if len(p.Manifest.Backend.Requires) > 0 {
			demanding = append(demanding, p)
		}
	}
	release()
	for _, p := range demanding {
		if k.requires.GetUnmet(ctx, p.Manifest.Backend) == "" {
			k.Keep(ctx, p.Manifest.ID)
			continue
		}
		k.Drop(ctx, p.Manifest.ID)
		k.stop(ctx, p.Manifest.ID)
	}
}

func (k *Keeper) stop(ctx context.Context, pluginID string) {
	if err := k.stopper.StopPlugin(ctx, pluginID); err != nil {
		k.report(fmt.Errorf("stop plugin %s: %w", pluginID, err))
	}
}
