package resultcache

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

// Problem is one problem of a query. Line and Column count from 1. EndLine and EndColumn are the last character
// of a problem that covers more than one, else zero.
type Problem struct {
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	EndLine   int    `json:"endLine,omitempty"`
	EndColumn int    `json:"endColumn,omitempty"`
	Message   string `json:"message"`
}
