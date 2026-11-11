package vectors

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/embedding"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/jiva-studio/numen/libs/core/kernel/port/embeddingport"
)

// EmbedQueries makes the vectors of search queries with an embedder.
type EmbedQueries struct {
	documents embeddingport.Embedder
	queries   embeddingport.QueryEmbedder
}

// NewEmbedQueries makes the vectors of queries with queries where the model tells a query from a
// document, and with documents, which has one kind of vector, where queries is nil.
func NewEmbedQueries(documents embeddingport.Embedder, queries embeddingport.QueryEmbedder) *EmbedQueries {
	return &EmbedQueries{documents: documents, queries: queries}
}

// Execute returns one vector per query, made for searching with the query as the question.
func (u *EmbedQueries) Execute(ctx context.Context, queries []string) ([][]float32, error) {
	if u.queries != nil {
		return u.queries.EmbedQueries(ctx, queries)
	}
	return u.documents.Embed(ctx, queries)
}

// EmbedTexts embeds the texts with the model in use. More than MaxEmbeddedTexts texts, or a text
// longer than MaxEmbeddedBytes, returns ErrLimitExceeded.
func (u *EmbedQueries) EmbedTexts(
	ctx context.Context, getModel func() embedding.Model, texts []string,
) (plugin.EmbeddedTexts, error) {
	recipe := getModel().Recipe()
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
	if after := getModel().Recipe(); after != recipe {
		return plugin.EmbeddedTexts{}, fmt.Errorf("%w: the model changed while the texts were embedded", plugin.ErrRecipeMismatch)
	}
	return plugin.EmbeddedTexts{Recipe: recipe, Vectors: vectors}, nil
}
