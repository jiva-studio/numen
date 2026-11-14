// Package settings is how a proofreading cuts a text into batches and how far it lets a correction
// move a line, read from the settings the host passes.
package settings

import (
	"strconv"

	"github.com/jiva-studio/numen/plugins/tools-proofreading/backend/domain/numbered"
)

// Config is how a proofreading cuts a text into batches and how far it lets a correction move a
// line.
type Config struct {
	BatchSize       int
	Overlap         int
	InFlight        int
	MaxEditDistance float64
}

// Default proofreading sizes.
const (
	DefaultBatchSize = 40
	DefaultOverlap   = 4
	DefaultInFlight  = 4
)

// DefaultConfig is the configuration before the host passes one.
func DefaultConfig() Config {
	return Config{
		BatchSize:       DefaultBatchSize,
		Overlap:         DefaultOverlap,
		InFlight:        DefaultInFlight,
		MaxEditDistance: numbered.MaxEditDistance,
	}
}

// ParseConfig returns the configuration with the sizes the settings name. A value that is not a
// number, or is below its least, leaves the size as it is.
func ParseConfig(values map[string]string, base Config) Config {
	base.BatchSize = parseInt(values, "batchSize", base.BatchSize, 1)
	base.Overlap = parseInt(values, "overlap", base.Overlap, 0)
	base.InFlight = parseInt(values, "inFlight", base.InFlight, 1)
	if v, ok := values["maxEditDistance"]; ok {
		if d, err := strconv.ParseFloat(v, 64); err == nil && d >= 0 {
			base.MaxEditDistance = d
		}
	}
	return base
}

func parseInt(values map[string]string, key string, current, least int) int {
	if v, ok := values[key]; ok {
		if n, err := strconv.Atoi(v); err == nil && n >= least {
			return n
		}
	}
	return current
}
