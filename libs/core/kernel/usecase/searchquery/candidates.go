package searchquery

import (
	"context"
	"fmt"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/note/domain"
)

type placeKey struct {
	path          string
	start, length int
}

func (p *plan) getPlaces(ctx context.Context, ids []domain.ChunkID) (map[domain.ChunkID]domain.SearchResult, error) {
	var missing []domain.ChunkID
	for _, id := range ids {
		if _, found := p.places[id]; !found {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		found, err := p.u.places.GetEnclosingResults(ctx, p.v.ID, missing)
		if err != nil {
			return nil, err
		}
		for _, place := range found {
			p.places[place.ChunkID] = place
		}
	}
	return p.places, nil
}

func (p *plan) filterScope(ctx context.Context, s searchquery.Scope, ids []domain.ChunkID) (map[domain.ChunkID]bool, error) {
	places, err := p.getPlaces(ctx, ids)
	if err != nil {
		return nil, err
	}
	inScope := make(map[domain.ChunkID]bool, len(ids))
	for _, id := range ids {
		place, found := places[id]
		inScope[id] = found && s.IsDocumentMatch(p.markSets(p.u.getDocument(place.Path, place.Kind)))
	}
	return inScope, nil
}

func (p *plan) unite(ctx context.Context, lists [][]domain.ChunkID) ([]domain.SearchResult, error) {
	var all []domain.ChunkID
	for _, list := range lists {
		all = append(all, list...)
	}
	places, err := p.getPlaces(ctx, all)
	if err != nil {
		return nil, err
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
	if limit := p.u.Limits.MaxCandidates; len(united) > limit {
		p.warnings = append(p.warnings, fmt.Sprintf("the conditions found more than %d passages; the query was applied to the first %d", limit, limit))
		united = united[:limit]
	}
	return united, nil
}

func hasTurn(lists [][]domain.ChunkID, turn int) bool {
	for _, list := range lists {
		if turn < len(list) {
			return true
		}
	}
	return false
}
