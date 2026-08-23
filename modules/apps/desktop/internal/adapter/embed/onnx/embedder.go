// Package onnx embeds text with a model running on this machine: an ONNX
// sentence encoder on GoMLX's Go backend. There is no C dependency and no
// network, so it builds for every platform from one machine and works on an
// installation nobody configured.
package onnx

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/gomlx/compute"
	_ "github.com/gomlx/compute/gobackend" // the pure-Go backend, registered under "go"
	"github.com/gomlx/go-huggingface/hub"
	"github.com/gomlx/go-huggingface/tokenizers/api"
	"github.com/gomlx/go-huggingface/tokenizers/hftokenizer"
	"github.com/gomlx/gomlx/core/graph"
	"github.com/gomlx/gomlx/core/tensors"
	"github.com/gomlx/gomlx/ml/model"
	onnxgomlx "github.com/gomlx/onnx-gomlx/onnx"
	"github.com/gomlx/onnx-gomlx/onnx/parser"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/embed"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
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
	name       string
	from       string
	dimensions int
	maxTokens  int
	pooling    string
	batchTexts int

	tokenizer api.Tokenizer
	pad       int
	net       onnxgomlx.Model
	exec      *model.Exec
	typeIDs   bool
	// pooled says the model's output is already one vector per text.
	pooled bool
	// head says the vector is the token that opens a text rather than the
	// average of them.
	head bool

	// One compiled graph, one execution at a time.
	mu sync.Mutex
}

// Open loads the model and compiles it. It is expensive — the weights are read
// and converted — and the result is reusable for the life of the process.
func Open(is embed.Identity, cfg embed.LocalModel) (*Embedder, error) {
	if is.Dimensions <= 0 {
		return nil, fmt.Errorf("%s: dimensions must be known before a vector is stored", cfg.Name)
	}
	if is.Pooling != "" && is.Pooling != embed.PoolMean && is.Pooling != embed.PoolHead {
		return nil, fmt.Errorf("%s is pooled %q, and a model is pooled %q or %q",
			is.Name, is.Pooling, embed.PoolMean, embed.PoolHead)
	}
	paths, err := locate(cfg)
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
		name:       is.Name,
		from:       paths.model,
		dimensions: is.Dimensions,
		maxTokens:  max(is.MaxTokens, tokenStep),
		pooling:    is.Pooling,
		batchTexts: max(cfg.BatchTexts, 1),
		tokenizer:  tokenizer,
		net:        net,
		head:       is.Pooling == embed.PoolHead,
	}
	if pad, err := tokenizer.SpecialTokenID(api.TokPad); err == nil {
		e.pad = pad
	}

	names, _ := net.Inputs()
	for _, name := range names {
		switch name {
		case inputIDs, attentionMask:
		case tokenTypeIDs:
			e.typeIDs = true
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
		if e.typeIDs {
			in[tokenTypeIDs] = inputs[2]
		}
		return net.CallGraph(scope, inputs[0].Graph(), in, output)
	})
	if err != nil {
		return nil, err
	}
	// One entry per sequence length the buckets allow.
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

	ids := make([][]int64, len(batch))
	mask := make([][]int64, len(batch))
	types := make([][]int64, len(batch))
	for row, tokens := range batch {
		ids[row] = make([]int64, seq)
		mask[row] = make([]int64, seq)
		types[row] = make([]int64, seq)
		for i := range ids[row] {
			ids[row][i] = int64(e.pad)
		}
		for i, id := range tokens {
			ids[row][i] = int64(id)
			mask[row][i] = 1
		}
	}

	args := []any{ids, mask}
	if e.typeIDs {
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
		return split(flat, len(batch), e.dimensions)
	}
	if want := len(batch) * seq * e.dimensions; len(flat) != want {
		return nil, fmt.Errorf("%s returned %d values for %s", e.name, len(flat), outputs[0].Shape())
	}
	if e.head {
		return headPool(flat, len(batch), seq, e.dimensions), nil
	}
	return meanPool(flat, mask, e.dimensions), nil
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

// paths are the files a model is made of. tokenizer.json defines the tokeniser
// in full, special tokens included.
type paths struct {
	model     string
	tokenizer string
}

// Where a repository keeps the models it publishes, and which of them is run
// when the configuration names none.
const (
	modelFolder = "onnx"
	modelFile   = "model.onnx"
	tokenFile   = "tokenizer.json"
)

// locate finds the model's files: in a directory the configuration names, or in
// the download cache. A named directory is what an installation with no network
// uses.
func locate(cfg embed.LocalModel) (paths, error) {
	file := cfg.File
	if file == "" {
		file = modelFile
	}
	if cfg.Dir != "" {
		p := paths{
			model:     filepath.Join(cfg.Dir, file),
			tokenizer: filepath.Join(cfg.Dir, tokenFile),
		}
		for _, required := range []string{p.model, p.tokenizer} {
			if _, err := os.Stat(required); err != nil {
				return paths{}, fmt.Errorf("model directory %s: %w", cfg.Dir, err)
			}
		}
		return p, nil
	}
	if cfg.Name == "" {
		return paths{}, errors.New("no model to run: name a repository or a directory")
	}
	if !cfg.Download {
		return paths{}, fmt.Errorf("%s is not on this machine: set local.dir to where it is, or local.download to fetch it", cfg.Name)
	}

	repo := hub.New(cfg.Name).WithProgressBar(false)
	folder, err := published(repo)
	if err != nil {
		return paths{}, err
	}
	if !slices.Contains(folder, file) {
		return paths{}, fmt.Errorf("%s publishes no %s/%s: it has %v", cfg.Name, modelFolder, file, folder)
	}

	p := paths{}
	for _, name := range wanted(folder, file) {
		at, err := repo.DownloadFile(modelFolder + "/" + name)
		if err != nil {
			return paths{}, err
		}
		if name == file {
			p.model = at
		}
		if name == tokenFile {
			p.tokenizer = at
		}
	}
	// The tokeniser stands at the root of a repository, and beside the models
	// in some.
	if repo.HasFile(tokenFile) {
		if p.tokenizer, err = repo.DownloadFile(tokenFile); err != nil {
			return paths{}, err
		}
	}
	if p.tokenizer == "" {
		return paths{}, fmt.Errorf("%s publishes no %s", cfg.Name, tokenFile)
	}
	return p, nil
}

// published is what a repository holds beside its models, by the names they
// have inside that folder.
func published(repo *hub.Repo) ([]string, error) {
	var out []string
	for name, err := range repo.IterFileNames() {
		if err != nil {
			return nil, err
		}
		if rest, inside := strings.CutPrefix(name, modelFolder+"/"); inside && rest != "" {
			out = append(out, rest)
		}
	}
	return out, nil
}

// wanted is everything the model named is made of, out of what stands beside
// it: the model, and every file that is not another model's.
//
// A model too large for one file keeps its weights in a second under its own
// name, and a graph may point at a constant in a third. A folder holds the full
// build and the quantised ones together, and taking one means leaving the
// gigabytes belonging to the others.
func wanted(folder []string, named string) []string {
	var others []string
	for _, name := range folder {
		if name != named && strings.HasSuffix(name, ".onnx") {
			others = append(others, name)
		}
	}
	out := []string{named}
	for _, name := range folder {
		if name == named || theirs(name, others) {
			continue
		}
		out = append(out, name)
	}
	return out
}

// theirs says a file belongs to one of the models given: it is that model, or
// it stands beside it under that model's name.
func theirs(name string, models []string) bool {
	for _, model := range models {
		if strings.HasPrefix(name, model) {
			return true
		}
	}
	return false
}
