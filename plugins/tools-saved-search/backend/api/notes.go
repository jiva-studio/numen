package api

type CreateInput struct {
	Title  string `json:"title" jsonschema:"the title of the saved search note; the file is named after it"`
	Folder string `json:"folder,omitempty" jsonschema:"the folder of the vault to put the note in, relative to the vault folder; default: the folder of the setting, else the vault folder"`
	Query  string `json:"query" jsonschema:"the search query, as library_query takes it"`
}

type UpdateInput struct {
	Path  string  `json:"path" jsonschema:"the path of the saved search note, relative to the vault folder"`
	Query *string `json:"query,omitempty" jsonschema:"the new query, as library_query takes it"`
}

type SavedOutput struct {
	Path        string    `json:"path,omitempty" jsonschema:"the path of the note written; absent when nothing was written"`
	URI         string    `json:"uri,omitempty" jsonschema:"the address of the note; use it as the target of a markdown link"`
	ResultCount int       `json:"resultCount" jsonschema:"how many results the first run found"`
	Order       Order     `json:"order,omitempty" jsonschema:"reranked when a reranking service ordered the results, fused when the scores of the query did"`
	ComputedAt  string    `json:"computedAt,omitempty" jsonschema:"when the results were computed, RFC 3339"`
	Warnings    []string  `json:"warnings,omitempty" jsonschema:"what limited the result, or why the run failed"`
	Errors      []Problem `json:"errors,omitempty" jsonschema:"the problems of a query that is not valid; nothing is written when there are any"`
}
