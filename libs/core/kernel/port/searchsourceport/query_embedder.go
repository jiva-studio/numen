package searchsourceport

import "context"

// QueryEmbedder makes the vectors of queries the way a source compares a query with its place. A plugin
// that calibrates the floor of a vault measures it with them.
type QueryEmbedder interface {
	// EmbedQueries returns one vector per query, in the order the queries arrived.
	EmbedQueries(ctx context.Context, queries []string) ([][]float32, error)
}
