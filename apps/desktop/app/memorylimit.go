package app

import (
	"math"
)

const bytesPerMB = 1 << 20

// applyMemoryLimit sets the soft memory limit of the Go runtime to limitMB megabytes. The collector works harder near a soft limit and allocation never fails. It does nothing when limitMB is zero, when GOMEMLIMIT is set, or when a limit is already in force.
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
