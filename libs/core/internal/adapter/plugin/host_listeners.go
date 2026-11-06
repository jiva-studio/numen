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

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/portproxy"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/transport"
	"github.com/jiva-studio/numen/libs/plugin-catalog/manifest"
)

const listenerAddressHost = "127.0.0.1"

var errListenerNotReported = errors.New("the ready line does not report the listener")

// checkReportedListeners returns an error naming the first listener the manifest declares that the
// ready line does not report.
func checkReportedListeners(m manifest.Manifest, reported map[string]int) error {
	for _, declared := range m.Contributes.Listeners {
		if _, ok := reported[declared.Name]; !ok {
			return fmt.Errorf("listener %s: %w", declared.Name, errListenerNotReported)
		}
	}
	return nil
}

var (
	errListenerPortValue = errors.New("the port setting does not hold a port from 1024 to 65535")
	// errListenerBind marks a failure of the plugin that a sync of its listeners recorded.
	errListenerBind = errors.New("listener port not bound")
)

// listenerBinding is a bound listener port and the proxy that forwards its connections.
type listenerBinding struct {
	port  int
	proxy *portproxy.Proxy
}

// getWantedPortsLocked returns the port the host binds for each listener of the plugin: the fixed
// port of the manifest, or the value in force of the listener's port setting. It returns none while
// the plugin is not registered or disabled, or the host is closed or in safe mode.
func (h *Host) getWantedPortsLocked(pluginID string) (map[string]int, error) {
	p, ok := h.plugins[pluginID]
	if !ok || h.isClosed || h.safeMode || !h.isPluginEnabledLocked(pluginID) {
		return nil, nil
	}
	ports := make(map[string]int, len(p.Manifest.Contributes.Listeners))
	settings := h.getPluginSettingsLocked(pluginID)
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

// readPort returns the integer from 1024 to 65535 a setting value holds.
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

// syncAllListeners brings the bound ports of every registered plugin, and of every plugin that has
// bound ports, in line with its manifest, its settings and its state.
func (h *Host) syncAllListeners(ctx context.Context) {
	h.listenersMu.Lock()
	ids := slices.Collect(maps.Keys(h.listeners))
	h.listenersMu.Unlock()
	h.mu.RLock()
	for id := range h.plugins {
		if !slices.Contains(ids, id) {
			ids = append(ids, id)
		}
	}
	h.mu.RUnlock()
	slices.Sort(ids)
	for _, id := range ids {
		h.syncListeners(ctx, id)
	}
}

// syncListeners binds 127.0.0.1:<port> for each listener of the plugin while it is registered and
// enabled, and closes its ports otherwise. A listener whose port changed is bound on the new port
// first and its old port closed after. A port that cannot be bound, or a port setting that holds no
// port, fails the plugin and closes all its ports.
func (h *Host) syncListeners(ctx context.Context, pluginID string) {
	h.listenersMu.Lock()
	defer h.listenersMu.Unlock()

	h.mu.RLock()
	wanted, err := h.getWantedPortsLocked(pluginID)
	h.mu.RUnlock()

	current := h.listeners[pluginID]
	if err == nil && isSamePorts(current, wanted) {
		return
	}
	var next map[string]listenerBinding
	if err == nil {
		next, err = h.openChangedListeners(context.WithoutCancel(ctx), pluginID, current, wanted)
	}
	if err != nil {
		h.closeListenersLocked(pluginID)
		h.failListenerBind(pluginID, err)
		return
	}
	for name, binding := range current {
		if next[name] != binding {
			h.closeListener(pluginID, binding.proxy)
		}
	}
	h.clearListenerFailure(pluginID)
	if len(next) == 0 {
		delete(h.listeners, pluginID)
		return
	}
	h.listeners[pluginID] = next
}

// isSamePorts reports whether the bound listeners are the wanted ones on the wanted ports.
func isSamePorts(current map[string]listenerBinding, wanted map[string]int) bool {
	if len(current) != len(wanted) {
		return false
	}
	for name, port := range wanted {
		if binding, ok := current[name]; !ok || binding.port != port {
			return false
		}
	}
	return true
}

// openChangedListeners returns the listeners to keep bound: those of current already on their wanted
// port, and a new binding for every other wanted listener. A failed bind closes the bindings it made.
func (h *Host) openChangedListeners(
	ctx context.Context, pluginID string, current map[string]listenerBinding, wanted map[string]int,
) (map[string]listenerBinding, error) {
	next := make(map[string]listenerBinding, len(wanted))
	var opened []*portproxy.Proxy
	for _, name := range slices.Sorted(maps.Keys(wanted)) {
		port := wanted[name]
		if binding, ok := current[name]; ok && binding.port == port {
			next[name] = binding
			continue
		}
		addr := net.JoinHostPort(listenerAddressHost, strconv.Itoa(port))
		proxy, err := portproxy.Open(ctx, addr, h.createListenerTarget(pluginID, name), h.reportError)
		if err != nil {
			for _, proxy := range opened {
				h.closeListener(pluginID, proxy)
			}
			return nil, fmt.Errorf("bind listener %s of plugin %s: %w", name, pluginID, err)
		}
		opened = append(opened, proxy)
		next[name] = listenerBinding{port: port, proxy: proxy}
	}
	return next, nil
}

// failListenerBind records a port that cannot be bound as a failed activation of the plugin, once
// while the plugin is failed.
func (h *Host) failListenerBind(pluginID string, err error) {
	h.mu.RLock()
	sup, ok := h.supervisors[pluginID]
	isFailed := ok && sup.IsActivationFailed()
	h.mu.RUnlock()
	if isFailed {
		return
	}
	h.recordActivationFailure(pluginID, fmt.Errorf("%w: %w", errListenerBind, err))
	h.deliverPluginFailures()
}

// clearListenerFailure forgets the failure of the plugin when a sync of its listeners recorded it.
func (h *Host) clearListenerFailure(pluginID string) {
	h.mu.RLock()
	sup, ok := h.supervisors[pluginID]
	h.mu.RUnlock()
	if !ok {
		return
	}
	if isFailed, cause := sup.GetActivationFailure(); isFailed && errors.Is(cause, errListenerBind) {
		h.clearFailure(pluginID, sup)
	}
}

// closeListenersLocked closes and forgets the bound ports of the plugin. The caller holds listenersMu.
func (h *Host) closeListenersLocked(pluginID string) {
	for _, binding := range h.listeners[pluginID] {
		h.closeListener(pluginID, binding.proxy)
	}
	delete(h.listeners, pluginID)
}

func (h *Host) closeListener(pluginID string, proxy *portproxy.Proxy) {
	if err := proxy.Close(); err != nil {
		h.reportError(fmt.Errorf("close a listener of plugin %s: %w", pluginID, err))
	}
}

// closeAllListeners closes the bound ports of every plugin.
func (h *Host) closeAllListeners() {
	h.listenersMu.Lock()
	defer h.listenersMu.Unlock()
	for id := range h.listeners {
		h.closeListenersLocked(id)
	}
}

// createListenerTarget returns the target of a connection to a listener: the plugin is started and
// held until the connection ends, and the connection goes to the port its ready line reported.
func (h *Host) createListenerTarget(pluginID, name string) portproxy.Target {
	return func(ctx context.Context) (string, func(), error) {
		endpoint, err := h.EnsureActive(ctx, pluginID)
		if err != nil {
			return "", nil, fmt.Errorf("activate plugin %s for listener %s: %w", pluginID, name, err)
		}
		release := func() { h.releaseEndpoint(pluginID, endpoint) }
		h.mu.RLock()
		port, ok := h.listenerPorts[pluginID][name]
		h.mu.RUnlock()
		if !ok {
			release()
			return "", nil, fmt.Errorf("plugin %s listener %s: %w", pluginID, name, errListenerNotReported)
		}
		return net.JoinHostPort(listenerAddressHost, strconv.Itoa(port)), release, nil
	}
}

// releaseEndpoint gives back a lease taken with EnsureActive on the process that served endpoint. A
// lease of a process that has stopped is not given back: the process that runs now holds its own.
func (h *Host) releaseEndpoint(pluginID string, endpoint transport.Endpoint) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	sup, ok := h.supervisors[pluginID]
	if !ok || h.endpoints[pluginID].Token != endpoint.Token {
		return
	}
	sup.Release(h.clock.Now())
}
