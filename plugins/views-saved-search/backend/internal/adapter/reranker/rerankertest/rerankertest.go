// Package rerankertest holds the rows of the connections setting and the reranking models of the tests of
// the saved search: the rows a test sets, each served in the process of the test by a stand-in model.
package rerankertest

import (
	"context"
	"maps"
	"slices"
	"sync"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm/llmtest"
)

// Request is one request a stand-in received: the query, the documents and the options of the row.
type Request struct {
	Query     string
	Documents []string
	Options   map[string]string
}

// StandIn is a reranking model in the process of the test. It scores a document from its map, 0 for a
// document the map does not hold, and records each request.
type StandIn struct {
	mu       sync.Mutex
	scores   map[string]float32
	err      error
	requests []Request
}

var _ llm.Reranker = (*StandIn)(nil)

// NewStandIn returns a model that scores documents from scores.
func NewStandIn(scores map[string]float32) *StandIn {
	return &StandIn{mu: sync.Mutex{}, scores: maps.Clone(scores), err: nil, requests: nil}
}

// SetScores replaces the scores of the model.
func (s *StandIn) SetScores(scores map[string]float32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.scores = maps.Clone(scores)
}

// Fail makes every request fail with err, until it is called with nil.
func (s *StandIn) Fail(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.err = err
}

func (s *StandIn) Rerank(_ context.Context, call connection.Call, request *llm.RerankRequest) (*llm.RerankResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, Request{
		Query: request.GetQuery(), Documents: slices.Clone(request.GetDocuments()), Options: maps.Clone(call.Options),
	})
	if s.err != nil {
		return nil, s.err
	}
	result := &llm.RerankResult{}
	for _, document := range request.GetDocuments() {
		result.Scores = append(result.Scores, s.scores[document])
	}
	return result, nil
}

// GetRequests returns the requests the model received, in order.
func (s *StandIn) GetRequests() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.requests)
}

// GetDocumentCount returns how many documents the model received over all its requests.
func (s *StandIn) GetDocumentCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, request := range s.requests {
		count += len(request.Documents)
	}
	return count
}

// ClearRequests forgets the requests received so far.
func (s *StandIn) ClearRequests() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = nil
}

// Rows is the rows of the plugin's connections setting for llm.rerank, as the host lists them, and the
// models that serve each connection.
type Rows struct {
	mu        sync.Mutex
	rows      []connection.Row
	models    map[string]llm.Reranker
	operation []string
	preferred []connection.PreferredRow
}

// NewRows returns a setting with no row.
func NewRows() *Rows {
	return &Rows{mu: sync.Mutex{}, rows: nil, models: map[string]llm.Reranker{}, operation: nil, preferred: nil}
}

// Add appends a row of the connection, with the options of its row fields, served by model.
func (r *Rows) Add(connectionID, connectionName string, options map[string]string, model llm.Reranker) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = append(r.rows, connection.Row{ConnectionID: connectionID, ConnectionName: connectionName, Options: maps.Clone(options)})
	r.models[connectionID] = model
}

// SetOptions replaces the options of the row of the connection, as the person choosing another model does.
func (r *Rows) SetOptions(connectionID string, options map[string]string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.rows {
		if r.rows[i].ConnectionID == connectionID {
			r.rows[i].Options = maps.Clone(options)
		}
	}
}

// Clear removes every row.
func (r *Rows) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rows = nil
}

func (r *Rows) ListRows(ctx context.Context, operation string) ([]connection.Row, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.operation = append(r.operation, operation)
	rows := make([]connection.Row, len(r.rows))
	for i, row := range r.rows {
		row.Options = maps.Clone(row.Options)
		rows[i] = row
	}
	return rows, nil
}

func (r *Rows) Prefer(row connection.PreferredRow) connection.Caller {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.preferred = append(r.preferred, row)
	roles := llm.Roles{}
	if model, isKnown := r.models[row.ConnectionID]; isKnown {
		roles.Reranker = model
	}
	return llmtest.NewCaller(llm.Provide(nil, roles), connection.Call{ConnectionID: row.ConnectionID, Options: maps.Clone(row.Options)})
}

// GetOperations returns the operations ListRows was asked for, in order.
func (r *Rows) GetOperations() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.operation)
}

// GetPreferredRows returns the rows Prefer was given, in order.
func (r *Rows) GetPreferredRows() []connection.PreferredRow {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.preferred)
}
