// Package profile holds the profiles of the cross-encoders the plugin runs: the shipped table, the
// person's profile files merged over it, and the score of a pair.
package profile

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

// DefaultModel is the profile a row with no model runs.
const DefaultModel = "onnx-community/bge-reranker-v2-m3-ONNX"

// ProfilesFolder is the folder, inside the data directory of the plugin, that holds the person's profile files.
const ProfilesFolder = "profiles"

// fewestTokens is the largest maxTokens that cannot hold the special tokens of a pair template and one
// token of each text.
const fewestTokens = 8

// ErrNoProfile is returned for a model that no profile lists.
var ErrNoProfile = errors.New("no profile for the model")

// Profile holds what a person, not the model file, decides about a cross-encoder: the longest pair it
// reads and where its files are. The score of a pair is the sigmoid of its single logit. A shipped
// profile names in Dependencies the id under which the manifest declares each file of the repository;
// a profile of the owner pins its files by Revision and SHA256.
type Profile struct {
	ID        string            `json:"id"`
	Aliases   []string          `json:"aliases,omitempty"`
	Title     string            `json:"title"`
	Tier      string            `json:"tier,omitempty"`
	SizeMB    int               `json:"sizeMB,omitempty"`
	MaxTokens int               `json:"maxTokens"`
	OnnxFile  string            `json:"onnxFile"`
	Revision  string            `json:"revision,omitempty"`
	SHA256    map[string]string `json:"sha256,omitempty"`

	Dependencies map[string]string `json:"dependencies,omitempty"`
}

// Validate returns an error when the profile cannot run a model.
func (p Profile) Validate() error {
	if strings.TrimSpace(p.ID) == "" {
		return errors.New("a profile needs an id")
	}
	if p.MaxTokens <= fewestTokens {
		return fmt.Errorf("%s: maxTokens %d is not above %d", p.ID, p.MaxTokens, fewestTokens)
	}
	if p.OnnxFile == "" {
		return fmt.Errorf("%s: a profile needs onnxFile", p.ID)
	}
	return nil
}

// GetOptionTitle returns the name the list of models shows for the profile.
func (p Profile) GetOptionTitle() string {
	return fmt.Sprintf("%s (%s, %d MB)", p.Title, p.Tier, p.SizeMB)
}

// GetScore returns the score of a pair from the logit of the model: its sigmoid, in [0, 1].
func GetScore(logit float32) float32 {
	return float32(1 / (1 + math.Exp(-float64(logit))))
}
