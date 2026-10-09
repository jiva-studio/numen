// Package onnx embeds text with a model running on this machine: an ONNX
// sentence encoder through ONNX Runtime, reached by name at run time.
//
// The runtime is the process's, opened once and shared with everything else
// here that runs a model.
package onnx

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"runtime"
	"slices"
	"sync"

	ort "github.com/getcharzp/onnxruntime_purego"
	"github.com/gomlx/go-huggingface/tokenizers/api"
	"github.com/gomlx/go-huggingface/tokenizers/hftokenizer"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/internal/onnxruntime"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// The inputs a sentence encoder exported to ONNX asks for.
const (
	inputIDs      = "input_ids"
	attentionMask = "attention_mask"
	tokenTypeIDs  = "token_type_ids"
)

// The outputs a vector is read from, best first. A model naming neither is read
// from the one output it has.
var outputs = []string{"sentence_embedding", "last_hidden_state"}

// Embedder is one model, loaded.
type Embedder struct {
	name string
	// origin is where these vectors are made, which is part of what they are.
	origin     string
	dimensions int
	maxTokens  int
	pooling    string
	batchTexts int
	poolSize   int

	tokenizer api.Tokenizer
	pad       int
	output    string
	isTyped   bool
	// isHeadPooled says the vector is the token that opens a text rather than the
	// average of them.
	isHeadPooled bool

	// Pool of inference sessions for concurrent passes.
	sessions    chan *ort.Session
	allSessions []*ort.Session
	mu          sync.Mutex
	isClosed    bool
}

// Open loads the model, fetching it first where this machine does not hold it.
// It is expensive — the weights are read — and the result is reusable for the
// life of the process.
//
// is is the identity the vectors this model returns are kept under, which the
// settings decide.
func Open(ctx context.Context, identity port.EmbeddingModel, cfg embed.LocalModel, progress embed.FetchProgress) (*Embedder, error) {
	if identity.Dimensions <= 0 {
		return nil, fmt.Errorf("%s: dimensions must be known before a vector is stored", cfg.Name)
	}
	// Where a text is cut off is part of what a vector is, and this machine is
	// what does the cutting. A model run here says where.
	if identity.MaxTokens <= 0 {
		return nil, fmt.Errorf("%s: where a text is cut off must be known before a vector is stored", cfg.Name)
	}
	if identity.Pooling != "" && identity.Pooling != embed.PoolMean && identity.Pooling != embed.PoolHead {
		return nil, fmt.Errorf("%s is pooled %q, and a model is pooled %q or %q",
			identity.Name, identity.Pooling, embed.PoolMean, embed.PoolHead)
	}
	paths, err := embed.Locate(ctx, cfg, progress)
	if err != nil {
		return nil, err
	}
	tokenizer, err := hftokenizer.NewFromFile(nil, paths.Tokenizer)
	if err != nil {
		return nil, err
	}

	// The runtime's own cache is where it is looked for. What the settings call
	// a directory here holds the weights, which is another folder.
	engine, _, err := onnxruntime.Open(ctx, onnxruntime.Settings{
		Section:        "indexing.embedding",
		Runtime:        cfg.Runtime,
		ShouldDownload: cfg.ShouldDownload,
		Fetching:       createFetchListener(progress),
	})
	if err != nil {
		return nil, err
	}
	options, _, err := onnxruntime.NewSessionOptions(engine, onnxruntime.SessionSettings{
		Provider: cfg.GetProvider(),
		Threads:  cfg.GetThreads(),
	})
	if err != nil {
		return nil, err
	}
	defer options.Destroy()

	poolSize := cfg.GetSessions()
	if poolSize < 1 {
		poolSize = 1
	}

	sessions := make([]*ort.Session, 0, poolSize)
	destroySessions := func() {
		for _, s := range sessions {
			if s != nil {
				s.Destroy()
			}
		}
	}

	for range poolSize {
		session, err := onnxruntime.NewSession(engine, paths.Model, options)
		if err != nil {
			destroySessions()
			return nil, fmt.Errorf("reading %s: %w", paths.Model, err)
		}
		sessions = append(sessions, session)
	}

	first := sessions[0]
	e := &Embedder{
		name:         identity.Name,
		origin:       identity.From,
		dimensions:   identity.Dimensions,
		maxTokens:    identity.MaxTokens,
		pooling:      identity.Pooling,
		batchTexts:   max(cfg.BatchTexts, 1),
		poolSize:     poolSize,
		tokenizer:    tokenizer,
		sessions:     make(chan *ort.Session, poolSize),
		allSessions:  sessions,
		isHeadPooled: identity.Pooling == embed.PoolHead,
	}
	for _, s := range sessions {
		e.sessions <- s
	}

	if pad, err := tokenizer.SpecialTokenID(api.TokPad); err == nil {
		e.pad = pad
	}

	for _, name := range first.InputNames {
		switch name {
		case inputIDs, attentionMask:
		case tokenTypeIDs:
			e.isTyped = true
		default:
			destroySessions()
			return nil, fmt.Errorf("%s asks for an input this adapter does not have: %s", cfg.Name, name)
		}
	}
	if !slices.Contains(first.InputNames, inputIDs) || !slices.Contains(first.InputNames, attentionMask) {
		destroySessions()
		return nil, fmt.Errorf("%s takes %v, and a sentence encoder takes tokens and a mask", cfg.Name, first.InputNames)
	}
	if e.output, err = chooseOutput(first.OutputNames); err != nil {
		destroySessions()
		return nil, fmt.Errorf("%s: %w", cfg.Name, err)
	}
	return e, nil
}

// Close releases the model. The runtime it ran on is the process's and stays.
func (e *Embedder) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.isClosed {
		return nil
	}
	e.isClosed = true
	close(e.sessions)
	for _, s := range e.allSessions {
		if s != nil {
			s.Destroy()
		}
	}
	e.allSessions = nil
	return nil
}

// Concurrency is how many batches this embedder runs in parallel.
func (e *Embedder) Concurrency() int {
	return e.poolSize
}

// PoolSize is how many sessions the pool holds.
func (e *Embedder) PoolSize() int {
	return e.poolSize
}

// acquireSession borrows one session from the pool.
func (e *Embedder) acquireSession(ctx context.Context) (*ort.Session, error) {
	e.mu.Lock()
	if e.isClosed {
		e.mu.Unlock()
		return nil, errors.New("the embedder is closed")
	}
	e.mu.Unlock()

	if ctx != nil {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case s, ok := <-e.sessions:
			if !ok || s == nil {
				return nil, errors.New("the embedder is closed")
			}
			return s, nil
		}
	}

	s, ok := <-e.sessions
	if !ok || s == nil {
		return nil, errors.New("the embedder is closed")
	}
	return s, nil
}

// releaseSession returns one session to the pool.
func (e *Embedder) releaseSession(s *ort.Session) {
	if s == nil {
		return
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.isClosed {
		return
	}
	e.sessions <- s
}

func (e *Embedder) Model() port.EmbeddingModel {
	return port.EmbeddingModel{
		Name: e.name, Dimensions: e.dimensions, MaxTokens: e.maxTokens, Pooling: e.pooling,
		From: e.origin,
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
	tokens := e.tokenize(texts)
	if len(texts) == 1 {
		return e.forward(ctx, tokens)
	}

	items := make([]sequenceItem, len(tokens))
	for i, tok := range tokens {
		items[i] = sequenceItem{index: i, tokens: tok}
	}
	slices.SortStableFunc(items, func(a, b sequenceItem) int {
		return cmp.Compare(len(a.tokens), len(b.tokens))
	})

	out := make([][]float32, len(texts))
	batchTokens := make([][]int, 0, e.batchTexts)
	for start := 0; start < len(items); start += e.batchTexts {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		end := min(start+e.batchTexts, len(items))
		batchItems := items[start:end]
		batchTokens = batchTokens[:0]
		for _, item := range batchItems {
			batchTokens = append(batchTokens, item.tokens)
		}
		vectors, err := e.forward(ctx, batchTokens)
		if err != nil {
			return nil, err
		}
		for i, item := range batchItems {
			out[item.index] = vectors[i]
		}
	}
	return out, nil
}

// tokenize turns each text into tokens.
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

// encode tokenises one text and truncates it, keeping the token that marks the
// end so that the model sees a complete sentence.
func (e *Embedder) encode(text string) []int {
	ids := e.tokenizer.Encode(text)
	if len(ids) <= e.maxTokens {
		return ids
	}
	cut := slices.Clone(ids[:e.maxTokens])
	cut[len(cut)-1] = ids[len(ids)-1]
	return cut
}

// forward is one pass of the model over one batch.
//
// A batch is laid out at the length of its longest text. The runtime takes a
// shape as it comes.
func (e *Embedder) forward(ctx context.Context, batch [][]int) ([][]float32, error) {
	rows, seq, padded := padBatch(batch, e.pad)
	defer releasePadding(padded)

	session, err := e.acquireSession(ctx)
	if err != nil {
		return nil, err
	}
	defer e.releaseSession(session)

	// The library keeps a pointer into each of these and nothing else does, so
	// they are held until the run is over.
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
	out, err := held.GetShape()
	if err != nil {
		return nil, err
	}
	if wide := out[len(out)-1]; wide != int64(e.dimensions) {
		return nil, fmt.Errorf("%s answered with %d dimensions and the index holds %d", e.name, wide, e.dimensions)
	}
	flat, err := ort.GetTensorData[float32](held)
	if err != nil {
		return nil, err
	}

	// A model that pools for itself answers one vector a text. One that does not
	// answers one a token, and the tokens the mask drops are not part of what a
	// text says.
	if len(out) == 2 {
		return split(flat, rows, e.dimensions)
	}
	if want := rows * seq * e.dimensions; len(flat) != want {
		return nil, fmt.Errorf("%s returned %d values for %v", e.name, len(flat), out)
	}
	if e.isHeadPooled {
		return headPool(flat, rows, seq, e.dimensions), nil
	}
	return meanPool(flat, padded.mask, rows, seq, e.dimensions), nil
}

func split(flat []float32, rows, dimensions int) ([][]float32, error) {
	if len(flat) != rows*dimensions {
		return nil, fmt.Errorf("got %d values for %d vectors of %d dimensions", len(flat), rows, dimensions)
	}
	out := make([][]float32, rows)
	for row := range out {
		out[row] = slices.Clone(flat[row*dimensions : (row+1)*dimensions])
	}
	return out, nil
}

// chooseOutput picks what the vector is read from. The runtime names a session's
// outputs and shapes none of them until it has run, so a model naming neither of
// the two this reads is taken at its only output and refused where it has
// several.
func chooseOutput(named []string) (string, error) {
	for _, want := range outputs {
		if slices.Contains(named, want) {
			return want, nil
		}
	}
	if len(named) == 1 {
		return named[0], nil
	}
	return "", fmt.Errorf("answers with %v, and a vector is read from one of %v", named, outputs)
}

// createFetchListener is what fetching the runtime reports to. What is coming
// down is named to the runtime's own listener and not to this one, which is
// told about a model.
func createFetchListener(progress embed.FetchProgress) func(string, int64, int64) {
	if progress == nil {
		return nil
	}
	return func(_ string, done, total int64) { progress(done, total) }
}
