package engine

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/onnx"
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

	// Revision is the commit of the repository the files are fetched at. Empty means the default branch.
	Revision string `json:"revision"`
	// Sums maps a path inside the repository to the sha256 its file must hold.
	Sums map[string]string `json:"sums"`
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

// DefaultBGEM3Config returns the default configuration for BGE-M3.
func DefaultBGEM3Config() (ModelInfo, LocalModel) {
	info := ModelInfo{
		Name:        "bge-m3",
		Dimensions:  1024,
		MaxTokens:   8192,
		Pooling:     PoolHead,
		From:        "plugin:md.numen.engines.embeddings",
		Concurrency: bgeM3Passes,
	}
	local := LocalModel{
		Name:           "Xenova/bge-m3",
		File:           "model_int8.onnx",
		BatchTexts:     bgeM3BatchTexts,
		PassBudget:     bgeM3Passes,
		ShouldDownload: true,
		Revision:       bgeM3Revision,
		Sums: map[string]string{
			ModelFolder + "/model_int8.onnx": "a206e10e995aa2a833924bcd725ba5dd6c3425cd34bac3cf2b5677cd2a1c51d6",
			TokenizerFile:                    "6710678b12670bc442b99edc952c4d996ae309a7020c1fa0096dd245c2faf790",
		},
	}
	return info, local
}

// bgeM3BatchTexts and bgeM3Passes are the most texts in a pass, and the most passes at once, that
// keep the resident memory of the plugin under the limit its manifest sets.
const (
	bgeM3BatchTexts = 8
	bgeM3Passes     = 1
)

// bgeM3Revision is the commit of Xenova/bge-m3 the pinned sums belong to.
const bgeM3Revision = "4de13258303883538bd53b696b452bf8099f0858"

// WithRepository returns the model fetched from another repository. The
// revision and sums belong to the repository they were pinned for, so they are
// dropped.
func (m LocalModel) WithRepository(name string) LocalModel {
	m.Name = name
	m.Revision = ""
	m.Sums = nil
	return m
}
