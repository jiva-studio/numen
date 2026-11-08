package service

import (
	"math"
	"strconv"
	"strings"
)

// The key and bounds of the setting that names the shortest piece of text given a vector. They
// repeat the setting in manifest.json.
const (
	settingMinChunkLength = "minChunkLength"
	defaultMinChunkLength = 40
	maxMinChunkLength     = 1000
)

func readMinChunkLength(values map[string]string) int {
	raw, ok := values[settingMinChunkLength]
	// gomutants:disable-next-line BRANCH_IF reason="missing key yields empty raw string that ParseFloat rejects"
	if !ok {
		return defaultMinChunkLength
	}
	// gomutants:disable-next-line INTEGER_DECREMENT,INTEGER_INCREMENT reason="ParseFloat treats bitSize other than 32 as 64"
	number, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || math.IsNaN(number) {
		return defaultMinChunkLength
	}
	return int(min(max(math.Trunc(number), 0), maxMinChunkLength))
}
