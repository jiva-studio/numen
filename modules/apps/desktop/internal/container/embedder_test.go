package container_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/task"
)

// A container nobody configured builds no embedder, and the machine's own
// settings are not read here.
func TestAContainerNobodyGaveSettingsEmbedsWithNothing(t *testing.T) {
	embedder, close, why := container.Config{}.Embedder(t.Context())
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

	embedder, close, why := container.Config{Embedding: serving("bge-m3")}.Embedder(t.Context())
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

// Saying nothing about questions is asking the way the vault was indexed.
func TestOnePlacementIsOneModelSeenTwoWays(t *testing.T) {
	t.Setenv(embed.KeyEnvVar, "sk-test")

	indexing, asking, close, why := container.Config{Embedding: serving("bge-m3")}.Embedders(t.Context(), nil)
	if why != nil {
		t.Fatal(why)
	}
	if close != nil {
		defer func() { _ = close() }()
	}
	if indexing == nil || asking == nil {
		t.Fatalf("got %v and %v", indexing, asking)
	}
	if a, b := indexing.Model().Recipe(), asking.Model().Recipe(); a != b {
		t.Errorf("%s and %s", a, b)
	}
}

func TestAQuestionIsEmbeddedWhereTheSettingsSay(t *testing.T) {
	t.Setenv(embed.KeyEnvVar, "sk-test")
	cfg := serving("bge-m3")
	cfg.Query.Use = embed.UseService
	cfg.Query.Service.Name = "reached-another-way"
	cfg.Query.Service.BaseURL = nowhere

	indexing, asking, close, why := container.Config{Embedding: cfg}.Embedders(t.Context(), nil)
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

// A run in a terminal has no list of what is being done, and asks for a model
// on this machine like any other.
func TestARunWithNoListToTellStillOpensAModel(t *testing.T) {
	cfg := embed.Defaults()
	// A folder with nothing in it: the model is looked for and not found,
	// which is what this asks about. Nothing reaches a network.
	cfg.Indexing.Local.Dir = t.TempDir()
	cfg.Indexing.Local.Download = false
	held := container.Config{Embedding: cfg}

	embedder, close, why := held.Embedder(t.Context())
	if why != nil {
		t.Fatal(why)
	}
	if close != nil {
		defer func() { _ = close() }()
	}
	if embedder == nil {
		t.Fatal("no embedder")
	}
	// The model never turns up, and asking says why rather than panicking on
	// the way there.
	if _, err := embedder.Embed(t.Context(), []string{"anything"}); err == nil {
		t.Error("a model that is not on this machine embedded something")
	}

	indexing, asking, closeBoth, why := held.Embedders(t.Context(), nil)
	if why != nil {
		t.Fatal(why)
	}
	if closeBoth != nil {
		defer func() { _ = closeBoth() }()
	}
	if indexing == nil || asking == nil {
		t.Fatalf("got %v and %v", indexing, asking)
	}
}

// A word for a placement that nobody implements is a reason, not a vault
// quietly searched by its words.
func TestAPlacementNobodyImplementsIsARefusal(t *testing.T) {
	cfg := serving("bge-m3")
	cfg.Query.Use = "grcp"

	if _, _, _, why := (container.Config{Embedding: cfg}).Embedders(t.Context(), nil); why == nil {
		t.Fatal("want a reason")
	}
	cfg = serving("bge-m3")
	cfg.Indexing.Use = "sevrice"
	if _, _, _, why := (container.Config{Embedding: cfg}).Embedders(t.Context(), nil); why == nil {
		t.Fatal("want a reason")
	}
}

// missing is a placement for a model on this machine that is not on it: a
// folder with nothing in it, looked in and not fetched. Nothing reaches a
// network.
func missing(t *testing.T) embed.Placement {
	t.Helper()
	where := embed.Defaults().Indexing
	where.Local.Dir = t.TempDir()
	where.Local.Download = false
	return where
}

// waited is the list of what is being done, once it holds what is asked of it.
func waited(t *testing.T, tasks *task.Tasks, enough func([]task.Task) bool) []task.Task {
	t.Helper()
	for range 400 {
		if held := tasks.List(); enough(held) {
			return held
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("the list holds %+v", tasks.List())
	return nil
}

// uncompared is two placements the comparison between them never got an answer
// out of: the model the vault would be indexed by is not on this machine, and
// questions are placed with a service. It answers with the list.
func uncompared(t *testing.T) *task.Tasks {
	t.Helper()
	t.Setenv(embed.KeyEnvVar, "sk-test")
	cfg := embed.Defaults()
	cfg.Indexing = missing(t)
	// A service placement is reached by the model it asks for, and names no
	// repository at all.
	cfg.Query = embed.Placement{Use: embed.UseService}
	cfg.Query.Service.Name = "reached-another-way"
	cfg.Query.Service.BaseURL = nowhere

	tasks := task.New()
	_, _, close, why := container.Config{Embedding: cfg}.Embedders(t.Context(), tasks)
	if why != nil {
		t.Fatal(why)
	}
	if close != nil {
		t.Cleanup(func() { _ = close() })
	}
	return tasks
}

// failing is the list once the number of things that stopped badly is reached.
func failing(t *testing.T, tasks *task.Tasks, want int) []task.Task {
	t.Helper()
	return waited(t, tasks, func(held []task.Task) bool {
		got := 0
		for _, at := range held {
			if at.Failed != "" {
				got++
			}
		}
		return got == want
	})
}

// A comparison that could not be made is not agreement: the placement that
// answers questions is let go of, and it is said.
func TestTwoPlacementsThatCouldNotBeComparedAreNotOneModel(t *testing.T) {
	held := failing(t, uncompared(t), 2)
	if len(held) != 2 {
		t.Fatalf("the list holds %d pieces of work: %+v", len(held), held)
	}
}

// A placement is called by the name it is reached by, whichever kind it is.
func TestAPlacementIsInTheListUnderItsOwnName(t *testing.T) {
	tasks := uncompared(t)
	held := failing(t, tasks, 2)

	for _, at := range held {
		if at.About == "reached-another-way" {
			return
		}
	}
	t.Errorf("the placement that answers questions is not in the list: %+v", held)
}

// Two placements naming one repository are two lines, and how far one has got
// is not written over by the other.
func TestTwoPlacementsOfOneRepositoryAreTwoLines(t *testing.T) {
	cfg := embed.Defaults()
	cfg.Indexing = missing(t)
	cfg.Query = missing(t)
	cfg.Query.Use = embed.UseLocal

	tasks := task.New()
	_, _, close, why := container.Config{Embedding: cfg}.Embedders(t.Context(), tasks)
	if why != nil {
		t.Fatal(why)
	}
	if close != nil {
		defer func() { _ = close() }()
	}

	held := waited(t, tasks, func(held []task.Task) bool {
		failed := 0
		for _, at := range held {
			if at.Failed != "" {
				failed++
			}
		}
		return failed == 2
	})
	if len(held) != 2 {
		t.Fatalf("two placements are %d lines: %+v", len(held), held)
	}
	if held[0].ID == held[1].ID {
		t.Errorf("two placements share the line %q", held[0].ID)
	}
}

// A placement that cannot be built is the whole thing not being built, and
// what was opened before it is let go of.
func TestAQuestionWithNowhereToBeEmbeddedIsAReason(t *testing.T) {
	t.Setenv(embed.KeyEnvVar, "sk-test")
	cfg := serving("bge-m3")
	cfg.Query.Use = embed.UseService
	cfg.Query.Service.BaseURL = ""

	indexing, asking, close, why := container.Config{Embedding: cfg}.Embedders(t.Context(), nil)
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
