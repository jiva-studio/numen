// Package onnx embeds text with a model running on this machine: an ONNX
// sentence encoder on GoMLX's Go backend. There is no C dependency and no
// network, so it builds for every platform from one machine and works on an
// installation nobody configured.
package onnx

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

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
	// from is where these vectors are made, which is part of what they are.
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

// Fetching is how much of a model is here and how much is wanted, told while it
// comes down. Nothing is told for a model that is already on this machine.
type Fetching func(done, total int64)

// Open loads the model and compiles it, fetching it first where this machine
// does not hold it. It is expensive — the weights are read and converted — and
// the result is reusable for the life of the process.
//
// is is the identity the vectors this model returns are kept under, which the
// settings decide.
func Open(ctx context.Context, is port.EmbeddingModel, cfg embed.LocalModel, tell Fetching) (*Embedder, error) {
	if is.Dimensions <= 0 {
		return nil, fmt.Errorf("%s: dimensions must be known before a vector is stored", cfg.Name)
	}
	// Where a text is cut off is part of what a vector is, and this machine is
	// what does the cutting. A model run here says where.
	if is.MaxTokens <= 0 {
		return nil, fmt.Errorf("%s: where a text is cut off must be known before a vector is stored", cfg.Name)
	}
	if is.Pooling != "" && is.Pooling != embed.PoolMean && is.Pooling != embed.PoolHead {
		return nil, fmt.Errorf("%s is pooled %q, and a model is pooled %q or %q",
			is.Name, is.Pooling, embed.PoolMean, embed.PoolHead)
	}
	paths, err := locate(ctx, cfg, tell)
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
		from:       is.From,
		dimensions: is.Dimensions,
		maxTokens:  is.MaxTokens,
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
		From: e.from,
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
		vectors, err := split(flat, rows, e.dimensions)
		if err != nil {
			return nil, err
		}
		return vectors[:len(batch)], nil
	}
	if want := rows * seq * e.dimensions; len(flat) != want {
		return nil, fmt.Errorf("%s returned %d values for %s", e.name, len(flat), outputs[0].Shape())
	}
	if e.head {
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
	modelFile   = embed.ModelFile
	tokenFile   = "tokenizer.json"
)

// locate finds the model's files: in a directory the configuration names, or in
// the download cache. A named directory is what an installation with no network
// uses.
func locate(ctx context.Context, cfg embed.LocalModel, tell Fetching) (paths, error) {
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
	folder, sizes, err := published(repo)
	if err != nil {
		return paths{}, fmt.Errorf("what %s publishes: %w", cfg.Name, err)
	}
	if !slices.Contains(folder, file) {
		return paths{}, fmt.Errorf("%s publishes no %s/%s: it has %v", cfg.Name, modelFolder, file, folder)
	}

	files := wanted(folder, file)
	var total int64
	for _, name := range files {
		total += sizes[name]
	}
	if dir, err := repo.CacheDir(); err == nil {
		defer arriving(dir, files, sizes, total, tell)()
	}

	p := paths{}
	for _, name := range files {
		at, err := repo.DownloadFileCtx(ctx, modelFolder+"/"+name)
		if err != nil {
			return paths{}, fmt.Errorf("fetching %s/%s: %w", modelFolder, name, err)
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
		if p.tokenizer, err = repo.DownloadFileCtx(ctx, tokenFile); err != nil {
			return paths{}, fmt.Errorf("fetching %s: %w", tokenFile, err)
		}
	}
	if p.tokenizer == "" {
		return paths{}, fmt.Errorf("%s publishes no %s", cfg.Name, tokenFile)
	}
	return p, nil
}

// published is what a repository holds beside its models, by the names they
// have inside that folder, and how large each is.
func published(repo *hub.Repo) ([]string, map[string]int64, error) {
	var out []string
	sizes := map[string]int64{}
	for info, err := range repo.IterFileInfos() {
		if err != nil {
			return nil, nil, err
		}
		rest, inside := strings.CutPrefix(info.Name, modelFolder+"/")
		if !inside || rest == "" {
			continue
		}
		out = append(out, rest)
		sizes[rest] = info.Size
	}
	return out, sizes, nil
}

// arriving reports how much of the model is on this machine while it comes
// down, and hands back what stops the reporting.
//
// What is counted is the bytes under the repository's own place in the cache,
// which is what has arrived.
func arriving(dir string, files []string, sizes map[string]int64, total int64, tell Fetching) func() {
	if tell == nil || total <= 0 {
		return func() {}
	}
	done := make(chan struct{})
	over := make(chan struct{})
	go func() {
		defer close(over)
		tick := time.NewTicker(time.Second)
		defer tick.Stop()
		for {
			select {
			case <-done:
				return
			case <-tick.C:
				tell(min(weighed(dir, files, sizes), total), total)
			}
		}
	}()
	return func() {
		close(done)
		<-over
	}
}

// weighed is how many bytes of the files named stand under a folder. A folder
// holding another build of the same model holds bytes that are not this one's,
// and a file part-written counts for no more than the size it will take.
//
// A cache files one copy of a model and hangs its names off it, so a name is
// weighed as what it points at.
func weighed(dir string, files []string, sizes map[string]int64) int64 {
	wanted := make(map[string]int64, len(files))
	for _, name := range files {
		wanted[name] = sizes[name]
	}
	var sum int64
	_ = filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return nil
		}
		was, ours := wanted[strings.TrimSuffix(filepath.Base(path), ".incomplete")]
		if !ours {
			return nil
		}
		if info, err := os.Stat(path); err == nil {
			sum += min(info.Size(), was)
		}
		return nil
	})
	return sum
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
