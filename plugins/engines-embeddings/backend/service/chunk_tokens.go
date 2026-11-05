package service

import (
	"math"
	"strconv"
	"strings"
)

// The key and bounds of the setting that names the longest chunk of a note, in tokens. They repeat
// the setting in manifest.json.
const (
	settingChunkTokens = "chunkTokens"
	defaultChunkTokens = 256
	minChunkTokens     = 16
	maxChunkTokens     = 8192
)

// ReadChunkTokens is the requested chunk size: the default for a value that is no number, the
// whole part of a fraction, and the nearest bound for a number outside the bounds.
func ReadChunkTokens(values map[string]string) int {
	number, err := strconv.ParseFloat(strings.TrimSpace(values[settingChunkTokens]), 64)
	if err != nil || math.IsNaN(number) {
		return defaultChunkTokens
	}
	return int(min(max(math.Trunc(number), minChunkTokens), maxChunkTokens))
}

// ClampChunkTokens is the requested size limited to the model's own limit. A limit of zero or less
// is a model that named none, and holds nothing.
func ClampChunkTokens(requested, modelMaxTokens int) int {
	if modelMaxTokens <= 0 {
		return requested
	}
	return min(requested, modelMaxTokens)
}
