package refresh_test

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/clock/clocktest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest/searchtest"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/domain/resultcache"
	"github.com/jiva-studio/numen/plugins/tools-saved-search/backend/usecase/refresh"
)

const testVault domain.VaultID = "harbor-vault"

const (
	stormsID    = "01JCX6Q4ZB3W8N2V7T5R9K1M0P"
	stormsPath  = "Searches/Winter storms.md"
	galesID     = "01JCX6Q4ZB3W8N2V7T5R9K1M0Q"
	galesPath   = "Searches/Gales.md"
	stormsQuery = `meaning("storm").rerank("Stories of storms at the harbor")`
	galesQuery  = `words("storm")`
	brokenQuery = `meaning("storm") &&`

	galeText     = "A storm at the harbor mouth."
	mooringText  = "The storm broke two moorings."
	winterText   = "Storm after storm all winter."
	mooringsText = "The storm broke three moorings."
)

func getStartTime() time.Time {
	return time.Date(2026, 11, 14, 9, 30, 0, 0, time.UTC)
}

func formatSearchNote(id, query string) string {
	var b strings.Builder
	b.WriteString("---\n")
	if id != "" {
		b.WriteString("id: " + id + "\n")
	}
	b.WriteString("type: search\n---\n```numen-query\n" + query + "\n```\n")
	return b.String()
}

func passage(path, text string, score float64) host.SearchResult {
	return host.SearchResult{Path: path, Title: strings.TrimSuffix(path[strings.LastIndex(path, "/")+1:], ".md"), Kind: "note",
		Text: text, Start: 0, Length: 400, Score: &score}
}

func getStormsAnswer(results ...host.SearchResult) host.SearchState {
	return host.SearchState{
		Results: results, ScoreKind: "rerank", Floor: 0.4, IndexGeneration: "gen-7", Warnings: []string{},
		Reranks: []host.SearchRerank{{ConnectionName: "Harbor reranker", SentCount: len(results), ReusedCount: 0}},
	}
}

// countingSearch is the search of the core, counting the searches on their way between their start and
// their removal. onStart, when set, runs at each start; failure, when set, is the answer of every read.
type countingSearch struct {
	*searchtest.Searches
	mu        sync.Mutex
	active    int
	maxActive int
	onStart   func()
	failure   error
}

func (c *countingSearch) StartSearch(ctx context.Context, start host.SearchStart) (string, error) {
	c.mu.Lock()
	c.active++
	c.maxActive = max(c.maxActive, c.active)
	hook := c.onStart
	c.mu.Unlock()
	if hook != nil {
		hook()
	}
	return c.Searches.StartSearch(ctx, start)
}

func (c *countingSearch) GetSearch(ctx context.Context, id string, read host.SearchRead) (host.SearchState, error) {
	c.mu.Lock()
	failure := c.failure
	c.mu.Unlock()
	if failure != nil {
		return host.SearchState{}, failure
	}
	return c.Searches.GetSearch(ctx, id, read)
}

func (c *countingSearch) DeleteSearch(ctx context.Context, id string) error {
	c.mu.Lock()
	c.active--
	c.mu.Unlock()
	return c.Searches.DeleteSearch(ctx, id)
}

func (c *countingSearch) getMaxActive() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.maxActive
}

func (c *countingSearch) getActiveCount() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.active
}

func (c *countingSearch) setHook(hook func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onStart = hook
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
	searches *searchtest.Searches
	host     *countingSearch
	tasks    *recordingTasks
	clock    *clocktest.Fake
	mu       sync.Mutex
	settings refresh.Settings
	runner   *refresh.Runner
}

func newWorld(t *testing.T, refreshMinutes int) *world {
	t.Helper()
	searches := searchtest.NewSearches()
	fake := new(clocktest.Fake)
	fake.Set(getStartTime())
	w := &world{
		vault:    hosttest.New(string(testVault)),
		searches: searches,
		host:     &countingSearch{Searches: searches},
		tasks:    &recordingTasks{},
		clock:    fake,
		settings: refresh.Settings{RefreshMinutes: refreshMinutes},
	}
	w.vault.WriteFile(stormsPath, formatSearchNote(stormsID, stormsQuery))
	w.setStorms(passage("Harbor/Gale.md", galeText, 0.9), passage("Logbooks/1911.md", mooringText, 0.6))
	searches.SetAnswer(galesQuery, host.SearchState{Results: []host.SearchResult{passage("Harbor/Gale.md", galeText, 1)}, ScoreKind: "words"})
	searches.SetAnswer(brokenQuery, host.SearchState{Errors: []host.SearchProblem{{Line: 1, Column: 20, Message: "Syntax error: unexpected end of input"}}})
	w.runner = w.newRunner()
	return w
}

func (w *world) newRunner() *refresh.Runner {
	return refresh.New(refresh.Dependencies{Open: w.open, Tasks: w.tasks, Log: slog.Default()}, w.getSettings, w.clock)
}

func (w *world) open(id domain.VaultID) refresh.Vault {
	return refresh.Vault{Notes: w.vault.With(id), Lookup: w.vault.With(id), Search: w.host, Store: w.vault.Store().With(id)}
}

func (w *world) getSettings() refresh.Settings {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.settings
}

func (w *world) setStorms(results ...host.SearchResult) {
	w.searches.SetAnswer(stormsQuery, getStormsAnswer(results...))
}

func (w *world) addWinterNote() {
	w.vault.WriteFile("Harbor/Winter.md", "# Winter\n\n"+winterText+"\n")
	w.setStorms(passage("Harbor/Gale.md", galeText, 0.9), passage("Harbor/Winter.md", winterText, 0.8), passage("Logbooks/1911.md", mooringText, 0.6))
}

func (w *world) changeMooringChunk() {
	w.setStorms(passage("Harbor/Gale.md", galeText, 0.9), passage("Logbooks/1911.md", mooringsText, 0.7))
}

func (w *world) getStartCount() int {
	return len(w.searches.ListStarted())
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
	record, err := w.runner.RunNow(t.Context(), testVault, notePath)
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
		Path: "Logbooks/1911.pdf", Title: "1911", Kind: "pdf", Start: 48210, Length: 2410, MatchLine: 379,
		Text: "The harbor master rowed out while the storm rose.", Selector: `{"page":14}`, PlaceLabel: "p. 14",
	}
}
