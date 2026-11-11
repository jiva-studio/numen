package model

import (
	"fmt"
	"path/filepath"
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
	BatchTexts     int    `json:"batchTexts"`
	Runtime        string `json:"runtime"`
	Threads        int    `json:"threads"`
	Sessions       int    `json:"sessions"`
	PassBudget     int    `json:"passBudget"`
	PassMemory     int64  `json:"passMemory"`
	Provider       string `json:"provider"`
	ShouldDownload bool   `json:"download"`

	Revision string            `json:"revision"`
	Sums     map[string]string `json:"sums"`
}

// ModelInfo describes the embedding model properties exposed to clients.
type ModelInfo struct {
	Name           string   `json:"name"`
	Dimensions     int      `json:"dimensions"`
	MaxTokens      int      `json:"maxTokens"`
	Pooling        string   `json:"pooling"`
	From           string   `json:"from"`
	Concurrency    int      `json:"concurrency"`
	FloorHint      float64  `json:"floorHint"`
	Handling       Handling `json:"handling"`
	MinChunkLength int      `json:"minChunkLength"`
	ChunkTokens    int      `json:"chunkTokens"`
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
