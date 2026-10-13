package onnx

import (
	"context"
	"strings"
	"testing"

	"github.com/gomlx/go-huggingface/tokenizers/api"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapterlib/onnxruntime"
)

// The tokens a fake tokenizer opens and ends every text with.
const (
	openToken = 1
	endToken  = 2
)

// wordTokenizer gives one token a word, its value the word's first letter,
// between an opening and an ending token.
type wordTokenizer struct {
	api.Tokenizer
}

func (wordTokenizer) Encode(text string) []int {
	ids := []int{openToken}
	for _, word := range strings.Fields(text) {
		ids = append(ids, int(word[0]))
	}
	return append(ids, endToken)
}

// passRecord is what a fake model saw in one forward pass.
type passRecord struct {
	rows, seq int
}

// recordPasses is a fake model that lays each batch out as the model takes it,
// answers each row with the tokens its mask keeps and the last of them, and
// records the shape of every pass.
func recordPasses(limit int, passes *[]passRecord) func(context.Context, [][]int) ([][]float32, error) {
	return func(_ context.Context, batch [][]int) ([][]float32, error) {
		rows, seq, in := padBatch(batch, 0, limit)
		defer releasePadding(in)
		*passes = append(*passes, passRecord{rows: rows, seq: seq})
		out := make([][]float32, rows)
		for row := range rows {
			kept, last := 0, int64(0)
			for token := range seq {
				if in.mask[row*seq+token] == 1 {
					kept++
					last = in.ids[row*seq+token]
				}
			}
			out[row] = []float32{float32(kept), float32(last)}
		}
		return out, nil
	}
}

func createText(words int) string {
	return strings.Repeat("w ", words)
}

// A text longer than 512 tokens reaches the model whole up to the window, and
// the token that ends it survives the cut.
func TestALongTextReachesTheModelUpToTheWindow(t *testing.T) {
	const limit = 8192
	e := &Embedder{tokenizer: wordTokenizer{}, maxTokens: limit, window: limit, batchTexts: 32}
	texts := []string{createText(1000), createText(limit * 2), createText(3)}
	var passes []passRecord
	got, err := embedTokens(t.Context(), e.tokenize(texts), e.batchTexts, e.window, recordPasses(limit, &passes))
	if err != nil {
		t.Fatal(err)
	}
	for i, want := range []int{1002, limit, 5} {
		if int(got[i][0]) != want {
			t.Errorf("text %d reached the model at %d tokens, want %d", i, int(got[i][0]), want)
		}
		if int(got[i][1]) != endToken {
			t.Errorf("text %d reached the model ending in %d, want the end token %d", i, int(got[i][1]), endToken)
		}
	}
}

// No forward pass holds more than the configured batch or the budget derived
// from it, a full batch at the reference length always fits, and a row too long
// for the budget is run by itself.
func TestEveryPassStaysWithinTheBudget(t *testing.T) {
	const limit = 8192
	for _, batchTexts := range []int{16, 32, 64, 256} {
		var texts [][]int
		for _, length := range []int{10, 300, 512, 513, 700, 1024, 1500, 2048, 4000, 8192} {
			for range 300 {
				texts = append(texts, make([]int, length))
			}
		}
		for _, one := range texts {
			one[len(one)-1] = endToken
		}
		var passes []passRecord
		got, err := embedTokens(t.Context(), texts, batchTexts, limit, recordPasses(limit, &passes))
		if err != nil {
			t.Fatal(err)
		}
		if len(got) != len(texts) {
			t.Fatalf("batch %d: got %d vectors for %d texts", batchTexts, len(got), len(texts))
		}
		rowsSeen := 0
		for _, p := range passes {
			rowsSeen += p.rows
			if p.rows > batchTexts {
				t.Errorf("batch %d: a pass held %d rows", batchTexts, p.rows)
			}
			if p.rows > 1 && p.rows*p.seq*p.seq > getAttentionBudget(batchTexts) {
				t.Errorf("batch %d: a pass of %d rows at %d tokens is past the budget", batchTexts, p.rows, p.seq)
			}
		}
		if rowsSeen != len(texts) {
			t.Errorf("batch %d: the passes held %d rows for %d texts", batchTexts, rowsSeen, len(texts))
		}
		if !hasPass(passes, batchTexts, referenceLength) {
			t.Errorf("batch %d: no pass held a full batch at %d tokens: %v", batchTexts, referenceLength, passes)
		}
		if !hasPass(passes, 1, limit) {
			t.Errorf("batch %d: no pass held one row at the window", batchTexts)
		}
	}
}

// A CoreML session keeps at most the reference length, end token included;
// every other provider keeps the model's window.
func TestTheWindowDependsOnTheProvider(t *testing.T) {
	for _, c := range []struct {
		provider  onnxruntime.Provider
		maxTokens int
		want      int
	}{
		{onnxruntime.ProviderCoreML, 8192, referenceLength},
		{onnxruntime.ProviderCoreML, 256, 256},
		{onnxruntime.ProviderCPU, 8192, 8192},
		{onnxruntime.ProviderCUDA, 8192, 8192},
		{onnxruntime.ProviderDirectML, 8192, 8192},
	} {
		if got := getWindow(c.provider, c.maxTokens); got != c.want {
			t.Errorf("%s with %d tokens keeps %d, want %d", c.provider, c.maxTokens, got, c.want)
		}
	}
}

// Under CoreML a long text is cut at the reference length with its end token
// kept, and the model is run at no length past it.
func TestACoreMLSessionCutsALongTextAtTheReferenceLength(t *testing.T) {
	const maxTokens = 8192
	e := &Embedder{
		tokenizer:  wordTokenizer{},
		maxTokens:  maxTokens,
		window:     getWindow(onnxruntime.ProviderCoreML, maxTokens),
		batchTexts: 4,
	}
	var passes []passRecord
	got, err := embedTokens(t.Context(), e.tokenize([]string{createText(3000), createText(3)}), e.batchTexts, e.window, recordPasses(e.window, &passes))
	if err != nil {
		t.Fatal(err)
	}
	if int(got[0][0]) != referenceLength || int(got[0][1]) != endToken {
		t.Errorf("the long text reached the model as %v, want %d tokens ending %d", got[0], referenceLength, endToken)
	}
	for _, p := range passes {
		if p.seq > referenceLength {
			t.Errorf("a pass ran at %d tokens, past %d", p.seq, referenceLength)
		}
	}
}

func hasPass(passes []passRecord, rows, seq int) bool {
	for _, p := range passes {
		if p.rows == rows && p.seq == seq {
			return true
		}
	}
	return false
}

// Texts split across passes by length come back in the order they were given.
func TestSplitPassesKeepTheOrderOfTheTexts(t *testing.T) {
	const limit = 8192
	lengths := []int{4000, 3, 1500, 600, 8192, 64, 2048, 3, 513, 7000, 100, 1024}
	texts := make([][]int, 0, len(lengths)*5)
	for round := range 5 {
		for _, length := range lengths {
			one := make([]int, length)
			one[len(one)-1] = len(texts)*10 + round
			texts = append(texts, one)
		}
	}
	var passes []passRecord
	got, err := embedTokens(t.Context(), texts, 32, limit, recordPasses(limit, &passes))
	if err != nil {
		t.Fatal(err)
	}
	if len(passes) < 2 {
		t.Fatalf("the texts went in %d pass, and the test needs them split", len(passes))
	}
	for i, one := range texts {
		if int(got[i][0]) != len(one) || int(got[i][1]) != one[len(one)-1] {
			t.Errorf("text %d came back as %v, want length %d ending %d", i, got[i], len(one), one[len(one)-1])
		}
	}
}
