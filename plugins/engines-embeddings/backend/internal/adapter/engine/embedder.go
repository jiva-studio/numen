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

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/domain/model"
	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/internal/adapter/engine/onnx"
)

// The inputs a sentence encoder exported to ONNX requires.
const (
	inputIDs      = "input_ids"
	attentionMask = "attention_mask"
	tokenTypeIDs  = "token_type_ids"
)

func getOutputs(pooling string) []string {
	if pooling == model.PoolOutput {
		return []string{"sentence_embedding"}
	}
	return []string{"last_hidden_state", "sentence_embedding"}
}

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
	isPositioned bool
	isNormalized bool

	sessions *sessionPool[*ort.Session]
	paddings *sync.Pool
}

// Open loads the model, fetching it first where this machine does not hold it.
func Open(
	ctx context.Context, library *onnx.Library, identity model.ModelInfo, cfg model.LocalModel, progress FetchProgress,
) (*Embedder, error) {
	if err := model.CheckIdentity(identity, cfg.Name); err != nil {
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

	engine, _, err := library.Open(ctx, onnx.ResolveLibrary(cfg.Runtime, ""))
	if err != nil {
		return nil, err
	}
	options, provider, err := library.NewSessionOptions(engine, onnx.SessionSettings{
		Provider: getProvider(cfg),
		Threads:  getThreads(cfg),

		ComputeUnits: onnx.ComputeUnitsDefault,
		ModelPath:    paths.Model,
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

	session, err := onnx.NewSession(engine, paths.Model, options)
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
		pad:          0,
		output:       "",
		isTyped:      false,
		isPositioned: false,
		sessions:     newSessionPool(sessions, passes, (*ort.Session).Destroy),
		paddings:     newPaddingPool(),
		isNormalized: identity.Handling.Normalise,
	}
	if pad, err := tokenizer.SpecialTokenID(api.TokPad); err == nil {
		e.pad = pad
	}

	inputs, err := checkInputs(cfg.Name, first.InputNames, identity.Handling.ExtraInputs)
	if err != nil {
		destroySessions()
		return nil, err
	}
	e.isTyped, e.isPositioned = inputs.isTyped, inputs.isPositioned
	if e.output, err = chooseOutput(first.OutputNames, e.pooling); err != nil {
		destroySessions()
		return nil, fmt.Errorf("%s: %w", cfg.Name, err)
	}
	return e, nil
}

// modelInputs says which of the optional inputs a model takes.
type modelInputs struct {
	isTyped, isPositioned bool
}

func checkInputs(owner string, names, extra []string) (modelInputs, error) {
	var inputs modelInputs
	for _, name := range names {
		switch name {
		case inputIDs, attentionMask:
		case tokenTypeIDs:
			inputs.isTyped = true
		case model.InputPositionIDs:
			if !slices.Contains(extra, model.InputPositionIDs) {
				return modelInputs{}, fmt.Errorf("%s takes %s: list it in extraInputs of its profile", owner, name)
			}
			inputs.isPositioned = true
		default:
			return modelInputs{}, fmt.Errorf("%s requires an unsupported input: %s", owner, name)
		}
	}
	if !slices.Contains(names, inputIDs) || !slices.Contains(names, attentionMask) {
		return modelInputs{}, fmt.Errorf("%s takes %v, expected tokens and mask", owner, names)
	}
	if slices.Contains(extra, model.InputPositionIDs) && !inputs.isPositioned {
		return modelInputs{}, fmt.Errorf("%s takes no %s, but its profile lists it in extraInputs", owner, model.InputPositionIDs)
	}
	return inputs, nil
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

func (e *Embedder) GetModel() model.ModelInfo {
	return model.ModelInfo{
		Name:           e.name,
		Dimensions:     e.dimensions,
		MaxTokens:      e.maxTokens,
		Pooling:        e.pooling,
		From:           e.origin,
		Concurrency:    e.passes,
		FloorHint:      0,
		MinChunkLength: 0,
		ChunkTokens:    0,
		Handling:       model.Handling{QueryPrefix: "", DocumentPrefix: "", File: "", Normalise: e.isNormalized, ExtraInputs: nil},
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
	windowTokens := slices.Clone(ids[:e.window])
	windowTokens[len(windowTokens)-1] = ids[len(ids)-1]
	return windowTokens
}

func (e *Embedder) forward(ctx context.Context, batch [][]int) ([][]float32, error) {
	rows, seq, padded := padBatch(e.paddings, batch, e.pad, e.window)
	defer e.paddings.Put(padded)

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
		{model.InputPositionIDs, padded.positions},
	} {
		if (one.name == tokenTypeIDs && !e.isTyped) || (one.name == model.InputPositionIDs && !e.isPositioned) {
			continue
		}
		value, err := ort.NewTensor(shape, one.flat)
		if err != nil {
			return nil, err
		}
		defer value.Destroy()
		in[one.name] = value
	}

	reply, err := session.Run(in)
	runtime.KeepAlive(padded)
	if err != nil {
		return nil, err
	}
	for _, v := range reply {
		defer v.Destroy()
	}

	target, ok := reply[e.output]
	if !ok {
		return nil, fmt.Errorf("%s answered without %s", e.name, e.output)
	}
	return e.readVectors(target, rows, seq, padded.mask)
}

func (e *Embedder) readVectors(target *ort.Value, rows, seq int, mask []int64) ([][]float32, error) {
	out, err := target.GetShape()
	if err != nil {
		return nil, err
	}
	if dims := out[len(out)-1]; dims != int64(e.dimensions) {
		return nil, fmt.Errorf("%s answered with %d dimensions, expected %d", e.name, dims, e.dimensions)
	}
	flat, err := ort.GetTensorData[float32](target)
	if err != nil {
		return nil, err
	}

	if len(out) == 2 {
		return splitRows(flat, rows, e.dimensions, e.isNormalized)
	}
	if want := rows * seq * e.dimensions; len(flat) != want {
		return nil, fmt.Errorf("%s returned %d values for %v", e.name, len(flat), out)
	}
	return pool(e.pooling, tokenOutput{flat: flat, mask: mask, rows: rows, seq: seq, dimensions: e.dimensions}, e.isNormalized)
}

func splitRows(flat []float32, rows, dimensions int, isNormalized bool) ([][]float32, error) {
	if len(flat) != rows*dimensions {
		return nil, fmt.Errorf("got %d values for %d vectors of %d dimensions", len(flat), rows, dimensions)
	}
	out := make([][]float32, rows)
	for row := range out {
		vector := slices.Clone(flat[row*dimensions : (row+1)*dimensions])
		out[row] = finishVector(vector, isNormalized)
	}
	return out, nil
}

func finishVector(vector []float32, isNormalized bool) []float32 {
	if isNormalized {
		return model.Normalise(vector)
	}
	return vector
}

func chooseOutput(named []string, pooling string) (string, error) {
	want := getOutputs(pooling)
	for _, name := range want {
		if slices.Contains(named, name) {
			return name, nil
		}
	}
	if len(named) == 1 {
		return named[0], nil
	}
	return "", fmt.Errorf("answers with %v, expected one of %v", named, want)
}
