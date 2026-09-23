package embed

import (
	"slices"
	"testing"
)

// What BAAI/bge-m3 publishes beside its model: weights in a second file, and a
// constant the graph points at in a third.
var bge = []string{
	"Constant_7_attr__value",
	"config.json",
	"model.onnx",
	"model.onnx_data",
	"sentencepiece.bpe.model",
	"tokenizer.json",
}

// What intfloat/multilingual-e5-large publishes: the full build with its
// weights beside it, and two smaller builds of the same model.
var e5 = []string{
	"config.json",
	"model.onnx",
	"model.onnx_data",
	"model_O4.onnx",
	"model_qint8_avx512_vnni.onnx",
	"tokenizer.json",
}

func TestAModelInSeveralFilesComesWithAllOfThem(t *testing.T) {
	got := selectModelFiles(bge, "model.onnx")
	want := []string{
		"model.onnx",
		"Constant_7_attr__value",
		"config.json",
		"model.onnx_data",
		"sentencepiece.bpe.model",
		"tokenizer.json",
	}
	if !slices.Equal(got, want) {
		t.Errorf("got %v", got)
	}
}

func TestAQuantisedBuildDoesNotDragDownTheFullOne(t *testing.T) {
	// The weights beside model.onnx belong to a model nobody asked for.
	got := selectModelFiles(e5, "model_qint8_avx512_vnni.onnx")
	want := []string{"model_qint8_avx512_vnni.onnx", "config.json", "tokenizer.json"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v", got)
	}
}

func TestTheFullBuildComesWithItsOwnWeightsAndNoOtherBuild(t *testing.T) {
	got := selectModelFiles(e5, "model.onnx")
	want := []string{"model.onnx", "config.json", "model.onnx_data", "tokenizer.json"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v", got)
	}
}
