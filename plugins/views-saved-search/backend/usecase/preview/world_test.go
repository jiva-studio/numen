package preview_test

import (
	"context"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm/llmtest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/api"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/internal/adapter/reranker"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/usecase/preview"
)

const (
	testVault = "harbor-vault"
	recipe    = "harbor-recipe"
)

// The fake model gives a text one dimension for each topic, the count of the topic's word in it.
var topics = []string{"tide", "lighthouse", "storm", "harbor"}

func embedByTopic(text string) []float32 {
	lower := strings.ToLower(text)
	vector := make([]float32, len(topics))
	hasTopic := false
	for i, topic := range topics {
		vector[i] = float32(strings.Count(lower, topic))
		hasTopic = hasTopic || vector[i] > 0
	}
	if !hasTopic {
		return nil
	}
	return vector
}

// recordingSearch is the host search of the vault, recording the calls of the plugin. It refuses a words
// query of refusedWords as FTS5 does and leaves out of its scores the chunks the index forgot.
type recordingSearch struct {
	*hosttest.Search
	mu           sync.Mutex
	queries      []host.SearchResultQuery
	scoredChunks [][]string
	refusedWords string
	forgotten    map[string]bool
}

func (r *recordingSearch) SearchText(ctx context.Context, query host.SearchResultQuery) ([]host.SearchResult, error) {
	r.mu.Lock()
	r.queries = append(r.queries, query)
	refused := r.refusedWords != "" && query.Mode == host.SearchModeWords && query.Query == r.refusedWords
	r.mu.Unlock()
	if refused {
		return nil, host.ErrInvalidWords
	}
	return r.Search.SearchText(ctx, query)
}

func (r *recordingSearch) ScoreChunks(ctx context.Context, query host.ScoreQuery) (host.ChunkScores, error) {
	r.mu.Lock()
	r.scoredChunks = append(r.scoredChunks, slices.Clone(query.ChunkIDs))
	refused := r.refusedWords != "" && slices.ContainsFunc(query.Conditions, func(c host.ScoreCondition) bool { return c.Words == r.refusedWords })
	kept := slices.DeleteFunc(slices.Clone(query.ChunkIDs), func(id string) bool { return r.forgotten[id] })
	r.mu.Unlock()
	if refused {
		return host.ChunkScores{}, host.ErrInvalidWords
	}
	query.ChunkIDs = kept
	return r.Search.ScoreChunks(ctx, query)
}

func (r *recordingSearch) getQueries() []host.SearchResultQuery {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.queries)
}

func (r *recordingSearch) getScoredChunks() [][]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.scoredChunks)
}

// standIn is a reranking model in the process of the test: it scores a document from a fixed map and
// records each request.
type standIn struct {
	mu       sync.Mutex
	scores   map[string]float32
	count    int
	err      error
	requests []*llm.RerankRequest
}

func (s *standIn) Rerank(_ context.Context, _ connection.Call, request *llm.RerankRequest) (*llm.RerankResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.requests = append(s.requests, request)
	if s.err != nil {
		return nil, s.err
	}
	result := &llm.RerankResult{}
	for _, document := range request.GetDocuments() {
		result.Scores = append(result.Scores, s.scores[document])
	}
	if s.count > 0 {
		result.Scores = result.Scores[:s.count]
	}
	return result, nil
}

func (s *standIn) getRequests() []*llm.RerankRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.requests)
}

type world struct {
	vault   *hosttest.Vault
	search  *hosttest.Search
	host    *recordingSearch
	standIn *standIn
	caller  connection.Caller
	config  preview.Config
}

// newWorld is a vault with a model and no row for reranking.
func newWorld(t *testing.T) *world {
	t.Helper()
	search := hosttest.NewSearch(testVault)
	search.SetModel(hosttest.SearchModel{Name: "harbor-model", Recipe: recipe, FloorHint: 0.4})
	search.SetEmbedder(embedByTopic)
	return &world{
		vault:  hosttest.New(testVault),
		search: search,
		host:   &recordingSearch{Search: search.With(testVault), forgotten: map[string]bool{}},
		caller: llmtest.NewCaller(llm.Provide(nil, llm.Roles{}), connection.Call{ConnectionID: "none"}),
		config: preview.DefaultConfig(),
	}
}

// withReranker gives the plugin one row for reranking, served by the stand-in.
func (w *world) withReranker(scores map[string]float32) *standIn {
	w.standIn = &standIn{scores: scores}
	w.caller = llmtest.NewCaller(llm.Provide(nil, llm.Roles{Reranker: w.standIn}), connection.Call{ConnectionID: "harbor-reranker"})
	return w.standIn
}

func (w *world) index(results ...host.SearchResult) {
	w.search.With(testVault).SetIndexedResults(results)
}

func (w *world) execute(ctx context.Context, input api.PreviewInput) (api.PreviewOutput, error) {
	useCase := preview.New(preview.Dependencies{Search: w.host, Notes: w.vault, Reranker: reranker.New(w.caller)}, w.config)
	return useCase.Execute(ctx, input)
}

func (w *world) run(t *testing.T, input api.PreviewInput) api.PreviewOutput {
	t.Helper()
	out, err := w.execute(t.Context(), input)
	if err != nil {
		t.Fatalf("Execute(%+v) = %v", input, err)
	}
	if len(out.Errors) > 0 {
		t.Fatalf("Execute(%+v) refused the query: %+v", input, out.Errors)
	}
	return out
}

func chunk(id, path string, start, length int, text string) host.SearchResult {
	return host.SearchResult{ChunkID: id, Path: path, Start: start, Length: length, Text: text}
}

func getPaths(out api.PreviewOutput) []string {
	paths := make([]string, len(out.Results))
	for i, result := range out.Results {
		paths[i] = result.Path
	}
	return paths
}

func getTexts(out api.PreviewOutput) []string {
	texts := make([]string, len(out.Results))
	for i, result := range out.Results {
		texts[i] = result.Text
	}
	return texts
}

func isScore(score *float64, want float64) bool {
	return score != nil && *score == want
}

func hasWarning(out api.PreviewOutput, text string) bool {
	return slices.ContainsFunc(out.Warnings, func(warning string) bool { return strings.Contains(warning, text) })
}
