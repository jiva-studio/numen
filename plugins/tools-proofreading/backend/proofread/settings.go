package proofread

import "strconv"

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
