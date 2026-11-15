package searchquery

import "github.com/jiva-studio/numen/libs/core/kernel/domain/modelcall"

// Request is one query. Limit is the count of results: zero takes Limits.DefaultLimit, or no cut when the
// query has a top stage; a count above Limits.MaxLimit is cut to it. Priority is whose action the calls of
// a rerank stage serve. ReportPartial, when set, receives the results the first rerank stage reranks, once,
// before that stage asks the reranking service.
type Request struct {
	Text          string
	Limit         int
	Priority      modelcall.Priority
	ReportPartial func(SearchResults)
}
