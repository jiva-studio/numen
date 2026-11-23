package service

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/searchfunction/searchfunctiontest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server/servertest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/todos/backend/ports"
	"github.com/jiva-studio/numen/plugins/todos/backend/testsupport"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/complete"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/edit"
	"github.com/jiva-studio/numen/plugins/todos/backend/wire/queryfns"
	"github.com/jiva-studio/numen/plugins/todos/backend/wire/tasktools"
)

type testClock struct{}

func (testClock) Now() time.Time {
	return time.Date(2026, time.November, 14, 12, 0, 0, 0, time.UTC)
}

// twoVaults is vault a, which the host shows and the plugin keeps the index of, and vault b, which it does not.
type twoVaults struct {
	a, b   *hosttest.Vault
	vaults *Vaults
}

func newTwoVaults(t *testing.T) *twoVaults {
	t.Helper()
	w := &twoVaults{a: hosttest.New("a")}
	w.a.WriteFile("n.md", "- [ ] in a due:2026-11-01\n")
	added, err := w.a.AddVault(t.Context(), "/vaults/b", "b")
	if err != nil {
		t.Fatal(err)
	}
	w.b = w.a.With(added.ID)
	w.b.WriteFile("n.md", "- [ ] in b\n- [ ] more in b\n")
	tasks := testsupport.NewIndex(w.a, w.a, w.a)
	if err := tasks.Scan(t.Context()); err != nil {
		t.Fatal(err)
	}
	lock := ports.NewWriteLock()
	scope := tasktools.Scope{Tasks: tasks, Editor: edit.New(w.a, w.a, tasks, lock), Completer: complete.New(w.a, w.a, tasks, lock)}
	w.vaults = NewVaults(func(context.Context) (domain.VaultID, error) { return "a", nil }, tasks, scope)
	return w
}

func (w *twoVaults) invoke(t *testing.T, vaultID, name string, args any) (string, bool) {
	t.Helper()
	registry, err := tasktools.New(w.vaults, testClock{}).BuildRegistry()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	res, err := registry.InvokeTool(t.Context(), tools.Call{Name: name, Arguments: raw, VaultID: vaultID})
	if err != nil {
		t.Fatal(err)
	}
	result := res.GetResult()
	if result.GetIsError() {
		return result.GetText(), true
	}
	return string(result.GetStructured()), false
}

func content(t *testing.T, v *hosttest.Vault, path string) string {
	t.Helper()
	text, _ := v.ReadContent(path)
	return text
}

func TestToolsListTheTasksOfTheShownVault(t *testing.T) {
	w := newTwoVaults(t)

	got, isError := w.invoke(t, "a", "task_list", map[string]any{})

	if isError || !strings.Contains(got, `"total":1`) || strings.Contains(got, "in b") {
		t.Errorf("task_list of a = %s, want the task of a only", got)
	}
}

func TestToolsWriteOnlyTheShownVault(t *testing.T) {
	b := "- [ ] in b\n- [ ] more in b\n"
	cases := []struct {
		name string
		tool string
		args map[string]any
		want string
	}{
		{"add", "task_add", map[string]any{"path": "n.md", "text": "new"}, "- [ ] in a due:2026-11-01\n- [ ] new\n"},
		{"tick", "task_set_done", map[string]any{"path": "n.md", "line": 1, "done": true, "expectedText": "in a"}, "- [x] in a due:2026-11-01 done:2026-11-14\n"},
		{"edit", "task_edit", map[string]any{"path": "n.md", "line": 1, "text": "changed", "expectedText": "in a"}, "- [ ] changed\n"},
		{"remove", "task_remove", map[string]any{"path": "n.md", "line": 1, "expectedText": "in a"}, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := newTwoVaults(t)

			text, isError := w.invoke(t, "a", c.tool, c.args)

			if isError {
				t.Fatal(text)
			}
			if got := content(t, w.a, "n.md"); got != c.want {
				t.Errorf("vault a = %q, want %q", got, c.want)
			}
			if got := content(t, w.b, "n.md"); got != b {
				t.Errorf("vault b = %q, want it unchanged", got)
			}
		})
	}
}

func TestToolsRefuseACallWithNoVaultAndTouchNothing(t *testing.T) {
	calls := map[string]map[string]any{
		"task_list":     {},
		"task_add":      {"path": "n.md", "text": "new"},
		"task_set_done": {"path": "n.md", "line": 1, "done": true, "expectedText": "in a"},
		"task_edit":     {"path": "n.md", "line": 1, "text": "x", "expectedText": "in a"},
		"task_remove":   {"path": "n.md", "line": 1, "expectedText": "in a"},
	}
	for name, args := range calls {
		t.Run(name, func(t *testing.T) {
			w := newTwoVaults(t)

			text, isError := w.invoke(t, "", name, args)

			if !isError || !strings.Contains(text, "names no vault") {
				t.Errorf("%s = %q, want a refusal naming no vault", name, text)
			}
			if got := content(t, w.a, "n.md"); got != "- [ ] in a due:2026-11-01\n" {
				t.Errorf("vault a = %q, want it unchanged", got)
			}
		})
	}
}

func TestToolsRefuseAVaultThatIsNotShown(t *testing.T) {
	for _, vaultID := range []string{"b", "missing"} {
		t.Run(vaultID, func(t *testing.T) {
			w := newTwoVaults(t)

			text, isError := w.invoke(t, vaultID, "task_add", map[string]any{"path": "n.md", "text": "new"})

			if !isError || !strings.Contains(text, "does not show") {
				t.Errorf("task_add = %q, want a refusal of a vault that is not shown", text)
			}
			if content(t, w.a, "n.md") != "- [ ] in a due:2026-11-01\n" || content(t, w.b, "n.md") != "- [ ] in b\n- [ ] more in b\n" {
				t.Error("a refused call wrote a vault")
			}
		})
	}
}

func newFunctionClient(t *testing.T, vaults *Vaults) *searchfunctiontest.Client {
	t.Helper()
	registry, err := queryfns.New(vaults, testClock{})
	if err != nil {
		t.Fatal(err)
	}
	handler, err := servertest.NewHandler(map[string]string{"NUMEN_PLUGIN_TOKEN": "t"}, func(s *server.Server) error {
		registry.Mount(s)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	web := httptest.NewServer(handler)
	t.Cleanup(web.Close)
	return searchfunctiontest.NewClient(web.Client(), web.URL, "t")
}

func TestFunctionsAnswerForTheShownVault(t *testing.T) {
	w := newTwoVaults(t)
	client := newFunctionClient(t, w.vaults)
	call := searchfunctiontest.Call{Function: "hasTaskOverdue", VaultID: "a"}

	got, _, err := client.MatchDocuments(t.Context(), call, []string{"n.md"})

	if err != nil || !slices.Equal(got, []string{"n.md"}) {
		t.Errorf("overdue in a = %v, %v, want n.md", got, err)
	}
}

func TestFunctionsRefuseACallWithNoVaultOrAVaultThatIsNotShown(t *testing.T) {
	w := newTwoVaults(t)
	client := newFunctionClient(t, w.vaults)
	for vaultID, want := range map[string]string{"": "names no vault", "b": "does not show"} {
		call := searchfunctiontest.Call{Function: "hasOpenTask", VaultID: vaultID}

		_, _, matchErr := client.MatchDocuments(t.Context(), call, []string{"n.md"})
		_, _, _, findErr := client.FindDocuments(t.Context(), call, 10)

		for _, err := range []error{matchErr, findErr} {
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("vault %q: err = %v, want a refusal with %q", vaultID, err, want)
			}
		}
	}
}
