// Package hostsearch lends the search index and the search floor of the vaults to the plugins that
// call the host.
package hostsearch

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	indexport "github.com/jiva-studio/numen/libs/core/usecase/indexing/indexcontract"
	pluginport "github.com/jiva-studio/numen/libs/core/usecase/plugincommand/plugincontract"
	"github.com/jiva-studio/numen/libs/core/usecase/search"
	searchpolicyport "github.com/jiva-studio/numen/libs/core/usecase/search/searchpolicycontract"
	embeddingport "github.com/jiva-studio/numen/libs/core/usecase/vectors/embeddingcontract"
)

// Search is pluginport.PluginSearch over the index of the installation and the model that fills it.
type Search struct {
	// Embedder is the model that fills the index. Nil is an installation with no model, and every
	// call that needs one returns ErrNoModel.
	Embedder embeddingport.Embedder
	Samples  searchpolicyport.Samples
	Nearest  searchpolicyport.Nearest
	Progress indexport.IndexProgress
	Policy   search.FloorPolicy
}

// getRecipe is the recipe of the model in use, or ErrNoModel.
func (s Search) getRecipe() (string, error) {
	if s.Embedder == nil {
		return "", pluginport.ErrNoModel
	}
	return s.Embedder.GetModel().Recipe(), nil
}

// checkRecipe returns nil when the recipe is the one of the model in use.
func (s Search) checkRecipe(recipe string) error {
	current, err := s.getRecipe()
	if err != nil {
		return err
	}
	if recipe != current {
		return fmt.Errorf("%w: %q is not %q", pluginport.ErrRecipeMismatch, recipe, current)
	}
	return nil
}

// GetEmbeddingState reads the model in use and the progress of the vault under it from the index.
func (s Search) GetEmbeddingState(ctx context.Context, v domain.Vault) (pluginport.PluginEmbeddingState, error) {
	recipe, err := s.getRecipe()
	if err != nil {
		return pluginport.PluginEmbeddingState{}, err
	}
	held, embedded, err := s.Progress.GetProgress(ctx, v.ID, recipe)
	if err != nil {
		return pluginport.PluginEmbeddingState{}, fmt.Errorf("read the embedding progress of %s: %w", v.ID, err)
	}
	return pluginport.PluginEmbeddingState{
		Recipe:        recipe,
		Model:         s.Embedder.GetModel().Name,
		FloorHint:     s.Policy.Get(ctx, v, recipe).Hint,
		ChunkCount:    held,
		EmbeddedCount: embedded,
	}, nil
}

// EmbedTexts embeds the texts with the model in use. More than MaxEmbeddedTexts texts, or a text
// longer than MaxEmbeddedBytes, returns ErrLimitExceeded.
func (s Search) EmbedTexts(ctx context.Context, texts []string) (pluginport.PluginEmbeddedTexts, error) {
	recipe, err := s.getRecipe()
	if err != nil {
		return pluginport.PluginEmbeddedTexts{}, err
	}
	if len(texts) > pluginport.MaxEmbeddedTexts {
		return pluginport.PluginEmbeddedTexts{}, fmt.Errorf("%w: %d texts, at most %d", pluginport.ErrLimitExceeded, len(texts), pluginport.MaxEmbeddedTexts)
	}
	for i, text := range texts {
		if len(text) > pluginport.MaxEmbeddedBytes {
			return pluginport.PluginEmbeddedTexts{}, fmt.Errorf("%w: text %d holds %d bytes, at most %d", pluginport.ErrLimitExceeded, i, len(text), pluginport.MaxEmbeddedBytes)
		}
	}
	if len(texts) == 0 {
		return pluginport.PluginEmbeddedTexts{Recipe: recipe}, nil
	}
	vectors, err := s.Embedder.Embed(ctx, texts)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return pluginport.PluginEmbeddedTexts{}, ctxErr
		}
		return pluginport.PluginEmbeddedTexts{}, fmt.Errorf("%w: %w", pluginport.ErrEmbeddingFailed, err)
	}
	if len(vectors) != len(texts) {
		return pluginport.PluginEmbeddedTexts{}, fmt.Errorf("%w: %d vectors for %d texts", pluginport.ErrEmbeddingFailed, len(vectors), len(texts))
	}
	if after := s.Embedder.GetModel().Recipe(); after != recipe {
		return pluginport.PluginEmbeddedTexts{}, fmt.Errorf("%w: the model changed while the texts were embedded", pluginport.ErrRecipeMismatch)
	}
	return pluginport.PluginEmbeddedTexts{Recipe: recipe, Vectors: vectors}, nil
}

// ListNoteTitles lists the titles of the vault, at most MaxListedTitles.
func (s Search) ListNoteTitles(ctx context.Context, v domain.Vault, limit int) ([]pluginport.PluginNoteTitle, error) {
	found, err := s.Samples.ListNoteTitles(ctx, v, clampLimit(limit, pluginport.MaxListedTitles))
	if err != nil {
		return nil, err
	}
	out := make([]pluginport.PluginNoteTitle, len(found))
	for i, one := range found {
		out[i] = pluginport.PluginNoteTitle{Source: one.Source, Title: one.Text}
	}
	return out, nil
}

// ListSmallChunkVectors lists the stored vectors of the small chunks of the vault under the recipe,
// at most MaxListedVectors.
func (s Search) ListSmallChunkVectors(ctx context.Context, v domain.Vault, recipe string, limit int) ([]pluginport.PluginChunkVector, error) {
	if err := s.checkRecipe(recipe); err != nil {
		return nil, err
	}
	found, err := s.Samples.ListSmallChunkVectors(ctx, v, recipe, clampLimit(limit, pluginport.MaxListedVectors))
	if err != nil {
		return nil, err
	}
	out := make([]pluginport.PluginChunkVector, len(found))
	for i, one := range found {
		out[i] = pluginport.PluginChunkVector{Source: one.Source, Vector: one.Vector}
	}
	return out, nil
}

// GetBestSimilarity is the similarity of the passage of the vault nearest the query.
func (s Search) GetBestSimilarity(ctx context.Context, v domain.Vault, recipe string, query []float32) (float64, error) {
	if err := s.checkRecipe(recipe); err != nil {
		return 0, err
	}
	return s.Nearest.GetBestSimilarity(ctx, v.ID, recipe, query)
}

// GetSearchFloor is the floor a search applies to the vault under the recipe.
func (s Search) GetSearchFloor(ctx context.Context, v domain.Vault, recipe string) (pluginport.PluginSearchFloor, error) {
	if err := s.checkRecipe(recipe); err != nil {
		return pluginport.PluginSearchFloor{}, err
	}
	got := s.Policy.Get(ctx, v, recipe)
	return pluginport.PluginSearchFloor{
		Floor: got.Floor, Source: convertSource(got.Source),
		Stored: got.Stored, Hint: got.Hint, Default: got.Default,
	}, nil
}

// SetSearchFloor stores the floor of the vault under the recipe.
func (s Search) SetSearchFloor(ctx context.Context, v domain.Vault, recipe string, floor float64) error {
	if err := s.checkRecipe(recipe); err != nil {
		return err
	}
	if err := s.Policy.Set(ctx, v, recipe, floor); err != nil {
		if errors.Is(err, search.ErrInvalidFloor) {
			return fmt.Errorf("%w: %w", pluginport.ErrInvalidFloor, err)
		}
		return err
	}
	return nil
}

// clampLimit is the limit a list is read with: the maximum for a limit of zero or less, or above it.
func clampLimit(limit, maximum int) int {
	if limit <= 0 || limit > maximum {
		return maximum
	}
	return limit
}

func convertSource(source search.FloorSourceKind) pluginport.PluginFloorSource {
	switch source {
	case search.FloorFromStore:
		return pluginport.PluginFloorStored
	case search.FloorFromModel:
		return pluginport.PluginFloorModelHint
	default:
		return pluginport.PluginFloorDefault
	}
}
