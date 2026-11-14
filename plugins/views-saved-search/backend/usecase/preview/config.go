package preview

import "strconv"

// Config holds the numbers of the plugin's settings. A search returns DefaultLimit results when the input
// names none and at most MaxLimit. ResultsPerSearch is the limit of each search of a meaning or words leaf
// and its limit of results of one file. A query is applied to at most MaxCandidates passages, and its host
// filter names at most MaxPathPrefixes folders. RerankedPerResult times the limit, at most MaxReranked,
// results go to the reranker. Passages of one file at most MergeGapBytes apart are joined into one result.
// A query has at most MaxQueryBytes bytes.
type Config struct {
	DefaultLimit      int
	MaxLimit          int
	ResultsPerSearch  int
	MaxCandidates     int
	MaxPathPrefixes   int
	RerankedPerResult int
	MaxReranked       int
	MergeGapBytes     int
	MaxQueryBytes     int
}

// DefaultConfig returns the defaults the manifest declares.
func DefaultConfig() Config {
	return Config{
		DefaultLimit: 20, MaxLimit: 100, ResultsPerSearch: 100, MaxCandidates: 1000, MaxPathPrefixes: 8,
		RerankedPerResult: 3, MaxReranked: 100, MergeGapBytes: 200, MaxQueryBytes: 16384,
	}
}

// ParseConfig reads the values of the settings over base. A value that is not a whole number at or above
// the least value of its setting keeps the value of base.
func ParseConfig(values map[string]string, base Config) Config {
	base.DefaultLimit = parseNumber(values, "defaultLimit", base.DefaultLimit, 1)
	base.MaxLimit = parseNumber(values, "maxLimit", base.MaxLimit, 1)
	base.ResultsPerSearch = parseNumber(values, "resultsPerSearch", base.ResultsPerSearch, 1)
	base.MaxCandidates = parseNumber(values, "maxCandidates", base.MaxCandidates, 1)
	base.MaxPathPrefixes = parseNumber(values, "maxPathPrefixes", base.MaxPathPrefixes, 1)
	base.RerankedPerResult = parseNumber(values, "rerankedPerResult", base.RerankedPerResult, 1)
	base.MaxReranked = parseNumber(values, "maxReranked", base.MaxReranked, 1)
	base.MergeGapBytes = parseNumber(values, "mergeGapBytes", base.MergeGapBytes, 0)
	base.MaxQueryBytes = parseNumber(values, "maxQueryBytes", base.MaxQueryBytes, 1)
	return base
}

func parseNumber(values map[string]string, key string, current, least int) int {
	if value, found := values[key]; found {
		if number, err := strconv.Atoi(value); err == nil && number >= least {
			return number
		}
	}
	return current
}
