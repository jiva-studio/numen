// Package preview runs a saved search query over the index of one vault and returns its results, without
// saving anything.
package preview

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/query"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/ranking"
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

// Run is a run of a query and what a saved search keeps of it: for each result of Output.Results the chunk
// ids of its passages, the row that reranked, nil in fused order, and the rerank query sent.
type Run struct {
	Output   api.PreviewOutput
	ChunkIDs [][]string
	Reranker *ports.RerankerID
	Rerank   string
}

// Plan is what a run of an input uses before it searches: the limit after the default and the cut, the
// rerank query, and the problems of the query.
type Plan struct {
	Limit    int
	Rerank   string
	Problems []api.Problem
}

// Execute returns the output of Run.
func (u *UseCase) Execute(ctx context.Context, in api.PreviewInput) (api.PreviewOutput, error) {
	run, err := u.Run(ctx, in)
	return run.Output, err
}

// Run compiles the query and searches the index once for each meaning and words condition that is not
// negated, in its mode and in each folder of the filter, one search after another, for the passages of
// the numbers of Config: those of a reranked run when the reranking setting has a row. It joins the result
// lists in turns, the first result of every list, then the second, each passage once, up to MaxCandidates;
// scores every candidate against every condition in batches of host.MaxScoredChunks, dropping a candidate
// the answer leaves out; and keeps the candidates the query matches. The first of them in fused order, as
// many as Config says, go to the reranker with the whole text of their chunks and the rest are dropped,
// whether the reranker answers or not; its query is the rerank text of the input, else the texts of the
// conditions that are not negated, one per line. A reranker that fails leaves the fused order with a warning.
// Neighbors of one file are joined and the results returned best first. A query that is not valid, or a
// words condition the index rejects, returns its problems in the output with a nil error; the leaf the
// index rejects is found by scoring each words condition alone.
func (u *UseCase) Run(ctx context.Context, in api.PreviewInput) (Run, error) {
	run := Run{
		Output: api.PreviewOutput{
			Query: in.Query, Results: []api.PreviewResult{}, Order: api.OrderFused, Floor: 0, CandidateCount: 0,
			Warnings: nil, Errors: nil,
		},
		ChunkIDs: [][]string{}, Reranker: nil, Rerank: getRerankQuery(in.Rerank, nil),
	}
	if err := ctx.Err(); err != nil {
		return run, err
	}
	limit, err := u.getLimit(in.Limit)
	if err != nil {
		return run, err
	}
	compiled, problems, err := u.compile(in.Query)
	if err != nil || len(problems) > 0 {
		run.Output.Errors = problems
		return run, err
	}
	run.Rerank = getRerankQuery(in.Rerank, compiled.Leaves)
	if compiled.Filter.IsEveryKindExcluded {
		run.Output.Warnings = append(run.Output.Warnings, warningKindsExcluded)
		return run, nil
	}
	return u.run(ctx, compiled, limit, run)
}

// Plan returns what a run of the input uses, without a call to the host or the reranker.
func (u *UseCase) Plan(in api.PreviewInput) (Plan, error) {
	limit, err := u.getLimit(in.Limit)
	if err != nil {
		return Plan{Limit: 0, Rerank: "", Problems: nil}, err
	}
	compiled, problems, err := u.compile(in.Query)
	if err != nil || len(problems) > 0 {
		return Plan{Limit: limit, Rerank: getRerankQuery(in.Rerank, nil), Problems: problems}, err
	}
	return Plan{Limit: limit, Rerank: getRerankQuery(in.Rerank, compiled.Leaves), Problems: nil}, nil
}

func (u *UseCase) run(ctx context.Context, compiled query.Query, limit int, run Run) (Run, error) {
	gathered, err := u.getPool(ctx)
	if err != nil {
		return run, err
	}
	found, err := u.fetchCandidates(ctx, compiled, gathered)
	if problem, isProblem := getWordsProblem(err); isProblem {
		run.Output.Errors = []api.Problem{problem}
		return run, nil
	}
	if err != nil {
		return run, err
	}
	run.Output.CandidateCount = found.count
	run.Output.Warnings = append(run.Output.Warnings, found.warnings...)
	kept, floor, err := u.keepMatches(ctx, compiled, found.candidates)
	if problem, isProblem := getWordsProblem(err); isProblem {
		run.Output.Errors = []api.Problem{problem}
		return run, nil
	}
	if err != nil {
		return run, err
	}
	run.Output.Floor = floor
	ranking.Sort(kept)
	reranked, err := u.rerank(ctx, rerankInput{text: run.Rerank, kept: kept, limit: limit, floor: floor, gathered: gathered})
	if err != nil {
		return run, err
	}
	run.Output.Order = reranked.order
	run.Reranker = reranked.reranker
	run.Output.Warnings = append(run.Output.Warnings, reranked.warnings...)
	merged := ranking.Merge(reranked.candidates, u.config.MergeGapBytes)
	ranking.Sort(merged)
	if len(merged) > limit {
		merged = merged[:limit]
	}
	run.ChunkIDs = make([][]string, len(merged))
	for i, candidate := range merged {
		run.ChunkIDs[i] = candidate.ChunkIDs
	}
	run.Output.Results, err = u.writeResults(ctx, merged)
	return run, err
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
	if compileError, isCompileError := errors.AsType[*query.CompileError](err); isCompileError {
		problems := make([]api.Problem, len(compileError.Problems))
		for i, problem := range compileError.Problems {
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
