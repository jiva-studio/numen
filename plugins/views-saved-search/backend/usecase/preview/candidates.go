package preview

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/api"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/query"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/ranking"
)

// wordsError is a words leaf whose full-text query the index refuses.
type wordsError struct {
	text string
	term query.Term
}

func (e *wordsError) Error() string {
	return fmt.Sprintf("words(%q) is not a valid full-text query", e.text)
}

func (e *wordsError) Unwrap() error { return host.ErrInvalidWords }

func getWordsProblem(err error) (api.Problem, bool) {
	refused, isRefused := errors.AsType[*wordsError](err)
	if !isRefused {
		return api.Problem{Line: 0, Column: 0, Message: ""}, false
	}
	return api.Problem{Line: refused.term.Line, Column: refused.term.Column, Message: refused.Error()}, true
}

// candidates are the union of the results of the searches of a query.
type candidates struct {
	candidates []ranking.Candidate
	count      int
	warnings   []string
}

func (u *UseCase) fetchCandidates(ctx context.Context, compiled query.Query, gathered pool) (candidates, error) {
	prefixes := compiled.Filter.Prefixes
	if len(prefixes) == 0 {
		prefixes = []string{""}
	}
	var lists [][]host.SearchResult
	for i, leaf := range compiled.Leaves {
		if !leaf.IsPositive {
			continue
		}
		for _, prefix := range prefixes {
			if err := ctx.Err(); err != nil {
				return candidates{candidates: nil, count: 0, warnings: nil}, err
			}
			found, err := u.search.SearchText(ctx, u.newSearchQuery(leaf, compiled.Filter.Kinds, prefix, gathered))
			if err != nil {
				return candidates{candidates: nil, count: 0, warnings: nil}, convertHostError(compiled, i, err, "search the library")
			}
			lists = append(lists, found)
		}
	}
	return unite(lists, u.config.MaxCandidates), nil
}

func (u *UseCase) newSearchQuery(leaf query.Leaf, kinds []string, prefix string, gathered pool) host.SearchResultQuery {
	mode := host.SearchModeWords
	if leaf.IsMeaning {
		mode = host.SearchModeMeaning
	}
	var sourceKinds []domain.SourceKind
	for _, kind := range kinds {
		sourceKinds = append(sourceKinds, domain.SourceKind(kind))
	}
	return host.SearchResultQuery{
		Query: leaf.Text, Limit: gathered.perSearch, ResultsPerSource: gathered.perSearch,
		Kinds: sourceKinds, PathPrefix: prefix, Mode: mode, IsRawWords: !leaf.IsMeaning, IsWholeText: gathered.isReranked,
	}
}

func convertHostError(compiled query.Query, leafIndex int, err error, action string) error {
	switch {
	case errors.Is(err, host.ErrNoModel):
		return errNoModel
	case errors.Is(err, host.ErrInvalidWords) && leafIndex >= 0:
		return &wordsError{text: compiled.Leaves[leafIndex].Text, term: findLeafTerm(compiled.Root, leafIndex)}
	}
	return fmt.Errorf("%s: %w", action, err)
}

type place struct {
	path          string
	start, length int
}

// union is the passages of several result lists, each once, in the order they were added.
type union struct {
	united    candidates
	positions map[place]int
	limit     int
}

func unite(lists [][]host.SearchResult, limit int) candidates {
	u := union{united: candidates{candidates: nil, count: 0, warnings: nil}, positions: map[place]int{}, limit: limit}
	longest := 0
	for _, list := range lists {
		longest = max(longest, len(list))
	}
	for rank := range longest {
		for _, list := range lists {
			if rank < len(list) {
				u.add(list[rank])
			}
		}
	}
	if u.united.count > limit {
		u.united.warnings = append(u.united.warnings,
			fmt.Sprintf("the searches found more than %d passages; the query was applied to the first %d", limit, limit))
	}
	return u.united
}

func (u *union) add(result host.SearchResult) {
	key := place{path: result.Path, start: result.Start, length: result.Length}
	if position, isSeen := u.positions[key]; isSeen {
		if position >= 0 {
			candidate := &u.united.candidates[position]
			candidate.WordScore = getHigherScore(candidate.WordScore, result.WordScore)
		}
		return
	}
	u.united.count++
	if len(u.united.candidates) == u.limit {
		u.positions[key] = -1
		return
	}
	u.positions[key] = len(u.united.candidates)
	u.united.candidates = append(u.united.candidates, newCandidate(result))
}

func newCandidate(result host.SearchResult) ranking.Candidate {
	return ranking.Candidate{
		ChunkID: result.ChunkID, Path: result.Path, Start: result.Start, Length: result.Length,
		Text: result.Text, WholeText: result.WholeText, Selector: result.Selector, PlaceLabel: result.PlaceLabel, MatchLine: result.MatchLine,
		Score: nil, WordScore: result.WordScore, ChunkCount: 1, ChunkIDs: []string{result.ChunkID},
	}
}

func getHigherScore(a, b *float64) *float64 {
	if a == nil || (b != nil && *b > *a) {
		return b
	}
	return a
}
