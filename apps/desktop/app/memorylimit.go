package app

import (
	"math"
)

const bytesPerMB = 1 << 20

func applyMemoryLimit(limitMB int64, setLimit func(int64) int64, lookupEnv func(string) (string, bool)) {
	if limitMB <= 0 {
		return
	}
	if value, isSet := lookupEnv("GOMEMLIMIT"); isSet && value != "" {
		return
	}
	if setLimit(-1) != math.MaxInt64 {
		return
	}
	setLimit(limitMB * bytesPerMB)
}
