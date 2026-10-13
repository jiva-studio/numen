package onnxruntime_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func appendVarint(out []byte, v uint64) []byte {
	for v >= 0x80 {
		out = append(out, byte(v)|0x80)
		v >>= 7
	}
	return append(out, byte(v))
}

func appendBytesField(out []byte, field int, body []byte) []byte {
	out = appendVarint(out, uint64(field)<<3|wireBytes)
	out = appendVarint(out, uint64(len(body)))
	return append(out, body...)
}

func appendVarintField(out []byte, field int, v uint64) []byte {
	out = appendVarint(out, uint64(field)<<3|wireVarint)
	return appendVarint(out, v)
}

// encodeValueInfo writes a tensor input the way an ONNX file does. A dimension
// of dynamicDim is written as a named parameter.
func encodeValueInfo(name string, element uint64, dims ...int64) []byte {
	var shape []byte
	for i, extent := range dims {
		var dim []byte
		if extent == dynamicDim {
			dim = appendBytesField(dim, 2, []byte("axis"+string(rune('a'+i))))
		} else {
			dim = appendVarintField(dim, 1, uint64(extent))
		}
		shape = appendBytesField(shape, 1, dim)
	}
	var tensor []byte
	tensor = appendVarintField(tensor, 1, element)
	tensor = appendBytesField(tensor, 2, shape)
	typed := appendBytesField(nil, 1, tensor)

	info := appendBytesField(nil, 1, []byte(name))
	return appendBytesField(info, 2, typed)
}

func TestTheSignatureOfAModelIsReadFromItsFile(t *testing.T) {
	weight := appendVarintField(nil, 1, 4)
	weight = appendBytesField(weight, 8, []byte("bias"))
	weight = appendBytesField(weight, 9, make([]byte, 4096))

	var graph []byte
	graph = appendBytesField(graph, 5, weight)
	graph = appendBytesField(graph, 11, encodeValueInfo("ids", uint64(elementInt64), dynamicDim, 7))
	graph = appendBytesField(graph, 11, encodeValueInfo("bias", uint64(elementFloat32), 4))
	graph = appendBytesField(graph, 12, encodeValueInfo("scores", uint64(elementFloat32), dynamicDim, 3, 9))

	model := appendVarintField(nil, 1, 8)
	model = appendBytesField(model, 7, graph)

	path := filepath.Join(t.TempDir(), "m.onnx")
	if err := os.WriteFile(path, model, 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readModelSignature(path)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Inputs) != 1 {
		t.Fatalf("inputs = %+v, want ids alone: bias is a weight", got.Inputs)
	}
	ids := got.Inputs[0]
	if ids.Name != "ids" || ids.Type != elementInt64 || !slices.Equal(ids.Dims, []int64{dynamicDim, 7}) || ids.Params[0] != "axisa" {
		t.Errorf("ids = %+v", ids)
	}
	if len(got.Outputs) != 1 || !slices.Equal(got.Outputs[0].Dims, []int64{dynamicDim, 3, 9}) {
		t.Errorf("outputs = %+v", got.Outputs)
	}
}

func TestAShapeKeepsTheFixedExtentsAndFillsTheDynamicOnes(t *testing.T) {
	in := tensorSignature{Name: "x", Dims: []int64{dynamicDim, 3, dynamicDim, dynamicDim}, Params: []string{"n", "", "h", "w"}}

	got, err := shapeOf(in, map[int]int64{0: 2, 2: 48, 3: 160}, 0)
	if err != nil || !slices.Equal(got, []int64{2, 3, 48, 160}) {
		t.Errorf("shapeOf = %v, %v", got, err)
	}

	got, err = shapeOf(in, nil, 1)
	if err != nil || !slices.Equal(got, []int64{1, 3, 1, 1}) {
		t.Errorf("shapeOf with a rest extent = %v, %v", got, err)
	}

	if _, err = shapeOf(in, map[int]int64{0: 1}, 0); err == nil {
		t.Error("a dynamic dimension left without an extent was accepted")
	}

	fixed := tensorSignature{Name: "x", Dims: []int64{dynamicDim, 3, 48, dynamicDim}, Params: []string{"n", "", "", "w"}}
	got, err = shapeOf(fixed, map[int]int64{0: 1, 2: 99, 3: 10}, 0)
	if err != nil || !slices.Equal(got, []int64{1, 3, 48, 10}) {
		t.Errorf("a fixed dimension took an extent: %v, %v", got, err)
	}
}

func TestAScalarHoldsOneElement(t *testing.T) {
	if got := getElementCount(nil); got != 1 {
		t.Errorf("getElementCount(nil) = %d, want 1", got)
	}
	if got := getElementCount([]int64{2, 3, 4}); got != 24 {
		t.Errorf("getElementCount = %d, want 24", got)
	}
}

func TestAModelIsRecognisedByWhatItDeclares(t *testing.T) {
	image := func(dims ...int64) tensorSignature { return tensorSignature{Name: "x", Dims: dims} }
	named := func(names ...string) []tensorSignature {
		var out []tensorSignature
		for _, name := range names {
			out = append(out, tensorSignature{Name: name})
		}
		return out
	}
	for _, tc := range []struct {
		name string
		sig  modelSignature
		want modelRole
	}{
		{"encoder", modelSignature{Inputs: named("audio_signal", "length")}, roleEncoder},
		{"decoder", modelSignature{Inputs: named("targets", "target_length", "states.1")}, roleDecoder},
		{"joiner", modelSignature{Inputs: named("encoder_outputs", "decoder_outputs")}, roleJoiner},
		{"voice activity", modelSignature{Inputs: named("input", "state", "sr")}, roleVAD},
		{"layout", modelSignature{Inputs: named("im_shape", "image", "scale_factor")}, roleLayout},
		{"detection", modelSignature{
			Inputs:  []tensorSignature{image(-1, 3, -1, -1)},
			Outputs: []tensorSignature{image(-1, 1, -1, -1)},
		}, roleDetect},
		{"recognition", modelSignature{
			Inputs:  []tensorSignature{image(-1, 3, 48, -1)},
			Outputs: []tensorSignature{{Name: "y", Dims: []int64{-1, -1, 6906}}},
		}, roleRecognise},
		{"none", modelSignature{Inputs: named("a", "b")}, roleOther},
	} {
		if got := classifyModel(tc.sig); got != tc.want {
			t.Errorf("%s: classified as %q, want %q", tc.name, got, tc.want)
		}
	}
}
