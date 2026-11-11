package plugin

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"math"
	"net"
	"slices"
	"strconv"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/portproxy"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/supervision"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/core/kernel/port/machineport"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const listenerAddressHost = "127.0.0.1"

var (
	errListenerPortValue = errors.New("port setting is not a port from 1024 to 65535")
	// errListenerBind marks a failure of the plugin that a sync of its listeners recorded.
	errListenerBind = errors.New("listener port not bound")
)

// listenedPlugins is what the listeners read of the set of plugins.
type listenedPlugins interface {
	view() (setView, func())
	getKnown(pluginID string) (manifest.Discovered, bool)
	getName(pluginID string) string
	listIDs() []string
}

// Listeners binds the ports the plugins declare as listeners and forwards each connection to the
// plugin, which it starts when a connection arrives.
type Listeners struct {
	set       listenedPlugins
	mode      hostState
	sups      *supervision.Supervisors
	activator pluginActivator
	failures  *supervision.Failures
	clock     machineport.Clock
	report    func(error)

	mu       sync.Mutex
	bindings map[string]map[string]listenerBinding
}

// listenerBinding is a bound listener port and the proxy that forwards its connections.
type listenerBinding struct {
	port  int
	proxy *portproxy.Proxy
}

func (l *Listeners) getBindable(pluginID string) (manifest.Discovered, map[string]any, bool) {
	v, release := l.set.view()
	defer release()
	p, ok := v.get(pluginID)
	if !ok || l.mode.IsClosed() || l.mode.IsSafeMode() || !v.isEnabled(pluginID) {
		return manifest.Discovered{}, nil, false
	}
	return p, v.settings(pluginID), true
}

func (l *Listeners) getWantedPorts(pluginID string) (map[string]int, error) {
	p, settings, ok := l.getBindable(pluginID)
	if !ok {
		return nil, nil
	}
	ports := make(map[string]int, len(p.Manifest.Contributes.Listeners))
	for _, declared := range p.Manifest.Contributes.Listeners {
		if declared.PortSetting == "" {
			ports[declared.Name] = declared.Port
			continue
		}
		port, isPort := readPort(settings[declared.PortSetting])
		if !isPort {
			return nil, fmt.Errorf("listener %s of plugin %s: setting %s = %v: %w",
				declared.Name, pluginID, declared.PortSetting, settings[declared.PortSetting], errListenerPortValue)
		}
		ports[declared.Name] = port
	}
	return ports, nil
}

func readPort(value any) (int, bool) {
	var number float64
	switch kind := value.(type) {
	case float64:
		number = kind
	case int:
		number = float64(kind)
	case int64:
		number = float64(kind)
	default:
		return 0, false
	}
	if number != math.Trunc(number) || number < manifest.MinListenerPort || number > manifest.MaxListenerPort {
		return 0, false
	}
	return int(number), true
}

func (l *Listeners) checkPort(ctx context.Context, pluginID, key string, value any) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	p, ok := l.set.getKnown(pluginID)
	_, _, isBindable := l.getBindable(pluginID)
	isBinding := ok && isBindable
	if !ok {
		return nil
	}
	for _, declared := range p.Manifest.Contributes.Listeners {
		if declared.PortSetting != key {
			continue
		}
		port, isPort := readPort(value)
		if !isPort {
			return fmt.Errorf("listener %s of plugin %s: setting %s = %v: %w",
				declared.Name, pluginID, key, value, errListenerPortValue)
		}
		if !isBinding || l.bindings[pluginID][declared.Name].port == port {
			continue
		}
		addr := net.JoinHostPort(listenerAddressHost, strconv.Itoa(port))
		listener, err := new(net.ListenConfig).Listen(ctx, "tcp", addr)
		if err != nil {
			return fmt.Errorf("listener %s of plugin %s: bind %s: %w", declared.Name, pluginID, addr, err)
		}
		if err := listener.Close(); err != nil {
			return fmt.Errorf("listener %s of plugin %s: close %s: %w", declared.Name, pluginID, addr, err)
		}
	}
	return nil
}

func (l *Listeners) syncAll(ctx context.Context) {
	l.mu.Lock()
	ids := slices.Collect(maps.Keys(l.bindings))
	l.mu.Unlock()
	for _, id := range l.set.listIDs() {
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	for _, id := range ids {
		l.RefreshListeners(ctx, id)
	}
}

// RefreshListeners syncs the listeners of the plugin with its settings and reports a failure that is not
// already recorded as the failure of the plugin.
func (l *Listeners) RefreshListeners(ctx context.Context, pluginID string) {
	if err := l.SyncListeners(ctx, pluginID); err != nil && !errors.Is(err, errListenerBind) {
		l.report(err)
	}
}

// SyncListeners binds the ports the settings of the plugin ask for and closes the ones they no longer ask
// for. A port that cannot be bound fails the plugin, and the error is returned.
func (l *Listeners) SyncListeners(ctx context.Context, pluginID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	wantedPorts, err := l.getWantedPorts(pluginID)

	current := l.bindings[pluginID]
	if err == nil && isSamePorts(current, wantedPorts) {
		return nil
	}
	var next map[string]listenerBinding
	if err == nil {
		next, err = l.openChangedListeners(context.WithoutCancel(ctx), pluginID, current, wantedPorts)
	}
	if err != nil {
		if len(current) > 0 {
			return err
		}
		err = fmt.Errorf("%w: %w", errListenerBind, err)
		l.closeListenersLocked(pluginID)
		l.failListenerBind(pluginID, err)
		return err
	}
	for name, binding := range current {
		if next[name] != binding {
			l.closeListener(pluginID, binding.proxy)
		}
	}
	l.clearListenerFailure(pluginID)
	if len(next) == 0 {
		delete(l.bindings, pluginID)
		return nil
	}
	l.bindings[pluginID] = next
	return nil
}

func isSamePorts(current map[string]listenerBinding, wantedPorts map[string]int) bool {
	if len(current) != len(wantedPorts) {
		return false
	}
	for name, port := range wantedPorts {
		if binding, ok := current[name]; !ok || binding.port != port {
			return false
		}
	}
	return true
}

func (l *Listeners) openChangedListeners(
	ctx context.Context, pluginID string, current map[string]listenerBinding, wantedPorts map[string]int,
) (map[string]listenerBinding, error) {
	next := make(map[string]listenerBinding, len(wantedPorts))
	var openedProxies []*portproxy.Proxy
	for _, name := range slices.Sorted(maps.Keys(wantedPorts)) {
		port := wantedPorts[name]
		if binding, ok := current[name]; ok && binding.port == port {
			next[name] = binding
			continue
		}
		addr := net.JoinHostPort(listenerAddressHost, strconv.Itoa(port))
		proxy, err := portproxy.Open(ctx, addr, l.createTarget(pluginID, name), l.report)
		if err != nil {
			for _, proxy := range openedProxies {
				l.closeListener(pluginID, proxy)
			}
			return nil, fmt.Errorf("bind listener %s of plugin %s: %w", name, pluginID, err)
		}
		openedProxies = append(openedProxies, proxy)
		next[name] = listenerBinding{port: port, proxy: proxy}
	}
	return next, nil
}

func (l *Listeners) failListenerBind(pluginID string, err error) {
	sup, ok := l.sups.Get(pluginID)
	if ok && sup.Failure.IsFailed() {
		return
	}
	l.failures.RecordActivationFailure(pluginID, l.set.getName(pluginID), err)
	l.failures.Deliver()
}

func (l *Listeners) clearListenerFailure(pluginID string) {
	sup, ok := l.sups.Get(pluginID)
	if !ok {
		return
	}
	if cause := sup.Failure.GetCause(); cause != nil && errors.Is(cause, errListenerBind) {
		l.failures.Clear(pluginID, sup)
	}
}

func (l *Listeners) closeListenersLocked(pluginID string) {
	for _, binding := range l.bindings[pluginID] {
		l.closeListener(pluginID, binding.proxy)
	}
	delete(l.bindings, pluginID)
}

func (l *Listeners) closeListener(pluginID string, proxy *portproxy.Proxy) {
	if err := proxy.Close(); err != nil {
		l.report(fmt.Errorf("close a listener of plugin %s: %w", pluginID, err))
	}
}

func (l *Listeners) closeAll() {
	l.mu.Lock()
	defer l.mu.Unlock()
	for id := range l.bindings {
		l.closeListenersLocked(id)
	}
}

func (l *Listeners) createTarget(pluginID, name string) portproxy.Target {
	return func(ctx context.Context) (string, func(), error) {
		endpoint, err := l.activator.EnsureActive(ctx, pluginID)
		isReported := l.failures.MarkTargetResult(pluginID, err)
		if err != nil {
			err = fmt.Errorf("activate plugin %s for listener %s: %w", pluginID, name, err)
			if isReported {
				return "", nil, fmt.Errorf("%w: %w", portproxy.ErrTargetReported, err)
			}
			return "", nil, err
		}
		release := func() { l.releaseEndpoint(pluginID, endpoint) }
		port, ok := l.sups.GetListenerPort(pluginID, name)
		if !ok {
			release()
			return "", nil, fmt.Errorf("plugin %s listener %s: %w", pluginID, name, supervision.ErrListenerNotReported)
		}
		return net.JoinHostPort(listenerAddressHost, strconv.Itoa(port)), release, nil
	}
}

func (l *Listeners) releaseEndpoint(pluginID string, endpoint transport.Endpoint) {
	l.sups.Release(pluginID, endpoint.Token, l.clock.Now())
}
