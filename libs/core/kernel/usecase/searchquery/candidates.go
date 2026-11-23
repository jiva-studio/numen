package searchquery

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

type placeKey struct {
	path          string
	start, length int
}

type scopeFilter func(ctx context.Context, s searchquery.Scope, ids []domain.ChunkID) (map[domain.ChunkID]bool, error)

type placeCache struct {
	source      ChunkPlaces
	vaultID     domain.VaultID
	found       map[domain.ChunkID]domain.SearchResult
	getDocument func(path string, sourceKind domain.SourceKind) searchquery.Document
	load        func(ctx context.Context, paths []string) error
}

func (c *placeCache) getPlaces(ctx context.Context, ids []domain.ChunkID) (map[domain.ChunkID]domain.SearchResult, error) {
	var missing []domain.ChunkID
	for _, id := range ids {
		if _, found := c.found[id]; !found {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		found, err := c.source.GetEnclosingResults(ctx, c.vaultID, missing)
		if err != nil {
			return nil, err
		}
		for _, place := range found {
			c.found[place.ChunkID] = place
		}
	}
	return c.found, nil
}

func (c *placeCache) filterScope(ctx context.Context, s searchquery.Scope, ids []domain.ChunkID) (map[domain.ChunkID]bool, error) {
	places, err := c.getPlaces(ctx, ids)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, id := range ids {
		if place, found := places[id]; found {
			paths = append(paths, place.Path)
		}
	}
	if err := c.load(ctx, paths); err != nil {
		return nil, err
	}
	inScope := make(map[domain.ChunkID]bool, len(ids))
	for _, id := range ids {
		place, found := places[id]
		inScope[id] = found && s.IsDocumentMatch(c.getDocument(place.Path, place.Kind))
	}
	return inScope, nil
}

func (c *placeCache) mergeLists(ctx context.Context, lists [][]domain.ChunkID, limit int) ([]domain.SearchResult, bool, error) {
	var all []domain.ChunkID
	for _, list := range lists {
		all = append(all, list...)
	}
	places, err := c.getPlaces(ctx, all)
	if err != nil {
		return nil, false, err
	}
	seen := map[placeKey]bool{}
	var united []domain.SearchResult
	for turn := 0; hasTurn(lists, turn); turn++ {
		for _, list := range lists {
			if turn >= len(list) {
				continue
			}
			place, found := places[list[turn]]
			key := placeKey{path: place.Path, start: place.Start, length: place.Length}
			if !found || seen[key] {
				continue
			}
			seen[key] = true
			united = append(united, place)
		}
	}
	if len(united) > limit {
		return united[:limit], true, nil
	}
	return united, false, nil
}

func hasTurn(lists [][]domain.ChunkID, turn int) bool {
	for _, list := range lists {
		if turn < len(list) {
			return true
		}
	}
	return false
}
