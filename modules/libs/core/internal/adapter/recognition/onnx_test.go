package recognition

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// The messages below are the few fields of an ONNX file that a graph of
// constants and one concatenation needs, written the way protobuf writes them.

func pbVarint(x uint64) []byte {
	var out []byte
	for x >= 0x80 {
		out = append(out, byte(x)|0x80)
		x >>= 7
	}
	return append(out, byte(x))
}

func pbBytes(field int, body []byte) []byte {
	out := pbVarint(uint64(field)<<3 | 2)
	out = append(out, pbVarint(uint64(len(body)))...)
	return append(out, body...)
}

func pbString(field int, s string) []byte { return pbBytes(field, []byte(s)) }

func pbInt(field int, v int) []byte {
	return append(pbVarint(uint64(field)<<3), pbVarint(uint64(v))...)
}

func join(parts ...[]byte) []byte {
	var out []byte
	for _, p := range parts {
		out = append(out, p...)
	}
	return out
}

func onnxFloatTensor(dims []int, values []float32) []byte {
	var packed []byte
	for _, v := range values {
		packed = binary.LittleEndian.AppendUint32(packed, math.Float32bits(v))
	}
	var out []byte
	for _, d := range dims {
		out = append(out, pbInt(1, d)...)
	}
	out = append(out, pbInt(2, 1)...)
	return append(out, pbBytes(4, packed)...)
}

func onnxValueInfo(name string, dims ...int) []byte {
	var shape []byte
	for _, d := range dims {
		shape = append(shape, pbBytes(1, pbInt(1, d))...)
	}
	tensor := join(pbInt(1, 1), pbBytes(2, shape))
	return join(pbString(1, name), pbBytes(2, pbBytes(1, tensor)))
}

func onnxConstant(output string, dims []int, values []float32) []byte {
	attribute := join(pbString(1, "value"), pbBytes(5, onnxFloatTensor(dims, values)), pbInt(20, 4))
	return join(pbString(2, output), pbString(4, "Constant"), pbBytes(5, attribute))
}

func onnxConcat(output string, inputs ...string) []byte {
	var node []byte
	for _, in := range inputs {
		node = append(node, pbString(1, in)...)
	}
	axis := join(pbString(1, "axis"), pbInt(3, 1), pbInt(20, 2))
	return join(node, pbString(2, output), pbString(4, "Concat"), pbBytes(5, axis))
}

func onnxModel(graph []byte, metadata map[string]string) []byte {
	out := join(pbInt(1, 8), pbBytes(8, pbInt(2, 17)), pbBytes(7, graph))
	for k, v := range metadata {
		out = append(out, pbBytes(14, join(pbString(1, k), pbString(2, v)))...)
	}
	return out
}

// layoutInputs are the inputs the layout model is asked, declared at the size
// it reads, and unused by the graphs below.
func layoutInputs(imageSide int) []byte {
	return join(
		pbBytes(11, onnxValueInfo(inputShape, 1, 2)),
		pbBytes(11, onnxValueInfo(inputImage, 1, 3, imageSide, imageSide)),
		pbBytes(11, onnxValueInfo(inputScale, 1, 2)),
	)
}

// echoModel answers one row: the shape and the scale it was given, then three
// numbers of its own.
func echoModel(own [3]float32) []byte {
	graph := join(
		pbBytes(1, onnxConstant("own", []int{1, 3}, own[:])),
		pbBytes(1, onnxConcat(outputBoxes, inputShape, inputScale, "own")),
		pbString(2, "echo"),
		layoutInputs(layoutSide),
		pbBytes(12, onnxValueInfo(outputBoxes, 1, 7)),
	)
	return onnxModel(graph, nil)
}

// tableModel answers the same numbers whatever it is asked.
func tableModel(output string, imageSide int, values []float32) []byte {
	graph := join(
		pbBytes(1, onnxConstant(output, []int{len(values)}, values)),
		pbString(2, "table"),
		layoutInputs(imageSide),
		pbBytes(12, onnxValueInfo(output, len(values))),
	)
	return onnxModel(graph, nil)
}

func writeModel(t *testing.T, dir, name string, body []byte) string {
	t.Helper()
	at := filepath.Join(dir, name)
	if err := os.WriteFile(at, body, 0o644); err != nil {
		t.Fatal(err)
	}
	return at
}
