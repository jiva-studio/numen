package gomlx

import (
	"encoding/binary"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"google.golang.org/protobuf/encoding/protowire"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// The fixture is a sentence encoder small enough to write out by hand. Its
// vocabulary is eight words: [UNK] [CLS] [SEP] [PAD] a b c d, in that order of
// id. Every input is looked up in a table of unit vectors, so the vector a text
// comes back as counts what the model was given:
//
//	column id        how many times that token was fed
//	column vocab     positions the mask marks as padding
//	column vocab+1   positions the mask marks as text
//	column vocab+2   positions, when the model takes token types
//
// The pooled model sums the positions of a text; the other returns them one by
// one.
const (
	fixtureVocab = 8
	fixtureWidth = fixtureVocab + 3

	idUnknown = 0
	idStart   = 1
	idEnd     = 2
	idPad     = 3
	idA       = 4
	idB       = 5
	idC       = 6
	idD       = 7
)

const (
	onnxFloat  = 1
	onnxInt64  = 7
	onnxDouble = 11

	attrInt  = 2
	attrInts = 7
)

type dim struct {
	value int64
	param string
}

type modelSpec struct {
	// typed adds the token_type_ids input.
	typed bool
	// pooled makes the model return one vector per text.
	pooled bool
	// double returns float64.
	double bool
	// fixedSeq builds a graph that holds for sequences of 64 tokens only.
	fixedSeq bool
	// declared replaces the shape the output says it has.
	declared []dim
	// badWeights gives the first table fewer bytes than its shape holds.
	badWeights bool
	// inputs, when set, makes a model that only names these inputs.
	inputs []string
}

type tokenizerSpec struct {
	noPad bool
	// wrap adds [CLS] before and [SEP] after every text.
	wrap bool
}

type fixture struct {
	model      modelSpec
	tokenizer  tokenizerSpec
	maxTokens  int
	batchTexts int
	pooling    string
}

func (f fixture) getIdentity() port.EmbeddingModel {
	maxTokens := f.maxTokens
	if maxTokens == 0 {
		maxTokens = 256
	}
	return port.EmbeddingModel{
		Name: "tiny", Dimensions: fixtureWidth, MaxTokens: maxTokens, Pooling: f.pooling, From: "this machine",
	}
}

func (f fixture) getLocal(dir string) embed.LocalModel {
	return embed.LocalModel{Name: "tiny", Dir: dir, BatchTexts: f.batchTexts}
}

// isolate keeps any path a mutated build resolves inside a temporary directory.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
}

func writeFile(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
}

// writeFixture writes the model and its tokeniser into a new directory.
func writeFixture(t *testing.T, f fixture) string {
	t.Helper()
	isolate(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, embed.ModelFile), buildModel(f.model))
	writeFile(t, filepath.Join(dir, embed.TokenizerFile), buildTokenizer(f.tokenizer))
	return dir
}

// openFixture opens the fixture model and closes it when the test ends.
func openFixture(t *testing.T, f fixture) *Embedder {
	t.Helper()
	dir := writeFixture(t, f)
	e, err := Open(t.Context(), f.getIdentity(), f.getLocal(dir), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = e.Close() })
	return e
}

func buildTokenizer(spec tokenizerSpec) []byte {
	added := []map[string]any{
		{"id": idUnknown, "content": "[UNK]", "special": true},
		{"id": idStart, "content": "[CLS]", "special": true},
		{"id": idEnd, "content": "[SEP]", "special": true},
	}
	if !spec.noPad {
		added = append(added, map[string]any{"id": idPad, "content": "[PAD]", "special": true})
	}
	doc := map[string]any{
		"version":       "1.0",
		"added_tokens":  added,
		"pre_tokenizer": map[string]any{"type": "WhitespaceSplit"},
		"model": map[string]any{
			"type":      "WordPiece",
			"unk_token": "[UNK]",
			"vocab": map[string]int{
				"[UNK]": idUnknown, "[CLS]": idStart, "[SEP]": idEnd, "[PAD]": idPad,
				"a": idA, "b": idB, "c": idC, "d": idD,
			},
		},
	}
	if spec.wrap {
		doc["post_processor"] = map[string]any{
			"type": "TemplateProcessing",
			"single": []map[string]any{
				{"SpecialToken": map[string]any{"id": "[CLS]", "type_id": 0}},
				{"Sequence": map[string]any{"id": "A", "type_id": 0}},
				{"SpecialToken": map[string]any{"id": "[SEP]", "type_id": 0}},
			},
			"special_tokens": map[string]any{
				"[CLS]": map[string]any{"id": "[CLS]", "ids": []int{idStart}, "tokens": []string{"[CLS]"}},
				"[SEP]": map[string]any{"id": "[SEP]", "ids": []int{idEnd}, "tokens": []string{"[SEP]"}},
			},
		}
	}
	out, err := json.Marshal(doc)
	if err != nil {
		panic(err)
	}
	return out
}

type node struct {
	op   string
	in   []string
	out  string
	attr []attribute
}

type attribute struct {
	name string
	i    int64
	ints []int64
}

// buildModel is the ONNX file of a model: a table lookup for each input, the
// lookups added, and the sum over positions where the model pools.
func buildModel(spec modelSpec) []byte {
	seqDim := []dim{{param: "batch"}, {param: "seq"}}
	inputNames := []string{"input_ids", "attention_mask"}
	if spec.typed {
		inputNames = append(inputNames, "token_type_ids")
	}
	if spec.inputs != nil {
		inputNames = spec.inputs
	}
	var inputs [][]byte
	for _, name := range inputNames {
		inputs = append(inputs, encodeValueInfo(name, onnxInt64, seqDim))
	}

	outName := "last_hidden_state"
	declared := []dim{{param: "batch"}, {param: "seq"}, {value: fixtureWidth}}
	if spec.pooled {
		outName = "sentence_embedding"
		declared = []dim{{param: "batch"}, {value: fixtureWidth}}
	}
	if spec.declared != nil {
		declared = spec.declared
	}
	outType := uint64(onnxFloat)
	if spec.double {
		outType = onnxDouble
	}
	outputs := [][]byte{encodeValueInfo(outName, outType, declared)}
	if spec.inputs != nil {
		return encodeModel(nil, nil, inputs, outputs)
	}

	ids := oneHotTable([]int{0, 1, 2, 3, 4, 5, 6, 7})
	if spec.badWeights {
		ids = ids[:len(ids)-4]
	}
	inits := [][]byte{
		encodeTensor("ids_table", onnxFloat, []int64{fixtureVocab, fixtureWidth}, ids),
		encodeTensor("mask_table", onnxFloat, []int64{2, fixtureWidth}, oneHotTable([]int{fixtureVocab, fixtureVocab + 1})),
	}
	nodes := []node{
		{op: "Gather", in: []string{"ids_table", "input_ids"}, out: "ids_h"},
		{op: "Gather", in: []string{"mask_table", "attention_mask"}, out: "mask_h"},
		{op: "Add", in: []string{"ids_h", "mask_h"}, out: "sum"},
	}
	last := "sum"
	if spec.typed {
		inits = append(inits, encodeTensor("types_table", onnxFloat, []int64{2, fixtureWidth},
			oneHotTable([]int{fixtureVocab + 2, fixtureVocab + 2})))
		nodes = append(nodes,
			node{op: "Gather", in: []string{"types_table", "token_type_ids"}, out: "types_h"},
			node{op: "Add", in: []string{"sum", "types_h"}, out: "typed"})
		last = "typed"
	}
	if spec.fixedSeq {
		inits = append(inits, encodeTensor("position", onnxFloat, []int64{1, 64, fixtureWidth}, make([]byte, 4*64*fixtureWidth)))
		nodes = append(nodes, node{op: "Add", in: []string{last, "position"}, out: "positioned"})
		last = "positioned"
	}
	if spec.pooled {
		nodes = append(nodes, node{
			op: "ReduceSum", in: []string{last}, out: "pooled",
			attr: []attribute{{name: "axes", ints: []int64{1}}, {name: "keepdims", i: 0}},
		})
		last = "pooled"
	}
	if spec.double {
		nodes = append(nodes, node{op: "Cast", in: []string{last}, out: outName, attr: []attribute{{name: "to", i: onnxDouble}}})
	} else {
		nodes = append(nodes, node{op: "Identity", in: []string{last}, out: outName})
	}
	return encodeModel(nodes, inits, inputs, outputs)
}

// oneHotTable has a row for each column given, with a one in that column.
func oneHotTable(columns []int) []byte {
	raw := make([]byte, 0, 4*len(columns)*fixtureWidth)
	for _, column := range columns {
		for c := range fixtureWidth {
			value := float32(0)
			if c == column {
				value = 1
			}
			raw = binary.LittleEndian.AppendUint32(raw, math.Float32bits(value))
		}
	}
	return raw
}

func encodeVarint(num protowire.Number, v uint64) []byte {
	return protowire.AppendVarint(protowire.AppendTag(nil, num, protowire.VarintType), v)
}

func encodeBytes(num protowire.Number, data []byte) []byte {
	return protowire.AppendBytes(protowire.AppendTag(nil, num, protowire.BytesType), data)
}

func encodeString(num protowire.Number, s string) []byte {
	return encodeBytes(num, []byte(s))
}

func concat(parts ...[]byte) []byte {
	return slices.Concat(parts...)
}

func encodeTensor(name string, dtype uint64, dims []int64, raw []byte) []byte {
	var out []byte
	for _, d := range dims {
		out = concat(out, encodeVarint(1, uint64(d)))
	}
	return concat(out, encodeVarint(2, dtype), encodeString(8, name), encodeBytes(9, raw))
}

func encodeValueInfo(name string, dtype uint64, dims []dim) []byte {
	var shape []byte
	for _, d := range dims {
		inner := encodeVarint(1, uint64(d.value))
		if d.param != "" {
			inner = encodeString(2, d.param)
		}
		shape = concat(shape, encodeBytes(1, inner))
	}
	tensor := concat(encodeVarint(1, dtype), encodeBytes(2, shape))
	return concat(encodeString(1, name), encodeBytes(2, encodeBytes(1, tensor)))
}

func encodeNode(n node) []byte {
	var out []byte
	for _, in := range n.in {
		out = concat(out, encodeString(1, in))
	}
	out = concat(out, encodeString(2, n.out), encodeString(3, n.out), encodeString(4, n.op))
	for _, a := range n.attr {
		enc := encodeString(1, a.name)
		if a.ints != nil {
			enc = concat(enc, encodeVarint(20, attrInts))
			for _, v := range a.ints {
				enc = concat(enc, encodeVarint(8, uint64(v)))
			}
		} else {
			enc = concat(enc, encodeVarint(20, attrInt), encodeVarint(3, uint64(a.i)))
		}
		out = concat(out, encodeBytes(5, enc))
	}
	return out
}

func encodeModel(nodes []node, inits, inputs, outputs [][]byte) []byte {
	var graph []byte
	for _, n := range nodes {
		graph = concat(graph, encodeBytes(1, encodeNode(n)))
	}
	graph = concat(graph, encodeString(2, "tiny"))
	for _, init := range inits {
		graph = concat(graph, encodeBytes(5, init))
	}
	for _, in := range inputs {
		graph = concat(graph, encodeBytes(11, in))
	}
	for _, out := range outputs {
		graph = concat(graph, encodeBytes(12, out))
	}
	return concat(encodeVarint(1, 8), encodeBytes(8, encodeVarint(2, 17)), encodeBytes(7, graph))
}
