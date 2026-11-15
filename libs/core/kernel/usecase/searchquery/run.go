package searchquery

import (
	"context"
	"fmt"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/note/domain"
)

func (u *EvaluateQuery) newRequest(
	ctx context.Context, v domain.Vault, q searchquery.Query, zone *time.Location, limits searchquery.Limits,
) *searchquery.Evaluation {
	req := &searchquery.Evaluation{Vault: v, Query: q, Scopes: searchquery.GetScopes(q), Limits: limits, Zone: zone}
	if u.hasModel() {
		req.Recipe = u.source.Describe().Recipe()
		if hasMeaning(q.Conditions) {
			req.Floor = u.floors.GetFloor(ctx, v, req.Recipe)
		}
	}
	return req
}

func (u *EvaluateQuery) run(ctx context.Context, req *searchquery.Evaluation, docs *documentReader, warns *warnings) (searchquery.SearchResults, error) {
	if err := docs.findSets(ctx); err != nil {
		return searchquery.SearchResults{}, err
	}
	documents, err := docs.listCandidateDocuments(ctx)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	places := &placeCache{
		source: u.chunks, vaultID: req.Vault.ID, found: map[domain.ChunkID]domain.SearchResult{}, document: docs.getDocument, load: docs.loadDocuments,
	}
	meaning := &meaningSearch{vectors: u.chunks, source: u.source, filter: places.filterScope, req: req, warnings: warns}
	vectors, err := meaning.embed(ctx)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	words := &wordSearch{words: u.chunks, filter: places.filterScope, req: req, warnings: warns}
	lists, err := findConditionCandidates(ctx, req, docs, meaning, words, vectors)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	candidates, isCut, err := places.unite(ctx, lists, req.Limits.MaxCandidates)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	if isCut {
		limit := req.Limits.MaxCandidates
		warns.add(fmt.Sprintf("the conditions found more than %d passages; the query was applied to the first %d", limit, limit))
	}
	if err := docs.loadDocuments(ctx, getCandidatePaths(candidates)); err != nil {
		return searchquery.SearchResults{}, err
	}
	texts, err := textReader{readers: u.readers, texts: u.texts, vault: req.Vault}.readTexts(ctx, candidates, documents)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	scorer := scorer{places: u.chunks, matcher: u.matcher, words: u.chunks, req: req}
	scores, err := scorer.score(ctx, candidates, texts, vectors)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	results := searchquery.SearchResults{
		ScoreKind: searchquery.GetScoreKind(req.Query.Conditions), Floor: req.Floor, Candidates: len(candidates),
	}
	results.Results = evaluator{req: req, document: docs.getDocument}.evaluate(candidates, scores, texts, documents, results.ScoreKind)
	results.Warnings = warns.list
	searchquery.Sort(&results)
	return results, nil
}

func findConditionCandidates(
	ctx context.Context, req *searchquery.Evaluation, docs *documentReader, meaning *meaningSearch, words *wordSearch, vectors map[int][]float32,
) ([][]domain.ChunkID, error) {
	var lists [][]domain.ChunkID
	for i, c := range req.Query.Conditions {
		if !c.IsPositive {
			continue
		}
		documents, isListed, err := docs.listScopeDocuments(ctx, req.Scopes[i])
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
