package api

type QueryInput struct {
	Query string   `json:"query" jsonschema:"a query in the search language; the tool description lists its functions"`
	Wait  *float64 `json:"wait,omitempty" jsonschema:"seconds to wait for the results, 0 to 20 (default: 20)"`
}

type QueryResultsInput struct {
	ID        string   `json:"id" jsonschema:"the id library_query returned"`
	Wait      *float64 `json:"wait,omitempty" jsonschema:"seconds to wait for a running query, 0 to 20 (default: 20)"`
	PageToken string   `json:"pageToken,omitempty" jsonschema:"the nextPageToken of the previous page"`
}

// QueryResult is a passage or a whole file a query found.
type QueryResult struct {
	Path        string   `json:"path" jsonschema:"the path of the file, relative to the vault folder"`
	Kind        string   `json:"kind" jsonschema:"note, pdf, epub, audio, video, image or file"`
	Title       string   `json:"title,omitempty" jsonschema:"the title of the note, else the name of the file"`
	Place       string   `json:"place,omitempty" jsonschema:"the place of the passage in the source, such as a page or a time"`
	Text        string   `json:"text,omitempty" jsonschema:"the text around the hit, or the start of a whole file, truncated to a fixed size"`
	URI         string   `json:"uri" jsonschema:"the address of the passage or the file; use it as the target of a markdown link"`
	Score       *float64 `json:"score,omitempty" jsonschema:"the score of the result; absent for a file matched by conditions on the file only"`
	IsWholeFile bool     `json:"isWholeFile,omitempty" jsonschema:"the result is a whole file, not a passage"`
}

// QueryProblem is an error of a query.
type QueryProblem struct {
	Line       int    `json:"line,omitempty" jsonschema:"the line of the problem in the query, from 1"`
	Column     int    `json:"column,omitempty" jsonschema:"the column of the problem in the query, from 1"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty" jsonschema:"a name of the language close to the one written"`
}

type QueryProgress struct {
	Step string `json:"step" jsonschema:"queued, candidates, reranking or done"`
}

type QueryOutput struct {
	ID            string         `json:"id" jsonschema:"the id of the query, for library_query_results"`
	IsRunning     bool           `json:"isRunning" jsonschema:"the query is still running: call library_query_results with the id"`
	IsPartial     bool           `json:"isPartial" jsonschema:"the results are those before the rerank stage"`
	IsStale       bool           `json:"isStale" jsonschema:"the index of the vault changed after the query started"`
	Progress      QueryProgress  `json:"progress"`
	Results       []QueryResult  `json:"results" jsonschema:"passages and whole files, best first"`
	Errors        []QueryProblem `json:"errors" jsonschema:"the errors of the query; a query with errors has no results"`
	Warnings      []string       `json:"warnings"`
	NextPageToken string         `json:"nextPageToken,omitempty" jsonschema:"pass it to library_query_results for the next page"`
	TotalResults  int            `json:"totalResults" jsonschema:"the count of results of the query"`
	Message       string         `json:"message,omitempty"`
}
