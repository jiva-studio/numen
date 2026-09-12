package container_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/task"
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

// Two services no test reaches. One model name is answered to by both, and
// which of them was asked is what a vector is kept under.
const (
	nowhere   = "http://127.0.0.1:1/v1"
	elsewhere = "http://127.0.0.1:2/v1"
)

func serving(name string) embed.Config {
	cfg := embed.Defaults()
	cfg.Model.Name = name
	cfg.Indexing = at(cfg.Indexing, name, nowhere)
	return cfg
}

// at is a provider asking a service at a base URL for the model it calls name.
// Everything else about the service is left as the settings hold it.
func at(where embed.Provider, name, baseURL string) embed.Provider {
	where.Use = embed.UseService
	service, _ := where.Service()
	service.Name, service.BaseURL = name, baseURL
	return where.Serving(service)
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
func TestOneProviderIsOneModelSeenTwoWays(t *testing.T) {
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
	cfg.Query = at(cfg.Query, "reached-another-way", nowhere)

	indexing, asking, close, why := container.Config{Embedding: cfg}.Embedders(t.Context(), nil)
	if why != nil {
		t.Fatal(why)
	}
	if close != nil {
		defer func() { _ = close() }()
	}
	if indexing == asking {
		t.Fatal("one embedder for two providers")
	}
	// Two providers of one model keep their vectors under one recipe.
	if a, b := indexing.Model().Recipe(), asking.Model().Recipe(); a != b {
		t.Errorf("%s and %s", a, b)
	}
}

// recipe is what an installation keeps its vectors under.
func recipe(t *testing.T, cfg embed.Config) string {
	t.Helper()
	embedder, close, why := container.Config{Embedding: cfg}.Embedder(t.Context())
	if why != nil {
		t.Fatal(why)
	}
	if close != nil {
		t.Cleanup(func() { _ = close() })
	}
	if embedder == nil {
		t.Fatal("no embedder")
	}
	return embedder.Model().Recipe()
}

// Two services answering to one model name are two sets of vectors.
//
// A service is pointed at by a base URL, and what is served there is whatever
// that address serves. Told only the name, a search would read one service's
// vectors as answers to a question the other was asked.
func TestTwoServicesServingOneNameKeepTheirOwnVectors(t *testing.T) {
	t.Setenv(embed.KeyEnvVar, "sk-test")

	one := serving("bge-m3")
	other := serving("bge-m3")
	other.Indexing = at(other.Indexing, "bge-m3", elsewhere)

	if a, b := recipe(t, one), recipe(t, other); a == b {
		t.Errorf("two services keep their vectors under one key: %s", a)
	}
}

// A model run on this machine and the same name served are two sets of vectors.
//
// A repository and a service call one model by one name, and the weights behind
// each are the address's own.
func TestAModelRunHereAndOneServedKeepTheirOwnVectors(t *testing.T) {
	t.Setenv(embed.KeyEnvVar, "sk-test")

	here := embed.Defaults()
	local, _ := here.Indexing.Local()
	// The repository is named and not fetched, so nothing reaches a network.
	local.Download = false
	here.Indexing = here.Indexing.Running(local)
	served := serving(here.Model.Name)
	served.Indexing = at(served.Indexing, local.Name, nowhere)

	if a, b := recipe(t, here), recipe(t, served); a == b {
		t.Errorf("a model run here and one served keep their vectors under one key: %s", a)
	}
}

// A run that only asks questions claims the vectors the index holds.
//
// The provider that fills an index is what its vectors were made by, and a
// question placed elsewhere is answered from those rows.
func TestARunThatOnlyAsksClaimsWhatTheIndexWasFilledWith(t *testing.T) {
	t.Setenv(embed.KeyEnvVar, "sk-test")

	cfg := serving("bge-m3")
	cfg.Query = at(cfg.Query, "bge-m3", elsewhere)

	asking, close, why := container.Config{Embedding: cfg}.Asking(t.Context())
	if why != nil {
		t.Fatal(why)
	}
	if close != nil {
		defer func() { _ = close() }()
	}
	if asking == nil {
		t.Fatal("no embedder")
	}
	if a, b := recipe(t, cfg), asking.Model().Recipe(); a != b {
		t.Errorf("the index was filled under %s and is asked under %s", a, b)
	}
}

// A run in a terminal has no list of what is being done, and asks for a model
// on this machine like any other.
func TestARunWithNoListToTellStillOpensAModel(t *testing.T) {
	cfg := embed.Defaults()
	// A folder with nothing in it: the model is looked for and not found,
	// which is what this asks about. Nothing reaches a network.
	cfg.Indexing = missing(t)
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

// A word for a provider that nobody implements is a reason, not a vault
// quietly searched by its words.
func TestAProviderNobodyImplementsIsARefusal(t *testing.T) {
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

// missing is a provider for a model on this machine that is not on it: a
// folder with nothing in it, looked in and not fetched. Nothing reaches a
// network.
func missing(t *testing.T) embed.Provider {
	t.Helper()
	where := embed.Defaults().Indexing
	local, _ := where.Local()
	local.Dir, local.Download = t.TempDir(), false
	return where.Running(local)
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

// uncompared is two providers the comparison between them never got an answer
// out of: the model the vault would be indexed by is not on this machine, and
// questions are placed with a service. It answers with the list.
func uncompared(t *testing.T) *task.Tasks {
	t.Helper()
	t.Setenv(embed.KeyEnvVar, "sk-test")
	cfg := embed.Defaults()
	cfg.Indexing = missing(t)
	// A provider on a service is reached by the model it asks for, and names no
	// repository at all.
	cfg.Query = embed.Provider{}.Serving(
		embed.ServiceModel{Name: "reached-another-way", BaseURL: nowhere})

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
			if at.Error != "" {
				got++
			}
		}
		return got == want
	})
}

// A provider is called by the name it is reached by, whichever kind it is.
func TestAProviderIsInTheListUnderItsOwnName(t *testing.T) {
	tasks := uncompared(t)
	held := failing(t, tasks, 2)

	for _, at := range held {
		if at.About == "reached-another-way" {
			return
		}
	}
	t.Errorf("the provider that answers questions is not in the list: %+v", held)
}

// A provider that cannot be built is the whole thing not being built, and
// what was opened before it is let go of.
func TestAQuestionWithNowhereToBeEmbeddedIsAReason(t *testing.T) {
	t.Setenv(embed.KeyEnvVar, "sk-test")
	cfg := serving("bge-m3")
	cfg.Query.Use = embed.UseService
	service, _ := cfg.Query.Service()
	service.BaseURL = ""
	cfg.Query = cfg.Query.Serving(service)

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

// An installation that says nothing about questions asks the way it indexed. A
// vault indexed over a network prepares nothing on this machine, so no model
// stands in the list of what is being done.
func TestAnInstallationSilentAboutQuestionsPreparesNoModelHere(t *testing.T) {
	t.Setenv(embed.KeyEnvVar, "sk-test")

	cfg := serving("baai/bge-m3")
	cfg.Query.Use = ""

	tasks := task.New()
	indexing, asking, close, why := container.Config{Embedding: cfg}.Embedders(t.Context(), tasks)
	if why != nil {
		t.Fatal(why)
	}
	if close != nil {
		defer func() { _ = close() }()
	}
	if indexing == nil || asking == nil {
		t.Fatalf("got %v and %v", indexing, asking)
	}
	// Both halves are the provider the settings named, and neither is this
	// machine's own model.
	if from := indexing.Model().From; from != cfg.Indexing.From() {
		t.Errorf("the index is filled from %q", from)
	}
	if from := asking.Model().From; from != cfg.Indexing.From() {
		t.Errorf("a question is embedded from %q", from)
	}
	if held := tasks.List(); len(held) != 0 {
		t.Errorf("a vault indexed over a network is getting a model ready: %+v", held)
	}
}
