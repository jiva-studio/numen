package api

// Service is the service of the saved search tab, and the names of its methods.
const (
	Service            = "md.numen.views.saved.search"
	MethodGetResults   = "GetResults"
	MethodWatchResults = "WatchResults"
	MethodUpdateSearch = "UpdateSearch"
)

type GetResultsRequest struct {
	Path string `json:"path"`
}

// GetResultsResponse is a saved search as the tab shows it. Query, Rerank, Limit, Title and Errors are
// those of the note as it reads now; the results, their time and order, the candidates, the warnings and
// the failed last run are those of the cache. ComputedAt is the time of the last good run and FailedAt of
// a run that failed after it, each "" when there is none. IsStale is true when a run will replace the
// results.
type GetResultsResponse struct {
	Path           string          `json:"path"`
	Title          string          `json:"title"`
	Query          string          `json:"query"`
	Rerank         string          `json:"rerank"`
	Limit          int             `json:"limit"`
	ComputedAt     string          `json:"computedAt"`
	FailedAt       string          `json:"failedAt"`
	Failure        string          `json:"failure"`
	IsStale        bool            `json:"isStale"`
	IsRunning      bool            `json:"isRunning"`
	Order          string          `json:"order"`
	CandidateCount int             `json:"candidateCount"`
	Results        []PreviewResult `json:"results"`
	Warnings       []string        `json:"warnings"`
	Errors         []Problem       `json:"errors"`
}

type WatchResultsRequest struct {
	Path string `json:"path"`
}

// ResultsChange is a notice that the results of a note changed or a run of it started; the tab reads
// GetResults on each.
type ResultsChange struct {
	ComputedAt string `json:"computedAt"`
	IsRunning  bool   `json:"isRunning"`
}

// UpdateSearchRequest is the query, the rerank text and the limit the tab writes into the note; an empty
// rerank text and a limit of 0 remove their key.
type UpdateSearchRequest struct {
	Path   string `json:"path"`
	Query  string `json:"query"`
	Rerank string `json:"rerank"`
	Limit  int    `json:"limit"`
}

// UpdateSearchResponse is the problems of a query that was not written, counted from the first line of
// the query; empty when the note was written.
type UpdateSearchResponse struct {
	Errors []Problem `json:"errors"`
}
