package plugin

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/delivery"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/lifecycle"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/portproxy"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

const SettingEnabled = settingEnabled

var (
	ErrBuiltInRegistered = errBuiltInRegistered
	ErrHostClosed        = errHostClosed
	ErrListenerPortValue = errListenerPortValue
	ErrNotAFile          = errNotAFile
	ErrNotAccepted       = errNotAccepted
	ErrNotForTarget      = errNotForTarget
	ErrPathNotFound      = errPathNotFound
	ErrPluginNotFound    = errPluginNotFound
	ErrProviderIDTaken   = errProviderIDTaken
)

// SetPlugin stores p in the set of the registry as it is, without the checks of a scan.
func SetPlugin(r *Registry, id string, p manifest.Discovered) {
	r.set.mu.Lock()
	defer r.set.mu.Unlock()
	r.set.plugins[id] = p
}

// GetRawPlugin returns the plugin of the set whether or not it is enabled.
func GetRawPlugin(r *Registry, id string) manifest.Discovered {
	p, _ := r.set.getKnownPlugin(id)
	return p
}

// AddProblem records a problem of a scan in the set of the registry.
func AddProblem(r *Registry, problem manifestjson.DiscoveryProblem) {
	r.set.mu.Lock()
	r.set.problems = append(r.set.problems, problem)
	r.set.mu.Unlock()
}

// GetName returns the name the set gives the plugin.
func GetName(r *Registry, id string) string { return r.set.getName(id) }

// GetRequirementIDs returns the ids of the requirements the plugin declares.
func GetRequirementIDs(r *Registry, id string) []string { return r.set.getRequirementIDs(id) }

// GetLaunch returns how the set launches p.
func GetLaunch(r *Registry, p manifest.Discovered) supervision.Launch { return r.set.getLaunch(p) }

func GetSupervisors(r *Registry) *supervision.Supervisors { return r.sups }

// SetPerformance sets the performance profile the push holds, without pushing it.
func SetPerformance(c *ConfigPush, profile string) { c.performance = profile }

func GetDelivery(c *ConfigPush) *delivery.Delivery { return c.delivery }

// HoldScan holds the gate against the start of a plugin, as a scan of the folders does.
func HoldScan(d *Discovery) { d.scan.holdScan() }

// ReleaseScan releases the hold HoldScan took.
func ReleaseScan(d *Discovery) { d.scan.releaseScan() }

func GetStopper(t *Teardown) *supervision.Stopper { return t.stopper }

// RunActivation starts the process of p under sup.
func RunActivation(ctx context.Context, a *Activation, p manifest.Discovered, sup *lifecycle.Supervisor) (transport.Endpoint, error) {
	return a.runActivation(ctx, p, sup, lifecycle.Lease{Kind: lifecycle.UseLease})
}

// CreateTarget returns the target the listener name of the plugin forwards its connections to.
func CreateTarget(l *Listeners, pluginID, name string) portproxy.Target {
	return l.createTarget(pluginID, name)
}
