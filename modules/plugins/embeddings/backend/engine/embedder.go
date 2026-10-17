package engine

import (
	"cmp"
	"context"
	"fmt"
	"runtime"
	"slices"
	"sync"

	ort "github.com/getcharzp/onnxruntime_purego"
	"github.com/gomlx/go-huggingface/tokenizers/api"
	"github.com/gomlx/go-huggingface/tokenizers/hftokenizer"

	"github.com/jiva-studio/numen/modules/plugins/embeddings/backend/onnxruntime"
)

// The inputs a sentence encoder exported to ONNX asks for.
const (
	inputIDs      = "input_ids"
	attentionMask = "attention_mask"
	tokenTypeIDs  = "token_type_ids"
)

// The outputs a vector is read from, best first.
var outputs = []string{"sentence_embedding", "last_hidden_state"}

// Embedder is one model, loaded.
type Embedder struct {
	name         string
	origin       string
	dimensions   int
	maxTokens    int
	window       int
	pooling      string
	batchTexts   int
	passes       int
	tokenizer    api.Tokenizer
	pad          int
	output       string
	isTyped      bool
	isHeadPooled bool

	sessions *sessionPool[*ort.Session]
}

// Open loads the model, fetching it first where this machine does not hold it.
func Open(ctx context.Context, identity ModelInfo, cfg LocalModel, progress FetchProgress) (*Embedder, error) {
	if err := CheckIdentity(identity, cfg.Name); err != nil {
		return nil, err
	}
	paths, err := Locate(ctx, cfg, progress)
	if err != nil {
		return nil, err
	}
	tokenizer, err := hftokenizer.NewFromFile(nil, paths.Tokenizer)
	if err != nil {
		return nil, err
	}

	engine, _, err := onnxruntime.Open(cfg.Runtime)
	if err != nil {
		return nil, err
	}
	options, provider, err := onnxruntime.NewSessionOptions(engine, onnxruntime.SessionSettings{
		Provider: cfg.GetProvider(),
		Threads:  cfg.GetThreads(),
	})
	if err != nil {
		return nil, err
	}
	defer options.Destroy()

	passes := resolvePasses(cfg, provider, identity)
	sessions := make([]*ort.Session, 0, 1)
	destroySessions := func() {
		for _, s := range sessions {
			if s != nil {
				s.Destroy()
			}
		}
	}

	session, err := onnxruntime.NewSession(engine, paths.Model, options)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", paths.Model, err)
	}
	sessions = append(sessions, session)

	first := sessions[0]
	e := &Embedder{
		name:         identity.Name,
		origin:       identity.From,
		dimensions:   identity.Dimensions,
		maxTokens:    identity.MaxTokens,
		window:       getWindow(provider, identity.MaxTokens),
		pooling:      identity.Pooling,
		batchTexts:   max(cfg.BatchTexts, 1),
		passes:       passes,
		tokenizer:    tokenizer,
		sessions:     newSessionPool(sessions, passes, (*ort.Session).Destroy),
		isHeadPooled: identity.Pooling == PoolHead,
	}
	if pad, err := tokenizer.SpecialTokenID(api.TokPad); err == nil {
		e.pad = pad
	}

	isTyped, err := checkInputs(cfg.Name, first.InputNames)
	if err != nil {
		destroySessions()
		return nil, err
	}
	e.isTyped = isTyped
	if e.output, err = chooseOutput(first.OutputNames); err != nil {
		destroySessions()
		return nil, fmt.Errorf("%s: %w", cfg.Name, err)
	}
	return e, nil
}

func checkInputs(model string, names []string) (bool, error) {
	isTyped := false
	for _, name := range names {
		switch name {
		case inputIDs, attentionMask:
		case tokenTypeIDs:
			isTyped = true
		default:
			return false, fmt.Errorf("%s asks for an unsupported input: %s", model, name)
		}
	}
	if !slices.Contains(names, inputIDs) || !slices.Contains(names, attentionMask) {
		return false, fmt.Errorf("%s takes %v, expected tokens and mask", model, names)
	}
	return isTyped, nil
}

// Close releases the model sessions.
func (e *Embedder) Close() error {
	if e.sessions != nil {
		e.sessions.close()
	}
	return nil
}

// Concurrency is how many batches this embedder runs in parallel.
func (e *Embedder) Concurrency() int {
	return e.passes
}

// GetModel returns the model description.
func (e *Embedder) GetModel() ModelInfo {
	return ModelInfo{
		Name:        e.name,
		Dimensions:  e.dimensions,
		MaxTokens:   e.maxTokens,
		Pooling:     e.pooling,
		From:        e.origin,
		Concurrency: e.passes,
	}
}

type sequenceItem struct {
	index  int
	tokens []int
}

// Embed runs the model over the texts, grouped by token length to minimize padding.
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	return embedTokens(ctx, e.tokenize(texts), e.batchTexts, e.window, e.forward)
}

func embedTokens(
	ctx context.Context,
	tokens [][]int,
	batchTexts, limit int,
	pass func(context.Context, [][]int) ([][]float32, error),
) ([][]float32, error) {
	items := make([]sequenceItem, len(tokens))
	for i, tok := range tokens {
		items[i] = sequenceItem{index: i, tokens: tok}
	}
	slices.SortStableFunc(items, func(a, b sequenceItem) int {
		return cmp.Compare(len(a.tokens), len(b.tokens))
	})

	out := make([][]float32, len(tokens))
	batchTokens := make([][]int, 0, batchTexts)
	for start := 0; start < len(items); {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		end := start + 1
		for end < len(items) && end-start < batchTexts && isWithinBudget(end-start+1, len(items[end].tokens), limit, batchTexts) {
			end++
		}
		batchItems := items[start:end]
		start = end
		batchTokens = batchTokens[:0]
		for _, item := range batchItems {
			batchTokens = append(batchTokens, item.tokens)
		}
		vectors, err := pass(ctx, batchTokens)
		if err != nil {
			return nil, err
		}
		if len(vectors) != len(batchItems) {
			return nil, fmt.Errorf("got %d vectors for %d texts", len(vectors), len(batchItems))
		}
		for i, item := range batchItems {
			out[item.index] = vectors[i]
		}
	}
	return out, nil
}

func isWithinBudget(rows, longest, limit, batchTexts int) bool {
	seq := roundUpToBucket(longest, limit)
	return rows*seq*seq <= getAttentionBudget(batchTexts)
}

func (e *Embedder) tokenize(texts []string) [][]int {
	tokens := make([][]int, len(texts))
	if len(texts) <= 1 {
		if len(texts) == 1 {
			tokens[0] = e.encode(texts[0])
		}
		return tokens
	}
	workers := min(len(texts), runtime.GOMAXPROCS(0))
	if workers <= 1 {
		for i, text := range texts {
			tokens[i] = e.encode(text)
		}
		return tokens
	}
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := range workers {
		start := (w * len(texts)) / workers
		end := ((w + 1) * len(texts)) / workers
		go func(start, end int) {
			defer wg.Done()
			for i := start; i < end; i++ {
				tokens[i] = e.encode(texts[i])
			}
		}(start, end)
	}
	wg.Wait()
	return tokens
}

func (e *Embedder) encode(text string) []int {
	ids := e.tokenizer.Encode(text)
	if len(ids) <= e.window {
		return ids
	}
	cut := slices.Clone(ids[:e.window])
	cut[len(cut)-1] = ids[len(ids)-1]
	return cut
}

func (e *Embedder) forward(ctx context.Context, batch [][]int) ([][]float32, error) {
	rows, seq, padded := padBatch(batch, e.pad, e.window)
	defer releasePadding(padded)

	session, err := e.sessions.acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer e.sessions.release(session)

	shape := []int64{int64(rows), int64(seq)}
	in := map[string]*ort.Value{}
	for _, one := range []struct {
		name string
		flat []int64
	}{
		{inputIDs, padded.ids},
		{attentionMask, padded.mask},
		{tokenTypeIDs, padded.types},
	} {
		if one.name == tokenTypeIDs && !e.isTyped {
			continue
		}
		value, err := ort.NewTensor(shape, one.flat)
		if err != nil {
			return nil, err
		}
		defer value.Destroy()
		in[one.name] = value
	}

	answered, err := session.Run(in)
	runtime.KeepAlive(padded)
	if err != nil {
		return nil, err
	}
	for _, v := range answered {
		defer v.Destroy()
	}

	held, ok := answered[e.output]
	if !ok {
		return nil, fmt.Errorf("%s answered without %s", e.name, e.output)
	}
	return e.readVectors(held, rows, seq, padded.mask)
}

func (e *Embedder) readVectors(held *ort.Value, rows, seq int, mask []int64) ([][]float32, error) {
	out, err := held.GetShape()
	if err != nil {
		return nil, err
	}
	if wide := out[len(out)-1]; wide != int64(e.dimensions) {
		return nil, fmt.Errorf("%s answered with %d dimensions, expected %d", e.name, wide, e.dimensions)
	}
	flat, err := ort.GetTensorData[float32](held)
	if err != nil {
		return nil, err
	}

	if len(out) == 2 {
		return splitRows(flat, rows, e.dimensions)
	}
	if want := rows * seq * e.dimensions; len(flat) != want {
		return nil, fmt.Errorf("%s returned %d values for %v", e.name, len(flat), out)
	}
	if e.isHeadPooled {
		return headPool(flat, rows, seq, e.dimensions), nil
	}
	return meanPool(flat, mask, rows, seq, e.dimensions), nil
}

func splitRows(flat []float32, rows, dimensions int) ([][]float32, error) {
	if len(flat) != rows*dimensions {
		return nil, fmt.Errorf("got %d values for %d vectors of %d dimensions", len(flat), rows, dimensions)
	}
	out := make([][]float32, rows)
	for row := range out {
		vector := slices.Clone(flat[row*dimensions : (row+1)*dimensions])
		out[row] = Normalise(vector)
	}
	return out, nil
}

func chooseOutput(named []string) (string, error) {
	for _, want := range outputs {
		if slices.Contains(named, want) {
			return want, nil
		}
	}
	if len(named) == 1 {
		return named[0], nil
	}
	return "", fmt.Errorf("answers with %v, expected one of %v", named, outputs)
}
