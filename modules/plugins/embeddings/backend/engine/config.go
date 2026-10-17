package engine

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/jiva-studio/numen/modules/plugins/embeddings/backend/onnxruntime"
)

// Pooling strategies supported by sentence encoders.
const (
	PoolMean = "mean"
	PoolHead = "head"
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
}

// ModelInfo describes the embedding model properties exposed to clients.
type ModelInfo struct {
	Name        string `json:"name"`
	Dimensions  int    `json:"dimensions"`
	MaxTokens   int    `json:"max_tokens"`
	Pooling     string `json:"pooling"`
	From        string `json:"from"`
	Concurrency int    `json:"concurrency"`
}

// CheckIdentity verifies the model dimension, tokens, and pooling.
func CheckIdentity(identity ModelInfo, repository string) error {
	if identity.Dimensions <= 0 {
		return fmt.Errorf("%s: dimensions must be positive", repository)
	}
	if identity.MaxTokens <= 0 {
		return fmt.Errorf("%s: max tokens must be positive", repository)
	}
	if identity.Pooling != "" && identity.Pooling != PoolMean && identity.Pooling != PoolHead {
		return fmt.Errorf("%s is pooled %q, expected %q or %q", identity.Name, identity.Pooling, PoolMean, PoolHead)
	}
	return nil
}

// GetProvider resolves the onnxruntime Provider.
func (m LocalModel) GetProvider() onnxruntime.Provider {
	// gomutants:disable-next-line BRANCH_IF reason="ParseProvider maps the empty string to ProviderAuto on the slow path"
	if m.Provider == "" {
		return onnxruntime.ProviderAuto
	}
	p, err := onnxruntime.ParseProvider(m.Provider)
	if err != nil {
		return onnxruntime.ProviderAuto
	}
	return p
}

// GetThreads resolves the thread count.
func (m LocalModel) GetThreads() int {
	if m.Threads > 0 {
		return m.Threads
	}
	return onnxruntime.DefaultThreads()
}

const (
	passLadderLow  = 2
	passLadderHigh = 4
)

// GetPasses calculates the concurrency level.
func (m LocalModel) GetPasses(provider onnxruntime.Provider) int {
	switch {
	case m.Sessions > 0:
		return m.Sessions
	case provider != onnxruntime.ProviderCPU:
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
	at := m.Name
	if m.Dir != "" {
		at = filepath.ToSlash(filepath.Clean(m.Dir))
	}
	file := m.File
	if file == "" {
		file = ModelFile
	}
	return "local:" + at + "/" + file
}

// DefaultBGEM3Config returns the default configuration for BGE-M3.
func DefaultBGEM3Config() (ModelInfo, LocalModel) {
	info := ModelInfo{
		Name:        "bge-m3",
		Dimensions:  1024,
		MaxTokens:   8192,
		Pooling:     PoolHead,
		From:        "plugin:md.numen.embeddings",
		Concurrency: 4,
	}
	local := LocalModel{
		Name:           "Xenova/bge-m3",
		File:           "model_int8.onnx",
		BatchTexts:     32,
		ShouldDownload: true,
	}
	return info, local
}
