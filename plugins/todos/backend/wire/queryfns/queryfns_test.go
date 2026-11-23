package queryfns_test

import (
	"context"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/searchfunction/searchfunctiontest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server/servertest"
	"github.com/jiva-studio/numen/plugins/todos/backend/testsupport"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/index"
	"github.com/jiva-studio/numen/plugins/todos/backend/wire/queryfns"
)

const (
	token = "todos-token"
	today = "2026-11-14"
)

type fixedClock struct{}

func (fixedClock) Now() time.Time {
	return time.Date(2026, time.November, 14, 12, 0, 0, 0, time.UTC)
}

// oneVault answers every call with the tasks of one vault.
type oneVault struct{ tasks *index.Index }

func (o oneVault) GetTasks(context.Context, string) (queryfns.Tasks, error) { return o.tasks, nil }

func getNotes() map[string]string {
	return map[string]string{
		"open.md":       "- [ ] plain\n",
		"done.md":       "- [x] finished done:2026-11-01 due:2026-11-02\n",
		"overdue.md":    "- [ ] late due:2026-11-10\n",
		"today.md":      "- [ ] now due:2026-11-14\n",
		"soon.md":       "- [ ] next week due:2026-11-20\n",
		"far.md":        "- [ ] much later due:2027-01-01\n",
		"project.md":    "- [ ] (B) ship +Work @desk Report\n",
		"donework.md":   "- [x] shipped +Work done:2026-11-01\n",
		"nothing.md":    "# no tasks here\n- plain list item\n",
		"upper.md":      "- [ ] CALL the Bank\n",
		"unreadable.md": "---\ntitle: never closed\n- [ ] hidden\n",
	}
}

func newClient(t *testing.T) *searchfunctiontest.Client {
	t.Helper()
	vault := hosttest.New("notes")
	for path, content := range getNotes() {
		vault.WriteFile(path, content)
	}
	tasks := testsupport.NewIndex(vault, vault, vault)
	if err := tasks.Scan(t.Context()); err != nil {
		t.Fatal(err)
	}
	registry, err := queryfns.New(oneVault{tasks: tasks}, fixedClock{})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := servertest.NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": token}, func(s *server.Server) error {
		registry.Mount(s)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)
	return searchfunctiontest.NewClient(httpServer.Client(), httpServer.URL, token)
}

func allPaths() []string {
	paths := make([]string, 0, len(getNotes()))
	for path := range getNotes() {
		paths = append(paths, path)
	}
	slices.Sort(paths)
	return paths
}

func call(function string, arguments map[string]any) searchfunctiontest.Call {
	return searchfunctiontest.Call{Function: function, VaultID: "notes", IndexGeneration: "g1", Arguments: arguments}
}

func TestFunctionsHoldForTheNotesWithAMatchingTask(t *testing.T) {
	cases := []struct {
		name string
		call searchfunctiontest.Call
		want []string
	}{
		{"open task", call("hasOpenTask", nil), []string{"far.md", "open.md", "overdue.md", "project.md", "soon.md", "today.md", "upper.md"}},
		{"done task", call("hasDoneTask", nil), []string{"done.md", "donework.md"}},
		{"overdue", call("hasTaskOverdue", nil), []string{"overdue.md"}},
		{"due today", call("hasTaskDueToday", nil), []string{"today.md"}},
		{"due within 0 days is today", call("hasTaskDueWithin", map[string]any{"days": int64(0)}), []string{"today.md"}},
		{"due within 7 days", call("hasTaskDueWithin", map[string]any{"days": int64(7)}), []string{"soon.md", "today.md"}},
		{"due before counts open and done", call("hasTaskDueBefore", map[string]any{"date": "2026-11-10"}), []string{"done.md", "overdue.md"}},
		{"due before includes the date", call("hasTaskDueBefore", map[string]any{"date": "2026-11-02"}), []string{"done.md"}},
		{"without due", call("hasTaskWithoutDue", nil), []string{"open.md", "project.md", "upper.md"}},
		{"project", call("hasTaskProject", map[string]any{"name": "Work"}), []string{"project.md"}},
		{"project with its plus sign", call("hasTaskProject", map[string]any{"name": "+Work"}), []string{"project.md"}},
		{"context", call("hasTaskContext", map[string]any{"name": "desk"}), []string{"project.md"}},
		{"priority", call("hasTaskPriority", map[string]any{"letter": "B"}), []string{"project.md"}},
		{"priority in lower case", call("hasTaskPriority", map[string]any{"letter": "b"}), []string{"project.md"}},
		{"text without case", call("hasTaskText", map[string]any{"text": "bank"}), []string{"upper.md"}},
		{"unknown project", call("hasTaskProject", map[string]any{"name": "Home"}), nil},
	}
	client := newClient(t)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			matched, warning, err := client.MatchDocuments(t.Context(), c.call, allPaths())
			if err != nil || warning != "" {
				t.Fatalf("MatchDocuments() = %v, %q, %v", matched, warning, err)
			}
			if !slices.Equal(matched, c.want) {
				t.Errorf("matched %v, want %v", matched, c.want)
			}
			found, count, _, err := client.FindDocuments(t.Context(), c.call, 100)
			if err != nil || !slices.Equal(found, c.want) || count != int64(len(c.want)) {
				t.Errorf("FindDocuments() = %v of %d, %v, want %v", found, count, err, c.want)
			}
		})
	}
}

func TestFindDocumentsLimitsThePathsAndKeepsTheCount(t *testing.T) {
	client := newClient(t)

	found, count, _, err := client.FindDocuments(t.Context(), call("hasOpenTask", nil), 2)

	if err != nil || !slices.Equal(found, []string{"far.md", "open.md"}) || count != 7 {
		t.Errorf("FindDocuments() = %v of %d, %v, want two of seven", found, count, err)
	}
}

func TestMatchDocumentsAnswersOnlyForThePathsAsked(t *testing.T) {
	client := newClient(t)

	matched, _, err := client.MatchDocuments(t.Context(), call("hasOpenTask", nil), []string{"done.md", "open.md", "missing.md"})

	if err != nil || !slices.Equal(matched, []string{"open.md"}) {
		t.Errorf("MatchDocuments() = %v, %v, want open.md", matched, err)
	}
}

func TestInvalidArgumentsFailTheQueryWithAMessage(t *testing.T) {
	cases := []struct {
		name string
		call searchfunctiontest.Call
		want string
	}{
		{"bad date", call("hasTaskDueBefore", map[string]any{"date": "2026-13-45"}), "YYYY-MM-DD"},
		{"empty date", call("hasTaskDueBefore", map[string]any{"date": ""}), "YYYY-MM-DD"},
		{"negative days", call("hasTaskDueWithin", map[string]any{"days": int64(-1)}), "number of days from 0"},
		{"empty project", call("hasTaskProject", map[string]any{"name": " + "}), "needs a name"},
		{"empty context", call("hasTaskContext", map[string]any{"name": ""}), "needs a name"},
		{"two letters", call("hasTaskPriority", map[string]any{"letter": "AB"}), "A to Z"},
		{"a digit", call("hasTaskPriority", map[string]any{"letter": "1"}), "A to Z"},
		{"empty text", call("hasTaskText", map[string]any{"text": "  "}), "needs text"},
	}
	client := newClient(t)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, _, matchErr := client.MatchDocuments(t.Context(), c.call, []string{"open.md"})
			_, _, _, findErr := client.FindDocuments(t.Context(), c.call, 10)

			for _, err := range []error{matchErr, findErr} {
				if err == nil || !strings.Contains(err.Error(), "invalid_argument") || !strings.Contains(err.Error(), c.want) {
					t.Errorf("err = %v, want invalid_argument naming %q", err, c.want)
				}
			}
		})
	}
}

func TestAnAnswerBeforeTheFirstScanCarriesAWarning(t *testing.T) {
	vault := hosttest.New("notes")
	registry, err := queryfns.New(oneVault{tasks: testsupport.NewIndex(vault, vault, vault)}, fixedClock{})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := servertest.NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": token}, func(s *server.Server) error {
		registry.Mount(s)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	httpServer := httptest.NewServer(handler)
	t.Cleanup(httpServer.Close)
	client := searchfunctiontest.NewClient(httpServer.Client(), httpServer.URL, token)

	_, warning, err := client.MatchDocuments(t.Context(), call("hasOpenTask", nil), []string{"a.md"})

	if err != nil || !strings.Contains(warning, "still being built") {
		t.Errorf("warning = %q, %v, want the not-ready warning", warning, err)
	}
}
