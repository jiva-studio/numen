package searchquery

// MaxStages is the most stages a query has.
const MaxStages = 8

// MaxRerankWindow is the largest window a rerank stage may name.
const MaxRerankWindow = 500

// StageLimits bounds the stages. RerankWindow is the count of documents a rerank stage takes when the query
// names none, MaxReranked the most chunks one rerank stage sends, reused ones included, and
// MaxRerankedBytes the most bytes of one chunk text sent.
type StageLimits struct {
	RerankWindow     int
	MaxReranked      int
	MaxRerankedBytes int
}

// DefaultStageLimits returns the limits of the stages when no setting changes them.
func DefaultStageLimits() StageLimits {
	return StageLimits{RerankWindow: 60, MaxReranked: 300, MaxRerankedBytes: 4096}
}
