// Package preview runs a saved search query over the index of one vault and returns its results, without
// saving anything.
package preview

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/api"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/query"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/ranking"
)

const (
	firstLine            = 1
	firstColumn          = 1
	warningKindsExcluded = "kind conditions exclude every kind"
)

var errNoModel = errors.New("the query uses meaning, and the installation has no embedding model; use words, or install a model")

// Dependencies are what a preview reads: the search index and the notes of one vault, and the reranking
// service.
type Dependencies struct {
	Search   ports.Search
	Notes    ports.NoteLookup
	Reranker ports.Reranker
}

// UseCase previews saved search queries over one vault.
type UseCase struct {
	search   ports.Search
	notes    ports.NoteLookup
	reranker ports.Reranker
	config   Config
}

// New creates a UseCase over the dependencies with the numbers of config.
func New(dependencies Dependencies, config Config) *UseCase {
	return &UseCase{search: dependencies.Search, notes: dependencies.Notes, reranker: dependencies.Reranker, config: config}
}

// Execute compiles the query and searches the index once for each meaning and words condition that is not
// negated, in its mode and in each folder of the filter, one search after another. It joins the result
// lists in turns, the first result of every list, then the second, each passage once, up to MaxCandidates;
// scores every candidate against every condition in batches of host.MaxScoredChunks, dropping a candidate
// the answer leaves out; and keeps the candidates the query matches. The first RerankedPerResult times the
// limit of them, at most MaxReranked, go to the reranker in fused order and the rest are dropped, whether
// the reranker answers or not; its query is the rerank text of the input, else the texts of the conditions
// that are not negated, one per line. A reranker that fails leaves the fused order with a warning.
// Neighbors of one file are joined and the results returned best first. A query that is not valid, or a
// words condition the index refuses, returns its problems in the output with a nil error; the leaf the
// index refuses is found by scoring each words condition alone.
func (u *UseCase) Execute(ctx context.Context, in api.PreviewInput) (api.PreviewOutput, error) {
	out := api.PreviewOutput{
		Query: in.Query, Results: []api.PreviewResult{}, Order: api.OrderFused, Floor: 0, CandidateCount: 0,
		Warnings: nil, Errors: nil,
	}
	if err := ctx.Err(); err != nil {
		return out, err
	}
	limit, err := u.getLimit(in.Limit)
	if err != nil {
		return out, err
	}
	compiled, problems, err := u.compile(in.Query)
	if err != nil || len(problems) > 0 {
		out.Errors = problems
		return out, err
	}
	if compiled.Filter.IsEveryKindExcluded {
		out.Warnings = append(out.Warnings, warningKindsExcluded)
		return out, nil
	}
	return u.run(ctx, in, compiled, limit, out)
}

func (u *UseCase) run(ctx context.Context, in api.PreviewInput, compiled query.Query, limit int, out api.PreviewOutput) (api.PreviewOutput, error) {
	found, err := u.fetchCandidates(ctx, compiled)
	if problem, isProblem := getWordsProblem(err); isProblem {
		out.Errors = []api.Problem{problem}
		return out, nil
	}
	if err != nil {
		return out, err
	}
	out.CandidateCount = found.count
	out.Warnings = append(out.Warnings, found.warnings...)
	kept, floor, err := u.keepMatches(ctx, compiled, found.candidates)
	if problem, isProblem := getWordsProblem(err); isProblem {
		out.Errors = []api.Problem{problem}
		return out, nil
	}
	if err != nil {
		return out, err
	}
	out.Floor = floor
	ranking.Sort(kept)
	reranked, err := u.rerank(ctx, in, compiled, kept, limit)
	if err != nil {
		return out, err
	}
	out.Order = reranked.order
	out.Warnings = append(out.Warnings, reranked.warnings...)
	merged := ranking.Merge(reranked.candidates, u.config.MergeGapBytes)
	ranking.Sort(merged)
	if len(merged) > limit {
		merged = merged[:limit]
	}
	out.Results, err = u.writeResults(ctx, merged)
	return out, err
}

func (u *UseCase) getLimit(limit int) (int, error) {
	switch {
	case limit < 0:
		return 0, fmt.Errorf("limit is %d: give a number of zero or more", limit)
	case limit == 0:
		return min(u.config.DefaultLimit, u.config.MaxLimit), nil
	}
	return min(limit, u.config.MaxLimit), nil
}

func (u *UseCase) compile(text string) (query.Query, []api.Problem, error) {
	if len(text) > u.config.MaxQueryBytes {
		message := fmt.Sprintf("a query has at most %d bytes", u.config.MaxQueryBytes)
		return query.Query{}, []api.Problem{{Line: firstLine, Column: firstColumn, Message: message}}, nil
	}
	compiled, err := query.Compile(text)
	if refused, isRefused := errors.AsType[*query.CompileError](err); isRefused {
		problems := make([]api.Problem, len(refused.Problems))
		for i, problem := range refused.Problems {
			problems[i] = api.Problem{Line: problem.Line, Column: problem.Column, Message: problem.Message}
		}
		return query.Query{}, problems, nil
	}
	if err != nil {
		return query.Query{}, nil, err
	}
	if prefixes := compiled.Filter.Prefixes; len(prefixes) > u.config.MaxPathPrefixes {
		term := findPathTerm(compiled.Root, prefixes[u.config.MaxPathPrefixes])
		message := fmt.Sprintf("a query searches at most %d folders: join at most %d path.startsWith conditions with ||",
			u.config.MaxPathPrefixes, u.config.MaxPathPrefixes)
		return query.Query{}, []api.Problem{{Line: term.Line, Column: term.Column, Message: message}}, nil
	}
	return compiled, nil, nil
}

func findPathTerm(node query.Node, prefix string) query.Term {
	return findTerm(node, func(term query.Term) bool { return term.Kind == query.TermPath && term.Prefix == prefix })
}

func findLeafTerm(node query.Node, leafIndex int) query.Term {
	return findTerm(node, func(term query.Term) bool {
		return (term.Kind == query.TermMeaning || term.Kind == query.TermWords) && term.LeafIndex == leafIndex
	})
}

func findTerm(node query.Node, isWanted func(query.Term) bool) query.Term {
	if term, isFound := searchTerm(node, isWanted); isFound {
		return term
	}
	return query.Term{Kind: query.TermWords, LeafIndex: 0, Kinds: nil, Prefix: "", Line: firstLine, Column: firstColumn}
}

func searchTerm(node query.Node, isWanted func(query.Term) bool) (query.Term, bool) {
	if node.Operator == query.OperatorLeaf {
		return node.Term, isWanted(node.Term)
	}
	for _, child := range node.Children {
		if term, isFound := searchTerm(child, isWanted); isFound {
			return term, true
		}
	}
	return node.Term, false
}
