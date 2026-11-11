package api

type NoteSearchInput struct {
	Query string `json:"query" jsonschema:"words or phrase to search for across note titles and content"`
	Limit int    `json:"limit,omitempty" jsonschema:"maximum number of matches to return (default: 20, max: 100)"`
}

type SearchMatch struct {
	Path    string `json:"path" jsonschema:"relative path to the note"`
	Title   string `json:"title,omitempty" jsonschema:"title of the note"`
	Snippet string `json:"snippet,omitempty" jsonschema:"matching text snippet or passage"`
	Line    int    `json:"line,omitempty" jsonschema:"line of the match in the note body, counted from 1 at the first line of the body"`
	Score   int    `json:"score" jsonschema:"relevance score"`
}

type NoteSearchOutput struct {
	Query   string        `json:"query"`
	Matches []SearchMatch `json:"matches"`
	Total   int           `json:"total"`
}
