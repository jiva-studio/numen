// Package api holds the JSON messages of the saved search tools: the input and output of each tool.
package api

// The values of PreviewOutput.Order.
const (
	OrderReranked = "reranked"
	OrderFused    = "fused"
)

type PreviewInput struct {
	Query  string `json:"query" jsonschema:"the search query in CEL: meaning(\"…\"), words(\"…\"), kind and path joined with &&, || and !"`
	Rerank string `json:"rerank,omitempty" jsonschema:"what makes a result relevant, in the user's words, for the reranking model; optional"`
	Limit  int    `json:"limit,omitempty" jsonschema:"how many results to return (default: 20, max: 100)"`
}

// Place is a place in a source: a page of a PDF, a position or range in a recording, or a position in
// an EPUB.
type Place struct {
	Page            int      `json:"page,omitempty" jsonschema:"PDF: the page, from 1"`
	PageCount       int      `json:"pageCount,omitempty" jsonschema:"PDF: the number of pages"`
	PositionSeconds *float64 `json:"positionSeconds,omitempty" jsonschema:"recording: seconds from the start"`
	EndSeconds      *float64 `json:"endSeconds,omitempty" jsonschema:"recording: seconds from the start to the end of the range"`
	CFI             string   `json:"cfi,omitempty" jsonschema:"EPUB: the CFI of the place"`
	Chapter         string   `json:"chapter,omitempty" jsonschema:"EPUB: the table-of-contents title at the place"`
}

// PreviewResult is one result of a query: one passage, or neighboring passages of one file joined.
type PreviewResult struct {
	Path       string   `json:"path" jsonschema:"the path of the source file, relative to the vault folder"`
	Title      string   `json:"title" jsonschema:"the title of the note, else the name of the file"`
	Place      *Place   `json:"place,omitempty" jsonschema:"the place of the best passage in the source; absent for a source with no pages, times or EPUB positions, such as a note"`
	Text       string   `json:"text" jsonschema:"the text of the best passage of the result, truncated to a fixed size, often mid-sentence"`
	Start      int      `json:"start" jsonschema:"the byte offset of the result in the text of the file"`
	Length     int      `json:"length" jsonschema:"the length of the result in bytes; neighboring passages of one file are joined into one result"`
	Line       int      `json:"line" jsonschema:"the line of the best passage, counted from 1"`
	URI        string   `json:"uri" jsonschema:"the address of this result; use it as the target of a markdown link"`
	Score      *float64 `json:"score,omitempty" jsonschema:"the relevance score, rounded to 3 decimals: the reranking score when order is reranked, else the similarity by meaning; absent for a result found by words, kind or path alone"`
	ChunkCount int      `json:"chunkCount" jsonschema:"how many passages were joined into this result"`
}

// Problem is one problem of a query that is not valid.
type Problem struct {
	Line    int    `json:"line" jsonschema:"the line of the problem in the query, counted from 1"`
	Column  int    `json:"column" jsonschema:"the column of the problem in the query, in characters, counted from 1"`
	Message string `json:"message" jsonschema:"what is wrong"`
}

type PreviewOutput struct {
	Query          string          `json:"query" jsonschema:"the query run"`
	Results        []PreviewResult `json:"results" jsonschema:"the results, best first"`
	Order          string          `json:"order" jsonschema:"reranked when the reranking model ordered the results, fused when they are ordered by the scores of the query"`
	Floor          float64         `json:"floor,omitempty" jsonschema:"the similarity a passage reaches for meaning to hold; absent for a query without meaning"`
	CandidateCount int             `json:"candidateCount" jsonschema:"how many passages the searches found before the query was applied"`
	Warnings       []string        `json:"warnings,omitempty" jsonschema:"what limited the result, such as a reranking service that did not answer"`
	Errors         []Problem       `json:"errors,omitempty" jsonschema:"the problems of a query that is not valid, each with its line and column; a query with problems returns no result"`
}
