// Package api holds the JSON messages of the saved search plugin: the input and output of each tool and
// the messages of the service of the tab.
package api

// Service is the service of the saved search tab, and the names of its methods.
const (
	Service            = "md.numen.tools.saved.search"
	MethodGetResults   = "GetResults"
	MethodWatchResults = "WatchResults"
	MethodUpdateSearch = "UpdateSearch"

	MethodListSearchFunctions = "ListSearchFunctions"
)

// Order is how the results of a run are ordered: OrderReranked when a reranking service ordered them,
// OrderFused when the scores of the query did.
type Order string

const (
	OrderReranked Order = "reranked"
	OrderFused    Order = "fused"
)

// Place is a place in a source: a page of a PDF, a position or range in a recording, or a position in
// an EPUB.
type Place struct {
	Page            int      `json:"page,omitempty"`
	PositionSeconds *float64 `json:"positionSeconds,omitempty"`
	EndSeconds      *float64 `json:"endSeconds,omitempty"`
	CFI             string   `json:"cfi,omitempty"`
	Chapter         string   `json:"chapter,omitempty"`
}

// Result is one result of a run: a passage of a file, or a whole file with IsWholeFile set. Line counts
// from 1.
type Result struct {
	Path        string   `json:"path"`
	Title       string   `json:"title"`
	Kind        string   `json:"kind"`
	Place       *Place   `json:"place,omitempty"`
	Text        string   `json:"text,omitempty"`
	Start       int      `json:"start"`
	Length      int      `json:"length"`
	Line        int      `json:"line"`
	URI         string   `json:"uri"`
	Score       *float64 `json:"score,omitempty"`
	IsWholeFile bool     `json:"isWholeFile,omitempty"`
}

// Problem is one problem of a query. Line and Column count from 1.
type Problem struct {
	Line    int    `json:"line" jsonschema:"the line of the problem, counted from 1"`
	Column  int    `json:"column" jsonschema:"the column of the problem, in characters, counted from 1"`
	Message string `json:"message" jsonschema:"what is wrong"`
}

type GetResultsRequest struct {
	Path string `json:"path"`
}

// GetResultsResponse is a saved search as the tab shows it. Query, Title and Errors are those of the note
// as it reads now and of its last run; the results, their time and order, the warnings and the failed
// last run are those of the cache. ComputedAt is the time of the last good run and FailedAt of a run that
// failed after it, each "" when there is none. IsStale is true when a run will replace the results.
type GetResultsResponse struct {
	Path       string    `json:"path"`
	Title      string    `json:"title"`
	Query      string    `json:"query"`
	ComputedAt string    `json:"computedAt"`
	FailedAt   string    `json:"failedAt"`
	Failure    string    `json:"failure"`
	IsStale    bool      `json:"isStale"`
	IsRunning  bool      `json:"isRunning"`
	Order      Order     `json:"order"`
	Results    []Result  `json:"results"`
	Warnings   []string  `json:"warnings"`
	Errors     []Problem `json:"errors"`
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

// UpdateSearchRequest is the query the tab writes into the note.
type UpdateSearchRequest struct {
	Path  string `json:"path"`
	Query string `json:"query"`
}

// UpdateSearchResponse is the problems of a query that was not written, counted from the first line of
// the query; empty when the note was written.
type UpdateSearchResponse struct {
	Errors []Problem `json:"errors"`
}

type ListSearchFunctionsRequest struct{}

// SearchFunction is a function, a variable or a stage of the search language. Kind is "function",
// "variable" or "stage".
type SearchFunction struct {
	Name        string   `json:"name"`
	Kind        string   `json:"kind"`
	Signatures  []string `json:"signatures"`
	Description string   `json:"description"`
	Example     string   `json:"example"`
}

// ListSearchFunctionsResponse is the catalog of the search language the core runs, with the functions
// plugins add to it.
type ListSearchFunctionsResponse struct {
	Functions []SearchFunction `json:"functions"`
}
