package service_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock/clocktest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/api"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/internal/adapter/reranker/rerankertest"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/internal/adapter/settings"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/ports"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/resultcache"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/service"
	"github.com/jiva-studio/numen/plugins/views-saved-search/backend/usecase/refresh"
)

const (
	testVault    = "harbor-vault"
	token        = "plugin-token"
	stormsID     = "01JCX6Q4ZB3W8N2V7T5R9K1M0P"
	stormsPath   = "Searches/Winter storms.md"
	stormsQuery  = `meaning("storm")`
	stormsRerank = "Stories of storms at the harbor"

	galeText     = "A storm at the harbor mouth."
	mooringText  = "The storm broke two moorings."
	mooringsText = "The storm broke three moorings."
	winterText   = "Storm after storm all winter."
)

func getStartTime() time.Time {
	return time.Date(2026, 11, 14, 9, 30, 0, 0, time.UTC)
}

func embedByTopic(text string) []float32 {
	// The fake model gives a text one dimension for each topic, the count of the topic's word in it.
	topics := []string{"tide", "lighthouse", "storm", "harbor"}
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

func formatSearchNote(id, query, rerank string, limit int) string {
	var b strings.Builder
	b.WriteString("---\n")
	if id != "" {
		b.WriteString("id: " + id + "\n")
	}
	b.WriteString("type: search\n")
	if rerank != "" {
		b.WriteString("rerank: " + rerank + "\n")
	}
	if limit > 0 {
		fmt.Fprintf(&b, "limit: %d\n", limit)
	}
	b.WriteString("---\n```numen-query\n" + query + "\n```\n")
	return b.String()
}

func getIndex(mooring string, others ...host.SearchResult) []host.SearchResult {
	return append([]host.SearchResult{
		{ChunkID: "c1", Path: "Notes/Tides.md", Text: "The tide at the quay.", Start: 0, Length: 21, MatchLine: 2},
		{ChunkID: "c2", Path: "Notes/Keeper.md", Text: "The lighthouse keeper.", Start: 0, Length: 22},
		{ChunkID: "c3", Path: "Harbor/Gale.md", Text: galeText, Start: 0, Length: 400},
		{ChunkID: "c4", Path: "Logbooks/1911.md", Text: mooring, Start: 0, Length: 400},
	}, others...)
}

// countingSearch is the host search of the vault, counting the searches of the plugin.
type countingSearch struct {
	ports.Search
	mu    sync.Mutex
	calls int
}

func (c *countingSearch) SearchText(ctx context.Context, query host.SearchResultQuery) ([]host.SearchResult, error) {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()
	return c.Search.SearchText(ctx, query)
}

func (c *countingSearch) ScoreChunks(ctx context.Context, query host.ScoreQuery) (host.ChunkScores, error) {
	c.mu.Lock()
	c.calls++
	c.mu.Unlock()
	return c.Search.ScoreChunks(ctx, query)
}

func (c *countingSearch) getCallCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

// fixture is the plugin over a host that shows a vault of tides and storms, with no reranking row.
type fixture struct {
	app      *service.App
	vault    *hosttest.Vault
	search   *hosttest.Search
	host     *countingSearch
	rows     *rerankertest.Rows
	standIn  *rerankertest.StandIn
	queue    *hosttest.Queue
	clock    *clocktest.Fake
	settings *settings.Store
	registry *tools.Registry
	web      *httptest.Server
}

func newFixture(t *testing.T, refreshMinutes int) *fixture {
	t.Helper()
	search := hosttest.NewSearch(testVault)
	search.SetModel(hosttest.SearchModel{Name: "harbor-model", Recipe: "harbor-recipe", FloorHint: 0.4})
	search.SetEmbedder(embedByTopic)
	search.With(testVault).SetIndexedResults(getIndex(mooringText))
	vault := hosttest.New(testVault)
	vault.File("Notes/Tides.md", "---\ntitle: Tide Tables\n---\nThe tide at the quay.")
	fake := new(clocktest.Fake)
	fake.Set(getStartTime())
	current := settings.NewStore()
	current.Update(map[string]string{"refreshMinutes": fmt.Sprint(refreshMinutes)})
	f := &fixture{
		vault: vault, search: search, host: &countingSearch{Search: search.With(testVault)},
		rows:    rerankertest.NewRows(),
		standIn: rerankertest.NewStandIn(map[string]float32{galeText: 0.9, mooringText: 0.6, mooringsText: 0.7, winterText: 0.8}),
		queue:   hosttest.NewQueue(), clock: fake, settings: current,
	}
	f.app = service.NewAppOver(service.Hosts{
		Open: func(id domain.VaultID) refresh.Vault {
			return refresh.Vault{Notes: vault.With(id), Lookup: vault.With(id), Search: f.host, Store: vault.Store().With(id)}
		},
		Window: vault, Rows: f.rows, Tasks: f.queue,
	}, current, fake)
	registry, err := f.app.NewRegistry()
	if err != nil {
		t.Fatal(err)
	}
	f.registry = registry
	handler, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": token}, f.app.Mount)
	if err != nil {
		t.Fatal(err)
	}
	f.web = httptest.NewServer(handler)
	t.Cleanup(f.web.Close)
	return f
}

func (f *fixture) addReranker() {
	f.rows.Add("conn-harbor", "Harbor reranker", map[string]string{"model": "harbor-rerank-a"}, f.standIn)
}

func (f *fixture) addStormsNote() {
	f.vault.File(stormsPath, formatSearchNote(stormsID, stormsQuery, stormsRerank, 0))
}

func (f *fixture) handleIndexChange(t *testing.T) {
	t.Helper()
	if err := f.app.HandleEvent(t.Context(), "onIndexChanged:"+testVault); err != nil {
		t.Fatalf("HandleEvent = %v", err)
	}
}

func (f *fixture) readRecord(t *testing.T, noteID string) resultcache.Record {
	t.Helper()
	raw, err := f.vault.Store().Read(t.Context(), noteID+".json")
	if err != nil {
		t.Fatalf("read the cache of %s: %v", noteID, err)
	}
	record, ok := resultcache.Decode(raw)
	if !ok {
		t.Fatalf("the cache of %s does not decode: %s", noteID, raw)
	}
	return record
}

func (f *fixture) newRequest(t *testing.T, method string, req any) *http.Request {
	t.Helper()
	body, err := json.Marshal(req)
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, f.web.URL+"/"+api.Service+"/"+method, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Authorization", "Bearer "+token)
	return request
}

func (f *fixture) call(t *testing.T, method string, req, res any) server.APIError {
	t.Helper()
	response, err := f.web.Client().Do(f.newRequest(t, method, req))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	var failure server.APIError
	if response.StatusCode != http.StatusOK {
		if err := json.NewDecoder(response.Body).Decode(&failure); err != nil {
			t.Fatalf("status %d: %v", response.StatusCode, err)
		}
		return failure
	}
	if err := json.NewDecoder(response.Body).Decode(res); err != nil {
		t.Fatal(err)
	}
	return failure
}

func (f *fixture) getResults(t *testing.T, notePath string) api.GetResultsResponse {
	t.Helper()
	var res api.GetResultsResponse
	if failure := f.call(t, api.MethodGetResults, api.GetResultsRequest{Path: notePath}, &res); failure.Code != "" {
		t.Fatalf("GetResults(%s) = %+v", notePath, failure)
	}
	return res
}

// event is one Server-Sent Event of a stream: its name, empty for a data event, and its data.
type event struct {
	name string
	data string
}

func (f *fixture) openStream(t *testing.T, method string, req any) <-chan event {
	t.Helper()
	ctx, cancel := context.WithCancel(t.Context())
	t.Cleanup(cancel)
	response, err := f.web.Client().Do(f.newRequest(t, method, req).WithContext(ctx))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := response.Body.Close(); err != nil {
			t.Log(err)
		}
	})
	events := make(chan event, 16)
	go func() {
		defer close(events)
		scanner := bufio.NewScanner(response.Body)
		var current event
		for scanner.Scan() {
			line := scanner.Text()
			switch {
			case strings.HasPrefix(line, "event: "):
				current.name = strings.TrimPrefix(line, "event: ")
			case strings.HasPrefix(line, "data: "):
				current.data = strings.TrimPrefix(line, "data: ")
			case line == "":
				events <- current
				current = event{}
			}
		}
	}()
	return events
}

func receiveEvent(t *testing.T, events <-chan event) event {
	t.Helper()
	select {
	case received, isOpen := <-events:
		if !isOpen {
			t.Fatal("the stream ended")
		}
		return received
	case <-time.After(5 * time.Second):
		t.Fatal("no event within 5 seconds")
		return event{}
	}
}

func (f *fixture) callTool(t *testing.T, vault, name, arguments string) (structured map[string]any, failure string) {
	t.Helper()
	res, err := f.registry.InvokeTool(t.Context(), tools.Call{VaultID: vault, Name: name, Arguments: []byte(arguments)})
	if err != nil {
		t.Fatal(err)
	}
	if res.GetResult().GetIsError() {
		return nil, res.GetResult().GetText()
	}
	if err := json.Unmarshal(res.GetResult().GetStructured(), &structured); err != nil {
		t.Fatalf("%v in %s", err, res.GetResult().GetStructured())
	}
	return structured, ""
}

func (f *fixture) watchFirstRun(t *testing.T) <-chan event {
	t.Helper()
	events := f.openStream(t, api.MethodWatchResults, api.WatchResultsRequest{Path: stormsPath})
	if first := receiveEvent(t, events); first.data != `{"computedAt":"","isRunning":false}` {
		t.Fatalf("first event = %+v, want a search never computed", first)
	}
	if start := receiveEvent(t, events); !strings.Contains(start.data, `"isRunning":true`) {
		t.Fatalf("event = %+v, want the start of the first run", start)
	}
	if write := receiveEvent(t, events); write.data != `{"computedAt":"2026-11-14T09:30:00Z","isRunning":false}` {
		t.Fatalf("event = %+v, want the write of the first run", write)
	}
	return events
}

func (f *fixture) configure(t *testing.T, values map[string]string) {
	t.Helper()
	body, err := json.Marshal(map[string]any{"values": values})
	if err != nil {
		t.Fatal(err)
	}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, f.web.URL+"/numen.v1.PluginLifecycleService/Configure", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := f.web.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("Configure = %d", response.StatusCode)
	}
}
