package tasktools_test

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/todos/backend/ports"
	"github.com/jiva-studio/numen/plugins/todos/backend/testsupport"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/complete"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/edit"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/index"
	"github.com/jiva-studio/numen/plugins/todos/backend/wire/tasktools"
)

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return time.Date(2026, time.November, 14, 12, 0, 0, 0, time.UTC)
}

// oneVault answers every call with the scope of one vault.
type oneVault struct{ scope tasktools.Scope }

func (o oneVault) GetScope(context.Context, string) (tasktools.Scope, error) { return o.scope, nil }

const homeNote = "# Home\n" +
	"- [ ] (A) call bank +money @phone due:2026-11-10\n" +
	"- [ ] water plants due:2026-11-14\n" +
	"- [x] buy milk done:2026-11-01\n" +
	"- [ ] read a book\n"

type session struct {
	registry *tools.Registry
	vault    *hosttest.Vault
	tasks    *index.Index
}

func newSession(t *testing.T) *session {
	t.Helper()
	vault := hosttest.New("notes")
	vault.WriteFile("Home.md", homeNote)
	vault.WriteFile("work/Plan.md", "- [ ] ship release +work due:2026-11-20\n")
	tasks := testsupport.NewIndex(vault, vault, vault)
	if err := tasks.Scan(t.Context()); err != nil {
		t.Fatal(err)
	}
	lock := ports.NewWriteLock()
	scope := tasktools.Scope{
		Tasks:     tasks,
		Editor:    edit.New(vault, vault, tasks, lock),
		Completer: complete.New(vault, vault, tasks, lock),
	}
	registry, err := tasktools.New(oneVault{scope: scope}, fixedClock{}).BuildRegistry()
	if err != nil {
		t.Fatal(err)
	}
	return &session{registry: registry, vault: vault, tasks: tasks}
}

func (s *session) invoke(t *testing.T, name string, args any) (text string, structured []byte, isError bool) {
	t.Helper()
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.registry.InvokeTool(t.Context(), tools.Call{Name: name, Arguments: raw, VaultID: "notes"})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	result := res.GetResult()
	return result.GetText(), result.GetStructured(), result.GetIsError()
}

func call[T any](t *testing.T, s *session, name string, args any) T {
	t.Helper()
	text, structured, isError := s.invoke(t, name, args)
	if isError {
		t.Fatalf("%s: %s", name, text)
	}
	var out T
	if err := json.Unmarshal(structured, &out); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return out
}

func getError(t *testing.T, s *session, name string, args any) string {
	t.Helper()
	text, _, isError := s.invoke(t, name, args)
	if !isError {
		t.Fatalf("%s: want an error", name)
	}
	return text
}

func lines(out tasktools.TaskListOutput) []string {
	var got []string
	for _, task := range out.Tasks {
		got = append(got, task.Path+":"+strings.TrimSpace(task.Text))
	}
	return got
}

func TestTaskListFiltersTasks(t *testing.T) {
	cases := []struct {
		name string
		args map[string]any
		want []string
	}{
		{"open by default", map[string]any{}, []string{"Home.md:call bank", "Home.md:water plants", "work/Plan.md:ship release", "Home.md:read a book"}},
		{"done", map[string]any{"status": "done"}, []string{"Home.md:buy milk"}},
		{"all", map[string]any{"status": "all", "path": "Home.md"}, []string{"Home.md:call bank", "Home.md:water plants", "Home.md:buy milk", "Home.md:read a book"}},
		{"overdue", map[string]any{"due": "overdue"}, []string{"Home.md:call bank"}},
		{"today", map[string]any{"due": "today"}, []string{"Home.md:water plants"}},
		{"this week", map[string]any{"due": "thisWeek"}, []string{"Home.md:water plants", "work/Plan.md:ship release"}},
		{"none", map[string]any{"due": "none"}, []string{"Home.md:read a book"}},
		{"range", map[string]any{"due": "range", "from": "2026-11-15", "to": "2026-11-30"}, []string{"work/Plan.md:ship release"}},
		{"project with its plus sign", map[string]any{"project": "+money"}, []string{"Home.md:call bank"}},
		{"context", map[string]any{"context": "phone"}, []string{"Home.md:call bank"}},
		{"priority in lower case", map[string]any{"priority": "a"}, []string{"Home.md:call bank"}},
		{"folder", map[string]any{"path": "work/"}, []string{"work/Plan.md:ship release"}},
		{"text without case", map[string]any{"text": "PLANTS"}, []string{"Home.md:water plants"}},
		{"limit", map[string]any{"limit": 1}, []string{"Home.md:call bank"}},
	}
	s := newSession(t)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := call[tasktools.TaskListOutput](t, s, "task_list", c.args)

			if got := lines(out); !slices.Equal(got, c.want) {
				t.Errorf("tasks = %v, want %v", got, c.want)
			}
		})
	}
}

func TestTaskListReturnsTheFieldsAndTheTotalBeforeTheLimit(t *testing.T) {
	s := newSession(t)

	out := call[tasktools.TaskListOutput](t, s, "task_list", map[string]any{"limit": 1})

	want := tasktools.TaskOutput{
		Path: "Home.md", Line: 2, Text: "call bank", Status: "open", Due: "2026-11-10", Priority: "A",
		Projects: []string{"money"}, Contexts: []string{"phone"},
	}
	if len(out.Tasks) != 1 || !reflect.DeepEqual(out.Tasks[0], want) || out.Total != 4 {
		t.Errorf("out = %+v, want %+v of 4", out, want)
	}
}

func TestTaskListRefusesInvalidFilters(t *testing.T) {
	cases := []struct {
		name string
		args map[string]any
		want string
	}{
		{"status", map[string]any{"status": "open+done"}, "status"},
		{"due", map[string]any{"due": "soon"}, "due"},
		{"date", map[string]any{"due": "range", "from": "2026-02-30"}, "YYYY-MM-DD"},
		{"range bounds without a range", map[string]any{"from": "2026-11-01"}, "need due range"},
		{"priority", map[string]any{"priority": "1"}, "A to Z"},
	}
	s := newSession(t)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := getError(t, s, "task_list", c.args); !strings.Contains(got, c.want) {
				t.Errorf("error = %q, want it to mention %q", got, c.want)
			}
		})
	}
}

func TestTaskAddAppendsALineTheIndexListsAtOnce(t *testing.T) {
	s := newSession(t)

	task := call[tasktools.TaskOutput](t, s, "task_add", map[string]any{"path": "Home.md", "text": "file taxes +money due:2026-12-01"})

	if got, _ := s.vault.ReadContent("Home.md"); got != homeNote+"- [ ] file taxes +money due:2026-12-01\n" {
		t.Errorf("note = %q", got)
	}
	if task.Line != 6 || task.Due != "2026-12-01" || task.Status != "open" {
		t.Errorf("task = %+v", task)
	}
	out := call[tasktools.TaskListOutput](t, s, "task_list", map[string]any{"text": "taxes"})
	if len(out.Tasks) != 1 {
		t.Errorf("listed %v after the add, want the new task", lines(out))
	}
	if got := getError(t, s, "task_add", map[string]any{"path": "Gone.md", "text": "x"}); !strings.Contains(got, "not a note") {
		t.Errorf("error = %q", got)
	}
	if got := getError(t, s, "task_add", map[string]any{"path": "Home.md", "text": "a\nb"}); !strings.Contains(got, "one non-empty line") {
		t.Errorf("error = %q", got)
	}
}

func TestTaskTickChangesTheCheckbox(t *testing.T) {
	s := newSession(t)

	ticked := call[tasktools.TaskOutput](t, s, "task_set_done", map[string]any{"path": "Home.md", "line": 3, "done": true, "expectedText": "water plants"})

	if got, _ := s.vault.ReadContent("Home.md"); !strings.Contains(got, "- [x] water plants due:2026-11-14 done:2026-11-14\n") {
		t.Errorf("note = %q", got)
	}
	if ticked.Status != "done" || ticked.Path != "Home.md" || ticked.Line != 3 {
		t.Errorf("ticked = %+v", ticked)
	}
	reopened := call[tasktools.TaskOutput](t, s, "task_set_done", map[string]any{"path": "Home.md", "line": 3, "done": false, "expectedText": "water plants"})
	if got, _ := s.vault.ReadContent("Home.md"); got != homeNote || reopened.Status != "open" {
		t.Errorf("note = %q, task = %+v, want the first note back", got, reopened)
	}
}

func TestTaskTickRejectsAChangeTheTaskDoesNotAllow(t *testing.T) {
	cases := []struct {
		name string
		args map[string]any
		want string
	}{
		{"a done task ticked again", map[string]any{"path": "Home.md", "line": 4, "done": true, "expectedText": "buy milk"}, "not open"},
		{"an open task unticked", map[string]any{"path": "Home.md", "line": 3, "done": false, "expectedText": "water plants"}, "not done"},
		{"a heading", map[string]any{"path": "Home.md", "line": 1, "done": true, "expectedText": "Home"}, "not a task"},
		{"a missing note", map[string]any{"path": "Gone.md", "line": 1, "done": true, "expectedText": "x"}, "not a note"},
		{"a stale line", map[string]any{"path": "Home.md", "line": 3, "done": true, "expectedText": "call bank"}, "not the expected text"},
		{"no expected text", map[string]any{"path": "Home.md", "line": 3, "done": true, "expectedText": ""}, "expectedText is required"},
	}
	s := newSession(t)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := getError(t, s, "task_set_done", c.args); !strings.Contains(got, c.want) {
				t.Errorf("error = %q, want it to mention %q", got, c.want)
			}
			if got, _ := s.vault.ReadContent("Home.md"); got != homeNote {
				t.Errorf("note changed to %q", got)
			}
		})
	}
}

func TestTaskEditReplacesTheTextAndRefusesAChangedTask(t *testing.T) {
	s := newSession(t)

	task := call[tasktools.TaskOutput](t, s, "task_edit", map[string]any{"path": "Home.md", "line": 5, "text": "read two books", "expectedText": "read a book"})

	if got, _ := s.vault.ReadContent("Home.md"); !strings.HasSuffix(got, "- [ ] read two books\n") || task.Text != "read two books" {
		t.Errorf("note = %q, task = %+v", got, task)
	}
	if got := getError(t, s, "task_edit", map[string]any{"path": "Home.md", "line": 5, "text": "x", "expectedText": "read a book"}); !strings.Contains(got, "expected text") {
		t.Errorf("error = %q", got)
	}
	if got := getError(t, s, "task_edit", map[string]any{"path": "Home.md", "line": 1, "text": "x", "expectedText": "x"}); !strings.Contains(got, "not a task") {
		t.Errorf("error = %q", got)
	}
}

func TestTaskRemoveDeletesTheLine(t *testing.T) {
	s := newSession(t)

	removed := call[tasktools.TaskOutput](t, s, "task_remove", map[string]any{"path": "Home.md", "line": 4, "expectedText": "buy milk"})

	want := strings.Replace(homeNote, "- [x] buy milk done:2026-11-01\n", "", 1)
	if got, _ := s.vault.ReadContent("Home.md"); got != want || removed.Text != "buy milk" {
		t.Errorf("note = %q, removed = %+v", got, removed)
	}
	if got := getError(t, s, "task_remove", map[string]any{"path": "Home.md", "line": 1, "expectedText": "x"}); !strings.Contains(got, "not a task") {
		t.Errorf("error = %q", got)
	}
}

func TestTheRegistryDeclaresTheFiveToolsWithTheirEffects(t *testing.T) {
	s := newSession(t)

	listed, err := s.registry.ListTools(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	got := map[string]tools.Effect{}
	for _, tool := range listed.Tools {
		got[tool.Name] = tool.Effect
	}
	want := map[string]tools.Effect{
		"task_list": tools.ReadOnly, "task_add": tools.ReadWrite, "task_set_done": tools.ReadWrite,
		"task_edit": tools.ReadWrite, "task_remove": tools.ReadWrite,
	}
	if len(got) != len(want) {
		t.Fatalf("tools = %v, want %v", got, want)
	}
	for name, effect := range want {
		if got[name] != effect {
			t.Errorf("tool %s has effect %v, want %v", name, got[name], effect)
		}
	}
}
