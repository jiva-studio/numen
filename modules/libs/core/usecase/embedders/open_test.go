package embedders_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/embedding"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/task"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/embedders"
)

// is the identity a provider's vectors are kept under. Nothing here reads a
// settings file: what a vector is has been settled before any of this is called.
func is(name string) port.EmbeddingModel {
	return port.EmbeddingModel{
		Name: name, Dimensions: 3, MaxTokens: 64, Pooling: "mean",
		From: "service:https://vectors.invalid/v1/" + name,
	}
}

// saying is a provider that answers every text with the one vector it was
// given. It counts the times it was asked the text two providers are compared
// over, and the times it was let go of.
type saying struct {
	model  port.EmbeddingModel
	vector []float32
	asked  atomic.Int64
	closed atomic.Int64
}

func answering(name string, vector ...float32) *saying {
	return &saying{model: is(name), vector: vector}
}

func (s *saying) Model() port.EmbeddingModel { return s.model }

func (s *saying) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i, one := range texts {
		if one == embedding.Asked {
			s.asked.Add(1)
		}
		out[i] = s.vector
	}
	return out, nil
}

func (s *saying) Close() error {
	s.closed.Add(1)
	return nil
}

// unfetched is a model that is not on this machine and is not to be fetched.
func unfetched(context.Context, func(done, total int64)) (port.Embedder, error) {
	return nil, errors.New("the model is not on this machine")
}

// An installation that places its index nowhere embeds with nothing, and there
// is nothing to let go of.
func TestAnInstallationNamingNoProviderEmbedsWithNothing(t *testing.T) {
	filling, asking, letGo := embedders.Open(t.Context(), task.New(),
		embedders.Provider{}, embedders.Provider{})
	if filling != nil || asking != nil {
		t.Errorf("got %v and %v", filling, asking)
	}
	if letGo != nil {
		t.Error("got something to close")
	}

	one, letGoOfOne := embedders.One(t.Context(), embedders.Provider{})
	if one != nil {
		t.Errorf("got an embedder: %v", one.Model())
	}
	if letGoOfOne != nil {
		t.Error("got something to close")
	}
}

// Saying nothing about questions is asking the way the vault was indexed: one
// provider is one model, waited for on the way in and not waited for on the way
// out.
func TestOneProviderIsOneModelSeenTwoWays(t *testing.T) {
	held := answering("one-provider", 1, 0, 0)

	filling, asking, letGo := embedders.Open(t.Context(), task.New(),
		embedders.Reached(held.Model(), "one-provider", held), embedders.Provider{})
	if filling == nil || asking == nil {
		t.Fatalf("got %v and %v", filling, asking)
	}
	if a, b := filling.Model().Recipe(), asking.Model().Recipe(); a != b {
		t.Errorf("%s and %s", a, b)
	}
	if err := letGo(); err != nil {
		t.Fatal(err)
	}
	if held.closed.Load() == 0 {
		t.Error("the provider was not let go of")
	}
}

// Two providers answering one text alike are one model, and both go on
// answering.
func TestTwoProvidersAnsweringAlikeAreOneModel(t *testing.T) {
	first := answering("indexed-here", 1, 0, 0)
	second := answering("asked-elsewhere", 1, 0, 0)

	tasks := task.New()
	filling, asking, letGo := embedders.Open(t.Context(), tasks,
		embedders.Reached(first.Model(), "indexed-here", first),
		embedders.Reached(second.Model(), "asked-elsewhere", second))
	defer func() { _ = letGo() }()

	compared(t, first, second)
	if listed := tasks.List(); len(listed) != 0 {
		t.Errorf("two providers of one model are in the list: %+v", listed)
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
// is let go of: a question embedded in another space finds nothing the first
// indexed. It is said in the list of what is being done.
func TestTwoProvidersThatDisagreeAreNotOneModel(t *testing.T) {
	first := answering("indexed-here", 1, 0, 0)
	second := answering("somewhere-else", 0, 1, 0)

	tasks := task.New()
	_, asking, letGo := embedders.Open(t.Context(), tasks,
		embedders.Reached(first.Model(), "indexed-here", first),
		embedders.Reached(second.Model(), "somewhere-else", second))
	defer func() { _ = letGo() }()

	held := failing(t, tasks, 1)
	if len(held) != 1 {
		t.Fatalf("the list holds %d pieces of work: %+v", len(held), held)
	}
	if held[0].About != "somewhere-else" {
		t.Errorf("the provider that answers questions is in the list as %q", held[0].About)
	}
	if _, err := asking.Embed(t.Context(), []string{"anything"}); err == nil {
		t.Error("a question was embedded by a model the index knows nothing about")
	}
	waiting(t, func() bool { return second.closed.Load() > 0 }, "the second provider was not let go of")
}

// A comparison nobody got an answer out of is not agreement. The model the
// index would be filled by never arrives, so the provider that answers
// questions is let go of, and both halves are in the list under what stopped
// them.
func TestTwoProvidersThatCouldNotBeComparedAreNotOneModel(t *testing.T) {
	second := answering("asked-elsewhere", 1, 0, 0)

	tasks := task.New()
	_, _, letGo := embedders.Open(t.Context(), tasks,
		embedders.Fetched(is("never-arrives"), "never-arrives", unfetched),
		embedders.Reached(second.Model(), "asked-elsewhere", second))
	defer func() { _ = letGo() }()

	if held := failing(t, tasks, 2); len(held) != 2 {
		t.Fatalf("the list holds %d pieces of work: %+v", len(held), held)
	}
	waiting(t, func() bool { return second.closed.Load() > 0 }, "the second provider was not let go of")
}

// Two providers naming one repository are two lines, and how far one has got is
// not written over by the other.
func TestTwoProvidersOfOneRepositoryAreTwoLines(t *testing.T) {
	tasks := task.New()
	_, _, letGo := embedders.Open(t.Context(), tasks,
		embedders.Fetched(is("one/repository"), "one/repository", unfetched),
		embedders.Fetched(is("one/repository"), "one/repository", unfetched))
	defer func() { _ = letGo() }()

	held := failing(t, tasks, 2)
	if len(held) != 2 {
		t.Fatalf("two providers are %d lines: %+v", len(held), held)
	}
	if held[0].ID == held[1].ID {
		t.Errorf("two providers share the line %q", held[0].ID)
	}
}

// A provider reached at once is nothing a person waits for, so it stands in no
// list.
func TestAProviderReachedAtOnceStandsInNoList(t *testing.T) {
	held := answering("reached-at-once", 1, 0, 0)

	tasks := task.New()
	_, _, letGo := embedders.Open(t.Context(), tasks,
		embedders.Reached(held.Model(), "reached-at-once", held), embedders.Provider{})
	defer func() { _ = letGo() }()

	if listed := tasks.List(); len(listed) != 0 {
		t.Errorf("a provider nobody waits for is getting ready: %+v", listed)
	}
}

// The model's own line draws no share until some of the model is here. A count
// of nothing against a size is a row sitting at nought per cent for the length
// of a download, which says less than a ring that turns.
func TestTheModelDrawsNoShareBeforeAnyOfItIsHere(t *testing.T) {
	told := make(chan func(done, total int64), 1)
	stop := make(chan struct{})
	opening := func(_ context.Context, tell func(done, total int64)) (port.Embedder, error) {
		told <- tell
		<-stop
		return nil, errors.New("none of the model is on this machine")
	}

	tasks := task.New()
	_, _, letGo := embedders.Open(t.Context(), tasks,
		embedders.Fetched(is("a/model"), "a/model", opening), embedders.Provider{})
	defer func() { _ = letGo() }()

	// Nothing is known yet: the line is in the list from the moment it starts.
	at := only(t, tasks)
	if at.Doing != "Preparing the model" || at.About != "a/model" {
		t.Errorf("the line is shown as %q about %q", at.Doing, at.About)
	}
	if at.Total != 0 {
		t.Errorf("a line that has counted nothing is drawn against %d", at.Total)
	}

	tell := <-told

	// The cache is read before the first bytes land, and the size is known.
	tell(0, 90_000_000)
	if at := only(t, tasks); at.Total != 0 || at.Count != 0 {
		t.Errorf("a model with none of it here is drawn as %d of %d", at.Count, at.Total)
	}

	tell(30_000_000, 90_000_000)
	at = only(t, tasks)
	if at.Count != 30_000_000 || at.Total != 90_000_000 {
		t.Errorf("the model is drawn as %d of %d", at.Count, at.Total)
	}
	if at.Unit != task.Bytes {
		t.Errorf("a model is counted as %v", at.Unit)
	}

	// The model never turns up, and the line stays in the list under the reason.
	close(stop)
	if held := failing(t, tasks, 1); held[0].Error == "" {
		t.Error("a model that never arrived left no reason")
	}
}

// A model that arrived is no longer being got ready: what is in the list is
// what is happening, not a record of what happened.
func TestAModelThatArrivedIsNoLongerBeingPreparedFor(t *testing.T) {
	held := answering("a/model", 1, 0, 0)
	opening := func(_ context.Context, tell func(done, total int64)) (port.Embedder, error) {
		tell(90_000_000, 90_000_000)
		return held, nil
	}

	tasks := task.New()
	filling, _, letGo := embedders.Open(t.Context(), tasks,
		embedders.Fetched(held.Model(), "a/model", opening), embedders.Provider{})
	defer func() { _ = letGo() }()

	if _, err := filling.Embed(t.Context(), []string{"anything"}); err != nil {
		t.Fatal(err)
	}
	waiting(t, func() bool { return len(tasks.List()) == 0 },
		"a model that is here is still being got ready")
}

// A run with no list to tell asks for a model like any other, and waits for it.
// A model that never arrives says why.
func TestARunWithNoListToTellStillOpensAModel(t *testing.T) {
	one, letGo := embedders.One(t.Context(),
		embedders.Fetched(is("never-arrives"), "never-arrives", unfetched))
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

// compared is the two providers having been asked the one text they are held to
// answering alike. The comparison runs behind the caller, so what it settles is
// waited for and not assumed.
func compared(t *testing.T, first, second *saying) {
	t.Helper()
	waiting(t, func() bool { return first.asked.Load() > 0 && second.asked.Load() > 0 },
		"the two providers were never compared")
}

// waiting is something happening somewhere else, waited for.
func waiting(t *testing.T, done func() bool, why string) {
	t.Helper()
	for range 400 {
		if done() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(why)
}

// only is the one piece of work in the list.
func only(t *testing.T, tasks *task.Tasks) task.Task {
	t.Helper()
	held := tasks.List()
	if len(held) != 1 {
		t.Fatalf("the list holds %d pieces of work: %+v", len(held), held)
	}
	return held[0]
}

// failing is the list once the number of things that stopped badly is reached.
func failing(t *testing.T, tasks *task.Tasks, want int) []task.Task {
	t.Helper()
	var held []task.Task
	waiting(t, func() bool {
		held = tasks.List()
		got := 0
		for _, at := range held {
			if at.Error != "" {
				got++
			}
		}
		return got == want
	}, "the list never held what stopped badly")
	return held
}
