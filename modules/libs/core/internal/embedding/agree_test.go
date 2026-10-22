package embedding_test

import (
	"math"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/embedding"
)

func TestTwoMachinesRunningOneModelAgree(t *testing.T) {
	// Two providers of one model differ by what arithmetic leaves.
	a := []float32{1, 0, 0}
	b := []float32{float32(math.Sqrt(1 - 1e-6)), 1e-3, 0}
	if !embedding.IsAgreed(a, b) {
		t.Error("one model was taken for two")
	}
}

func TestTwoModelsBelowOneNameAreCaughtByWhatTheySay(t *testing.T) {
	// Nothing in a settings file tells these apart: both are called bge-m3 and
	// both are three wide.
	if embedding.IsAgreed([]float32{1, 0, 0}, []float32{0, 1, 0}) {
		t.Error("two models were taken for one")
	}
}

func TestVectorsOfTwoWidthsAreTwoModels(t *testing.T) {
	if embedding.IsAgreed([]float32{1, 0, 0}, []float32{1, 0}) {
		t.Error("two widths were taken for one model")
	}
	if embedding.IsAgreed(nil, nil) {
		t.Error("nothing was taken for a model")
	}
}

func TestOneDimensionEachWayCanAgree(t *testing.T) {
	if !embedding.IsAgreed([]float32{1}, []float32{1}) {
		t.Error("one model was taken for two")
	}
}

func TestVectorsActiveExactlyAtTheAgreementAgree(t *testing.T) {
	// The dot product of these is exactly Agreement: a float32 head and two
	// float32 pieces that sum to the rest of it.
	target := float64(embedding.Agreement)
	head := float32(target)
	rest := target - float64(head)
	first := float32(rest)
	second := float32(rest - float64(first))
	if float64(head)+float64(first)+float64(second) != target {
		t.Fatal("the vectors do not sum to the agreement")
	}
	a := []float32{head, first, second}
	b := []float32{1, 1, 1}
	if !embedding.IsAgreed(a, b) {
		t.Error("vectors at the agreement were taken for two models")
	}
}
