package gomlx

import (
	"context"
	"errors"
	"math"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gomlx/compute/dtypes"
	"github.com/gomlx/compute/shapes"
	onnxgomlx "github.com/gomlx/onnx-gomlx/onnx"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// getSum is what the pooled fixture returns for a text of these tokens laid out
// over seq positions: a column for each token fed, for each padded position and
// for each position that holds text.
func getSum(ids []int, seq, pad int, typed bool) []float32 {
	v := make([]float32, fixtureWidth)
	for _, id := range ids {
		v[id]++
	}
	v[fixtureVocab+1] += float32(len(ids))
	for range seq - len(ids) {
		v[pad]++
		v[fixtureVocab]++
	}
	if typed {
		v[fixtureVocab+2] += float32(seq)
	}
	return v
}

// getUnit is the vector with these columns set, at unit length.
func getUnit(columns map[int]float32) []float32 {
	v := make([]float32, fixtureWidth)
	var sum float64
	for column, x := range columns {
		v[column] = x
		sum += float64(x) * float64(x)
	}
	for i := range v {
		v[i] = float32(float64(v[i]) / math.Sqrt(sum))
	}
	return v
}

func isClose(got, want []float32) bool {
	if len(got) != len(want) {
		return false
	}
	for i := range got {
		if math.Abs(float64(got[i]-want[i])) > 1e-6 {
			return false
		}
	}
	return true
}

func repeatWord(word string, n int) string {
	return strings.TrimSpace(strings.Repeat(word+" ", n))
}

func repeatID(id, n int) []int {
	return slices.Repeat([]int{id}, n)
}

func TestAPooledModelsVectorIsWhatTheModelReturned(t *testing.T) {
	e := openFixture(t, fixture{model: modelSpec{pooled: true}})
	got, err := e.Embed(t.Context(), []string{"a b c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !slices.Equal(got[0], getSum([]int{idA, idB, idC}, 64, idPad, false)) {
		t.Errorf("got %v", got)
	}
}

func TestALengthIsRoundedUpToTheNextStep(t *testing.T) {
	e := openFixture(t, fixture{model: modelSpec{pooled: true}})
	for _, c := range []struct{ words, seq int }{{1, 64}, {64, 64}, {65, 128}, {128, 128}, {129, 192}} {
		got, err := e.Embed(t.Context(), []string{repeatWord("a", c.words)})
		if err != nil {
			t.Fatal(err)
		}
		if want := getSum(repeatID(idA, c.words), c.seq, idPad, false); !slices.Equal(got[0], want) {
			t.Errorf("%d words: got %v, want %v", c.words, got[0], want)
		}
	}
}

func TestABatchIsLaidOutAtItsLongestText(t *testing.T) {
	e := openFixture(t, fixture{model: modelSpec{pooled: true}, batchTexts: 2})
	got, err := e.Embed(t.Context(), []string{"b", repeatWord("a", 65)})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got[0], getSum([]int{idB}, 128, idPad, false)) || !slices.Equal(got[1], getSum(repeatID(idA, 65), 128, idPad, false)) {
		t.Errorf("got %v", got)
	}
}

func TestALengthIsNeverPastWhatTheModelTakes(t *testing.T) {
	e := openFixture(t, fixture{model: modelSpec{pooled: true}, maxTokens: 100})
	got, err := e.Embed(t.Context(), []string{repeatWord("a", 70)})
	if err != nil {
		t.Fatal(err)
	}
	if want := getSum(repeatID(idA, 70), 100, idPad, false); !slices.Equal(got[0], want) {
		t.Errorf("got %v, want %v", got[0], want)
	}
}

func TestATokeniserWithoutAPadTokenPadsWithTheFirstID(t *testing.T) {
	e := openFixture(t, fixture{model: modelSpec{pooled: true}, tokenizer: tokenizerSpec{noPad: true}})
	got, err := e.Embed(t.Context(), []string{"a"})
	if err != nil {
		t.Fatal(err)
	}
	if want := getSum([]int{idA}, 64, idUnknown, false); !slices.Equal(got[0], want) {
		t.Errorf("got %v, want %v", got[0], want)
	}
}

func TestATokenTypesInputIsFedWhenTheModelAsksForIt(t *testing.T) {
	e := openFixture(t, fixture{model: modelSpec{pooled: true, typed: true}})
	got, err := e.Embed(t.Context(), []string{"a b"})
	if err != nil {
		t.Fatal(err)
	}
	if want := getSum([]int{idA, idB}, 64, idPad, true); !slices.Equal(got[0], want) {
		t.Errorf("got %v, want %v", got[0], want)
	}
}

func TestATextPastTheLimitKeepsItsClosingToken(t *testing.T) {
	e := openFixture(t, fixture{model: modelSpec{pooled: true}, tokenizer: tokenizerSpec{wrap: true}, maxTokens: 5})
	for _, c := range []struct {
		name, text string
		ids        []int
	}{
		{"cut", "a b c d a b c d a b", []int{idStart, idA, idB, idC, idEnd}},
		{"at the limit", "a b c", []int{idStart, idA, idB, idC, idEnd}},
		{"under it", "d", []int{idStart, idD, idEnd}},
	} {
		got, err := e.Embed(t.Context(), []string{c.text})
		if err != nil {
			t.Fatal(c.name, err)
		}
		if want := getSum(c.ids, 5, idPad, false); !slices.Equal(got[0], want) {
			t.Errorf("%s: got %v, want %v", c.name, got[0], want)
		}
	}
}

func TestTextsComeBackInOrderAcrossBatches(t *testing.T) {
	e := openFixture(t, fixture{model: modelSpec{pooled: true}, batchTexts: 2})
	got, err := e.Embed(t.Context(), []string{"a", "b", "c", "d", "a"})
	if err != nil {
		t.Fatal(err)
	}
	ids := []int{idA, idB, idC, idD, idA}
	if len(got) != len(ids) {
		t.Fatalf("got %d vectors", len(got))
	}
	for i, id := range ids {
		if want := getSum([]int{id}, 64, idPad, false); !slices.Equal(got[i], want) {
			t.Errorf("text %d: got %v, want %v", i, got[i], want)
		}
	}
}

func TestABatchOfNoTextsIsOneTextAtATime(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	e := openFixture(t, fixture{model: modelSpec{pooled: true}, batchTexts: 0})
	got, err := e.Embed(ctx, []string{"a", "b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 || !slices.Equal(got[2], getSum([]int{idC}, 64, idPad, false)) {
		t.Errorf("got %v", got)
	}
}

func TestNoTextsAreNoVectors(t *testing.T) {
	e := openFixture(t, fixture{model: modelSpec{pooled: true}})
	for _, texts := range [][]string{nil, {}} {
		got, err := e.Embed(t.Context(), texts)
		if got != nil || err != nil {
			t.Errorf("got %v, %v", got, err)
		}
	}
}

func TestACancelledContextEmbedsNothing(t *testing.T) {
	e := openFixture(t, fixture{model: modelSpec{pooled: true}})
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	got, err := e.Embed(ctx, []string{"a"})
	if !errors.Is(err, context.Canceled) || got != nil {
		t.Errorf("got %v, %v", got, err)
	}
}

func TestTokenVectorsAreAveragedOverTheTextAtUnitLength(t *testing.T) {
	e := openFixture(t, fixture{batchTexts: 2})
	got, err := e.Embed(t.Context(), []string{"a b", "c"})
	if err != nil {
		t.Fatal(err)
	}
	mean := getUnit(map[int]float32{idA: 0.5, idB: 0.5, fixtureVocab + 1: 1})
	single := getUnit(map[int]float32{idC: 1, fixtureVocab + 1: 1})
	if !isClose(got[0], mean) || !isClose(got[1], single) {
		t.Errorf("got %v", got)
	}
}

func TestTheFirstTokenIsTheVectorOfAModelPooledThatWay(t *testing.T) {
	e := openFixture(t, fixture{batchTexts: 2, pooling: embed.PoolHead})
	got, err := e.Embed(t.Context(), []string{"a b", "d c"})
	if err != nil {
		t.Fatal(err)
	}
	first := getUnit(map[int]float32{idA: 1, fixtureVocab + 1: 1})
	second := getUnit(map[int]float32{idD: 1, fixtureVocab + 1: 1})
	if !isClose(got[0], first) || !isClose(got[1], second) {
		t.Errorf("got %v", got)
	}
}

func TestOnlyTheTextsAskedForComeBackWhateverTheModelPools(t *testing.T) {
	for name, f := range map[string]fixture{
		"pooled":  {model: modelSpec{pooled: true}},
		"mean":    {},
		"head":    {pooling: embed.PoolHead},
		"typed":   {model: modelSpec{pooled: true, typed: true}},
		"batched": {batchTexts: 4},
	} {
		f.batchTexts = max(f.batchTexts, 3)
		e := openFixture(t, f)
		got, err := e.Embed(t.Context(), []string{"a"})
		if err != nil {
			t.Fatal(name, err)
		}
		if len(got) != 1 {
			t.Errorf("%s: got %d vectors", name, len(got))
		}
	}
}

func TestSequenceLengthsAreCompiledOnceEach(t *testing.T) {
	// The default allows 32 shapes, and 33 lengths are asked for.
	const lengths = 33
	e := openFixture(t, fixture{model: modelSpec{pooled: true}, maxTokens: 64 * lengths})
	for n := 1; n <= lengths; n++ {
		got, err := e.Embed(t.Context(), []string{repeatWord("a", 64*n)})
		if err != nil {
			t.Fatalf("%d tokens: %v", 64*n, err)
		}
		if len(got) != 1 {
			t.Fatalf("%d tokens: got %d vectors", 64*n, len(got))
		}
	}
}

func TestAGraphThatCannotTakeALengthIsAnErrorForThatText(t *testing.T) {
	e := openFixture(t, fixture{model: modelSpec{pooled: true, fixedSeq: true}})
	if _, err := e.Embed(t.Context(), []string{repeatWord("a", 65)}); err == nil {
		t.Error("a length the graph cannot take returned vectors")
	}
	got, err := e.Embed(t.Context(), []string{"a"})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got[0], getSum([]int{idA}, 64, idPad, false)) {
		t.Errorf("got %v", got[0])
	}
}

func TestAnOutputOfAnotherTypeIsAnError(t *testing.T) {
	e := openFixture(t, fixture{model: modelSpec{pooled: true, double: true}})
	_, err := e.Embed(t.Context(), []string{"a"})
	if err == nil || strings.Contains(err.Error(), "values") {
		t.Errorf("float64 output: got %v", err)
	}
}

func TestAnOutputOfAnotherSizeThanItSaysIsAnError(t *testing.T) {
	for _, c := range []struct {
		name     string
		spec     modelSpec
		contains string
	}{
		{"pooled by its shape", modelSpec{declared: []dim{{param: "batch"}, {value: fixtureWidth}}}, "got "},
		{"per token by its shape", modelSpec{pooled: true, declared: []dim{{param: "batch"}, {param: "seq"}, {value: fixtureWidth}}}, "tiny returned "},
	} {
		e := openFixture(t, fixture{model: c.spec})
		_, err := e.Embed(t.Context(), []string{"a"})
		if err == nil || !strings.Contains(err.Error(), c.contains) {
			t.Errorf("%s: got %v", c.name, err)
		}
	}
}

func TestAClosedEmbedderEmbedsNothing(t *testing.T) {
	e := openFixture(t, fixture{model: modelSpec{pooled: true}})
	if err := e.Close(); err != nil {
		t.Fatal(err)
	}
	got, err := e.Embed(t.Context(), []string{"a"})
	if err == nil || !strings.Contains(err.Error(), "closed") || got != nil {
		t.Errorf("got %v, %v", got, err)
	}
	if err := e.Close(); err != nil {
		t.Errorf("a second close returned %v", err)
	}
}

func TestEmbeddingWhileClosingEitherAnswersOrIsRefused(t *testing.T) {
	e := openFixture(t, fixture{model: modelSpec{pooled: true}})
	failures := make(chan error, 16)
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 5 {
				if _, err := e.Embed(t.Context(), []string{"a b"}); err != nil && !strings.Contains(err.Error(), "closed") {
					failures <- err
					return
				}
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := e.Close(); err != nil {
			failures <- err
		}
	}()
	wg.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
}

func TestAnEmbedderTellsTheModelItWasOpenedFor(t *testing.T) {
	f := fixture{model: modelSpec{pooled: true}, maxTokens: 77, pooling: embed.PoolHead}
	e := openFixture(t, f)
	if got, want := e.GetModel(), f.getIdentity(); got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestOpenRefusesWhatItCannotRun(t *testing.T) {
	for _, c := range []struct {
		name     string
		fixture  fixture
		change   func(dir string, id *port.EmbeddingModel, local *embed.LocalModel)
		contains string
		wraps    bool
	}{
		{
			name: "an identity without dimensions", fixture: fixture{model: modelSpec{pooled: true}}, contains: "dimensions must be known",
			change: func(_ string, id *port.EmbeddingModel, _ *embed.LocalModel) { id.Dimensions = 0 },
		},
		{
			name: "a directory that is not there", fixture: fixture{model: modelSpec{pooled: true}}, contains: "model directory",
			change: func(dir string, _ *port.EmbeddingModel, local *embed.LocalModel) {
				local.Dir = filepath.Join(dir, "missing")
			},
		},
		{
			name: "a tokeniser that is not one", fixture: fixture{model: modelSpec{pooled: true}}, contains: "tokenizer.json",
			change: func(dir string, _ *port.EmbeddingModel, _ *embed.LocalModel) {
				writeFile(t, filepath.Join(dir, embed.TokenizerFile), []byte("not json"))
			},
		},
		{
			name: "a model that is not one", fixture: fixture{model: modelSpec{pooled: true}}, wraps: true,
			contains: "reading ",
			change: func(dir string, _ *port.EmbeddingModel, _ *embed.LocalModel) {
				writeFile(t, filepath.Join(dir, embed.ModelFile), []byte{0xff, 0xff, 0xff, 0xff})
			},
		},
		{
			name: "weights that are short", fixture: fixture{model: modelSpec{pooled: true, badWeights: true}}, wraps: true,
			contains: "loading the weights of tiny",
		},
		{
			name: "no output of the width", fixture: fixture{model: modelSpec{pooled: true}},
			contains: "tiny has no output of 12 dimensions: [sentence_embedding]",
			change:   func(_ string, id *port.EmbeddingModel, _ *embed.LocalModel) { id.Dimensions = fixtureWidth + 1 },
		},
		{
			name: "an input it does not have", fixture: fixture{model: modelSpec{inputs: []string{"input_ids", "attention_mask", "position_ids"}}},
			contains: "asks for an input this adapter does not have: position_ids",
		},
		{
			name: "no mask", fixture: fixture{model: modelSpec{inputs: []string{"input_ids", "token_type_ids"}}},
			contains: "a sentence encoder takes tokens and a mask",
		},
		{
			name: "no tokens", fixture: fixture{model: modelSpec{inputs: []string{"attention_mask", "token_type_ids"}}},
			contains: "a sentence encoder takes tokens and a mask",
		},
		{
			name: "neither", fixture: fixture{model: modelSpec{inputs: []string{}}},
			contains: "a sentence encoder takes tokens and a mask",
		},
	} {
		dir := writeFixture(t, c.fixture)
		id, local := c.fixture.getIdentity(), c.fixture.getLocal(dir)
		if c.change != nil {
			c.change(dir, &id, &local)
		}
		e, err := Open(t.Context(), id, local, nil)
		if err == nil || !strings.Contains(err.Error(), c.contains) {
			t.Errorf("%s: got %v, want one that holds %q", c.name, err, c.contains)
			continue
		}
		if e != nil {
			t.Errorf("%s: returned an embedder with the error", c.name)
		}
		if c.wraps && errors.Unwrap(err) == nil {
			t.Errorf("%s: the cause is lost: %v", c.name, err)
		}
	}
}

// stubNet is a model that only knows its outputs and how it closes.
type stubNet struct {
	onnxgomlx.Model
	names    []string
	dshapes  []shapes.Shape
	closeErr error
	closes   int
}

func (s *stubNet) Outputs() ([]string, []shapes.Shape) { return s.names, s.dshapes }

func (s *stubNet) Close() error {
	s.closes++
	return s.closeErr
}

func TestTheOutputChosenIsTheOneThatHoldsTheVector(t *testing.T) {
	float := func(dims ...int) shapes.Shape { return shapes.Make(dtypes.Float32, dims...) }
	for _, c := range []struct {
		name       string
		names      []string
		dshapes    []shapes.Shape
		want       string
		wantPooled bool
	}{
		{"the sentence embedding before all", []string{"other", "last_hidden_state", "sentence_embedding"},
			[]shapes.Shape{float(2, 3, 4), float(2, 3, 4), float(2, 4)}, "sentence_embedding", true},
		{"the sentence embedding at the head of the outputs", []string{"sentence_embedding", "last_hidden_state"},
			[]shapes.Shape{float(2, 4), float(2, 3, 4)}, "sentence_embedding", true},
		{"the sentence embedding wherever it stands", []string{"last_hidden_state", "sentence_embedding"},
			[]shapes.Shape{float(2, 3, 4), float(2, 4)}, "sentence_embedding", true},
		{"the hidden state before the rest", []string{"other", "last_hidden_state"},
			[]shapes.Shape{float(2, 3, 4), float(2, 3, 4)}, "last_hidden_state", false},
		{"the first that fits when none is preferred", []string{"first", "second"},
			[]shapes.Shape{float(2, 3, 4), float(2, 3, 4)}, "first", false},
		{"an output of another width is passed over", []string{"sentence_embedding", "other"},
			[]shapes.Shape{float(2, 5), float(2, 4)}, "other", true},
		{"a vector with no batch is passed over", []string{"flat", "narrow", "other"},
			[]shapes.Shape{float(4), float(2, 5), float(2, 3, 4)}, "other", false},
		{"one vector for each token is not pooled", []string{"sentence_embedding"},
			[]shapes.Shape{float(2, 3, 4)}, "sentence_embedding", false},
	} {
		e := &Embedder{name: "tiny", dimensions: 4, net: &stubNet{names: c.names, dshapes: c.dshapes}}
		got, pooled, err := e.chooseOutput()
		if err != nil || got != c.want || pooled != c.wantPooled {
			t.Errorf("%s: got %q, %v, %v", c.name, got, pooled, err)
		}
	}
}

func TestNoOutputOfTheWidthIsAnErrorThatNamesWhatThereIs(t *testing.T) {
	e := &Embedder{name: "tiny", dimensions: 4, net: &stubNet{
		names:   []string{"flat", "narrow"},
		dshapes: []shapes.Shape{shapes.Make(dtypes.Float32, 4), shapes.Make(dtypes.Float32, 2, 5)},
	}}
	name, pooled, err := e.chooseOutput()
	if err == nil || name != "" || pooled || !strings.Contains(err.Error(), "tiny has no output of 4 dimensions: [flat narrow]") {
		t.Errorf("got %q, %v, %v", name, pooled, err)
	}
}

func TestRowsAreCutFromTheFlatOutputAndCopied(t *testing.T) {
	flat := []float32{1, 2, 3, 4, 5, 6}
	got, err := splitRows(flat, 3, 2)
	if err != nil {
		t.Fatal(err)
	}
	flat[0] = 9
	if len(got) != 3 || !slices.Equal(got[0], []float32{1, 2}) || !slices.Equal(got[1], []float32{3, 4}) || !slices.Equal(got[2], []float32{5, 6}) {
		t.Errorf("got %v", got)
	}
	for _, c := range [][]float32{flat[:5], append(slices.Clone(flat), 7)} {
		if _, err := splitRows(c, 3, 2); err == nil {
			t.Errorf("%d values for 3 vectors of 2 were split", len(c))
		}
	}
}

func TestClosingReleasesTheModelOnce(t *testing.T) {
	cause := errors.New("cannot release")
	net := &stubNet{closeErr: cause}
	e := &Embedder{net: net}
	if err := e.Close(); !errors.Is(err, cause) {
		t.Errorf("got %v", err)
	}
	if err := e.Close(); err != nil || net.closes != 1 {
		t.Errorf("a second close returned %v after %d releases", err, net.closes)
	}
}

func TestInputsAreTypedWhenTheModelTakesTokenTypes(t *testing.T) {
	for _, c := range []struct {
		names []string
		want  bool
	}{
		{[]string{"input_ids", "attention_mask", "token_type_ids"}, true},
		{[]string{"token_type_ids", "attention_mask", "input_ids"}, true},
		{[]string{"attention_mask", "input_ids"}, false},
	} {
		typed, err := checkInputs("tiny", c.names)
		if err != nil || typed != c.want {
			t.Errorf("%v: got %v, %v", c.names, typed, err)
		}
	}
}
