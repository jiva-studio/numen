package embedding_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	agreement "github.com/jiva-studio/numen/modules/libs/core/internal/embedding"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/embedding"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"

	"github.com/jiva-studio/numen/modules/libs/core/port/embeddingport"
)

// is the identity a provider's vectors are kept under. Nothing here reads a
// settings file: what a vector is has been settled before any of this is called.
func is(name string) embeddingport.EmbeddingModel {
	return embeddingport.EmbeddingModel{
		Name: name, Dimensions: 3, MaxTokens: 64, Pooling: "mean",
		From: "service:https://vectors.invalid/v1/" + name,
	}
}

// saying is a provider that answers every text with the one vector it was
// given. It counts the times it was asked the text two providers are compared
// over, and the times it was release.
type saying struct {
	model        embeddingport.EmbeddingModel
	vector       []float32
	requestCount atomic.Int64
	closed       atomic.Int64
}

func newSaying(name string, vector ...float32) *saying {
	return &saying{model: is(name), vector: vector}
}

func (s *saying) GetModel() embeddingport.EmbeddingModel { return s.model }

func (s *saying) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, one := range texts {
		if one == agreement.Asked {
			s.requestCount.Add(1)
		}
		out[i] = s.vector
	}
	return out, nil
}

func (s *saying) Close() error {
	s.closed.Add(1)
	return nil
}

// refuseFetch is a model that is not on this machine and is not to be fetched.
func refuseFetch(context.Context, func(finished, total int64)) (embeddingport.Embedder, error) {
	return nil, errors.New("the model is not on this machine")
}

// An installation that places its index nowhere embeds with nothing, and there
// is nothing to release.
func TestAnInstallationNamingNoProviderEmbedsWithNothing(t *testing.T) {
	filling, asking, letGo := embedding.Open(t.Context(), run.NewTasks(),
		embedding.Provider{}, embedding.Provider{})
	if filling != nil || asking != nil {
		t.Errorf("got %v and %v", filling, asking)
	}
	if letGo != nil {
		t.Error("got something to close")
	}

	one, letGoOfOne := embedding.OpenOne(t.Context(), embedding.Provider{})
	if one != nil {
		t.Errorf("got an embedder: %v", one.GetModel())
	}
	if letGoOfOne != nil {
		t.Error("got something to close")
	}
}

// Saying nothing about questions is asking the way the vault was indexed: one
// provider is one model, waited for on the way in and not waited for on the way
// out.
func TestOneProviderIsOneModelObservedTwoWays(t *testing.T) {
	target := newSaying("one-provider", 1, 0, 0)

	filling, asking, letGo := embedding.Open(t.Context(), run.NewTasks(),
		embedding.NewReached(target.GetModel(), "one-provider", target), embedding.Provider{})
	if filling == nil || asking == nil {
		t.Fatalf("got %v and %v", filling, asking)
	}
	if a, b := filling.GetModel().Recipe(), asking.GetModel().Recipe(); a != b {
		t.Errorf("%s and %s", a, b)
	}
	if err := letGo(); err != nil {
		t.Fatal(err)
	}
	if target.closed.Load() == 0 {
		t.Error("the provider was not let go of")
	}
}

// Two providers answering one text alike are one model, and both go on
// answering.
func TestTwoProvidersAnsweringAlikeAreOneModel(t *testing.T) {
	first := newSaying("indexed-here", 1, 0, 0)
	second := newSaying("asked-elsewhere", 1, 0, 0)

	tasks := run.NewTasks()
	filling, asking, letGo := embedding.Open(t.Context(), tasks,
		embedding.NewReached(first.GetModel(), "indexed-here", first),
		embedding.NewReached(second.GetModel(), "asked-elsewhere", second))
	defer func() { _ = letGo() }()

	waitForComparison(t, first, second)
	if items := tasks.List(); len(items) != 0 {
		t.Errorf("two providers of one model are in the list: %+v", items)
	}
	if second.closed.Load() != 0 {
		t.Error("a provider that agreed was let go of")
	}
	if _, err := filling.Embed(t.Context(), []string{"anything"}); err != nil {
		t.Errorf("the index is filled by nothing: %v", err)
	}
	if _, err := asking.Embed(t.Context(), []string{"anything"}); err != nil {
		t.Errorf("a question is embedded by nothing: %v", err)
	}
}

// Two providers that answer one text differently are two models, and the second
// is release: a question embedded in another space finds nothing the first
// indexed. It is said in the list of what is being done.
func TestTwoProvidersThatDisagreeAreNotOneModel(t *testing.T) {
	first := newSaying("indexed-here", 1, 0, 0)
	second := newSaying("somewhere-else", 0, 1, 0)

	tasks := run.NewTasks()
	_, asking, letGo := embedding.Open(t.Context(), tasks,
		embedding.NewReached(first.GetModel(), "indexed-here", first),
		embedding.NewReached(second.GetModel(), "somewhere-else", second))
	defer func() { _ = letGo() }()

	target := waitForFailures(t, tasks, 1)
	if len(target) != 1 {
		t.Fatalf("the list holds %d pieces of work: %+v", len(target), target)
	}
	if target[0].About != "somewhere-else" {
		t.Errorf("the provider that answers questions is in the list as %q", target[0].About)
	}
	if _, err := asking.Embed(t.Context(), []string{"anything"}); err == nil {
		t.Error("a question was embedded by a model the index knows nothing about")
	}
	waitFor(t, func() bool { return second.closed.Load() > 0 }, "the second provider was not let go of")
}

// A comparison nobody got an answer out of is not agreement. The model the
// index would be filled by never arrives, so the provider that answers
// questions is release, and both halves are in the list under what stopped
// them.
func TestTwoProvidersThatCouldNotBeComparedAreNotOneModel(t *testing.T) {
	second := newSaying("asked-elsewhere", 1, 0, 0)

	tasks := run.NewTasks()
	_, _, letGo := embedding.Open(t.Context(), tasks,
		embedding.NewFetched(is("never-arrives"), "never-arrives", refuseFetch),
		embedding.NewReached(second.GetModel(), "asked-elsewhere", second))
	defer func() { _ = letGo() }()

	if target := waitForFailures(t, tasks, 2); len(target) != 2 {
		t.Fatalf("the list holds %d pieces of work: %+v", len(target), target)
	}
	waitFor(t, func() bool { return second.closed.Load() > 0 }, "the second provider was not let go of")
}

// Two providers naming one repository are two lines, and how far one has got is
// not written over by the other.
func TestTwoProvidersOfOneRepositoryAreTwoLines(t *testing.T) {
	tasks := run.NewTasks()
	_, _, letGo := embedding.Open(t.Context(), tasks,
		embedding.NewFetched(is("one/repository"), "one/repository", refuseFetch),
		embedding.NewFetched(is("one/repository"), "one/repository", refuseFetch))
	defer func() { _ = letGo() }()

	target := waitForFailures(t, tasks, 2)
	if len(target) != 2 {
		t.Fatalf("two providers are %d lines: %+v", len(target), target)
	}
	if target[0].ID == target[1].ID {
		t.Errorf("two providers share the line %q", target[0].ID)
	}
}

// list.
func TestAProviderReachedAtOnceExistsInNoList(t *testing.T) {
	target := newSaying("reached-at-once", 1, 0, 0)

	tasks := run.NewTasks()
	_, _, letGo := embedding.Open(t.Context(), tasks,
		embedding.NewReached(target.GetModel(), "reached-at-once", target), embedding.Provider{})
	defer func() { _ = letGo() }()

	if items := tasks.List(); len(items) != 0 {
		t.Errorf("a provider nobody waits for is getting ready: %+v", items)
	}
}

// The model's own line draws no share until some of the model is here. A count
// of nothing against a size is a row sitting at nought per cent for the length
// of a download, which says less than a ring that turns.
func TestTheModelDrawsNoShareBeforeAnyOfItIsHere(t *testing.T) {
	progressCh := make(chan func(completed, total int64), 1)
	stop := make(chan struct{})
	opening := func(_ context.Context, tell func(processed, total int64)) (embeddingport.Embedder, error) {
		progressCh <- tell
		<-stop
		return nil, errors.New("none of the model is on this machine")
	}

	tasks := run.NewTasks()
	_, _, letGo := embedding.Open(t.Context(), tasks,
		embedding.NewFetched(is("a/model"), "a/model", opening), embedding.Provider{})
	defer func() { _ = letGo() }()

	// Nothing is known yet: the line is in the list from the moment it starts.
	pos := getOnlyTask(t, tasks)
	if pos.Title != "Preparing the model" || pos.About != "a/model" {
		t.Errorf("the line is shown as %q about %q", pos.Title, pos.About)
	}
	if pos.Total != 0 {
		t.Errorf("a line that has counted nothing is drawn against %d", pos.Total)
	}

	tell := <-progressCh

	// The cache is read before the first bytes land, and the size is known.
	tell(0, 90_000_000)
	if offset := getOnlyTask(t, tasks); offset.Total != 0 || offset.Count != 0 {
		t.Errorf("a model with none of it here is drawn as %d of %d", offset.Count, offset.Total)
	}

	tell(30_000_000, 90_000_000)
	pos = getOnlyTask(t, tasks)
	if pos.Count != 30_000_000 || pos.Total != 90_000_000 {
		t.Errorf("the model is drawn as %d of %d", pos.Count, pos.Total)
	}
	if pos.Unit != run.Bytes {
		t.Errorf("a model is counted as %v", pos.Unit)
	}

	// The model never turns up, and the line stays in the list under the reason.
	close(stop)
	if target := waitForFailures(t, tasks, 1); target[0].Error == "" {
		t.Error("a model that never arrived left no reason")
	}
}

// A model that arrived is no longer being got ready: what is in the list is
// what is happening, not a record of what happened.
func TestAModelThatArrivedIsNoLongerBeingPreparedFor(t *testing.T) {
	target := newSaying("a/model", 1, 0, 0)
	opening := func(_ context.Context, tell func(finishedVal, total int64)) (embeddingport.Embedder, error) {
		tell(90_000_000, 90_000_000)
		return target, nil
	}

	tasks := run.NewTasks()
	filling, _, letGo := embedding.Open(t.Context(), tasks,
		embedding.NewFetched(target.GetModel(), "a/model", opening), embedding.Provider{})
	defer func() { _ = letGo() }()

	if _, err := filling.Embed(t.Context(), []string{"anything"}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return len(tasks.List()) == 0 },
		"a model that is here is still being got ready")
}

// A run with no list to tell asks for a model like any other, and waits for it.
// A model that never arrives says why.
func TestARunWithNoListToTellStillOpensAModel(t *testing.T) {
	one, letGo := embedding.OpenOne(t.Context(),
		embedding.NewFetched(is("never-arrives"), "never-arrives", refuseFetch))
	if one == nil {
		t.Fatal("no embedder")
	}
	if letGo == nil {
		t.Fatal("nothing to close")
	}
	defer func() { _ = letGo() }()

	if _, err := one.Embed(t.Context(), []string{"anything"}); err == nil {
		t.Error("a model that is not on this machine embedded something")
	}
}

// waitForComparison is the two providers having been asked the one text they
// it settles is waited for and not assumed.
func waitForComparison(t *testing.T, first, second *saying) {
	t.Helper()
	waitFor(t, func() bool { return first.requestCount.Load() > 0 && second.requestCount.Load() > 0 },
		"the two providers were never compared")
}

// waitFor is something happening somewhere else, waited for.
func waitFor(t *testing.T, finishedVal func() bool, why string) {
	t.Helper()
	for range 400 {
		if finishedVal() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(why)
}

// getOnlyTask is the one piece of work in the list.
func getOnlyTask(t *testing.T, tasks *run.Tasks) run.Task {
	t.Helper()
	target := tasks.List()
	if len(target) != 1 {
		t.Fatalf("the list holds %d pieces of work: %+v", len(target), target)
	}
	return target[0]
}

// waitForFailures is the list once the number of things that stopped badly is
// reached.
func waitForFailures(t *testing.T, tasks *run.Tasks, want int) []run.Task {
	t.Helper()
	var target []run.Task
	waitFor(t, func() bool {
		target = tasks.List()
		got := 0
		for _, item := range target {
			if item.Error != "" {
				got++
			}
		}
		return got == want
	}, "the list never held what stopped badly")
	return target
}
