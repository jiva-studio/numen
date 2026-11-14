// Package chunking holds the settings that cut a note into pieces and the rules that keep them in
// bounds.
package chunking

import (
	"math"
	"strconv"
	"strings"
)

// The keys and bounds of the settings. They repeat the settings in manifest.json.
const (
	TokensKey        = "chunkTokens"
	DefaultTokens    = 256
	MinTokens        = 16
	MaxTokens        = 8192
	MinLengthKey     = "minChunkLength"
	DefaultMinLength = 40
	MaxMinLength     = 1000
)

// ReadTokens is the requested chunk size, in tokens: the default for a value that is no number,
// the whole part of a fraction, and the nearest bound for a number outside the bounds.
func ReadTokens(values map[string]string) int {
	number, err := strconv.ParseFloat(strings.TrimSpace(values[TokensKey]), 64)
	if err != nil || math.IsNaN(number) {
		return DefaultTokens
	}
	return int(min(max(math.Trunc(number), MinTokens), MaxTokens))
}

// ClampTokens is the requested size limited to the model's own limit. A limit of zero or less is a
// model that named none, and holds nothing.
func ClampTokens(requested, modelMaxTokens int) int {
	if modelMaxTokens <= 0 {
		return requested
	}
	return min(requested, modelMaxTokens)
}

// ReadMinLength is the length of the shortest piece of text given a vector, in characters.
func ReadMinLength(values map[string]string) int {
	raw, ok := values[MinLengthKey]
	// gomutants:disable-next-line BRANCH_IF reason="missing key yields empty raw string that ParseFloat rejects"
	if !ok {
		return DefaultMinLength
	}
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="ParseFloat treats bitSize other than 32 as 64"
	number, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.IsNaN(number) {
		return DefaultMinLength
	}
	return int(min(max(math.Trunc(number), 0), MaxMinLength))
}
