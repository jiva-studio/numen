package embedding_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/internal/embedding"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

var bge = port.EmbeddingModel{Name: "bge-m3", Dimensions: 2, MaxTokens: 512, Pooling: "head"}

// landed is a model that has finished arriving.
type landed struct{}

func (landed) Model() port.EmbeddingModel { return bge }

func (landed) Embed(_ context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, len(texts))
	for i := range out {
		out[i] = []float32{1, 0}
	}
	return out, nil
}

func (landed) Close() error { return nil }

// shut is a model that says when it was let go of.
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
func TestAModelSaysWhatItIsBeforeItIsHere(t *testing.T) {
	arriving := embedding.NewArriving(bge)
	if got := arriving.GetImpatientEmbedder().Model().Recipe(); got != bge.Recipe() {
		t.Errorf("got %s", got)
	}
	if got := arriving.GetWaitingEmbedder().Model().Recipe(); got != bge.Recipe() {
		t.Errorf("got %s", got)
	}
}

func TestAQuestionIsNotMadeToWaitForAModelStillArriving(t *testing.T) {
	arriving := embedding.NewArriving(bge)
	if _, err := arriving.GetImpatientEmbedder().Embed(t.Context(), []string{"anything"}); !errors.Is(err, embedding.ErrArriving) {
		t.Fatalf("got %v", err)
	}
}

func TestAPassFillingTheIndexWaitsForTheModel(t *testing.T) {
	arriving := embedding.NewArriving(bge)
	go func() {
		time.Sleep(10 * time.Millisecond)
		arriving.ReportArrival(landed{}, nil)
	}()
	got, err := arriving.GetWaitingEmbedder().Embed(t.Context(), []string{"anything"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !slices.Equal(got[0], []float32{1, 0}) {
		t.Errorf("got %v", got)
	}
}

func TestAPassIsNotLeftWaitingOnAModelThatWillNeverCome(t *testing.T) {
	unreachable := errors.New("dial tcp: network is unreachable")
	arriving := embedding.NewArriving(bge)
	arriving.ReportArrival(nil, unreachable)

	if _, err := arriving.GetWaitingEmbedder().Embed(t.Context(), []string{"anything"}); !errors.Is(err, unreachable) {
		t.Errorf("got %v", err)
	}
	if _, err := arriving.GetImpatientEmbedder().Embed(t.Context(), []string{"anything"}); !errors.Is(err, unreachable) {
		t.Errorf("got %v", err)
	}
}

// A model here and not the one whose vectors are stored is let go of, and
// nothing is asked of it again.
func TestADisownedModelIsLetGoOfAndNotAsked(t *testing.T) {
	wrong := errors.New("not one model")
	held := &shut{}
	arriving := embedding.NewArriving(bge)
	arriving.ReportArrival(held, nil)

	if err := arriving.Disown(wrong); err != nil {
		t.Fatal(err)
	}
	if !held.isClosed {
		t.Error("a model nothing will ask of is still loaded")
	}
	if _, err := arriving.GetImpatientEmbedder().Embed(t.Context(), []string{"anything"}); !errors.Is(err, wrong) {
		t.Errorf("got %v", err)
	}
	if _, err := arriving.GetWaitingEmbedder().Embed(t.Context(), []string{"anything"}); !errors.Is(err, wrong) {
		t.Errorf("got %v", err)
	}
}

// A model disowned before it turned up is one nobody waits for.
func TestAModelDisownedBeforeItLandsIsNotWaitedFor(t *testing.T) {
	wrong := errors.New("not one model")
	arriving := embedding.NewArriving(bge)
	if err := arriving.Disown(wrong); err != nil {
		t.Fatal(err)
	}
	if _, err := arriving.GetWaitingEmbedder().Embed(t.Context(), []string{"anything"}); !errors.Is(err, wrong) {
		t.Errorf("got %v", err)
	}
}

// A model let go of before it arrived is let go of when it does: the weights
// are compiled by then and nothing else holds them.
func TestAModelThatLandsAfterItWasDisownedIsLetGoOf(t *testing.T) {
	arriving := embedding.NewArriving(bge)
	if err := arriving.Disown(errors.New("not one model")); err != nil {
		t.Fatal(err)
	}

	held := &shut{}
	arriving.ReportArrival(held, nil)
	if !held.isClosed {
		t.Error("a model nothing will ask of is still loaded")
	}
}

func TestAModelThatLandsAfterEverythingWasClosedIsLetGoOf(t *testing.T) {
	arriving := embedding.NewArriving(bge)
	if err := arriving.Close(); err != nil {
		t.Fatal(err)
	}

	held := &shut{}
	arriving.ReportArrival(held, nil)
	if !held.isClosed {
		t.Error("a model nothing will ask of is still loaded")
	}
}

// Whoever was waiting on a model let go of is told why. The reason is in place
// before anybody can see the wait is over.
func TestAModelDisownedTellsWhoeverWasWaitingWhy(t *testing.T) {
	wrong := errors.New("not one model")
	for range 2000 {
		arriving := embedding.NewArriving(bge)

		waiting := make(chan struct{}, 8)
		waited := make(chan error, cap(waiting))
		for range cap(waited) {
			go func() {
				waiting <- struct{}{}
				waited <- arriving.Wait(t.Context())
			}()
		}
		for range cap(waiting) {
			<-waiting
		}

		if err := arriving.Disown(wrong); err != nil {
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
	arriving := embedding.NewArriving(bge)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := arriving.GetWaitingEmbedder().Embed(ctx, []string{"anything"}); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v", err)
	}
}
