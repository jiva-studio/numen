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

// nowhere is a service no test reaches.
const nowhere = "http://127.0.0.1:1/v1"

func serving(name string) embed.Config {
	cfg := embed.Defaults()
	cfg.Model.Name = name
	cfg.Indexing.Use = embed.UseService
	cfg.Indexing.Service.Name = name
	cfg.Indexing.Service.BaseURL = nowhere
	return cfg
}

func TestTheSettingsGivenAreTheOnesUsed(t *testing.T) {
	t.Setenv(embed.KeyEnvVar, "sk-test")

	embedder, close, why := container.Config{Embedding: serving("bge-m3")}.Embedder()
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

// Saying nothing about questions is asking the way the vault was indexed, and
// one placement is one model loaded once.
func TestOnePlacementIsOneEmbedder(t *testing.T) {
	t.Setenv(embed.KeyEnvVar, "sk-test")

	indexing, asking, close, why := container.Config{Embedding: serving("bge-m3")}.Embedders()
	if why != nil {
		t.Fatal(why)
	}
	if close != nil {
		defer func() { _ = close() }()
	}
	if indexing != asking {
		t.Errorf("two embedders for one placement: %v and %v", indexing, asking)
	}
}

func TestAQuestionIsEmbeddedWhereTheSettingsSay(t *testing.T) {
	t.Setenv(embed.KeyEnvVar, "sk-test")
	cfg := serving("bge-m3")
	cfg.Query.Use = embed.UseService
	cfg.Query.Service.Name = "reached-another-way"
	cfg.Query.Service.BaseURL = nowhere

	indexing, asking, close, why := container.Config{Embedding: cfg}.Embedders()
	if why != nil {
		t.Fatal(why)
	}
	if close != nil {
		defer func() { _ = close() }()
	}
	if indexing == asking {
		t.Fatal("one embedder for two placements")
	}
	// Two placements of one model keep their vectors under one recipe.
	if a, b := indexing.Model().Recipe(), asking.Model().Recipe(); a != b {
		t.Errorf("%s and %s", a, b)
	}
}

// A placement that cannot be built is the whole thing not being built, and
// what was opened before it is let go of.
func TestAQuestionWithNowhereToBeEmbeddedIsAReason(t *testing.T) {
	t.Setenv(embed.KeyEnvVar, "sk-test")
	cfg := serving("bge-m3")
	cfg.Query.Use = embed.UseService
	cfg.Query.Service.BaseURL = ""

	indexing, asking, close, why := container.Config{Embedding: cfg}.Embedders()
	if why == nil {
		t.Fatal("want a reason")
	}
	if indexing != nil || asking != nil {
		t.Errorf("got %v and %v", indexing, asking)
	}
	if close != nil {
		t.Error("got something to close")
	}
}
