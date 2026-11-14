package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/views-todos/backend/api"
	"github.com/jiva-studio/numen/plugins/views-todos/backend/todo"
	"github.com/jiva-studio/numen/plugins/views-todos/backend/usecase/index"
)

const today = "2026-11-14"

type app struct {
	vault *hosttest.Vault
	index *index.Index
	web   *httptest.Server
}

func start(t *testing.T, notes map[string]string) *app {
	t.Helper()
	vault := hosttest.New("notes")
	for path, text := range notes {
		vault.File(path, text)
	}
	tasks := index.New(vault, vault, todo.ParseNote)
	if err := tasks.Scan(t.Context()); err != nil {
		t.Fatal(err)
	}
	handler, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "plugin-token"}, func(s *server.Server) error {
		return mount(s, tasks)
	})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	t.Cleanup(web.Close)
	return &app{vault: vault, index: tasks, web: web}
}

func (a *app) request(ctx context.Context, t *testing.T, path, body string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.web.URL+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer plugin-token")
	res, err := a.web.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// call posts a request to a method and decodes the answer into out when the status is 200.
func (a *app) call(t *testing.T, method string, request, out any) (int, server.APIError) {
	t.Helper()
	body, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	res := a.request(t.Context(), t, "/"+api.Service+"/"+method, string(body))
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			t.Error(closeErr)
		}
	}()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	var failure server.APIError
	if res.StatusCode == http.StatusOK {
		if err := json.Unmarshal(data, out); err != nil {
			t.Fatalf("%s answered %s: %v", method, data, err)
		}
		return res.StatusCode, failure
	}
	if err := json.Unmarshal(data, &failure); err != nil {
		t.Fatalf("%s answered %d %s", method, res.StatusCode, data)
	}
	return res.StatusCode, failure
}

func (a *app) list(t *testing.T, f api.Filter, limit int) api.ListTasksResponse {
	t.Helper()
	var got api.ListTasksResponse
	if status, failure := a.call(t, api.MethodListTasks, api.ListTasksRequest{Today: today, Filter: f, Limit: limit}, &got); status != http.StatusOK {
		t.Fatalf("ListTasks = %d %+v", status, failure)
	}
	return got
}

func TestGetSummaryCountsTheOpenTasksOfTheDay(t *testing.T) {
	a := start(t, map[string]string{
		"a.md": "- [ ] late due:2026-11-01\n- [ ] now due:2026-11-14\n- [ ] floating\n- [x] finished",
		"b.md": "---\nbroken",
	})

	var got api.Summary
	status, _ := a.call(t, api.MethodGetSummary, api.GetSummaryRequest{Today: today}, &got)

	want := api.Summary{IsReady: true, OverdueCount: 1, TodayCount: 1, NoDateCount: 1, UnreadableCount: 1}
	if status != http.StatusOK || got != want {
		t.Errorf("GetSummary = %d %+v, want %+v", status, got, want)
	}
}

func TestListTasksAppliesTheFilterAndReportsTheTotalBeforeTheLimit(t *testing.T) {
	a := start(t, map[string]string{
		"a.md": "- [ ] one due:2026-11-01 +home\n- [ ] two due:2026-11-02 +home\n- [ ] three due:2026-11-03 +home\n- [ ] four +work",
	})

	got := a.list(t, api.Filter{Due: "overdue", Projects: []string{"home"}}, 2)

	if got.Total != 3 || len(got.Tasks) != 2 || got.Tasks[0].Text != "one" || got.Tasks[1].Text != "two" {
		t.Fatalf("ListTasks = %+v", got)
	}
	if first := got.Tasks[0]; first.Path != "a.md" || first.Line != 1 || first.Status != "open" || first.Due != "2026-11-01" || first.Projects[0] != "home" {
		t.Errorf("first task = %+v", first)
	}
}

func TestListTasksReturnsAtMostFiveHundredTasksWhenNoLimitIsAskedAndTwoThousandWhenMoreIsAsked(t *testing.T) {
	var lines []string
	for n := range 2100 {
		lines = append(lines, fmt.Sprintf("- [ ] task %d", n))
	}
	a := start(t, map[string]string{"big.md": strings.Join(lines, "\n")})

	if got := a.list(t, api.Filter{}, 0); len(got.Tasks) != 500 || got.Total != 2100 {
		t.Errorf("no limit: %d tasks of %d, want 500 of 2100", len(got.Tasks), got.Total)
	}
	if got := a.list(t, api.Filter{}, 9999); len(got.Tasks) != 2000 || got.Total != 2100 {
		t.Errorf("a limit of 9999: %d tasks of %d, want 2000 of 2100", len(got.Tasks), got.Total)
	}
}

func TestRefreshIndexReadsAChangedNoteAndTheNextListShowsIt(t *testing.T) {
	a := start(t, map[string]string{"a.md": "- [ ] first"})
	before := a.list(t, api.Filter{}, 10)

	a.vault.File("a.md", "- [ ] first\n- [ ] second")
	var empty api.RefreshIndexResponse
	status, failure := a.call(t, api.MethodRefreshIndex, api.RefreshIndexRequest{Paths: []string{"a.md"}, Renamed: []api.Rename{}}, &empty)
	after := a.list(t, api.Filter{}, 10)

	if status != http.StatusOK {
		t.Fatalf("RefreshIndex = %d %+v", status, failure)
	}
	if len(before.Tasks) != 1 || len(after.Tasks) != 2 || after.Revision <= before.Revision {
		t.Errorf("before = %+v, after = %+v", before, after)
	}
}

func TestRefreshIndexOfAHostWithNoVaultIsAFailedPrecondition(t *testing.T) {
	a := start(t, map[string]string{"a.md": "- [ ] first"})
	a.vault.Hide()

	var empty api.RefreshIndexResponse
	status, failure := a.call(t, api.MethodRefreshIndex, api.RefreshIndexRequest{Paths: []string{}, Renamed: []api.Rename{}}, &empty)

	if status != http.StatusPreconditionFailed || failure.Code != server.CodeFailedPrecondition {
		t.Errorf("RefreshIndex = %d %+v, want failed_precondition", status, failure)
	}
}

func TestTheHostConfiguresTheIgnoredFoldersAndTheNextListLeavesThemOut(t *testing.T) {
	a := start(t, map[string]string{"Daily.md": "- [ ] daily", "templates/Weekly.md": "- [ ] template"})

	res := a.request(t.Context(), t, "/numen.v1.PluginLifecycleService/Configure", `{"values":{"ignoredFolders":"templates"}}`)
	if err := res.Body.Close(); err != nil {
		t.Fatal(err)
	}
	got := a.list(t, api.Filter{}, 10)

	if res.StatusCode != http.StatusOK || len(got.Tasks) != 1 || got.Tasks[0].Path != "Daily.md" {
		t.Errorf("Configure = %d, tasks = %+v", res.StatusCode, got.Tasks)
	}
}

// events reads the data events of a stream until count are read.
func (a *app) events(ctx context.Context, t *testing.T, method string, count int) <-chan api.WatchTasksMessage {
	t.Helper()
	res := a.request(ctx, t, "/"+api.Service+"/"+method, `{}`)
	out := make(chan api.WatchTasksMessage, count)
	go func() {
		defer close(out)
		defer func() { _ = res.Body.Close() }()
		scanner := bufio.NewScanner(res.Body)
		for scanner.Scan() {
			data, found := strings.CutPrefix(scanner.Text(), "data: ")
			if !found {
				continue
			}
			var message api.WatchTasksMessage
			if json.Unmarshal([]byte(data), &message) != nil {
				return
			}
			out <- message
		}
	}()
	return out
}

func TestWatchTasksSendsOnceAtTheStartAndAgainAfterAChangeOfTheIndex(t *testing.T) {
	a := start(t, map[string]string{"a.md": "- [ ] first"})
	ctx, stop := context.WithCancel(t.Context())
	defer stop()
	messages := a.events(ctx, t, api.MethodWatchTasks, 4)
	next := func() api.WatchTasksMessage {
		t.Helper()
		select {
		case message, ok := <-messages:
			if !ok {
				t.Fatal("the stream ended")
			}
			return message
		case <-time.After(10 * time.Second):
			t.Fatal("no message was sent")
			return api.WatchTasksMessage{}
		}
	}

	first := next()
	if !first.IsReady || first.Revision == 0 {
		t.Errorf("first message = %+v, want a ready index with a revision", first)
	}

	a.vault.File("a.md", "- [ ] first\n- [ ] second")
	if err := a.index.Refresh(t.Context(), []string{"a.md"}, nil); err != nil {
		t.Fatal(err)
	}
	if second := next(); second.Revision <= first.Revision {
		t.Errorf("second message = %+v, want a revision after %d", second, first.Revision)
	}
}
