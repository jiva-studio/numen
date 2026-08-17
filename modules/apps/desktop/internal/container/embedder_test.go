package container_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
)

// A container nobody configured builds no embedder, and the machine's own
// settings are not read here.
func TestAContainerNobodyGaveSettingsEmbedsWithNothing(t *testing.T) {
	embedder, close, why := container.Config{}.Embedder()
	if why != nil {
		t.Fatalf("want no reason, got %v", why)
	}
	if embedder != nil {
		t.Errorf("got an embedder: %v", embedder.Model())
	}
	if close != nil {
		t.Error("got something to close")
	}
}

func TestTheSettingsGivenAreTheOnesUsed(t *testing.T) {
	cfg := embed.Defaults()
	cfg.Use = embed.UseService
	cfg.Service.Name = "bge-m3"
	// Nowhere: no test reaches a service.
	cfg.Service.BaseURL = "http://127.0.0.1:1/v1"
	t.Setenv(embed.KeyEnvVar, "sk-test")

	embedder, close, why := container.Config{Embedding: cfg}.Embedder()
	if why != nil {
		t.Fatal(why)
	}
	if close != nil {
		defer func() { _ = close() }()
	}
	if embedder == nil {
		t.Fatal("no embedder")
	}
	if got := embedder.Model().Name; got != "bge-m3" {
		t.Errorf("got %q", got)
	}
}
