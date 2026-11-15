package searchquery

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func (u *FindSearchResults) newRequest(ctx context.Context, v domain.Vault, q searchquery.Query, zone *time.Location) *request {
	req := &request{vault: v, query: q, limits: u.Limits, zone: zone}
	if u.hasModel() {
		req.recipe = u.source.Describe().Recipe()
		if hasMeaning(q.Conditions) {
			req.floor = u.floors.GetFloor(ctx, v, req.recipe)
		}
	}
	return req
}

func (u *FindSearchResults) run(
	ctx context.Context, req *request, docs *documentReader, warns *warnings, calls *functionCalls,
) (searchquery.SearchResults, error) {
	if err := docs.findSets(ctx); err != nil {
		return searchquery.SearchResults{}, err
	}
	if err := calls.find(ctx); err != nil {
		return searchquery.SearchResults{}, err
	}
	req.found = calls.found
	req.scopes = searchquery.GetScopes(req.query, calls.found)
	documents, err := docs.listCandidateDocuments(ctx)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	places := &placeCache{
		source: u.index.places, vaultID: req.vault.ID, found: map[domain.ChunkID]domain.SearchResult{}, document: docs.getDocument, load: docs.loadDocuments,
	}
	meaning := &meaningSearch{vectors: u.index.vectors, source: u.source, filter: places.filterScope, req: req, warnings: warns}
	vectors, err := meaning.embed(ctx)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	words := &wordSearch{words: u.index.words, filter: places.filterScope, req: req, warnings: warns}
	lists, err := findConditionCandidates(ctx, req, docs, meaning, words, vectors)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	candidates, isCut, err := places.unite(ctx, lists, req.limits.MaxCandidates)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	if isCut {
		limit := req.limits.MaxCandidates
		warns.add(fmt.Sprintf("the conditions found more than %d passages; the query was applied to the first %d", limit, limit))
	}
	if err := docs.loadDocuments(ctx, getCandidatePaths(candidates)); err != nil {
		return searchquery.SearchResults{}, err
	}
	texts, err := textReader{readers: u.readers, texts: u.texts, vault: req.vault}.readTexts(ctx, candidates, documents)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	scorer := scorer{places: u.index.places, matcher: u.matcher, words: u.index.words, req: req}
	scores, err := scorer.score(ctx, candidates, texts, vectors)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	if err := calls.match(ctx, getCandidatePaths(candidates)); err != nil {
		return searchquery.SearchResults{}, err
	}
	results := searchquery.SearchResults{
		ScoreKind: searchquery.GetScoreKind(req.query.Conditions), Floor: req.floor, Candidates: len(candidates),
	}
	results.Results = evaluator{req: req, document: docs.getDocument}.evaluate(candidates, scores, texts, documents, results.ScoreKind)
	results.Warnings = slices.Concat(warns.list, calls.warning)
	searchquery.Sort(&results)
	return results, nil
}

func findConditionCandidates(
	ctx context.Context, req *request, docs *documentReader, meaning *meaningSearch, words *wordSearch, vectors map[int][]float32,
) ([][]domain.ChunkID, error) {
	var lists [][]domain.ChunkID
	for i, c := range req.query.Conditions {
		if !c.IsPositive {
			continue
		}
		documents, isListed, err := docs.listScopeDocuments(ctx, req.scopes[i])
		if err != nil {
			return nil, err
		}
		var found []domain.ChunkID
		if c.IsMeaning {
			found, err = meaning.findMeaning(ctx, i, vectors[i], documents, isListed)
		} else {
			found, err = words.findWords(ctx, i, documents, isListed)
		}
		if err != nil {
			return nil, err
		}
		lists = append(lists, found)
	}
	return lists, nil
}

func getCandidatePaths(candidates []domain.SearchResult) []string {
	paths := make([]string, len(candidates))
	for i, c := range candidates {
		paths[i] = c.Path
	}
	return paths
}
