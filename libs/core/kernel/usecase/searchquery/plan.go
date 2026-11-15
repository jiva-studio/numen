package searchquery

import (
	"context"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// plan is one query being evaluated over the index of one vault.
type plan struct {
	u        *FindSearchResults
	v        domain.Vault
	query    searchquery.Query
	scopes   []searchquery.Scope
	recipe   string
	floor    float64
	vectors  map[int][]float32
	places   map[domain.ChunkID]domain.SearchResult
	sets     []map[string]bool
	warnings []string
}

func (u *FindSearchResults) newPlan(ctx context.Context, v domain.Vault, q searchquery.Query) *plan {
	p := &plan{u: u, v: v, query: q, scopes: searchquery.GetScopes(q), places: map[domain.ChunkID]domain.SearchResult{}}
	if u.hasModel() {
		p.recipe = u.source.Describe().Recipe()
		if hasMeaning(q.Conditions) {
			p.floor = u.floors.GetFloor(ctx, v, p.recipe)
		}
	}
	return p
}

func (p *plan) execute(ctx context.Context) (searchquery.SearchResults, error) {
	if err := p.computeSets(ctx); err != nil {
		return searchquery.SearchResults{}, err
	}
	documents, err := p.listCandidateDocuments(ctx)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	if err := p.embed(ctx); err != nil {
		return searchquery.SearchResults{}, err
	}
	lists, err := p.findConditionCandidates(ctx)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	candidates, err := p.unite(ctx, lists)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	texts, err := p.readTexts(ctx, candidates, documents)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	scores, err := p.score(ctx, candidates, texts)
	if err != nil {
		return searchquery.SearchResults{}, err
	}
	results := searchquery.SearchResults{
		ScoreKind: searchquery.GetScoreKind(p.query.Conditions), Floor: p.floor, Candidates: len(candidates),
	}
	results.Results = p.evaluate(candidates, scores, texts, documents, results.ScoreKind)
	results.Warnings = p.warnings
	searchquery.Sort(&results)
	return results, nil
}

func (p *plan) findConditionCandidates(ctx context.Context) ([][]domain.ChunkID, error) {
	var lists [][]domain.ChunkID
	for i, c := range p.query.Conditions {
		if !c.IsPositive {
			continue
		}
		documents, isListed, err := p.listScopeDocuments(ctx, p.scopes[i])
		if err != nil {
			return nil, err
		}
		var found []domain.ChunkID
		if c.IsMeaning {
			found, err = p.findMeaning(ctx, i, documents, isListed)
		} else {
			found, err = p.findWords(ctx, i, documents, isListed)
		}
		if err != nil {
			return nil, err
		}
		lists = append(lists, found)
	}
	return lists, nil
}

func (p *plan) evaluate(
	candidates []domain.SearchResult, scores [][]searchquery.ConditionScore, texts map[string]fileText,
	documents []searchquery.Document, kind searchquery.ScoreKind,
) []searchquery.SearchResult {
	var matched []searchquery.SearchResult
	for i, c := range candidates {
		d := p.markSets(p.u.getDocument(c.Path, c.Kind))
		o := searchquery.Evaluate(p.query.Root, p.query.Conditions, scores[i], p.floor, d)
		if !o.IsMatch {
			continue
		}
		matched = append(matched, newChunkResult(d, c, o, texts[c.Path]))
	}
	searchquery.Fuse(matched, kind)
	results := p.mergeFiles(matched)
	hasChunks := map[string]bool{}
	for _, r := range results {
		hasChunks[r.Document.Path] = true
	}
	for _, d := range documents {
		if !hasChunks[d.Path] {
			results = append(results, searchquery.SearchResult{Document: d, Text: getExcerpt(texts[d.Path].prose, 0, len(texts[d.Path].prose))})
		}
	}
	return results
}

func (p *plan) mergeFiles(matched []searchquery.SearchResult) []searchquery.SearchResult {
	byPath := map[string][]searchquery.SearchResult{}
	var paths []string
	for _, r := range matched {
		if _, found := byPath[r.Document.Path]; !found {
			paths = append(paths, r.Document.Path)
		}
		byPath[r.Document.Path] = append(byPath[r.Document.Path], r)
	}
	slices.Sort(paths)
	var merged []searchquery.SearchResult
	for _, path := range paths {
		merged = append(merged, searchquery.Merge(byPath[path], p.u.Limits.MergeGapBytes)...)
	}
	return merged
}

func newChunkResult(d searchquery.Document, c domain.SearchResult, o searchquery.Outcome, text fileText) searchquery.SearchResult {
	r := searchquery.SearchResult{
		Document: d, Chunks: []searchquery.ChunkRange{{ID: c.ChunkID, Start: c.Start, Length: c.Length}},
		Start: c.Start, Length: c.Length, PlaceLabel: c.PlaceLabel,
		MeaningScore: o.MeaningScore, WordScore: o.WordScore,
	}
	if text.isRead {
		r.Text = getExcerpt(text.prose, c.Start, c.Length)
		r.MatchLine = getLine(text.prose, c.Start+c.MatchOffset)
		if text.version != nil {
			r.Selector = text.version.GetSelector(c.Start + c.MatchOffset)
		}
	}
	return r
}
