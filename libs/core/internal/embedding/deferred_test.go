package embedding_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/embedding"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
)

var bge = embeddingport.EmbeddingModel{Name: "bge-m3", Dimensions: 2, MaxTokens: 512, Pooling: "head"}

// landed is a model that has finished arriving.
type landed struct{}

func (landed) GetModel() embeddingport.EmbeddingModel { return bge }

func (landed) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range out {
		out[i] = []float32{1, 0}
	}
	return out, nil
}

func (landed) Close() error { return nil }

// shut is a model that says when it was release.
type shut struct {
	landed
	isClosed bool
}

func (s *shut) Close() error {
	s.isClosed = true
	return nil
}

// What a model is, is known from the settings before the weights are here: the
// vector index is fitted to its width and a vector is claimed under its recipe
// while it comes down.
func TestAModelReturnsWhatItIsBeforeItIsHere(t *testing.T) {
	incoming := embedding.NewDeferredEmbedder(bge)
	if got := incoming.GetImpatientEmbedder().GetModel().Recipe(); got != bge.Recipe() {
		t.Errorf("got %s", got)
	}
	if got := incoming.GetWaitingEmbedder().GetModel().Recipe(); got != bge.Recipe() {
		t.Errorf("got %s", got)
	}
}

func TestAQuestionIsNotMadeToWaitForAModelStillArriving(t *testing.T) {
	incoming := embedding.NewDeferredEmbedder(bge)
	if _, err := incoming.GetImpatientEmbedder().Embed(t.Context(), []string{"anything"}); !errors.Is(err, embedding.ErrModelNotReady) {
		t.Fatalf("got %v", err)
	}
}

func TestAPassFillingTheIndexWaitsForTheModel(t *testing.T) {
	incoming := embedding.NewDeferredEmbedder(bge)
	go func() {
		time.Sleep(10 * time.Millisecond)
		incoming.SetModel(landed{}, nil)
	}()
	got, err := incoming.GetWaitingEmbedder().Embed(t.Context(), []string{"anything"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !slices.Equal(got[0], []float32{1, 0}) {
		t.Errorf("got %v", got)
	}
}

func TestAPassIsNotLeftWaitingOnAModelThatWillNeverCome(t *testing.T) {
	unreachable := errors.New("dial tcp: network is unreachable")
	incoming := embedding.NewDeferredEmbedder(bge)
	incoming.SetModel(nil, unreachable)

	if _, err := incoming.GetWaitingEmbedder().Embed(t.Context(), []string{"anything"}); !errors.Is(err, unreachable) {
		t.Errorf("got %v", err)
	}
	if _, err := incoming.GetImpatientEmbedder().Embed(t.Context(), []string{"anything"}); !errors.Is(err, unreachable) {
		t.Errorf("got %v", err)
	}
}

// A model here and not the one whose vectors are stored is release, and
// nothing is asked of it again.
func TestADisownedModelIsLetGoOfAndNotAsked(t *testing.T) {
	wrong := errors.New("not one model")
	target := &shut{}
	incoming := embedding.NewDeferredEmbedder(bge)
	incoming.SetModel(target, nil)

	if err := incoming.DiscardModel(wrong); err != nil {
		t.Fatal(err)
	}
	if !target.isClosed {
		t.Error("a model nothing will ask of is still loaded")
	}
	if _, err := incoming.GetImpatientEmbedder().Embed(t.Context(), []string{"anything"}); !errors.Is(err, wrong) {
		t.Errorf("got %v", err)
	}
	if _, err := incoming.GetWaitingEmbedder().Embed(t.Context(), []string{"anything"}); !errors.Is(err, wrong) {
		t.Errorf("got %v", err)
	}
}

// A model disowned before it turned up is one nobody waits for.
func TestAModelDisownedBeforeItLandsIsNotWaitedFor(t *testing.T) {
	wrong := errors.New("not one model")
	incoming := embedding.NewDeferredEmbedder(bge)
	if err := incoming.DiscardModel(wrong); err != nil {
		t.Fatal(err)
	}
	if _, err := incoming.GetWaitingEmbedder().Embed(t.Context(), []string{"anything"}); !errors.Is(err, wrong) {
		t.Errorf("got %v", err)
	}
}

// A model release before it arrived is release when it does: the weights
// are compiled by then and nothing else holds them.
func TestAModelThatLandsAfterItWasDisownedIsLetGoOf(t *testing.T) {
	incoming := embedding.NewDeferredEmbedder(bge)
	if err := incoming.DiscardModel(errors.New("not one model")); err != nil {
		t.Fatal(err)
	}

	target := &shut{}
	incoming.SetModel(target, nil)
	if !target.isClosed {
		t.Error("a model nothing will ask of is still loaded")
	}
}

func TestAModelThatLandsAfterEverythingWasClosedIsLetGoOf(t *testing.T) {
	incoming := embedding.NewDeferredEmbedder(bge)
	if err := incoming.Close(); err != nil {
		t.Fatal(err)
	}

	target := &shut{}
	incoming.SetModel(target, nil)
	if !target.isClosed {
		t.Error("a model nothing will ask of is still loaded")
	}
}

// Whoever was waiting on a model release is told why. The reason is in place
// before anybody can see the wait is over.
func TestAModelDisownedTellsWhoeverWasWaitingWhy(t *testing.T) {
	wrong := errors.New("not one model")
	for range 2000 {
		incoming := embedding.NewDeferredEmbedder(bge)

		waitCh := make(chan struct{}, 8)
		waited := make(chan error, cap(waitCh))
		for range cap(waited) {
			go func() {
				waitCh <- struct{}{}
				waited <- incoming.Wait(t.Context())
			}()
		}
		for range cap(waitCh) {
			<-waitCh
		}

		if err := incoming.DiscardModel(wrong); err != nil {
			t.Fatal(err)
		}
		for range cap(waited) {
			if err := <-waited; !errors.Is(err, wrong) {
				t.Fatalf("a model let go of permanently answered %v", err)
			}
		}
	}
}

func TestAPassStoppedWhileTheModelArrivesIsStopped(t *testing.T) {
	incoming := embedding.NewDeferredEmbedder(bge)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := incoming.GetWaitingEmbedder().Embed(ctx, []string{"anything"}); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v", err)
	}
}

// failing is a model whose closing fails.
type failing struct{ landed }

var errCloseFailed = errors.New("close failed")

func (failing) Close() error { return errCloseFailed }

// sized is a model that runs several batches at once.
type sized struct{ landed }

func (sized) Concurrency() int { return 3 }

// closesPromptly fails the test when Close does not return, which is what a
// lock left held does to it.
func closesPromptly(t *testing.T, e *embedding.Embedder) {
	t.Helper()
	isDone := make(chan struct{})
	go func() {
		_ = e.Close()
		close(isDone)
	}()
	select {
	case <-isDone:
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return")
	}
}

func TestTheEmbedderReturnsWhatItIs(t *testing.T) {
	if got := embedding.NewDeferredEmbedder(bge).GetModel(); got != bge {
		t.Errorf("got %v", got)
	}
}

func TestAnEmbedderClosedAfterTheModelLandedIsNotReady(t *testing.T) {
	e := embedding.NewDeferredEmbedder(bge)
	e.SetModel(landed{}, nil)
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
	if err := e.Wait(t.Context()); !errors.Is(err, embedding.ErrModelNotReady) {
		t.Errorf("Wait: got %v", err)
	}
	if _, err := e.GetImpatientEmbedder().Embed(t.Context(), []string{"a"}); !errors.Is(err, embedding.ErrModelNotReady) {
		t.Errorf("Embed: got %v", err)
	}
	if _, err := e.GetWaitingEmbedder().Embed(t.Context(), []string{"a"}); !errors.Is(err, embedding.ErrModelNotReady) {
		t.Errorf("waiting Embed: got %v", err)
	}
}

func TestAskingForVectorsPreservesTheEmbedderFreeToClose(t *testing.T) {
	for name, get := range map[string]func(*embedding.Embedder) embeddingport.Embedder{
		"waiting":   (*embedding.Embedder).GetWaitingEmbedder,
		"impatient": (*embedding.Embedder).GetImpatientEmbedder,
	} {
		t.Run(name, func(t *testing.T) {
			e := embedding.NewDeferredEmbedder(bge)
			e.SetModel(landed{}, nil)
			if _, err := get(e).Embed(t.Context(), []string{"a"}); err != nil {
				t.Fatal(err)
			}
			closesPromptly(t, e)
		})
	}
}

func TestAnUnreadyEmbedderAskedForVectorsPreservesItselfFreeToClose(t *testing.T) {
	e := embedding.NewDeferredEmbedder(bge)
	e.SetModel(landed{}, nil)
	_ = e.Close()
	_, _ = e.GetImpatientEmbedder().Embed(t.Context(), []string{"a"})
	closesPromptly(t, e)
}

func TestAWaitingEmbedderReturnsWhatStoppedTheModel(t *testing.T) {
	why := errors.New("download failed")
	e := embedding.NewDeferredEmbedder(bge)
	e.SetModel(nil, why)
	waiting, ok := e.GetWaitingEmbedder().(embedding.WaitingEmbedder)
	if !ok {
		t.Fatal("the waiting embedder cannot be waited on")
	}
	if err := waiting.Wait(t.Context()); !errors.Is(err, why) {
		t.Errorf("got %v", err)
	}
}

func TestEitherEmbedderReturnsWhatClosingTheModelSaid(t *testing.T) {
	for name, get := range map[string]func(*embedding.Embedder) embeddingport.Embedder{
		"waiting":   (*embedding.Embedder).GetWaitingEmbedder,
		"impatient": (*embedding.Embedder).GetImpatientEmbedder,
	} {
		t.Run(name, func(t *testing.T) {
			e := embedding.NewDeferredEmbedder(bge)
			e.SetModel(failing{}, nil)
			if err := get(e).Close(); !errors.Is(err, errCloseFailed) {
				t.Errorf("got %v", err)
			}
		})
	}
}

func TestConcurrencyIsWhatTheModelRunsAndOneUntilItArrives(t *testing.T) {
	type counter = interface{ Concurrency() int }
	for name, get := range map[string]func(*embedding.Embedder) any{
		"embedder":  func(e *embedding.Embedder) any { return e },
		"waiting":   func(e *embedding.Embedder) any { return e.GetWaitingEmbedder() },
		"impatient": func(e *embedding.Embedder) any { return e.GetImpatientEmbedder() },
	} {
		t.Run(name, func(t *testing.T) {
			counted := func(e any) int {
				c, ok := e.(counter)
				if !ok {
					t.Fatal("no Concurrency")
				}
				return c.Concurrency()
			}
			incoming := embedding.NewDeferredEmbedder(bge)
			if got := counted(get(incoming)); got != 1 {
				t.Errorf("before the model: got %d, want 1", got)
			}
			incoming.SetModel(sized{}, nil)
			if got := counted(get(incoming)); got != 3 {
				t.Errorf("sized model: got %d, want 3", got)
			}
			direct := embedding.NewDeferredEmbedder(bge)
			direct.SetModel(landed{}, nil)
			if got := counted(get(direct)); got != 1 {
				t.Errorf("model without a count: got %d, want 1", got)
			}
			closesPromptly(t, incoming)
		})
	}
}

func TestOnlyTheFirstModelToLandCounts(t *testing.T) {
	first, second := &shut{}, &shut{}
	e := embedding.NewDeferredEmbedder(bge)
	e.SetModel(first, nil)
	e.SetModel(second, nil)
	if first.isClosed {
		t.Error("the model that landed first was let go of")
	}
	if !second.isClosed {
		t.Error("the model that landed second was kept")
	}
	e.SetModel(nil, errors.New("late failure"))
	if err := e.Wait(t.Context()); err != nil {
		t.Errorf("a later failure replaced the model: %v", err)
	}
}
