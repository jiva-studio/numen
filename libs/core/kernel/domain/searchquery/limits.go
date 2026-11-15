package searchquery

// Limits bounds one query.
type Limits struct {
	MaxQueryBytes          int
	MaxConditions          int
	MaxConditionBytes      int
	MaxDepth               int
	MaxPathPrefixes        int
	CandidatesPerCondition int
	MaxCandidates          int
	MaxDocuments           int
	ExactScanChunks        int
	CoarseOversample       int
	MaxCoarseCandidates    int
	MergeGapBytes          int
	DefaultLimit           int
	MaxLimit               int
	MaxLinkDepth           int
}

// DefaultLimits returns the limits of a query when no setting changes them.
func DefaultLimits() Limits {
	return Limits{
		MaxQueryBytes:          16384,
		MaxConditions:          16,
		MaxConditionBytes:      8192,
		MaxDepth:               32,
		MaxPathPrefixes:        8,
		CandidatesPerCondition: 300,
		MaxCandidates:          1000,
		MaxDocuments:           10000,
		ExactScanChunks:        20000,
		CoarseOversample:       8,
		MaxCoarseCandidates:    4096,
		MergeGapBytes:          200,
		DefaultLimit:           20,
		MaxLimit:               100,
		MaxLinkDepth:           16,
	}
}
