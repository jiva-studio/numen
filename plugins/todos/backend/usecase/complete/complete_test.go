package complete

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/todos/backend/todo"
)

const today = "2026-11-14"

// writer is the vault of the test with its writes counted. Before each of the first stale
// writes, another window replaces the note with what edit returns, so that write finds the
// note changed since its read.
type writer struct {
	vault *hosttest.Vault
	edit  func(content string) string
	stale int

	mu     sync.Mutex
	writes int
}

func (w *writer) Write(ctx context.Context, path string, content []byte, baseline domain.FileStat) (domain.FileStat, error) {
	w.mu.Lock()
	w.writes++
	isStale := w.writes <= w.stale
	w.mu.Unlock()
	if isStale {
		now, _ := w.vault.Content(path)
		w.vault.File(path, w.edit(now))
	}
	return w.vault.Write(ctx, path, content, baseline)
}

func (w *writer) count() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writes
}

func setup(t *testing.T, path, content string) (*hosttest.Vault, *writer, *SetTaskDone) {
	t.Helper()
	vault := hosttest.New("notes")
	vault.File(path, content)
	w := &writer{vault: vault}
	return vault, w, New(vault, w)
}

func execute(t *testing.T, u *SetTaskDone, req Request) Result {
	t.Helper()
	result, err := u.Execute(t.Context(), req)
	if err != nil {
		t.Fatalf("Execute(%+v) error = %v", req, err)
	}
	return result
}

func content(t *testing.T, vault *hosttest.Vault, path string) string {
	t.Helper()
	text, ok := vault.Content(path)
	if !ok {
		t.Fatalf("%s is gone", path)
	}
	return text
}

func tick(path string, line int, raw string) Request {
	return Request{Path: path, Line: line, Raw: raw, IsDone: true, Today: today}
}

func TestExecuteTicksTheOneLineAndKeepsEveryOtherByte(t *testing.T) {
	note := "---\ntitle: Home\ntags: [a]\n---\n\n# Chores\n- [ ] fix the tap\n- [ ] buy milk +shop\n  - [ ] sub  task\n\ntext after\n"
	vault, w, u := setup(t, "Home.md", note)

	got := execute(t, u, tick("Home.md", 8, "- [ ] buy milk +shop"))

	want := "---\ntitle: Home\ntags: [a]\n---\n\n# Chores\n- [ ] fix the tap\n- [x] buy milk +shop done:2026-11-14\n  - [ ] sub  task\n\ntext after\n"
	if written := content(t, vault, "Home.md"); written != want {
		t.Fatalf("note = %q, want %q", written, want)
	}
	if got.Refusal != "" || got.Change != todo.ChangeCompleted {
		t.Errorf("result = %+v, want a completed change", got)
	}
	if task := got.Task; task.Path != "Home.md" || task.Line != 8 || task.Status != todo.StatusDone ||
		task.DoneOn != today || task.Raw != "- [x] buy milk +shop done:2026-11-14" || task.Text != "buy milk" {
		t.Errorf("task = %+v", task)
	}
	if string(got.Content) != want {
		t.Errorf("content = %q, want the written note", got.Content)
	}
	stat, err := vault.StatFile(t.Context(), "Home.md")
	if err != nil || !got.Stat.IsUnchanged(stat) {
		t.Errorf("stat = %+v, want the stat after the write %+v (%v)", got.Stat, stat, err)
	}
	if w.count() != 1 {
		t.Errorf("writes = %d, want the whole note written once", w.count())
	}
}

func TestExecuteKeepsTheLineEndingsTheByteOrderMarkAndTheFrontmatter(t *testing.T) {
	cases := []struct {
		name string
		note string
		line int
		raw  string
		want string
	}{
		{
			name: "CRLF endings and a byte order mark before the frontmatter",
			note: "\ufeff---\r\ntitle: Plan\r\n---\r\n- [ ] first\r\n- [ ] second\r\n",
			line: 5, raw: "- [ ] second",
			want: "\ufeff---\r\ntitle: Plan\r\n---\r\n- [ ] first\r\n- [x] second done:2026-11-14\r\n",
		},
		{
			name: "a byte order mark before a task on the first line",
			note: "\ufeff- [ ] first\n- [ ] second",
			line: 1, raw: "\ufeff- [ ] first",
			want: "\ufeff- [x] first done:2026-11-14\n- [ ] second",
		},
		{
			name: "a last line with no line break",
			note: "- [ ] first\n- [ ] last",
			line: 2, raw: "- [ ] last",
			want: "- [ ] first\n- [x] last done:2026-11-14",
		},
		{
			name: "a task in a callout keeps its quote prefix",
			note: "> [!todo] Today\r\n> - [ ] call the bank\r\n",
			line: 2, raw: "> - [ ] call the bank",
			want: "> [!todo] Today\r\n> - [x] call the bank done:2026-11-14\r\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vault, _, u := setup(t, "Plan.md", tc.note)

			got := execute(t, u, tick("Plan.md", tc.line, tc.raw))

			if written := content(t, vault, "Plan.md"); written != tc.want || got.Refusal != "" {
				t.Errorf("note = %q refusal %q, want %q", written, got.Refusal, tc.want)
			}
		})
	}
}

func TestExecuteUnticksATickedTaskAndRemovesItsCompletionDate(t *testing.T) {
	vault, _, u := setup(t, "a.md", "- [x] buy milk done:2026-11-10 ^milk\n- [ ] other\n")

	got := execute(t, u, Request{Path: "a.md", Line: 1, Raw: "- [x] buy milk done:2026-11-10 ^milk", IsDone: false, Today: today})

	if written := content(t, vault, "a.md"); written != "- [ ] buy milk ^milk\n- [ ] other\n" {
		t.Errorf("note = %q", written)
	}
	if got.Change != todo.ChangeReopened || got.Task.Status != todo.StatusOpen || got.Task.DoneOn != "" {
		t.Errorf("result = %+v", got)
	}
}

func TestExecuteFindsALineThatMovedByItsText(t *testing.T) {
	vault, _, u := setup(t, "a.md", "- [ ] one\n- [ ] two\n")
	vault.File("a.md", "new first\nnew second\n- [ ] one\n- [ ] two\n")

	got := execute(t, u, tick("a.md", 2, "- [ ] two"))

	if written := content(t, vault, "a.md"); written != "new first\nnew second\n- [ ] one\n- [x] two done:2026-11-14\n" {
		t.Errorf("note = %q", written)
	}
	if got.Task.Line != 4 {
		t.Errorf("task line = %d, want the line it moved to, 4", got.Task.Line)
	}
}

func TestExecuteFindsALineWhoseNumberIsPastTheEndOfTheNote(t *testing.T) {
	vault, _, u := setup(t, "a.md", "- [ ] one\n- [ ] two\n")

	execute(t, u, tick("a.md", 40, "- [ ] one"))

	if written := content(t, vault, "a.md"); written != "- [x] one done:2026-11-14\n- [ ] two\n" {
		t.Errorf("note = %q", written)
	}
}

func TestExecuteTicksTheIdenticalLineNearestToTheOldNumber(t *testing.T) {
	cases := []struct {
		name string
		note string
		line int
		want string
	}{
		{
			name: "the nearer one below",
			note: "- [ ] water\nx\nx\nx\nx\n- [ ] water\n",
			line: 4,
			want: "- [ ] water\nx\nx\nx\nx\n- [x] water done:2026-11-14\n",
		},
		{
			name: "the earlier one on a tie",
			note: "x\n- [ ] water\nx\nx\nx\n- [ ] water\n",
			line: 4,
			want: "x\n- [x] water done:2026-11-14\nx\nx\nx\n- [ ] water\n",
		},
		{
			name: "the one at the line number when its text is the same",
			note: "- [ ] water\n- [ ] water\n- [ ] water\n",
			line: 2,
			want: "- [ ] water\n- [x] water done:2026-11-14\n- [ ] water\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vault, _, u := setup(t, "a.md", tc.note)

			execute(t, u, tick("a.md", tc.line, "- [ ] water"))

			if written := content(t, vault, "a.md"); written != tc.want {
				t.Errorf("note = %q, want %q", written, tc.want)
			}
		})
	}
}

func TestExecuteRefusesAChangeTheNoteDoesNotAllowAndWritesNothing(t *testing.T) {
	cases := []struct {
		name string
		note string
		req  Request
		want Refusal
	}{
		{
			name: "a line whose text changed",
			note: "- [ ] buy milk and bread\n",
			req:  tick("a.md", 1, "- [ ] buy milk"),
			want: RefusalNotFound,
		},
		{
			name: "a note whose frontmatter is never closed",
			note: "---\ntitle: x\n- [ ] buy milk\n",
			req:  tick("a.md", 3, "- [ ] buy milk"),
			want: RefusalChanged,
		},
		{
			name: "a tick of a done task",
			note: "- [x] buy milk\n",
			req:  tick("a.md", 1, "- [x] buy milk"),
			want: RefusalNotOpen,
		},
		{
			name: "a tick of a cancelled task",
			note: "- [-] buy milk\n",
			req:  tick("a.md", 1, "- [-] buy milk"),
			want: RefusalNotOpen,
		},
		{
			name: "an untick of an open task",
			note: "- [ ] buy milk\n",
			req:  Request{Path: "a.md", Line: 1, Raw: "- [ ] buy milk", IsDone: false, Today: today},
			want: RefusalNotDone,
		},
		{
			name: "a line of the text inside a code fence",
			note: "```\n- [ ] buy milk\n```\n",
			req:  tick("a.md", 2, "- [ ] buy milk"),
			want: RefusalNotFound,
		},
		{
			name: "a line of the text inside an HTML comment",
			note: "<!--\n- [ ] buy milk\n-->\n",
			req:  tick("a.md", 2, "- [ ] buy milk"),
			want: RefusalNotFound,
		},
		{
			name: "a tick of a task with an invalid recurrence",
			note: "- [ ] buy milk rec:0w\n",
			req:  tick("a.md", 1, "- [ ] buy milk rec:0w"),
			want: RefusalInvalidRecurrence,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			vault, w, u := setup(t, "a.md", tc.note)

			got := execute(t, u, tc.req)

			if got.Refusal != tc.want {
				t.Errorf("refusal = %q, want %q", got.Refusal, tc.want)
			}
			if written := content(t, vault, "a.md"); written != tc.note || w.count() != 0 {
				t.Errorf("note = %q after %d writes, want %q untouched", written, w.count(), tc.note)
			}
		})
	}
}

func TestExecuteReadsANoteThatChangedSinceTheReadOnceMoreAndKeepsTheOtherChange(t *testing.T) {
	vault, w, u := setup(t, "a.md", "- [ ] one\n- [ ] two\n")
	w.edit = func(now string) string { return "added above\n" + now }
	w.stale = 1

	got := execute(t, u, tick("a.md", 2, "- [ ] two"))

	if written := content(t, vault, "a.md"); written != "added above\n- [ ] one\n- [x] two done:2026-11-14\n" {
		t.Errorf("note = %q, want the other change kept and the line ticked", written)
	}
	if got.Refusal != "" || got.Task.Line != 3 || w.count() != 2 {
		t.Errorf("result = %+v after %d writes", got, w.count())
	}
}

func TestExecuteRefusesANoteThatChangedSinceTheReadTwiceAsStale(t *testing.T) {
	vault, w, u := setup(t, "a.md", "- [ ] one\n")
	w.edit = func(now string) string { return now + "typed\n" }
	w.stale = 2

	got := execute(t, u, tick("a.md", 1, "- [ ] one"))

	if got.Refusal != RefusalStale {
		t.Errorf("refusal = %q, want %q", got.Refusal, RefusalStale)
	}
	if written := content(t, vault, "a.md"); written != "- [ ] one\ntyped\ntyped\n" || w.count() != 2 {
		t.Errorf("note = %q after %d writes, want the other window's text only", written, w.count())
	}
}

func TestExecuteReturnsTheHostFailureAsAnError(t *testing.T) {
	t.Run("no vault", func(t *testing.T) {
		vault, _, u := setup(t, "a.md", "- [ ] one\n")
		vault.Hide()

		_, err := u.Execute(t.Context(), tick("a.md", 1, "- [ ] one"))

		if !errors.Is(err, host.ErrNoVault) {
			t.Errorf("error = %v, want %v", err, host.ErrNoVault)
		}
	})
	t.Run("a note the plugin may not write", func(t *testing.T) {
		vault, _, u := setup(t, "a.md", "- [ ] one\n")
		vault.RefuseWriting("a.md", host.ErrPermissionDenied)

		_, err := u.Execute(t.Context(), tick("a.md", 1, "- [ ] one"))

		if !errors.Is(err, host.ErrPermissionDenied) {
			t.Errorf("error = %v, want %v", err, host.ErrPermissionDenied)
		}
		if written := content(t, vault, "a.md"); written != "- [ ] one\n" {
			t.Errorf("note = %q", written)
		}
	})
	t.Run("a note that is gone", func(t *testing.T) {
		vault, _, u := setup(t, "a.md", "- [ ] one\n")
		vault.Remove("a.md")

		got, err := u.Execute(t.Context(), tick("a.md", 1, "- [ ] one"))

		if err == nil && got.Refusal != RefusalNotFound {
			t.Errorf("result = %+v, want an error or %q", got, RefusalNotFound)
		}
	})
}

func TestExecuteWritesTwoTicksOfOneNoteBothAtOnce(t *testing.T) {
	var lines []string
	for _, name := range []string{"a", "b", "c", "d", "e", "f"} {
		lines = append(lines, "- [ ] "+name)
	}
	vault, _, u := setup(t, "a.md", strings.Join(lines, "\n")+"\n")

	var wg sync.WaitGroup
	results := make([]Result, len(lines))
	failures := make([]error, len(lines))
	for n, line := range lines {
		wg.Go(func() { results[n], failures[n] = u.Execute(t.Context(), tick("a.md", n+1, line)) })
	}
	wg.Wait()

	for n := range lines {
		if failures[n] != nil || results[n].Refusal != "" {
			t.Errorf("tick of line %d = %+v %v", n+1, results[n], failures[n])
		}
	}
	want := strings.ReplaceAll(strings.Join(lines, " done:2026-11-14\n")+" done:2026-11-14\n", "[ ]", "[x]")
	if written := content(t, vault, "a.md"); written != want {
		t.Errorf("note = %q, want every line ticked %q", written, want)
	}
}
