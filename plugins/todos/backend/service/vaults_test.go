package service

import (
	"encoding/json"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/searchfunction/searchfunctiontest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/server/servertest"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
	"github.com/jiva-studio/numen/plugins/todos/backend/wire/queryfns"
	"github.com/jiva-studio/numen/plugins/todos/backend/wire/tasktools"
)

type testClock struct{}

func (testClock) GetToday() string { return today }

// twoVaults is vault a, which the host shows, and vault b, with the vaults a call opened.
type twoVaults struct {
	a, b   *hosttest.Vault
	vaults *Vaults

	mu     sync.Mutex
	opened []domain.VaultID
}

func newTwoVaults(t *testing.T) *twoVaults {
	t.Helper()
	w := &twoVaults{a: hosttest.New("a")}
	w.a.File("n.md", "- [ ] in a due:2026-11-01\n")
	added, err := w.a.AddVault(t.Context(), "/vaults/b", "b")
	if err != nil {
		t.Fatal(err)
	}
	w.b = w.a.With(added.ID)
	w.b.File("n.md", "- [ ] in b\n- [ ] more in b\n")
	w.vaults = NewVaults(func(id domain.VaultID) Notes {
		w.mu.Lock()
		w.opened = append(w.opened, id)
		w.mu.Unlock()
		return w.a.With(id)
	})
	return w
}

func (w *twoVaults) invoke(t *testing.T, vaultID, name string, args any) (string, bool) {
	t.Helper()
	registry, err := tasktools.New(w.vaults, testClock{}).Registry()
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
	text, _ := v.Content(path)
	return text
}

func TestToolsListTheVaultTheCallNames(t *testing.T) {
	w := newTwoVaults(t)

	got, isError := w.invoke(t, "b", "task_list", map[string]any{})
	gotA, _ := w.invoke(t, "a", "task_list", map[string]any{})

	if isError || !strings.Contains(got, `"total":2`) || strings.Contains(got, "in a") {
		t.Errorf("task_list of b = %s, want the two tasks of b only", got)
	}
	if !strings.Contains(gotA, `"total":1`) || strings.Contains(gotA, "in b") {
		t.Errorf("task_list of a = %s, want the task of a only", gotA)
	}
}

func TestToolsWriteOnlyTheVaultTheCallNames(t *testing.T) {
	a := "- [ ] in a due:2026-11-01\n"
	cases := []struct {
		name string
		tool string
		args map[string]any
		want string
	}{
		{"add", "task_add", map[string]any{"path": "n.md", "text": "new"}, "- [ ] in b\n- [ ] more in b\n- [ ] new\n"},
		{"tick", "task_set_done", map[string]any{"path": "n.md", "line": 1, "done": true}, "- [x] in b done:2026-11-14\n- [ ] more in b\n"},
		{"edit", "task_edit", map[string]any{"path": "n.md", "line": 1, "text": "changed"}, "- [ ] changed\n- [ ] more in b\n"},
		{"remove", "task_remove", map[string]any{"path": "n.md", "line": 1}, "- [ ] more in b\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := newTwoVaults(t)

			text, isError := w.invoke(t, "b", c.tool, c.args)

			if isError {
				t.Fatal(text)
			}
			if got := content(t, w.b, "n.md"); got != c.want {
				t.Errorf("vault b = %q, want %q", got, c.want)
			}
			if got := content(t, w.a, "n.md"); got != a {
				t.Errorf("vault a = %q, want it unchanged", got)
			}
			if !slices.Equal(w.opened, []domain.VaultID{"b"}) {
				t.Errorf("opened %v, want only vault b", w.opened)
			}
		})
	}
}

func TestToolsRefuseACallWithNoVaultAndTouchNothing(t *testing.T) {
	calls := map[string]map[string]any{
		"task_list":     {},
		"task_add":      {"path": "n.md", "text": "new"},
		"task_set_done": {"path": "n.md", "line": 1, "done": true},
		"task_edit":     {"path": "n.md", "line": 1, "text": "x"},
		"task_remove":   {"path": "n.md", "line": 1},
	}
	for name, args := range calls {
		t.Run(name, func(t *testing.T) {
			w := newTwoVaults(t)

			text, isError := w.invoke(t, "", name, args)

			if !isError || !strings.Contains(text, "names no vault") {
				t.Errorf("%s = %q, want a refusal naming no vault", name, text)
			}
			if len(w.opened) != 0 || content(t, w.a, "n.md") != "- [ ] in a due:2026-11-01\n" {
				t.Errorf("opened %v, vault a = %q, want nothing read or written", w.opened, content(t, w.a, "n.md"))
			}
		})
	}
}

func TestToolsRefuseAnUnknownVaultWithoutFallingBackToTheShownOne(t *testing.T) {
	w := newTwoVaults(t)

	text, isError := w.invoke(t, "missing", "task_add", map[string]any{"path": "n.md", "text": "new"})

	if !isError {
		t.Errorf("task_add = %q, want an error", text)
	}
	if got := content(t, w.a, "n.md"); got != "- [ ] in a due:2026-11-01\n" {
		t.Errorf("vault a = %q, want it unchanged", got)
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

func TestFunctionsAnswerForTheVaultTheCallNames(t *testing.T) {
	w := newTwoVaults(t)
	client := newFunctionClient(t, w.vaults)
	overdue := searchfunctiontest.Call{Function: "hasTaskOverdue", VaultID: "a"}
	anyOpen := searchfunctiontest.Call{Function: "hasTaskWithoutDue", VaultID: "b"}

	inA, _, errA := client.MatchDocuments(t.Context(), overdue, []string{"n.md"})
	inB, _, errB := client.MatchDocuments(t.Context(), searchfunctiontest.Call{Function: "hasTaskOverdue", VaultID: "b"}, []string{"n.md"})
	withoutDue, _, errNoDue := client.MatchDocuments(t.Context(), anyOpen, []string{"n.md"})

	if errA != nil || !slices.Equal(inA, []string{"n.md"}) {
		t.Errorf("vault a overdue = %v, %v, want n.md", inA, errA)
	}
	if errB != nil || len(inB) != 0 {
		t.Errorf("vault b overdue = %v, %v, want none", inB, errB)
	}
	if errNoDue != nil || !slices.Equal(withoutDue, []string{"n.md"}) {
		t.Errorf("vault b without due = %v, %v, want n.md", withoutDue, errNoDue)
	}
}

func TestFunctionsRefuseACallWithNoVault(t *testing.T) {
	w := newTwoVaults(t)
	client := newFunctionClient(t, w.vaults)
	call := searchfunctiontest.Call{Function: "hasOpenTask", VaultID: ""}

	_, _, matchErr := client.MatchDocuments(t.Context(), call, []string{"n.md"})
	_, _, _, findErr := client.FindDocuments(t.Context(), call, 10)

	for _, err := range []error{matchErr, findErr} {
		if err == nil || !strings.Contains(err.Error(), "names no vault") {
			t.Errorf("err = %v, want a refusal naming no vault", err)
		}
	}
	if len(w.opened) != 0 {
		t.Errorf("opened %v, want no vault read", w.opened)
	}
}
