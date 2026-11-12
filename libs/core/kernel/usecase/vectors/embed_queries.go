package vectors

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchsource"
	"github.com/jiva-studio/numen/libs/core/kernel/port/searchsourceport"
)

// EmbedQueries makes the vectors of queries the way the search source compares a query with its place.
type EmbedQueries struct {
	queries searchsourceport.QueryEmbedder
}

// NewEmbedQueries makes the vectors of queries with the queries of a search source.
func NewEmbedQueries(queries searchsourceport.QueryEmbedder) *EmbedQueries {
	return &EmbedQueries{queries: queries}
}

// Execute returns one vector per query, made for searching with the query as the question.
func (u *EmbedQueries) Execute(ctx context.Context, queries []string) ([][]float32, error) {
	return u.queries.EmbedQueries(ctx, queries)
}

// EmbedTexts embeds the texts with the search source in use. More than MaxEmbeddedTexts texts, or a text
// longer than MaxEmbeddedBytes, returns ErrLimitExceeded.
func (u *EmbedQueries) EmbedTexts(
	ctx context.Context, describe func() searchsource.Description, texts []string,
) (plugin.EmbeddedTexts, error) {
	recipe := describe().Recipe()
	if len(texts) > plugin.MaxEmbeddedTexts {
		return plugin.EmbeddedTexts{}, fmt.Errorf("%w: %d texts, at most %d", plugin.ErrLimitExceeded, len(texts), plugin.MaxEmbeddedTexts)
	}
	for i, text := range texts {
		if len(text) > plugin.MaxEmbeddedBytes {
			return plugin.EmbeddedTexts{}, fmt.Errorf("%w: text %d holds %d bytes, at most %d", plugin.ErrLimitExceeded, i, len(text), plugin.MaxEmbeddedBytes)
		}
	}
	if len(texts) == 0 {
		return plugin.EmbeddedTexts{Recipe: recipe}, nil
	}
	vectors, err := u.Execute(ctx, texts)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return plugin.EmbeddedTexts{}, ctxErr
		}
		return plugin.EmbeddedTexts{}, fmt.Errorf("%w: %w", plugin.ErrEmbeddingFailed, err)
	}
	if len(vectors) != len(texts) {
		return plugin.EmbeddedTexts{}, fmt.Errorf("%w: %d vectors for %d texts", plugin.ErrEmbeddingFailed, len(vectors), len(texts))
	}
	if after := describe().Recipe(); after != recipe {
		return plugin.EmbeddedTexts{}, fmt.Errorf("%w: the search source changed while the texts were embedded", plugin.ErrRecipeMismatch)
	}
	return plugin.EmbeddedTexts{Recipe: recipe, Vectors: vectors}, nil
}
