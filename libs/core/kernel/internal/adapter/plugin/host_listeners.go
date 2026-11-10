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

	plugindomain "github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/plugin/portproxy"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
)

const listenerAddressHost = "127.0.0.1"

var errListenerNotReported = errors.New("the ready line does not report the listener")

func checkReportedListeners(m manifest.Manifest, reported map[string]int) error {
	for _, declared := range m.Contributes.Listeners {
		if _, ok := reported[declared.Name]; !ok {
			return fmt.Errorf("listener %s: %w", declared.Name, errListenerNotReported)
		}
	}
	return nil
}

var (
	errListenerPortValue = errors.New("port setting is not a port from 1024 to 65535")
	// errListenerBind marks a failure of the plugin that a sync of its listeners recorded.
	errListenerBind = errors.New("listener port not bound")
)

// listenerBinding is a bound listener port and the proxy that forwards its connections.
type listenerBinding struct {
	port  int
	proxy *portproxy.Proxy
}

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

func (h *Host) checkListenerPort(ctx context.Context, pluginID, key string, value any) error {
	h.listenersMu.Lock()
	defer h.listenersMu.Unlock()
	h.mu.RLock()
	p, ok := h.plugins[pluginID]
	isBinding := ok && !h.isClosed && !h.safeMode && h.isPluginEnabledLocked(pluginID)
	h.mu.RUnlock()
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
		if !isBinding || h.listeners[pluginID][declared.Name].port == port {
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
		h.syncListenersAndReport(ctx, id)
	}
}

func (h *Host) syncListenersAndReport(ctx context.Context, pluginID string) {
	if err := h.syncListeners(ctx, pluginID); err != nil && !errors.Is(err, errListenerBind) {
		h.reportError(err)
	}
}

func (h *Host) syncListeners(ctx context.Context, pluginID string) error {
	h.listenersMu.Lock()
	defer h.listenersMu.Unlock()

	h.mu.RLock()
	wantedPorts, err := h.getWantedPortsLocked(pluginID)
	h.mu.RUnlock()

	current := h.listeners[pluginID]
	if err == nil && isSamePorts(current, wantedPorts) {
		return nil
	}
	var next map[string]listenerBinding
	if err == nil {
		next, err = h.openChangedListeners(context.WithoutCancel(ctx), pluginID, current, wantedPorts)
	}
	if err != nil {
		if len(current) > 0 {
			return err
		}
		err = fmt.Errorf("%w: %w", errListenerBind, err)
		h.closeListenersLocked(pluginID)
		h.failListenerBind(pluginID, err)
		return err
	}
	for name, binding := range current {
		if next[name] != binding {
			h.closeListener(pluginID, binding.proxy)
		}
	}
	h.clearListenerFailure(pluginID)
	if len(next) == 0 {
		delete(h.listeners, pluginID)
		return nil
	}
	h.listeners[pluginID] = next
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

func (h *Host) openChangedListeners(
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
		proxy, err := portproxy.Open(ctx, addr, h.createListenerTarget(pluginID, name), h.reportError)
		if err != nil {
			for _, proxy := range openedProxies {
				h.closeListener(pluginID, proxy)
			}
			return nil, fmt.Errorf("bind listener %s of plugin %s: %w", name, pluginID, err)
		}
		openedProxies = append(openedProxies, proxy)
		next[name] = listenerBinding{port: port, proxy: proxy}
	}
	return next, nil
}

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

func (h *Host) clearListenerFailure(pluginID string) {
	h.mu.RLock()
	sup, ok := h.supervisors[pluginID]
	h.mu.RUnlock()
	if !ok {
		return
	}
	if cause := sup.GetActivationFailure(); cause != nil && errors.Is(cause, errListenerBind) {
		h.clearFailure(pluginID, sup)
	}
}

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

func (h *Host) closeAllListeners() {
	h.listenersMu.Lock()
	defer h.listenersMu.Unlock()
	for id := range h.listeners {
		h.closeListenersLocked(id)
	}
}

func (h *Host) createListenerTarget(pluginID, name string) portproxy.Target {
	return func(ctx context.Context) (string, func(), error) {
		endpoint, err := h.EnsureActive(ctx, pluginID)
		h.mu.Lock()
		_, isReported := h.reportedTargetFailures[pluginID]
		if err == nil {
			delete(h.reportedTargetFailures, pluginID)
		} else {
			h.reportedTargetFailures[pluginID] = struct{}{}
		}
		h.mu.Unlock()
		if err != nil {
			err = fmt.Errorf("activate plugin %s for listener %s: %w", pluginID, name, err)
			if isReported {
				return "", nil, fmt.Errorf("%w: %w", portproxy.ErrTargetReported, err)
			}
			return "", nil, err
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

func (h *Host) releaseEndpoint(pluginID string, endpoint plugindomain.Endpoint) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	sup, ok := h.supervisors[pluginID]
	if !ok || h.endpoints[pluginID].Token != endpoint.Token {
		return
	}
	sup.Release(h.clock.Now())
}

func (h *Host) recordActivationFailure(pluginID string, cause error) *plugindomain.Failure {
	h.mu.Lock()
	defer h.mu.Unlock()
	sup, ok := h.supervisors[pluginID]
	if !ok || !sup.RecordActivationFailure(h.clock.Now(), cause) {
		return nil
	}
	failure := h.newPluginFailureLocked(pluginID, sup, cause)
	h.failedPlugins = append(h.failedPlugins, failure)
	return &failure
}
