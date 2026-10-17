//go:build plugins_monolith

package plugin

import (
	"errors"
	"testing"
)

func TestAMonolithActivationThatLostTheRaceWithShutdownIsRejected(t *testing.T) {
	host := newDiscoveredHost(t)
	p, ok := host.GetPlugin("md.numen.ocr")
	if !ok {
		t.Fatal("plugin not registered")
	}
	host.mu.RLock()
	sup := host.supervisors["md.numen.ocr"]
	host.mu.RUnlock()

	endpoint, err := host.activatePlugin(t.Context(), p, sup)

	if !errors.Is(err, errActivationAborted) {
		t.Fatalf("expected errActivationAborted, got %v", err)
	}
	if !isEmptyEndpoint(endpoint) {
		t.Fatalf("expected no endpoint, got %+v", endpoint)
	}
	if sup.GetState() != StateDormant {
		t.Fatalf("expected StateDormant, got %v", sup.GetState())
	}
}
