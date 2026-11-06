package plugin

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net"
	"slices"
	"strconv"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/plugin/portproxy"
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

// getWantedListenersLocked returns the listeners the host binds for the plugin: those its manifest
// declares while it is registered and enabled, the host is open and not in safe mode, and none
// otherwise.
func (h *Host) getWantedListenersLocked(pluginID string) []manifest.Listener {
	p, ok := h.plugins[pluginID]
	if !ok || h.isClosed || h.safeMode || !h.isPluginEnabledLocked(pluginID) {
		return nil
	}
	return p.Manifest.Contributes.Listeners
}

// syncAllListeners brings the bound ports of every registered plugin, and of every plugin that has
// bound ports, in line with its manifest and state.
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

// syncListeners binds 127.0.0.1:<port> for each listener the plugin declares while it is
// registered and enabled, and closes its ports otherwise. A changed set of listeners is closed and
// bound again. A port that cannot be bound fails the plugin.
func (h *Host) syncListeners(ctx context.Context, pluginID string) {
	h.listenersMu.Lock()
	defer h.listenersMu.Unlock()

	h.mu.RLock()
	wanted := h.getWantedListenersLocked(pluginID)
	h.mu.RUnlock()

	if slices.Equal(h.listenerSets[pluginID], wanted) {
		return
	}
	h.closeListenersLocked(pluginID)
	if len(wanted) == 0 {
		return
	}
	proxies, err := h.openListeners(context.WithoutCancel(ctx), pluginID, wanted)
	if err != nil {
		h.failListenerBind(pluginID, err)
		return
	}
	h.listeners[pluginID] = proxies
	h.listenerSets[pluginID] = wanted
}

// openListeners binds every listener, or none: a failed bind closes the ports bound before it.
func (h *Host) openListeners(ctx context.Context, pluginID string, wanted []manifest.Listener) ([]*portproxy.Proxy, error) {
	proxies := make([]*portproxy.Proxy, 0, len(wanted))
	for _, declared := range wanted {
		addr := net.JoinHostPort(listenerAddressHost, strconv.Itoa(declared.Port))
		proxy, err := portproxy.Open(ctx, addr, h.createListenerTarget(pluginID, declared.Name), h.reportError)
		if err != nil {
			for _, opened := range proxies {
				h.closeListener(pluginID, opened)
			}
			return nil, fmt.Errorf("bind listener %s of plugin %s: %w", declared.Name, pluginID, err)
		}
		proxies = append(proxies, proxy)
	}
	return proxies, nil
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
	h.recordActivationFailure(pluginID, err)
	h.deliverPluginFailures()
}

// closeListenersLocked closes and forgets the bound ports of the plugin. The caller holds listenersMu.
func (h *Host) closeListenersLocked(pluginID string) {
	for _, proxy := range h.listeners[pluginID] {
		h.closeListener(pluginID, proxy)
	}
	delete(h.listeners, pluginID)
	delete(h.listenerSets, pluginID)
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
		if _, err := h.EnsureActive(ctx, pluginID); err != nil {
			return "", nil, fmt.Errorf("activate plugin %s for listener %s: %w", pluginID, name, err)
		}
		h.mu.RLock()
		port, ok := h.listenerPorts[pluginID][name]
		h.mu.RUnlock()
		if !ok {
			h.Release(pluginID)
			return "", nil, fmt.Errorf("plugin %s listener %s: %w", pluginID, name, errListenerNotReported)
		}
		return net.JoinHostPort(listenerAddressHost, strconv.Itoa(port)), func() { h.Release(pluginID) }, nil
	}
}
