// Package onnx embeds text with a model running on this machine: an ONNX
// sentence encoder on GoMLX's Go backend. There is no C dependency and no
// network, so it builds for every platform from one machine and works on an
// installation nobody configured.
package onnx

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/gomlx/compute"
	_ "github.com/gomlx/compute/gobackend" // the pure-Go backend, registered under "go"
	"github.com/gomlx/go-huggingface/tokenizers/api"
	"github.com/gomlx/go-huggingface/tokenizers/hftokenizer"
	"github.com/gomlx/gomlx/core/graph"
	"github.com/gomlx/gomlx/core/tensors"
	"github.com/gomlx/gomlx/ml/model"
	onnxgomlx "github.com/gomlx/onnx-gomlx/onnx"
	"github.com/gomlx/onnx-gomlx/onnx/parser"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// backendName is GoMLX's pure-Go backend: no C dependency, and one machine
// cross-compiles the application for every platform it ships to.
const backendName = "go"

// tokenStep is the granularity sequence lengths are rounded up to. A shape is
// compiled the first time it appears, so a run meets a handful of them.
const tokenStep = 64

// The inputs a sentence encoder exported to ONNX asks for.
const (
	inputIDs      = "input_ids"
	attentionMask = "attention_mask"
	tokenTypeIDs  = "token_type_ids"
)

// Embedder is one model, loaded and compiled.
type Embedder struct {
	name string
	// origin is where these vectors are made, which is part of what they are.
	origin     string
	dimensions int
	maxTokens  int
	pooling    string
	batchTexts int

	tokenizer api.Tokenizer
	pad       int
	net       onnxgomlx.Model
	exec      *model.Exec
	typed     bool
	// pooled says the model's output is already one vector per text.
	pooled bool
	// headPooled says the vector is the token that opens a text rather than the
	// average of them.
	headPooled bool

	// One compiled graph, one execution at a time.
	mu sync.Mutex
}

// Open loads the model and compiles it, fetching it first where this machine
// does not hold it. It is expensive — the weights are read and converted — and
// the result is reusable for the life of the process.
//
// is is the identity the vectors this model returns are kept under, which the
// settings decide.
func Open(ctx context.Context, identity port.EmbeddingModel, cfg embed.LocalModel, progress FetchProgress) (*Embedder, error) {
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
	paths, err := locate(ctx, cfg, progress)
	if err != nil {
		return nil, err
	}
	tokenizer, err := hftokenizer.NewFromFile(nil, paths.tokenizer)
	if err != nil {
		return nil, err
	}
	net, err := parser.ParseFile(paths.model)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", paths.model, err)
	}

	e := &Embedder{
		name:       identity.Name,
		origin:     identity.From,
		dimensions: identity.Dimensions,
		maxTokens:  identity.MaxTokens,
		pooling:    identity.Pooling,
		batchTexts: max(cfg.BatchTexts, 1),
		tokenizer:  tokenizer,
		net:        net,
		headPooled: identity.Pooling == embed.PoolHead,
	}
	if pad, err := tokenizer.SpecialTokenID(api.TokPad); err == nil {
		e.pad = pad
	}

	names, _ := net.Inputs()
	for _, name := range names {
		switch name {
		case inputIDs, attentionMask:
		case tokenTypeIDs:
			e.typed = true
		default:
			return nil, fmt.Errorf("%s asks for an input this adapter does not have: %s", cfg.Name, name)
		}
	}
	if !slices.Contains(names, inputIDs) || !slices.Contains(names, attentionMask) {
		return nil, fmt.Errorf("%s takes %v, and a sentence encoder takes tokens and a mask", cfg.Name, names)
	}

	output, pooled, err := e.chooseOutput()
	if err != nil {
		return nil, err
	}
	e.pooled = pooled

	backend, err := compute.NewWithConfig(backendName)
	if err != nil {
		return nil, err
	}
	store := model.NewStore()
	if err := net.VariablesToScope(store.RootScope()); err != nil {
		return nil, fmt.Errorf("loading the weights of %s: %w", cfg.Name, err)
	}
	exec, err := model.NewExec(backend, store, func(scope *model.Scope, inputs []*graph.Node) []*graph.Node {
		in := map[string]*graph.Node{inputIDs: inputs[0], attentionMask: inputs[1]}
		if e.typed {
			in[tokenTypeIDs] = inputs[2]
		}
		return net.CallGraph(scope, inputs[0].Graph(), in, output)
	})
	if err != nil {
		return nil, err
	}
	// One entry per sequence length the buckets allow, and two to spare.
	e.exec = exec.SetMaxCache(e.maxTokens/tokenStep + 2)
	return e, nil
}

// Close releases the model.
func (e *Embedder) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.exec != nil {
		e.exec.Finalize()
		e.exec = nil
	}
	if e.net != nil {
		err := e.net.Close()
		e.net = nil
		return err
	}
	return nil
}

func (e *Embedder) Model() port.EmbeddingModel {
	return port.EmbeddingModel{
		Name: e.name, Dimensions: e.dimensions, MaxTokens: e.maxTokens, Pooling: e.pooling,
		From: e.origin,
	}
}

// Embed runs the model over the texts, a batch at a time.
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	tokens := make([][]int, len(texts))
	for i, text := range texts {
		tokens[i] = e.encode(text)
	}
	out := make([][]float32, 0, len(texts))
	for start := 0; start < len(tokens); start += e.batchTexts {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		vectors, err := e.forward(tokens[start:min(start+e.batchTexts, len(tokens))])
		if err != nil {
			return nil, err
		}
		out = append(out, vectors...)
	}
	return out, nil
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
func (e *Embedder) forward(batch [][]int) ([][]float32, error) {
	longest := 0
	for _, ids := range batch {
		longest = max(longest, len(ids))
	}
	seq := bucket(longest, tokenStep, e.maxTokens)
	rows := max(e.batchTexts, len(batch))

	ids, mask, types := padded(batch, rows, seq, e.pad)

	args := []any{ids, mask}
	if e.typed {
		args = append(args, types)
	}

	e.mu.Lock()
	defer e.mu.Unlock()
	if e.exec == nil {
		return nil, errors.New("the embedder is closed")
	}
	outputs, err := e.exec.Call(args...)
	if err != nil {
		return nil, err
	}
	defer func() {
		for _, t := range outputs {
			_ = t.FinalizeAll()
		}
	}()

	flat, err := tensors.CopyFlatData[float32](outputs[0])
	if err != nil {
		return nil, err
	}
	if e.pooled {
		vectors, err := split(flat, rows, e.dimensions)
		if err != nil {
			return nil, err
		}
		return vectors[:len(batch)], nil
	}
	if want := rows * seq * e.dimensions; len(flat) != want {
		return nil, fmt.Errorf("%s returned %d values for %s", e.name, len(flat), outputs[0].Shape())
	}
	if e.headPooled {
		return headPool(flat, rows, seq, e.dimensions)[:len(batch)], nil
	}
	return meanPool(flat, mask, e.dimensions)[:len(batch)], nil
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

// chooseOutput picks what the vector is read from, and says whether it is
// already one vector per text. The width is checked here: what the index stores
// is the width the configuration claims.
func (e *Embedder) chooseOutput() (name string, pooled bool, err error) {
	names, dshapes := e.net.Outputs()
	preferred := []string{"sentence_embedding", "last_hidden_state"}
	order := make([]int, 0, len(names))
	for _, want := range preferred {
		if i := slices.Index(names, want); i >= 0 {
			order = append(order, i)
		}
	}
	for i := range names {
		if !slices.Contains(order, i) {
			order = append(order, i)
		}
	}
	for _, i := range order {
		shape := dshapes[i]
		if shape.Rank() < 2 {
			continue
		}
		if shape.Dimensions[shape.Rank()-1] != e.dimensions {
			continue
		}
		return names[i], shape.Rank() == 2, nil
	}
	return "", false, fmt.Errorf("%s has no output of %d dimensions: %v %v", e.name, e.dimensions, names, dshapes)
}
