package searchquery

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
)

const (
	// matchBatchSize is the most paths one MatchDocuments call asks about.
	matchBatchSize = 1000
	// matchCallsAtOnce is the most MatchDocuments calls of one query that wait on one plugin at a time.
	matchCallsAtOnce = 4
)

// ErrNoFunctions is a query that calls a plugin function in a core that answers none.
var ErrNoFunctions = errors.New("plugin functions are not answered here")

// calls are the answers of the plugin functions of one query: every path of the calls found whole, and the
// paths asked about of the others.
type calls struct {
	found   searchquery.FoundPaths
	sets    map[int]map[string]bool
	known   []map[string]bool
	mu      sync.Mutex
	warning []string
}

func (p *plan) newCall(index int) searchquery.FunctionCall {
	c := p.query.Calls[index]
	return searchquery.FunctionCall{
		PluginID: c.PluginID, Function: c.Function, Arguments: c.Arguments, VaultID: string(p.v.ID), IndexGeneration: p.generation,
	}
}

// findCalls asks the plugin of each positive call for its documents, and keeps every path of a call that
// holds for at most MaxDocuments of them.
func (p *plan) findCalls(ctx context.Context) error {
	p.calls = &calls{found: searchquery.FoundPaths{}, sets: map[int]map[string]bool{}, known: make([]map[string]bool, len(p.query.Calls))}
	for i := range p.query.Calls {
		p.calls.known[i] = map[string]bool{}
	}
	if len(p.query.Calls) > 0 && p.u.functions == nil {
		return ErrNoFunctions
	}
	for i, c := range p.query.Calls {
		if !c.IsPositive {
			continue
		}
		answer, err := p.u.functions.FindDocuments(ctx, p.newCall(i), p.u.Limits.MaxDocuments+1)
		if err != nil {
			return p.getCallError(ctx, i, err)
		}
		p.calls.addWarning(c.Function, answer.Warning)
		if len(answer.Paths) > p.u.Limits.MaxDocuments {
			continue
		}
		p.calls.found[i] = answer.Paths
		set := make(map[string]bool, len(answer.Paths))
		for _, path := range answer.Paths {
			set[path] = true
		}
		p.calls.sets[i] = set
	}
	return nil
}

// matchCalls asks the plugins of the calls not found whole about the paths not asked about yet.
func (p *plan) matchCalls(ctx context.Context, paths []string) error {
	for i := range p.query.Calls {
		if _, isFound := p.calls.found[i]; isFound {
			continue
		}
		if err := p.matchCall(ctx, i, paths); err != nil {
			return err
		}
	}
	return nil
}

func (p *plan) matchCall(ctx context.Context, index int, paths []string) error {
	known := p.calls.known[index]
	var asked []string
	isAsked := map[string]bool{}
	for _, path := range paths {
		if _, isKnown := known[path]; !isKnown && !isAsked[path] {
			asked, isAsked[path] = append(asked, path), true
		}
	}
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(matchCallsAtOnce)
	for batch := range slices.Chunk(asked, matchBatchSize) {
		group.Go(func() error {
			answer, err := p.u.functions.MatchDocuments(groupCtx, p.newCall(index), batch)
			if err != nil {
				return err
			}
			p.calls.keepMatched(index, batch, answer)
			p.calls.addWarning(p.query.Calls[index].Function, answer.Warning)
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return p.getCallError(ctx, index, err)
	}
	return nil
}

func (c *calls) keepMatched(index int, batch []string, answer searchquery.FunctionAnswer) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, path := range batch {
		c.known[index][path] = false
	}
	for _, path := range answer.Paths {
		if _, isAsked := c.known[index][path]; isAsked {
			c.known[index][path] = true
		}
	}
}

func (c *calls) addWarning(function, warning string) {
	if warning == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if text := function + ": " + warning; !slices.Contains(c.warning, text) {
		c.warning = append(c.warning, text)
	}
}

// withCalls returns the document with whether each call holds for it.
func (p *plan) withCalls(d searchquery.Document) searchquery.Document {
	if len(p.query.Calls) == 0 {
		return d
	}
	d.MatchedCalls = make([]bool, len(p.query.Calls))
	for i := range p.query.Calls {
		if set, isFound := p.calls.sets[i]; isFound {
			d.MatchedCalls[i] = set[d.Path]
			continue
		}
		d.MatchedCalls[i] = p.calls.known[i][d.Path]
	}
	return d
}

func (p *plan) getCallError(ctx context.Context, index int, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	c := p.query.Calls[index]
	if errors.Is(err, searchquery.ErrInvalidFunctionArguments) {
		return &searchquery.ParseError{Problems: []searchquery.Problem{{Position: c.Position, Message: err.Error()}}}
	}
	return fmt.Errorf("the function %s failed: %w", c.Function, err)
}
