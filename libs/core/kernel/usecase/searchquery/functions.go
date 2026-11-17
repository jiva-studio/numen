package searchquery

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/searchquery"
	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

const (
	// matchBatchSize is the most paths one MatchDocuments call asks about.
	matchBatchSize = 1000
	// matchCallsAtOnce is the most MatchDocuments calls of one query that wait on one plugin at a time.
	matchCallsAtOnce = 4
)

// ErrNoFunctions is a query that calls a plugin function in a core that answers none.
var ErrNoFunctions = errors.New("plugin functions are not answered here")

// functionCalls are the answers of the plugin functions of one query: every path of the calls found whole, and the
// paths asked about of the others.
type functionCalls struct {
	functions  pluginport.SearchFunctions
	query      searchquery.Query
	vaultID    string
	generation string
	limits     searchquery.Limits
	found      searchquery.FoundPaths
	sets       map[int]map[string]bool
	known      []map[string]bool
	mu         sync.Mutex
	warning    []string
}

func newFunctionCalls(functions pluginport.SearchFunctions, v domain.Vault, q searchquery.Query, limits searchquery.Limits, generation string) *functionCalls {
	known := make([]map[string]bool, len(q.Calls))
	for i := range known {
		known[i] = map[string]bool{}
	}
	return &functionCalls{
		functions: functions, query: q, vaultID: string(v.ID), generation: generation, limits: limits,
		found: searchquery.FoundPaths{}, sets: map[int]map[string]bool{}, known: known,
	}
}

func (c *functionCalls) newCall(index int) searchquery.FunctionCall {
	call := c.query.Calls[index]
	return searchquery.FunctionCall{
		PluginID: call.PluginID, Function: call.Function, Arguments: call.Arguments, VaultID: c.vaultID, IndexGeneration: c.generation,
	}
}

func (c *functionCalls) find(ctx context.Context) error {
	if len(c.query.Calls) > 0 && c.functions == nil {
		return ErrNoFunctions
	}
	for i, call := range c.query.Calls {
		if !call.IsPositive {
			continue
		}
		answer, err := c.functions.FindDocuments(ctx, c.newCall(i), c.limits.MaxDocuments+1)
		if err != nil {
			return c.getError(ctx, i, err)
		}
		c.addWarning(call.Function, answer.Warning)
		if len(answer.Paths) > c.limits.MaxDocuments {
			continue
		}
		c.found[i] = answer.Paths
		set := make(map[string]bool, len(answer.Paths))
		for _, path := range answer.Paths {
			set[path] = true
		}
		c.sets[i] = set
	}
	return nil
}

func (c *functionCalls) match(ctx context.Context, paths []string) error {
	for i := range c.query.Calls {
		if _, isFound := c.found[i]; isFound {
			continue
		}
		if err := c.matchCall(ctx, i, paths); err != nil {
			return err
		}
	}
	return nil
}

func (c *functionCalls) matchCall(ctx context.Context, index int, paths []string) error {
	known := c.known[index]
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
			answer, err := c.functions.MatchDocuments(groupCtx, c.newCall(index), batch)
			if err != nil {
				return err
			}
			c.keepMatched(index, batch, answer)
			c.addWarning(c.query.Calls[index].Function, answer.Warning)
			return nil
		})
	}
	if err := group.Wait(); err != nil {
		return c.getError(ctx, index, err)
	}
	return nil
}

func (c *functionCalls) keepMatched(index int, batch []string, answer searchquery.FunctionAnswer) {
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

func (c *functionCalls) addWarning(function, warning string) {
	if warning == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if text := function + ": " + warning; !slices.Contains(c.warning, text) {
		c.warning = append(c.warning, text)
	}
}

func (c *functionCalls) withCalls(d searchquery.Document) searchquery.Document {
	if len(c.query.Calls) == 0 {
		return d
	}
	d.MatchedCalls = make([]bool, len(c.query.Calls))
	for i := range c.query.Calls {
		if set, isFound := c.sets[i]; isFound {
			d.MatchedCalls[i] = set[d.Path]
			continue
		}
		d.MatchedCalls[i] = c.known[i][d.Path]
	}
	return d
}

func (c *functionCalls) getError(ctx context.Context, index int, err error) error {
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	call := c.query.Calls[index]
	if errors.Is(err, searchquery.ErrInvalidFunctionArguments) {
		return &searchquery.ParseError{Problems: []searchquery.Problem{{Position: call.Position, Message: err.Error()}}}
	}
	return fmt.Errorf("the function %s failed: %w", call.Function, err)
}
