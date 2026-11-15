package pluginport

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

// SearchFunctions answers the functions plugins contribute to the query language. An answer holds only paths
// the function holds for; ErrInvalidFunctionArguments wrapped in the error is arguments the plugin refuses.
type SearchFunctions interface {
	// MatchDocuments returns the paths among paths the call holds for.
	MatchDocuments(ctx context.Context, call searchquery.FunctionCall, paths []string) (searchquery.FunctionAnswer, error)
	// FindDocuments returns at most limit paths the call holds for and an estimate of their count.
	FindDocuments(ctx context.Context, call searchquery.FunctionCall, limit int) (searchquery.FunctionAnswer, error)
}
