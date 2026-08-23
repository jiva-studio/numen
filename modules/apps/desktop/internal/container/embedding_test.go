package container_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
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

// What a model is, is known from the settings before the weights are here: the
// vector index is fitted to its width and a vector is claimed under its recipe
// while it comes down.
func TestAModelSaysWhatItIsBeforeItIsHere(t *testing.T) {
	arriving := container.Arriving(bge)
	if got := arriving.Asking().Model().Recipe(); got != bge.Recipe() {
		t.Errorf("got %s", got)
	}
	if got := arriving.Filling().Model().Recipe(); got != bge.Recipe() {
		t.Errorf("got %s", got)
	}
}

func TestAQuestionIsNotMadeToWaitForAModelStillArriving(t *testing.T) {
	arriving := container.Arriving(bge)
	if _, err := arriving.Asking().Embed(t.Context(), []string{"anything"}); !errors.Is(err, container.ErrArriving) {
		t.Fatalf("got %v", err)
	}
}

func TestAPassFillingTheIndexWaitsForTheModel(t *testing.T) {
	arriving := container.Arriving(bge)
	go func() {
		time.Sleep(10 * time.Millisecond)
		arriving.Landed(landed{}, nil)
	}()
	got, err := arriving.Filling().Embed(t.Context(), []string{"anything"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !slices.Equal(got[0], []float32{1, 0}) {
		t.Errorf("got %v", got)
	}
}

func TestAPassIsNotLeftWaitingOnAModelThatWillNeverCome(t *testing.T) {
	unreachable := errors.New("dial tcp: network is unreachable")
	arriving := container.Arriving(bge)
	arriving.Landed(nil, unreachable)

	if _, err := arriving.Filling().Embed(t.Context(), []string{"anything"}); !errors.Is(err, unreachable) {
		t.Errorf("got %v", err)
	}
	if _, err := arriving.Asking().Embed(t.Context(), []string{"anything"}); !errors.Is(err, unreachable) {
		t.Errorf("got %v", err)
	}
}

// A model here and not the one whose vectors are stored is a model nothing is
// asked of.
func TestADisownedModelIsNotAsked(t *testing.T) {
	wrong := errors.New("not one model")
	arriving := container.Arriving(bge)
	arriving.Landed(landed{}, nil)
	arriving.Disown(wrong)

	if arriving.Here() {
		t.Error("still here")
	}
	if _, err := arriving.Asking().Embed(t.Context(), []string{"anything"}); !errors.Is(err, wrong) {
		t.Errorf("got %v", err)
	}
}

func TestAPassStoppedWhileTheModelArrivesIsStopped(t *testing.T) {
	arriving := container.Arriving(bge)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if _, err := arriving.Filling().Embed(ctx, []string{"anything"}); !errors.Is(err, context.Canceled) {
		t.Errorf("got %v", err)
	}
}
