// Package searchtest is the searches of the query language in memory, for the tests of a plugin that runs them.
package searchtest

import (
	"context"
	"slices"
	"strconv"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
)

// Searches is the searches of the query language in memory, as the host keeps them for one plugin. A query
// answers the results set for it at once, or, when held, once it is released; a read waits for it up to
// its Wait. Pages are cut as the host cuts them, with a page token of the offset.
type Searches struct {
	mu        sync.Mutex
	functions host.SearchFunctions
	listErr   error
	answers   map[string]host.SearchState
	held      map[string]chan struct{}
	searches  map[string]string
	removed   []string
	started   []host.SearchStart
}

// NewSearches returns searches with no query answered and an empty language.
func NewSearches() *Searches {
	return &Searches{answers: map[string]host.SearchState{}, held: map[string]chan struct{}{}, searches: map[string]string{}}
}

// SetFunctions sets the language ListSearchFunctions returns, and err its failure when not nil.
func (s *Searches) SetFunctions(functions host.SearchFunctions, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.functions, s.listErr = functions, err
}

// SetResults makes every search of the query text end with the results.
func (s *Searches) SetResults(query string, results []host.SearchResult) {
	s.SetAnswer(query, host.SearchState{Results: results})
}

// SetAnswer makes every search of the query text end as the state says: its results cut into pages, and
// its errors, warnings, reranks, score kind, floor, index generation and IsStale as they are.
func (s *Searches) SetAnswer(query string, state host.SearchState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.answers[query] = state
}

// HoldSearch keeps the searches of a query running until the returned function is called.
func (s *Searches) HoldSearch(query string) func() {
	s.mu.Lock()
	defer s.mu.Unlock()
	release := make(chan struct{})
	s.held[query] = release
	return sync.OnceFunc(func() { close(release) })
}

// ListRemoved returns the ids of the searches removed, in order.
func (s *Searches) ListRemoved() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.removed)
}

// ListStarted returns the starts received, in order.
func (s *Searches) ListStarted() []host.SearchStart {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.started)
}

func (s *Searches) StartSearch(_ context.Context, start host.SearchStart) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := "search-" + strconv.Itoa(len(s.started)+1)
	s.started = append(s.started, start)
	s.searches[id] = start.Query
	return id, nil
}

func (s *Searches) GetSearch(ctx context.Context, id string, read host.SearchRead) (host.SearchState, error) {
	s.mu.Lock()
	query, isKnown := s.searches[id]
	held := s.held[query]
	s.mu.Unlock()
	if !isKnown {
		return host.SearchState{}, host.ErrSearchNotFound
	}
	if held != nil && !s.isReleasedWithin(ctx, held, read.Wait) {
		if err := ctx.Err(); err != nil {
			return host.SearchState{}, err
		}
		return host.SearchState{ID: id, IsRunning: true, Step: "candidates", Results: []host.SearchResult{}}, nil
	}
	return s.cutPage(id, query, read)
}

func (s *Searches) isReleasedWithin(ctx context.Context, held chan struct{}, wait time.Duration) bool {
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-held:
		return true
	case <-timer.C:
	case <-ctx.Done():
	}
	select {
	case <-held:
		return true
	default:
		return false
	}
}

func (s *Searches) cutPage(id, query string, read host.SearchRead) (host.SearchState, error) {
	s.mu.Lock()
	answer := s.answers[query]
	s.mu.Unlock()
	results := answer.Results
	offset := 0
	if read.PageToken != "" {
		parsed, err := strconv.Atoi(read.PageToken)
		if err != nil || parsed < 0 || parsed > len(results) {
			return host.SearchState{}, host.ErrInvalidPageToken
		}
		offset = parsed
	}
	size := read.PageSize
	if size <= 0 {
		size = 20
	}
	end := min(offset+size, len(results))
	state := answer
	state.ID, state.IsRunning, state.Step, state.NextPageToken = id, false, "done", ""
	state.Results, state.TotalResults = slices.Clone(results[offset:end]), len(results)
	if state.Results == nil {
		state.Results = []host.SearchResult{}
	}
	if end < len(results) {
		state.NextPageToken = strconv.Itoa(end)
	}
	return state, nil
}

func (s *Searches) DeleteSearch(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, isKnown := s.searches[id]; !isKnown {
		return host.ErrSearchNotFound
	}
	delete(s.searches, id)
	s.removed = append(s.removed, id)
	return nil
}

func (s *Searches) ListSearchFunctions(context.Context) (host.SearchFunctions, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.functions, s.listErr
}
