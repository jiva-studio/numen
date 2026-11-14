package service

import (
	"encoding/json"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/contracts/llm"
	"github.com/jiva-studio/numen/plugins/connections-llm/backend/fakeopenai"
)

type rerankBody struct {
	Model     string   `json:"model"`
	Query     string   `json:"query"`
	Documents []string `json:"documents"`
	TopN      int      `json:"top_n"`
}

// recorder is a service that answers every request with a fixed body and keeps the path and the body of
// each request.
type recorder struct {
	mu     sync.Mutex
	paths  []string
	bodies [][]byte
}

func startRecorder(t *testing.T, answer string) (*recorder, string) {
	t.Helper()
	got := &recorder{}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		got.mu.Lock()
		got.paths, got.bodies = append(got.paths, r.URL.Path), append(got.bodies, body)
		got.mu.Unlock()
		if _, err := w.Write([]byte(answer)); err != nil {
			t.Error(err)
		}
	}))
	t.Cleanup(ts.Close)
	return got, ts.URL
}

func (r *recorder) getPaths() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.paths)
}

func (r *recorder) readBody(t *testing.T, i int) rerankBody {
	t.Helper()
	r.mu.Lock()
	defer r.mu.Unlock()
	var body rerankBody
	if err := json.Unmarshal(r.bodies[i], &body); err != nil {
		t.Fatalf("body %s: %v", r.bodies[i], err)
	}
	return body
}

func TestRerankReturnsScoresInDocumentOrder(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{APIKey: testKey})
	documents := []string{"Tide tables", "Ferry times", "Harbor map", "Lighthouse hours", "Ferry fares"}
	query := "when does the ferry leave?"

	result, err := newTestService(t).Rerank(newContext(t), newCall("openrouter", address, testKey, "cohere/rerank-v3.5"),
		&llm.RerankRequest{Query: query, Documents: documents})
	if err != nil {
		t.Fatal(err)
	}

	want := make([]float32, len(documents))
	for i, document := range documents {
		want[i] = fake.Score(query, document)
	}
	if !slices.Equal(result.GetScores(), want) {
		t.Errorf("scores = %v, want %v in the order of the documents", result.GetScores(), want)
	}
}

func TestRerankPostsModelQueryDocumentsAndTopNWithTheKey(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{APIKey: testKey})
	documents := []string{"Tide tables", "Ferry times", "Harbor map"}

	if _, err := newTestService(t).Rerank(newContext(t), newCall("openrouter", address, testKey, "cohere/rerank-v3.5"),
		&llm.RerankRequest{Query: "ferry", Documents: documents}); err != nil {
		t.Fatal(err)
	}

	requests := fake.Requests()
	if len(requests) != 1 || requests[0].Method != http.MethodPost || requests[0].Path != "/rerank" ||
		requests[0].Header.Get("Authorization") != "Bearer "+testKey {
		t.Fatalf("requests = %+v", requests)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(requests[0].Body, &fields); err != nil {
		t.Fatal(err)
	}
	if keys := slices.Sorted(maps.Keys(fields)); !slices.Equal(keys, []string{"documents", "model", "query", "top_n"}) {
		t.Errorf("body = %s, want model, query, documents and top_n alone", requests[0].Body)
	}
	var body rerankBody
	if err := json.Unmarshal(requests[0].Body, &body); err != nil {
		t.Fatal(err)
	}
	if body.Model != "cohere/rerank-v3.5" || body.Query != "ferry" || !slices.Equal(body.Documents, documents) || body.TopN != len(documents) {
		t.Errorf("body = %s", requests[0].Body)
	}
}

func TestRerankPlacesEachScoreAtTheIndexOfItsDocument(t *testing.T) {
	answer := `{"results":[{"index":2,"relevance_score":0.9},{"index":0,"relevance_score":0.5},{"index":1,"relevance_score":0.1}]}`
	_, address := startRecorder(t, answer)

	result, err := newTestService(t).Rerank(newContext(t), newCall("openai-compatible", address, "", "bge-reranker-v2-m3"),
		&llm.RerankRequest{Query: "ferry", Documents: []string{"a", "b", "c"}})

	if err != nil || !slices.Equal(result.GetScores(), []float32{0.5, 0.1, 0.9}) {
		t.Errorf("Rerank = %v, %v, want the scores in the order of the documents", result, err)
	}
}

func TestRerankPostsToTheRerankPathOfTheAddress(t *testing.T) {
	answer := `{"results":[{"index":0,"relevance_score":0.5}]}`
	for typeID, path := range map[string]string{"openai-compatible": "/v1", "openrouter": "/api/v1"} {
		got, address := startRecorder(t, answer)

		_, err := newTestService(t).Rerank(newContext(t), newCall(typeID, address+path, testKey, "m"),
			&llm.RerankRequest{Query: "ferry", Documents: []string{"a"}})

		if err != nil || !slices.Equal(got.getPaths(), []string{path + "/rerank"}) {
			t.Errorf("%s: err = %v, paths = %v, want %s/rerank", typeID, err, got.getPaths(), path)
		}
	}
}

func TestRerankPutsTheInstructionAndALineBreakBeforeTheQuery(t *testing.T) {
	answer := `{"results":[{"index":0,"relevance_score":0.5}]}`
	got, address := startRecorder(t, answer)
	svc := newTestService(t)
	call := newCall("openai-compatible", address+"/v1", "", "m")

	if _, err := svc.Rerank(newContext(t), call, &llm.RerankRequest{
		Instruction: "Find the notes that answer the question", Query: "when does the ferry leave?", Documents: []string{"a"},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Rerank(newContext(t), call, &llm.RerankRequest{Query: " when does the ferry leave? ", Documents: []string{"a"}}); err != nil {
		t.Fatal(err)
	}

	if query := got.readBody(t, 0).Query; query != "Find the notes that answer the question\nwhen does the ferry leave?" {
		t.Errorf("query with an instruction = %q", query)
	}
	if query := got.readBody(t, 1).Query; query != " when does the ferry leave? " {
		t.Errorf("query without an instruction = %q, want it unchanged", query)
	}
}

func TestRerankWithNoDocumentsSendsNothing(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})

	result, err := newTestService(t).Rerank(newContext(t), newCall("openai-compatible", address, "", "m"), &llm.RerankRequest{Query: "ferry"})

	if err != nil || len(result.GetScores()) != 0 {
		t.Errorf("Rerank = %v, %v", result, err)
	}
	if len(fake.Requests()) != 0 {
		t.Errorf("the service received %d requests", len(fake.Requests()))
	}
}

func TestRerankStopsOnAnAnswerThatDoesNotMatchTheDocuments(t *testing.T) {
	bodies := map[string]string{
		"fewer results than documents": `{"results":[{"index":0,"relevance_score":0.5}]}`,
		"a repeated index":             `{"results":[{"index":0,"relevance_score":0.5},{"index":0,"relevance_score":0.4}]}`,
		"an index out of range":        `{"results":[{"index":0,"relevance_score":0.5},{"index":2,"relevance_score":0.4}]}`,
		"a negative index":             `{"results":[{"index":0,"relevance_score":0.5},{"index":-1,"relevance_score":0.4}]}`,
		"a missing index":              `{"results":[{"index":0,"relevance_score":0.5},{"relevance_score":0.4}]}`,
		"a missing score":              `{"results":[{"index":0,"relevance_score":0.5},{"index":1}]}`,
		"no results":                   `{}`,
		"not JSON":                     `<html>`,
	}
	for name, body := range bodies {
		_, address := startRecorder(t, body)

		_, err := newTestService(t).Rerank(newContext(t), newCall("openai-compatible", address, "", "m"),
			&llm.RerankRequest{Query: "ferry", Documents: []string{"a", "b"}})

		if !connection.IsStop(err) || err.Error() != connection.ReasonUnavailable.String() {
			t.Errorf("%s: err = %v, want a stop of an unavailable service", name, err)
		}
	}
}

func TestRerankWithARejectedKeyIsTryNext(t *testing.T) {
	_, address := startFakeServer(t, fakeopenai.Config{APIKey: testKey})

	_, err := newTestService(t).Rerank(newContext(t), newCall("openrouter", address, "sk-other-key-9999", "m"),
		&llm.RerankRequest{Query: "ferry", Documents: []string{"a"}})

	checkFailure(t, err, failureWant{isTryNext: true, reason: connection.ReasonRejectedKey})
}

func TestRerankOnAServerWithoutRerankStopsWithAnInvalidRequest(t *testing.T) {
	ts := httptest.NewServer(http.NotFoundHandler())
	t.Cleanup(ts.Close)

	_, err := newTestService(t).Rerank(newContext(t), newCall("openai-compatible", ts.URL+"/v1", "", "m"),
		&llm.RerankRequest{Query: "ferry", Documents: []string{"a"}})

	checkFailure(t, err, failureWant{reason: connection.ReasonInvalidRequest})
}

func TestRerankOfAnOpenAIConnectionStopsWithAnInvalidRequestAndSendsNothing(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})

	_, err := newTestService(t).Rerank(newContext(t), newCall("openai", address, testKey, "m"),
		&llm.RerankRequest{Query: "ferry", Documents: []string{"a"}})

	checkFailure(t, err, failureWant{reason: connection.ReasonInvalidRequest})
	if len(fake.Requests()) != 0 {
		t.Errorf("the service received %d requests", len(fake.Requests()))
	}
}

func TestTheFirstRerankOfAnOpenAICompatibleModelOutlastsTheSettingAndTheNextOneDoesNot(t *testing.T) {
	fake, address := startFakeServer(t, fakeopenai.Config{})
	fake.Delay("/rerank", 300*time.Millisecond)
	svc := newTestService(t)
	svc.Configure(map[string]string{"requestTimeout": "0.1"})
	call := newCall("openai-compatible", address, "", "bge-reranker-v2-m3")
	request := &llm.RerankRequest{Query: "ferry", Documents: []string{"a"}}

	if _, err := svc.Rerank(newContext(t), call, request); err != nil {
		t.Fatalf("the first request: %v", err)
	}

	_, err := svc.Rerank(newContext(t), call, request)
	checkFailure(t, err, failureWant{isTryNext: true, reason: connection.ReasonUnavailable})
}
