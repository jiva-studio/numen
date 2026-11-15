package refresh_test

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock/clocktest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/connection"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/internal/adapter/reranker"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/internal/adapter/reranker/rerankertest"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/ports"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/resultcache"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/usecase/preview"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/usecase/refresh"
)

const testVault domain.VaultID = "harbor-vault"

const (
	stormsID         = "01JCX6Q4ZB3W8N2V7T5R9K1M0P"
	stormsPath       = "Searches/Winter storms.md"
	galesID          = "01JCX6Q4ZB3W8N2V7T5R9K1M0Q"
	galesPath        = "Searches/Gales.md"
	stormsQuery      = `meaning("storm")`
	stormsRerank     = "Stories of storms at the harbor"
	harborConnection = "conn-harbor"

	galeText     = "A storm at the harbor mouth."
	mooringText  = "The storm broke two moorings."
	lampText     = "The lighthouse lamp was lit."
	winterText   = "Storm after storm all winter."
	mooringsText = "The storm broke three moorings."
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

func chunk(id, path string, start, length int, text string) host.SearchResult {
	return host.SearchResult{ChunkID: id, Path: path, Start: start, Length: length, Text: text}
}

// countingSearch is the host search of the vault, counting the searches of the plugin. onSearch, when set,
// runs at the start of each search. failure, when set, is the answer of every search.
type countingSearch struct {
	ports.Search
	mu        sync.Mutex
	calls     int
	active    int
	maxActive int
	onSearch  func()
	failure   error
}

func (c *countingSearch) SearchText(ctx context.Context, query host.SearchResultQuery) ([]host.SearchResult, error) {
	c.mu.Lock()
	c.calls++
	c.active++
	c.maxActive = max(c.maxActive, c.active)
	hook, failure := c.onSearch, c.failure
	c.mu.Unlock()
	if hook != nil {
		hook()
	}
	defer func() {
		c.mu.Lock()
		c.active--
		c.mu.Unlock()
	}()
	if failure != nil {
		return nil, failure
	}
	return c.Search.SearchText(ctx, query)
}

func (c *countingSearch) getCallCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.calls
}

func (c *countingSearch) getMaxActive() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.maxActive
}

func (c *countingSearch) setHook(hook func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onSearch = hook
}

func (c *countingSearch) setFailure(failure error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.failure = failure
}

// recordingTasks is the task list of the host, recording every report and every finish.
type recordingTasks struct {
	mu       sync.Mutex
	reports  []host.Task
	finished []string
}

func (r *recordingTasks) ReportTask(_ context.Context, task host.Task) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.reports = append(r.reports, task)
	return nil
}

func (r *recordingTasks) FinishTask(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.finished = append(r.finished, id)
	return nil
}

func (r *recordingTasks) getReports() []host.Task {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.reports)
}

func (r *recordingTasks) getFinishedIDs() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.finished)
}

type world struct {
	vault    *hosttest.Vault
	search   *hosttest.Search
	host     *countingSearch
	rows     *rerankertest.Rows
	standIn  *rerankertest.StandIn
	tasks    *recordingTasks
	clock    *clocktest.Fake
	mu       sync.Mutex
	settings refresh.Settings
	runner   *refresh.Runner
}

func newWorld(t *testing.T, refreshMinutes int) *world {
	t.Helper()
	search := hosttest.NewSearch(testVault)
	search.SetModel(hosttest.SearchModel{Name: "harbor-model", Recipe: "harbor-recipe", FloorHint: 0.4})
	search.SetEmbedder(embedByTopic)
	fake := new(clocktest.Fake)
	fake.Set(getStartTime())
	w := &world{
		vault:    hosttest.New(string(testVault)),
		search:   search,
		host:     &countingSearch{Search: search.With(testVault)},
		rows:     rerankertest.NewRows(),
		standIn:  rerankertest.NewStandIn(map[string]float32{galeText: 0.9, mooringText: 0.6, winterText: 0.8, mooringsText: 0.7}),
		tasks:    &recordingTasks{},
		clock:    fake,
		settings: refresh.Settings{Preview: preview.DefaultConfig(), RefreshMinutes: refreshMinutes},
	}
	w.rows.Add(harborConnection, "Harbor reranker", map[string]string{"model": "harbor-rerank-a"}, w.standIn)
	w.vault.File(stormsPath, formatSearchNote(stormsID, stormsQuery, stormsRerank, 0))
	w.index(
		chunk("c1", "Harbor/Gale.md", 0, 400, galeText),
		chunk("c2", "Logbooks/1911.md", 0, 400, mooringText),
		chunk("c3", "Harbor/Lamp.md", 0, 400, lampText),
	)
	w.runner = w.newRunner()
	return w
}

func (w *world) newRunner() *refresh.Runner {
	return refresh.New(refresh.Dependencies{Open: w.open, Reranker: reranker.New(w.rows), Tasks: w.tasks}, w.getSettings, w.clock)
}

func (w *world) open(id domain.VaultID) refresh.Vault {
	return refresh.Vault{Notes: w.vault.With(id), Lookup: w.vault.With(id), Search: w.host, Store: w.vault.Store().With(id)}
}

func (w *world) getSettings() refresh.Settings {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.settings
}

func (w *world) setPreviewConfig(config preview.Config) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.settings.Preview = config
}

func (w *world) index(results ...host.SearchResult) {
	w.search.With(testVault).SetIndexedResults(results)
}

func (w *world) addWinterNote() {
	w.vault.File("Harbor/Winter.md", "# Winter\n\n"+winterText+"\n")
	w.index(
		chunk("c1", "Harbor/Gale.md", 0, 400, galeText),
		chunk("c2", "Logbooks/1911.md", 0, 400, mooringText),
		chunk("c3", "Harbor/Lamp.md", 0, 400, lampText),
		chunk("c4", "Harbor/Winter.md", 0, 400, winterText),
	)
}

func (w *world) changeMooringChunk() {
	w.index(
		chunk("c1", "Harbor/Gale.md", 0, 400, galeText),
		chunk("c2", "Logbooks/1911.md", 0, 400, mooringsText),
		chunk("c3", "Harbor/Lamp.md", 0, 400, lampText),
	)
}

func (w *world) runDueSearches(t *testing.T) {
	t.Helper()
	if err := w.runner.RunDue(t.Context(), testVault); err != nil {
		t.Fatalf("RunDue = %v", err)
	}
}

func (w *world) changeIndex(t *testing.T) {
	t.Helper()
	if err := w.runner.MarkStale(t.Context(), testVault); err != nil {
		t.Fatalf("MarkStale = %v", err)
	}
	w.runDueSearches(t)
}

func (w *world) runNow(t *testing.T, notePath string) resultcache.Record {
	t.Helper()
	record, err := w.runner.RunNow(t.Context(), testVault, notePath, connection.PriorityInteractive)
	if err != nil {
		t.Fatalf("RunNow(%s) = %v", notePath, err)
	}
	return record
}

func (w *world) hasRecord(t *testing.T, name string) bool {
	t.Helper()
	files, err := w.vault.Store().List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	return slices.ContainsFunc(files, func(file host.StoreFile) bool { return file.Name == name })
}

func (w *world) readRecord(t *testing.T, noteID string) resultcache.Record {
	t.Helper()
	raw, err := w.vault.Store().Read(t.Context(), noteID+".json")
	if err != nil {
		t.Fatalf("read the cache of %s: %v", noteID, err)
	}
	record, ok := resultcache.Decode(raw)
	if !ok {
		t.Fatalf("the cache of %s does not decode: %s", noteID, raw)
	}
	return record
}

func getResultPaths(record resultcache.Record) []string {
	paths := make([]string, len(record.Results))
	for i, result := range record.Results {
		paths[i] = result.Path
	}
	return paths
}

func getResultTexts(record resultcache.Record) []string {
	texts := make([]string, len(record.Results))
	for i, result := range record.Results {
		texts[i] = result.Text
	}
	return texts
}

func isAt(mark *time.Time, want time.Time) bool {
	return mark != nil && mark.Equal(want)
}

func formatMark(mark *time.Time) string {
	if mark == nil {
		return "unset"
	}
	return mark.Format(time.RFC3339)
}

func getLogbookPage() host.SearchResult {
	return host.SearchResult{
		ChunkID: "ch-1911-14", Path: "Logbooks/1911.pdf", Start: 48210, Length: 2410,
		Text: "The harbor master rowed out while the storm rose.", Selector: `{"page":14}`,
	}
}

func (c *countingSearch) getActiveCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.active
}
