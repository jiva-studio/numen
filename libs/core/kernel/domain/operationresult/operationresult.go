// Package operationresult names the stored results of connection operations, so that a run that asks the
// same of the same service again reuses the answer.
package operationresult

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"math"
	"maps"
	"slices"
)

// OperationRerank is the operation whose results a rerank stage stores.
const OperationRerank = "llm.rerank"

// RerankRecipe is what makes two rerank scores of one text comparable: the connection by its type and
// address, the version of the plugin that provides the type, the options of the row and the cut of a text.
type RerankRecipe struct {
	Fingerprint      string
	ProviderVersion  string
	Options          map[string]string
	MaxRerankedBytes int
}

// GetRerankRecipeHash returns the hex SHA-256 of the canonical JSON of the operation and the recipe, the
// options sorted by key.
func GetRerankRecipeHash(r RerankRecipe) string {
	options := make([][2]string, 0, len(r.Options))
	for _, key := range slices.Sorted(maps.Keys(r.Options)) {
		options = append(options, [2]string{key, r.Options[key]})
	}
	return getHash([]any{OperationRerank, r.Fingerprint, r.ProviderVersion, options, r.MaxRerankedBytes})
}

// GetRerankInputHash returns the hex SHA-256 of the canonical JSON of the query, the instruction and the
// text sent.
func GetRerankInputHash(query, instruction, text string) string {
	return getHash([]string{query, instruction, text})
}

func getHash(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}

// ErrInvalidScore is a stored output that is not a score.
var ErrInvalidScore = errors.New("the stored output is not a score")

// EncodeScore returns the stored output of a rerank score: IEEE 754 binary64, big-endian.
func EncodeScore(score float64) []byte {
	return binary.BigEndian.AppendUint64(nil, math.Float64bits(score))
}

// DecodeScore reads a score EncodeScore wrote.
func DecodeScore(output []byte) (float64, error) {
	if len(output) != 8 {
		return 0, ErrInvalidScore
	}
	score := math.Float64frombits(binary.BigEndian.Uint64(output))
	if math.IsNaN(score) || math.IsInf(score, 0) {
		return 0, ErrInvalidScore
	}
	return score, nil
}
