package embedding_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/embedding"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/run"
)

var errModelGone = errors.New("the model is gone")

// failWith is a model that is not on this machine, and says why.
func failWith(why error) embedding.OpenModel {
	return func(context.Context, func(done, total int64)) (port.Embedder, error) { return nil, why }
}

// failing is a provider that is reached and fails every text it is asked.
type failing struct {
	*saying
	why error
}

func (f failing) Embed(context.Context, []string) ([][]float32, error) { return nil, f.why }

// counting is a provider that answers every text with a fixed number of copies
// of its vector.
type counting struct {
	*saying
	vectors int
}

func (c counting) Embed(context.Context, []string) ([][]float32, error) {
	out := make([][]float32, c.vectors)
	for i := range out {
		out[i] = c.vector
	}
	return out, nil
}

// closing is a provider that says what letting go of it said.
type closing struct {
	*saying
	why error
}

func (c closing) Close() error {
	_ = c.saying.Close()
	return c.why
}

// waitForDisowned is the provider that answers questions having been release.
func waitForDisowned(t *testing.T, second *saying) {
	t.Helper()
	waitFor(t, func() bool { return second.closed.Load() > 0 }, "the second provider was not let go of")
}

// A question asked of a provider that was release is told why, and the reason
// is the one that stopped the comparison.
func TestAProviderLetGoOfReturnsWhatStoppedTheComparison(t *testing.T) {
	for _, c := range []struct {
		name          string
		first, second func(second *saying) embedding.Provider
	}{
		{
			name: "the first never arrives",
			first: func(*saying) embedding.Provider {
				return embedding.NewFetched(is("first"), "first", failWith(errModelGone))
			},
			second: func(s *saying) embedding.Provider { return embedding.NewReached(s.model, "second", s) },
		},
		{
			name: "the first fails to answer",
			first: func(*saying) embedding.Provider {
				f := failing{saying: newSaying("first", 1, 0, 0), why: errModelGone}
				return embedding.NewReached(f.model, "first", f)
			},
			second: func(s *saying) embedding.Provider { return embedding.NewReached(s.model, "second", s) },
		},
		{
			name: "the second fails to answer",
			first: func(*saying) embedding.Provider {
				f := newSaying("first", 1, 0, 0)
				return embedding.NewReached(f.model, "first", f)
			},
			second: func(s *saying) embedding.Provider {
				return embedding.NewReached(s.model, "second", failing{saying: s, why: errModelGone})
			},
		},
	} {
		t.Run(c.name, func(t *testing.T) {
			second := newSaying("second", 1, 0, 0)
			_, asking, letGo := embedding.Open(t.Context(), run.NewTasks(), c.first(second), c.second(second))
			defer func() { _ = letGo() }()

			waitForDisowned(t, second)
			_, err := asking.Embed(t.Context(), []string{"anything"})
			if !errors.Is(err, errModelGone) {
				t.Errorf("a question was answered with %v, want the reason the comparison stopped", err)
			}
		})
	}
}

// A first model that is known not to come is not waited out by the second: the
// second is release at once.
func TestASecondProviderIsNotWaitedForOnceTheFirstNeverArrives(t *testing.T) {
	tasks := run.NewTasks()
	waitFn := func(ctx context.Context, _ func(done, total int64)) (port.Embedder, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	_, asking, letGo := embedding.Open(t.Context(), tasks,
		embedding.NewFetched(is("first"), "first", failWith(errModelGone)),
		embedding.NewFetched(is("second"), "second", waitFn))
	defer func() { _ = letGo() }()

	waitFor(t, func() bool {
		_, err := asking.Embed(t.Context(), []string{"anything"})
		return err != nil && strings.Contains(err.Error(), "not compared as one model")
	}, "the second provider was waited for")
}

// A second model that is known not to come is not waited out by an answer of
// the first.
func TestAFirstProviderIsNotAskedOnceTheSecondNeverArrives(t *testing.T) {
	first := newSaying("first", 1, 0, 0)
	request := make(chan struct{})
	t.Cleanup(func() { close(request) })
	blocked := blocking{saying: first, until: request}

	_, asking, letGo := embedding.Open(t.Context(), run.NewTasks(),
		embedding.NewReached(first.model, "first", blocked),
		embedding.NewFetched(is("second"), "second", failWith(errModelGone)))
	defer func() { _ = letGo() }()

	waitFor(t, func() bool {
		_, err := asking.Embed(t.Context(), []string{"anything"})
		return err != nil && strings.Contains(err.Error(), "not compared as one model")
	}, "the first provider was asked before the second was known to come")
}

// blocking is a provider that answers nothing until it is released.
type blocking struct {
	*saying
	until chan struct{}
}

func (b blocking) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	select {
	case <-b.until:
	case <-ctx.Done():
		return nil, ctx.Err()
	}
	return b.saying.Embed(ctx, texts)
}

// A model that answers the text with other than one vector is not one model with
// the first, whichever of the two does it.
func TestProvidersAnsweringWithOtherThanOneVectorAreNotOneModel(t *testing.T) {
	for _, c := range []struct {
		name         string
		said, backed int
	}{
		{name: "the first answers none", said: 0, backed: 1},
		{name: "the first answers two", said: 2, backed: 1},
		{name: "the second answers none", said: 1, backed: 0},
		{name: "the second answers two", said: 1, backed: 2},
	} {
		t.Run(c.name, func(t *testing.T) {
			first, second := newSaying("first", 1, 0, 0), newSaying("second", 1, 0, 0)

			tasks := run.NewTasks()
			_, asking, letGo := embedding.Open(t.Context(), tasks,
				embedding.NewReached(first.model, "first", counting{saying: first, vectors: c.said}),
				embedding.NewReached(second.model, "second", counting{saying: second, vectors: c.backed}))
			defer func() { _ = letGo() }()

			waitForDisowned(t, second)
			_, err := asking.Embed(t.Context(), []string{"anything"})
			if err == nil || !strings.Contains(err.Error(), "not one model") {
				t.Errorf("a question was answered with %v, want the models called two", err)
			}
		})
	}
}

// A comparison ended by the run being stopped is no disagreement: the second
// provider is kept and nothing is said against it.
func TestAComparisonEndedByTheRunStoppingLetsNobodyGo(t *testing.T) {
	second := newSaying("second", 1, 0, 0)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	waitFn := func(ctx context.Context, _ func(done, total int64)) (port.Embedder, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}

	tasks := run.NewTasks()
	_, _, letGo := embedding.Open(ctx, tasks,
		embedding.NewFetched(is("first"), "first", waitFn),
		embedding.NewReached(second.model, "second", second))
	defer func() { _ = letGo() }()

	cancel()
	time.Sleep(100 * time.Millisecond)

	if second.closed.Load() != 0 {
		t.Error("the second provider was let go of by a run that was stopped")
	}
	for _, at := range tasks.List() {
		if at.About == "second" && at.Error != "" {
			t.Errorf("the second provider is in the list as %q", at.Error)
		}
	}
}

// A run with no list to tell opens a model that arrives, and tells nobody.
func TestAModelOpenedWithNoListToTellArrives(t *testing.T) {
	target := newSaying("a/model", 1, 0, 0)
	opening := func(_ context.Context, tell func(done, total int64)) (port.Embedder, error) {
		tell(1, 10)
		return target, nil
	}

	filling, _, letGo := embedding.Open(t.Context(), nil,
		embedding.NewFetched(target.model, "a/model", opening), embedding.Provider{})
	defer func() { _ = letGo() }()

	if _, err := filling.Embed(t.Context(), []string{"anything"}); err != nil {
		t.Fatal(err)
	}
}

// The first byte of a model that is here is a share drawn, and a line that has
// counted nothing draws none.
func TestTheFirstByteOfTheModelIsGeneratedAsAShare(t *testing.T) {
	progressCh := make(chan func(done, total int64), 1)
	stop := make(chan struct{})
	opening := func(_ context.Context, tell func(done, total int64)) (port.Embedder, error) {
		progressCh <- tell
		<-stop
		return nil, errors.New("none of the model is on this machine")
	}

	tasks := run.NewTasks()
	_, _, letGo := embedding.Open(t.Context(), tasks,
		embedding.NewFetched(is("a/model"), "a/model", opening), embedding.Provider{})
	defer func() { _ = letGo() }()
	defer close(stop)

	if at := getOnlyTask(t, tasks); at.Count != 0 || at.Unit != run.Things {
		t.Errorf("a line that has counted nothing is drawn as %d %v", at.Count, at.Unit)
	}

	(<-progressCh)(1, 10)
	at := getOnlyTask(t, tasks)
	if at.Count != 1 || at.Total != 10 || at.Unit != run.Bytes {
		t.Errorf("the first byte is drawn as %d of %d %v", at.Count, at.Total, at.Unit)
	}
}

// Letting go of two providers lets go of both, and says what the second said
// before what the first said.
func TestLettingGoOfTwoProvidersLetsGoOfBothAndReturnsWhyOnce(t *testing.T) {
	errFirst, errSecond := errors.New("the first would not close"), errors.New("the second would not close")
	for _, c := range []struct {
		name          string
		first, second error
		want          error
	}{
		{name: "both close", want: nil},
		{name: "the first would not", first: errFirst, want: errFirst},
		{name: "the second would not", second: errSecond, want: errSecond},
		{name: "neither would", first: errFirst, second: errSecond, want: errSecond},
	} {
		t.Run(c.name, func(t *testing.T) {
			first, second := newSaying("first", 1, 0, 0), newSaying("second", 1, 0, 0)
			_, _, letGo := embedding.Open(t.Context(), run.NewTasks(),
				embedding.NewReached(first.model, "first", closing{saying: first, why: c.first}),
				embedding.NewReached(second.model, "second", closing{saying: second, why: c.second}))

			waitForComparison(t, first, second)
			err := letGo()
			if !errors.Is(err, c.want) || (c.want == nil && err != nil) {
				t.Errorf("let go = %v, want %v", err, c.want)
			}
			if first.closed.Load() != 1 || second.closed.Load() != 1 {
				t.Errorf("closed %d and %d, want each once", first.closed.Load(), second.closed.Load())
			}
		})
	}
}
