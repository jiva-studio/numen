package service

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/plugins/todos/backend/api"
	"github.com/jiva-studio/numen/plugins/todos/backend/ports"
	"github.com/jiva-studio/numen/plugins/todos/backend/todo"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/complete"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/index"
)

const today = "2026-11-14"

type app struct {
	vault *hosttest.Vault
	index *index.Index
	web   *httptest.Server
}

func start(t *testing.T, notes map[string]string) *app {
	t.Helper()
	return startWithWriter(t, notes, nil)
}

func startWithWriter(t *testing.T, notes map[string]string, getWriter func(*hosttest.Vault) ports.FileWriter) *app {
	t.Helper()
	vault := hosttest.New("notes")
	for path, text := range notes {
		vault.File(path, text)
	}
	var writer ports.FileWriter = vault
	if getWriter != nil {
		writer = getWriter(vault)
	}
	tasks := index.New(vault, vault, todo.ParseNote)
	if err := tasks.Scan(t.Context()); err != nil {
		t.Fatal(err)
	}
	handler, err := server.NewLinkedHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "plugin-token"}, func(s *server.Server) error {
		return mount(s, tasks, complete.New(vault, writer))
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

func TestTheHostConfiguresTheIgnoredFoldersAndTheNextListOmitsThem(t *testing.T) {
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

func (a *app) events(ctx context.Context, t *testing.T, method string, count int) <-chan api.WatchTasksMessage {
	t.Helper()
	res := a.request(ctx, t, "/"+api.Service+"/"+method, `{}`)
	t.Cleanup(func() {
		if err := res.Body.Close(); err != nil {
			t.Error(err)
		}
	})
	out := make(chan api.WatchTasksMessage, count)
	go func() {
		defer close(out)
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

func (a *app) setTaskDone(t *testing.T, req api.SetTaskDoneRequest) (int, api.SetTaskDoneResponse, server.APIError) {
	t.Helper()
	var got api.SetTaskDoneResponse
	status, failure := a.call(t, api.MethodSetTaskDone, req, &got)
	return status, got, failure
}

func tickRequest(path string, line int, raw string) api.SetTaskDoneRequest {
	return api.SetTaskDoneRequest{Path: path, Line: line, Raw: raw, IsDone: true, Today: today}
}

func TestTickOverHTTPWritesTheLineAndTheNextListShowsTheTaskTicked(t *testing.T) {
	a := start(t, map[string]string{"Home.md": "# Chores\n- [ ] fix the tap +home\n- [ ] buy milk\n"})
	before := a.list(t, api.Filter{}, 10)

	status, got, failure := a.setTaskDone(t, tickRequest("Home.md", 2, "- [ ] fix the tap +home"))

	if status != http.StatusOK {
		t.Fatalf("SetTaskDone = %d %+v", status, failure)
	}
	if got.Error != "" || got.Change != "completed" {
		t.Errorf("response = %+v, want a completed change", got)
	}
	if task := got.Task; task.Path != "Home.md" || task.Line != 2 || task.Status != "done" || task.DoneOn != today ||
		task.Raw != "- [x] fix the tap +home done:2026-11-14" || task.Projects[0] != "home" {
		t.Errorf("task = %+v", task)
	}
	if note, _ := a.vault.Content("Home.md"); note != "# Chores\n- [x] fix the tap +home done:2026-11-14\n- [ ] buy milk\n" {
		t.Errorf("note = %q", note)
	}
	open := a.list(t, api.Filter{}, 10)
	done := a.list(t, api.Filter{Status: []string{"done"}}, 10)
	if len(open.Tasks) != 1 || open.Tasks[0].Text != "buy milk" || open.Revision <= before.Revision {
		t.Errorf("open tasks after the tick = %+v, before = revision %d", open, before.Revision)
	}
	if len(done.Tasks) != 1 || done.Tasks[0].Line != 2 {
		t.Errorf("done tasks after the tick = %+v", done.Tasks)
	}
}

func TestUntickOverHTTPReopensATickedTask(t *testing.T) {
	a := start(t, map[string]string{"a.md": "- [x] buy milk done:2026-11-10\n"})

	status, got, failure := a.setTaskDone(t, api.SetTaskDoneRequest{
		Path: "a.md", Line: 1, Raw: "- [x] buy milk done:2026-11-10", IsDone: false, Today: today,
	})

	if status != http.StatusOK || got.Error != "" || got.Change != "reopened" || got.Task.Status != "open" {
		t.Fatalf("SetTaskDone = %d %+v %+v", status, got, failure)
	}
	if note, _ := a.vault.Content("a.md"); note != "- [ ] buy milk\n" {
		t.Errorf("note = %q", note)
	}
}

// staleWriter is a vault that another window changes before each write.
type staleWriter struct{ vault *hosttest.Vault }

func (w staleWriter) Write(ctx context.Context, path string, content []byte, baseline domain.FileStat) (domain.FileStat, error) {
	now, _ := w.vault.Content(path)
	w.vault.File(path, now+"typed\n")
	return w.vault.Write(ctx, path, content, baseline)
}

func TestTickOverHTTPAnswersAChangeTheNoteDoesNotAllowWithItsErrorWord(t *testing.T) {
	cases := []struct {
		name      string
		note      string
		req       api.SetTaskDoneRequest
		getWriter func(*hosttest.Vault) ports.FileWriter
		want      string
	}{
		{"a line whose text changed", "- [ ] buy milk and bread\n", tickRequest("a.md", 1, "- [ ] buy milk"), nil, "notFound"},
		{"a note whose frontmatter is never closed", "---\n- [ ] buy milk\n", tickRequest("a.md", 2, "- [ ] buy milk"), nil, "changed"},
		{"a tick of a done task", "- [x] buy milk\n", tickRequest("a.md", 1, "- [x] buy milk"), nil, "notOpen"},
		{"a tick of a task with an invalid recurrence", "- [ ] buy milk rec:0w\n", tickRequest("a.md", 1, "- [ ] buy milk rec:0w"), nil, "invalidRecurrence"},
		{
			"an untick of an open task", "- [ ] buy milk\n",
			api.SetTaskDoneRequest{Path: "a.md", Line: 1, Raw: "- [ ] buy milk", IsDone: false, Today: today}, nil, "notDone",
		},
		{
			"a note that changes before each write", "- [ ] buy milk\n", tickRequest("a.md", 1, "- [ ] buy milk"),
			func(v *hosttest.Vault) ports.FileWriter { return staleWriter{vault: v} }, "stale",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := startWithWriter(t, map[string]string{"a.md": tc.note}, tc.getWriter)

			status, got, failure := a.setTaskDone(t, tc.req)

			if status != http.StatusOK || got.Error != tc.want || got.Change != "" {
				t.Errorf("SetTaskDone = %d %+v %+v, want error %q", status, got, failure, tc.want)
			}
			if note, _ := a.vault.Content("a.md"); strings.Contains(note, "[x] buy milk done:") {
				t.Errorf("note = %q, want no tick written", note)
			}
		})
	}
}

func TestTickOverHTTPAnswersAHostFailureWithAnAPIError(t *testing.T) {
	cases := []struct {
		name    string
		arrange func(*hosttest.Vault)
		status  int
		code    server.ErrorCode
	}{
		{"no vault", func(v *hosttest.Vault) { v.Hide() }, http.StatusPreconditionFailed, server.CodeFailedPrecondition},
		{"a note the plugin may not write", func(v *hosttest.Vault) { v.RefuseWriting("a.md", host.ErrPermissionDenied) }, http.StatusForbidden, server.CodePermissionDenied},
		{"a failure the host does not name", func(v *hosttest.Vault) { v.RefuseWriting("a.md", errors.New("disk full")) }, http.StatusInternalServerError, server.CodeInternal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a := start(t, map[string]string{"a.md": "- [ ] buy milk\n"})
			tc.arrange(a.vault)

			status, _, failure := a.setTaskDone(t, tickRequest("a.md", 1, "- [ ] buy milk"))

			if status != tc.status || failure.Code != tc.code {
				t.Errorf("SetTaskDone = %d %+v, want %d %s", status, failure, tc.status, tc.code)
			}
		})
	}
}
