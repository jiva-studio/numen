package preview

import "strconv"

// Config holds the numbers of the plugin's settings. A search returns DefaultLimit results when the input
// names none and at most MaxLimit. A query is applied to at most MaxCandidates passages, and its host
// filter names at most MaxPathPrefixes folders. Passages of one file at most MergeGapBytes apart are joined
// into one result. A query has at most MaxQueryBytes bytes.
//
// Without a reranking connection each search of a meaning or words leaf asks for ResultsPerSearch
// passages, also its limit of passages of one file, and the first FusedPerResult times the limit, at most
// MaxFused, of the passages the query keeps are joined into results. With a connection each search asks for
// RerankResultsPerSearch passages with the whole text of their chunks, and the first RerankedPerResult times
// the limit, at most MaxReranked, go to the reranker, each text cut to its first MaxRerankedBytes bytes.
type Config struct {
	DefaultLimit           int
	MaxLimit               int
	ResultsPerSearch       int
	RerankResultsPerSearch int
	MaxCandidates          int
	MaxPathPrefixes        int
	FusedPerResult         int
	MaxFused               int
	RerankedPerResult      int
	MaxReranked            int
	MaxRerankedBytes       int
	MergeGapBytes          int
	MaxQueryBytes          int
}

// DefaultConfig returns the defaults the manifest declares.
func DefaultConfig() Config {
	return Config{
		DefaultLimit: 20, MaxLimit: 100, ResultsPerSearch: 100, RerankResultsPerSearch: 300, MaxCandidates: 1000,
		MaxPathPrefixes: 8, FusedPerResult: 3, MaxFused: 100, RerankedPerResult: 10, MaxReranked: 300,
		MaxRerankedBytes: 4096, MergeGapBytes: 200, MaxQueryBytes: 16384,
	}
}

// ParseConfig reads the values of the settings over base. A value that is not a whole number at or above
// the least value of its setting keeps the value of base.
func ParseConfig(values map[string]string, base Config) Config {
	base.DefaultLimit = parseNumber(values, "defaultLimit", base.DefaultLimit, 1)
	base.MaxLimit = parseNumber(values, "maxLimit", base.MaxLimit, 1)
	base.ResultsPerSearch = parseNumber(values, "resultsPerSearch", base.ResultsPerSearch, 1)
	base.RerankResultsPerSearch = parseNumber(values, "rerankResultsPerSearch", base.RerankResultsPerSearch, 1)
	base.MaxCandidates = parseNumber(values, "maxCandidates", base.MaxCandidates, 1)
	base.MaxPathPrefixes = parseNumber(values, "maxPathPrefixes", base.MaxPathPrefixes, 1)
	base.FusedPerResult = parseNumber(values, "fusedPerResult", base.FusedPerResult, 1)
	base.MaxFused = parseNumber(values, "maxFused", base.MaxFused, 1)
	base.RerankedPerResult = parseNumber(values, "rerankedPerResult", base.RerankedPerResult, 1)
	base.MaxReranked = parseNumber(values, "maxReranked", base.MaxReranked, 1)
	base.MaxRerankedBytes = parseNumber(values, "maxRerankedBytes", base.MaxRerankedBytes, 1)
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
