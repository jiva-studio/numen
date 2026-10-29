package floor

import (
	"context"
	"fmt"
	"strings"

	"github.com/jiva-studio/numen/libs/core/port/embeddingport"
	"github.com/jiva-studio/numen/libs/core/port/floorport"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// Probe is how near the best passage of a vault stands to a query, to be compared
// with the floor in force.
type Probe struct {
	nearest  floorport.Nearest
	resolve  Resolve
	embedder embeddingport.Embedder
}

// ProbeResult is what a probe measured.
type ProbeResult struct {
	// BestSimilarity is the similarity of the nearest passage, zero where the
	// vault has none.
	BestSimilarity float64
	// Floor is the floor in force, and DoesPass whether the nearest passage reaches it.
	Floor    float64
	DoesPass bool
}

// NewProbe returns the probe over the embedder given. A nil embedder has no
// model, and every probe then fails with ErrNoModel.
func NewProbe(nearest floorport.Nearest, resolve Resolve, embedder embeddingport.Embedder) Probe {
	return Probe{nearest: nearest, resolve: resolve, embedder: embedder}
}

// Execute embeds the query and measures the nearest passage of the vault. A query
// of no words is not embedded and has no passage near it.
func (u Probe) Execute(ctx context.Context, v domain.Vault, query string) (ProbeResult, error) {
	if u.embedder == nil {
		return ProbeResult{}, ErrNoModel
	}
	recipe := u.embedder.GetModel().Recipe()
	floor := u.resolve.GetFloor(ctx, v, recipe)
	if strings.TrimSpace(query) == "" {
		return ProbeResult{Floor: floor}, nil
	}

	vectors, err := u.embedder.Embed(ctx, []string{query})
	if err != nil {
		return ProbeResult{}, fmt.Errorf("%w: embed the query: %w", ErrEmbedding, err)
	}
	if len(vectors) != 1 {
		return ProbeResult{}, fmt.Errorf("%w: embed the query: %d vectors for one query", ErrEmbedding, len(vectors))
	}
	best, err := u.nearest.GetBestSimilarity(ctx, v.ID, recipe, vectors[0])
	if err != nil {
		return ProbeResult{}, err
	}
	return ProbeResult{BestSimilarity: best, Floor: floor, DoesPass: best >= floor}, nil
}
