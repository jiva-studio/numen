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

func TestTwoModelsUnderOneNameAreCaughtByWhatTheySay(t *testing.T) {
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
