package embedding_test

import (
	"context"
	"errors"
	"math"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/embedding"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
)

// pointing is an embedder answering every text with the direction given.
type pointing struct {
	is        port.EmbeddingModel
	direction []float32
	why       error
}

func (p pointing) Model() port.EmbeddingModel { return p.is }

func (p pointing) Embed(context.Context, []string) ([][]float32, error) {
	if p.why != nil {
		return nil, p.why
	}
	return [][]float32{slices.Clone(p.direction)}, nil
}

var bge = port.EmbeddingModel{Name: "bge-m3", Dimensions: 3, MaxTokens: 512, Pooling: "head"}

func TestTwoPlacementsOfOneModelAgree(t *testing.T) {
	// Two machines running one model differ by what arithmetic leaves.
	a := pointing{is: bge, direction: []float32{1, 0, 0}}
	b := pointing{is: bge, direction: []float32{float32(math.Sqrt(1 - 1e-6)), 1e-3, 0}}
	if err := embedding.Agree(t.Context(), a, b); err != nil {
		t.Fatal(err)
	}
}

func TestTwoModelsUnderOneNameAreCaughtByWhatTheySay(t *testing.T) {
	// Nothing in a configuration file tells these apart: both are called
	// bge-m3 and both are 3 wide.
	a := pointing{is: bge, direction: []float32{1, 0, 0}}
	b := pointing{is: bge, direction: []float32{0, 1, 0}}
	err := embedding.Agree(t.Context(), a, b)
	if err == nil || !strings.Contains(err.Error(), "not one model") {
		t.Fatalf("got %v", err)
	}
}

func TestAnEmbedderAskedAboutNothingIsNotChecked(t *testing.T) {
	a := pointing{is: bge, direction: []float32{1, 0, 0}}
	if err := embedding.Agree(t.Context(), a, nil); err != nil {
		t.Fatal(err)
	}
	if err := embedding.Agree(t.Context(), nil, a); err != nil {
		t.Fatal(err)
	}
}

func TestAModelThatCannotBeReachedIsNotAModelThatDisagrees(t *testing.T) {
	unreachable := errors.New("dial tcp: network is unreachable")
	a := pointing{is: bge, direction: []float32{1, 0, 0}}
	b := pointing{is: bge, why: unreachable}
	if err := embedding.Agree(t.Context(), a, b); !errors.Is(err, unreachable) {
		t.Fatalf("got %v", err)
	}
}

func TestTwoRecipesAreTwoModelsBeforeEitherIsAsked(t *testing.T) {
	other := bge
	other.Pooling = "mean"
	a := pointing{is: bge, direction: []float32{1, 0, 0}, why: errors.New("must not be asked")}
	b := pointing{is: other, direction: []float32{1, 0, 0}, why: errors.New("must not be asked")}
	err := embedding.Agree(t.Context(), a, b)
	if err == nil || !strings.Contains(err.Error(), "asked for under") {
		t.Fatalf("got %v", err)
	}
}
