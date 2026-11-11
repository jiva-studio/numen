package engine

import (
	"math"
	"slices"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/engine/onnx"
)

func twoRows() (flat []float32, mask []int64) {
	flat = []float32{
		1, 0, 0, // row 0, token 0
		0, 3, 0, // row 0, token 1
		0, 0, 5, // row 0, token 2
		0, 2, 0, // row 1, token 0
		4, 0, 0, // row 1, token 1
		9, 9, 9, // row 1, padding
	}
	mask = []int64{1, 1, 1, 1, 1, 0}
	return flat, mask
}

func TestPoolKinds(t *testing.T) {
	flat, mask := twoRows()
	cases := []struct {
		kind string
		want [][]float32
	}{
		{model.PoolCLS, [][]float32{{1, 0, 0}, {0, 2, 0}}},
		{model.PoolLast, [][]float32{{0, 0, 5}, {4, 0, 0}}},
		{model.PoolMean, [][]float32{{1.0 / 3, 1, 5.0 / 3}, {2, 1, 0}}},
	}
	for _, tc := range cases {
		got, err := pool(tc.kind, tokenOutput{flat: flat, mask: mask, rows: 2, seq: 3, dimensions: 3}, false)
		if err != nil {
			t.Fatalf("%s: %v", tc.kind, err)
		}
		for row := range tc.want {
			if !isNear(got[row], tc.want[row]) {
				t.Errorf("%s row %d = %v, want %v", tc.kind, row, got[row], tc.want[row])
			}
		}
	}
}

func TestPoolNormalizesOnlyWhereTheProfileAsksForIt(t *testing.T) {
	flat, mask := twoRows()
	raw, err := pool(model.PoolLast, tokenOutput{flat: flat, mask: mask, rows: 2, seq: 3, dimensions: 3}, false)
	if err != nil {
		t.Fatal(err)
	}
	if raw[0][2] != 5 {
		t.Errorf("a vector that is not normalized is scaled: %v", raw[0])
	}
	unit, err := pool(model.PoolLast, tokenOutput{flat: flat, mask: mask, rows: 2, seq: 3, dimensions: 3}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !isNear(unit[0], []float32{0, 0, 1}) {
		t.Errorf("a normalized vector is %v, want unit length", unit[0])
	}
}

func TestPoolOutputIsNoPerTokenPooling(t *testing.T) {
	flat, mask := twoRows()
	if _, err := pool(model.PoolOutput, tokenOutput{flat: flat, mask: mask, rows: 2, seq: 3, dimensions: 3}, true); err == nil {
		t.Error("output pooling of a per-token tensor gave no error")
	}
}

func TestSplitRowsKeepsTheVectorsOfAnOutputThatIsPooled(t *testing.T) {
	flat := []float32{3, 4, 0, 6}
	got, err := splitRows(flat, 2, 2, true)
	if err != nil {
		t.Fatal(err)
	}
	if !isNear(got[0], []float32{0.6, 0.8}) || !isNear(got[1], []float32{0, 1}) {
		t.Errorf("normalized rows = %v", got)
	}
	raw, err := splitRows(flat, 2, 2, false)
	if err != nil {
		t.Fatal(err)
	}
	if !isNear(raw[0], []float32{3, 4}) {
		t.Errorf("raw rows = %v", raw)
	}
	if _, err := splitRows(flat, 3, 2, true); err == nil {
		t.Error("a wrong number of values gave no error")
	}
}

func TestPadBatchSetsPositionsOfTheRealTokensAndZeroForPadding(t *testing.T) {
	paddings := newPaddingPool()
	if got, ok := paddings.Get().(*padding); !ok || got == nil {
		t.Fatalf("newPaddingPool().Get() = %v", got)
	}
	rows, seq, in := padBatch(paddings, [][]int{{7, 8, 9}, {5}}, 99, 512)
	if rows != 2 || seq != 64 {
		t.Fatalf("rows, seq = %d, %d", rows, seq)
	}
	if got := in.ids[:5]; !slices.Equal(got, []int64{7, 8, 9, 99, 99}) {
		t.Errorf("ids of row 0 = %v", got)
	}
	if got := in.ids[seq : seq+3]; !slices.Equal(got, []int64{5, 99, 99}) {
		t.Errorf("ids of row 1 = %v", got)
	}
	if got := in.mask[:5]; !slices.Equal(got, []int64{1, 1, 1, 0, 0}) {
		t.Errorf("mask of row 0 = %v", got)
	}
	if got := in.mask[seq : seq+3]; !slices.Equal(got, []int64{1, 0, 0}) {
		t.Errorf("mask of row 1 = %v", got)
	}
	if len(in.types) != rows*seq || in.types[0] != 0 {
		t.Errorf("types = len %d", len(in.types))
	}
	if got := in.positions[:4]; !slices.Equal(got, []int64{0, 1, 2, 0}) {
		t.Errorf("positions of row 0 = %v", got)
	}
	if got := in.positions[seq : seq+3]; !slices.Equal(got, []int64{0, 0, 0}) {
		t.Errorf("positions of row 1 = %v", got)
	}
	in.types[0] = 7
	in.mask[3] = 7
	firstIDPtr := &in.ids[0]
	paddings.Put(in)
	_, _, again := padBatch(paddings, [][]int{{}}, 0, 512)
	if &again.ids[0] != firstIDPtr {
		t.Error("expected padBatch to reuse pooled backing slice")
	}
	if again.positions[1] != 0 || again.types[0] != 0 || again.mask[0] != 1 || again.mask[1] != 0 || again.mask[3] != 0 {
		t.Errorf("reused padding state: pos=%v types[0]=%d mask=%v", again.positions[:3], again.types[0], again.mask[:4])
	}

	if _, _, p1 := padBatch(&sync.Pool{}, [][]int{{1}}, 0, 512); p1 == nil || p1.ids[0] != 1 {
		t.Errorf("padBatch with empty sync.Pool = %+v", p1)
	}
	nilPool := &sync.Pool{New: func() any { return (*padding)(nil) }}
	if _, _, p2 := padBatch(nilPool, [][]int{{2}}, 0, 512); p2 == nil || p2.ids[0] != 2 {
		t.Errorf("padBatch with typed-nil sync.Pool = %+v", p2)
	}

	if got := getLongest(nil); got != 0 {
		t.Errorf("getLongest(nil) = %d, want 0", got)
	}
	if got := getLongest([][]int{{1}, {1, 2, 3, 4}}); got != 4 {
		t.Errorf("getLongest = %d, want 4", got)
	}

	exact := make([]int64, 0, 4)
	out := resize(exact, 4)
	if &out[0] != &exact[:1][0] {
		t.Error("resize allocated when cap equaled size")
	}

	if got := getWindow(onnx.ProviderCoreML, 1024); got != 512 {
		t.Errorf("getWindow(CoreML, 1024) = %d, want 512", got)
	}
	if got := getWindow(onnx.ProviderCPU, 1024); got != 1024 {
		t.Errorf("getWindow(CPU, 1024) = %d, want 1024", got)
	}

	unmasked := lastPool([]float32{3, 4, 9, 9}, []int64{0, 0}, 1, 2, 2, false)
	if !isNear(unmasked[0], []float32{3, 4}) {
		t.Errorf("lastPool with all-zero mask = %v, want [3 4]", unmasked[0])
	}
}

func TestChooseOutput(t *testing.T) {
	cases := []struct {
		name    string
		named   []string
		pooling string
		want    string
		isError bool
	}{
		{"per-token output of a mean model", []string{"last_hidden_state"}, model.PoolMean, "last_hidden_state", false},
		{"both, pooled by the model", []string{"last_hidden_state", "sentence_embedding"}, model.PoolOutput, "sentence_embedding", false},
		{"both, pooled here", []string{"sentence_embedding", "last_hidden_state"}, model.PoolLast, "last_hidden_state", false},
		{"only a sentence embedding", []string{"sentence_embedding"}, model.PoolMean, "sentence_embedding", false},
		{"output pooling without it", []string{"last_hidden_state", "other"}, model.PoolOutput, "", true},
		{"one unknown output", []string{"logits"}, model.PoolOutput, "logits", false},
		{"none we know", []string{"a", "b"}, model.PoolMean, "", true},
	}
	for _, tc := range cases {
		got, err := chooseOutput(tc.named, tc.pooling)
		if (err != nil) != tc.isError || got != tc.want {
			t.Errorf("%s: chooseOutput = %q, %v", tc.name, got, err)
		}
	}
}

func TestCheckInputs(t *testing.T) {
	base := []string{inputIDs, attentionMask}
	cases := []struct {
		name    string
		names   []string
		extra   []string
		want    modelInputs
		isError bool
	}{
		{"plain", base, nil, modelInputs{}, false},
		{"typed", append(slices.Clone(base), tokenTypeIDs), nil, modelInputs{isTyped: true}, false},
		{"positions asked and taken", append(slices.Clone(base), model.InputPositionIDs), []string{model.InputPositionIDs}, modelInputs{isPositioned: true}, false},
		{"positions taken but not asked", append(slices.Clone(base), model.InputPositionIDs), nil, modelInputs{}, true},
		{"positions asked but not taken", base, []string{model.InputPositionIDs}, modelInputs{}, true},
		{"cache inputs", append(slices.Clone(base), "past_key_values.0.key"), nil, modelInputs{}, true},
		{"no mask", []string{inputIDs}, nil, modelInputs{}, true},
		{"no ids", []string{attentionMask}, nil, modelInputs{}, true},
	}
	for _, tc := range cases {
		got, err := checkInputs("model", tc.names, tc.extra)
		if (err != nil) != tc.isError || got != tc.want {
			t.Errorf("%s: checkInputs = %+v, %v", tc.name, got, err)
		}
	}
}

func isNear(got, want []float32) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if math.Abs(float64(got[i]-want[i])) > 1e-5 {
			return false
		}
	}
	return true
}
