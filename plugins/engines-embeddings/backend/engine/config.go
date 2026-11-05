// Package engine runs embedding models locally on ONNX Runtime.
package engine

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/jiva-studio/numen/plugins/engines-embeddings/backend/onnx"
)

// Pooling strategies supported by sentence encoders.
const (
	PoolMean   = "mean"
	PoolCLS    = "cls"
	PoolLast   = "last"
	PoolOutput = "output"
)

// File conventions in Hugging Face model repositories.
const (
	ModelFile     = "model.onnx"
	TokenizerFile = "tokenizer.json"
	ModelFolder   = "onnx"
)

// LocalModel describes the location and execution parameters of an ONNX model.
type LocalModel struct {
	Name           string `json:"name"`
	Dir            string `json:"dir"`
	File           string `json:"file"`
	BatchTexts     int    `json:"batch_texts"`
	Runtime        string `json:"runtime"`
	Threads        int    `json:"threads"`
	Sessions       int    `json:"sessions"`
	PassBudget     int    `json:"pass_budget"`
	PassMemory     int64  `json:"pass_memory"`
	Provider       string `json:"provider"`
	ShouldDownload bool   `json:"download"`

	// Revision is the commit of the repository the files are fetched at. Empty means the default branch.
	Revision string `json:"revision"`
	// Sums maps a path inside the repository to the sha256 its file must hold.
	Sums map[string]string `json:"sums"`
}

// ModelInfo describes the embedding model properties exposed to clients.
type ModelInfo struct {
	Name        string  `json:"name"`
	Dimensions  int     `json:"dimensions"`
	MaxTokens   int     `json:"max_tokens"`
	Pooling     string  `json:"pooling"`
	From        string  `json:"from"`
	Concurrency int     `json:"concurrency"`
	FloorHint   float64 `json:"floor_hint"`
	// Handling is what is done to a text before the model reads it and to the output after.
	Handling Handling `json:"handling"`
	// MinChunkLength is the fewest bytes a piece of text holds to be given a vector, and 0 embeds every piece.
	MinChunkLength int `json:"min_chunk_length"`
	// ChunkTokens is the most tokens one chunk of a note may hold, and 0 names none.
	ChunkTokens int `json:"chunk_tokens"`
}

// CheckIdentity verifies the model dimension, tokens, and pooling.
func CheckIdentity(identity ModelInfo, repository string) error {
	if identity.Dimensions <= 0 {
		return fmt.Errorf("%s: dimensions must be positive", repository)
	}
	if identity.MaxTokens <= 0 {
		return fmt.Errorf("%s: max tokens must be positive", repository)
	}
	if identity.Pooling != "" && !isPooling(identity.Pooling) {
		return fmt.Errorf("%s is pooled %q, expected %q, %q, %q or %q", identity.Name, identity.Pooling, PoolMean, PoolCLS, PoolLast, PoolOutput)
	}
	return nil
}

// GetProvider resolves the onnxruntime Provider. A model that names none runs
// on the provider the host chose, which is auto when it chose none.
func (m LocalModel) GetProvider() onnx.Provider {
	name := m.Provider
	if name == "" {
		name = string(onnx.GetProvider())
	}
	p, err := onnx.ParseProvider(name)
	if err != nil {
		return onnx.ProviderAuto
	}
	return p
}

// GetThreads resolves the thread count.
func (m LocalModel) GetThreads() int {
	if m.Threads > 0 {
		return m.Threads
	}
	return onnx.GetDefaultThreads()
}

const (
	passLadderLow  = 2
	passLadderHigh = 4
)

// GetPasses calculates the concurrency level.
func (m LocalModel) GetPasses(provider onnx.Provider) int {
	switch {
	case m.Sessions > 0:
		return m.Sessions
	case provider != onnx.ProviderCPU:
		return 1
	case m.PassBudget > 0:
		return m.PassBudget
	}
	// gomutants:disable-next-line INTEGER_DECREMENT reason="GOMAXPROCS reads the setting for every argument below one"
	cpus := runtime.GOMAXPROCS(0)
	switch {
	case cpus <= passLadderLow:
		return 1
	case cpus <= passLadderHigh:
		return 2
	}
	return 3
}

// GetAddress returns a unique descriptor for the model source.
func (m LocalModel) GetAddress() string {
	pos := m.Name
	if m.Dir != "" {
		pos = filepath.ToSlash(filepath.Clean(m.Dir))
	}
	file := m.File
	if file == "" {
		file = ModelFile
	}
	return "local:" + pos + "/" + file
}

// NewConfig returns the model identity and the location and execution parameters of a local model
// from its profile.
func NewConfig(profile Profile) (ModelInfo, LocalModel) {
	info := ModelInfo{
		Name:           profile.ID,
		Dimensions:     profile.Dimensions,
		MaxTokens:      profile.GetTokens(),
		Pooling:        profile.Pooling,
		From:           "plugin:md.numen.engines.embeddings",
		Concurrency:    defaultPasses,
		FloorHint:      profile.FloorHint,
		MinChunkLength: 0,
		ChunkTokens:    0,
		Handling:       profile.GetHandling(),
	}
	local := LocalModel{
		Name:           profile.ID,
		Dir:            "",
		File:           profile.GetModelFile(),
		BatchTexts:     defaultBatchTexts,
		Runtime:        "",
		Threads:        0,
		Sessions:       0,
		PassBudget:     defaultPasses,
		PassMemory:     0,
		Provider:       "",
		ShouldDownload: true,
		Revision:       profile.Revision,
		Sums:           profile.SHA256,
	}
	return info, local
}

// defaultBatchTexts and defaultPasses are the most texts in a pass, and the most passes at once, that
// keep the resident memory of the plugin under the limit its manifest sets.
const (
	defaultBatchTexts = 8
	defaultPasses     = 1
)

// WithRepository returns the model fetched from another repository. The
// revision and sums belong to the repository they were pinned for, so they are
// dropped.
func (m LocalModel) WithRepository(name string) LocalModel {
	m.Name = name
	m.Revision = ""
	m.Sums = nil
	return m
}

// GetDefaultBatchTexts is the most texts in a pass of a local model.
func GetDefaultBatchTexts() int {
	return defaultBatchTexts
}
