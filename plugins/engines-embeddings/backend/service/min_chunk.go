package service

import (
	"math"
	"strconv"
	"strings"
)

// The key and bounds of the setting that names the shortest piece of text given a vector. They
// repeat the setting in manifest.json.
const (
	settingMinChunkLength = "min_chunk_length"
	defaultMinChunkLength = 40
	maxMinChunkLength     = 1000
)

// readMinChunkLength returns the shortest piece of text, in bytes, that is given a vector. A value
// that is no number gives the default, a number with a fraction is cut to its whole part, and a
// number outside the bounds of the setting is brought to the nearest bound.
func readMinChunkLength(values map[string]string) int {
	raw, ok := values[settingMinChunkLength]
	if !ok {
		return defaultMinChunkLength
	}
	number, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.IsNaN(number) {
		return defaultMinChunkLength
	}
	return int(min(max(math.Trunc(number), 0), maxMinChunkLength))
}
