package edit_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/hosttest"
	"github.com/jiva-studio/numen/plugins/todos/backend/ports"
	"github.com/jiva-studio/numen/plugins/todos/backend/todo"
	"github.com/jiva-studio/numen/plugins/todos/backend/usecase/edit"
)

type recordingStore struct {
	paths []string
}

func (s *recordingStore) StoreNote(path string, _ []byte, _ domain.FileStat) {
	s.paths = append(s.paths, path)
}

// racingWriter replaces the note before each of the first stale writes, so the write finds the
// note changed since its read.
type racingWriter struct {
	vault  *hosttest.Vault
	stale  int
	writes int
}

func (w *racingWriter) Write(ctx context.Context, path string, content []byte, baseline domain.FileStat) (domain.FileStat, error) {
	w.writes++
	if w.writes <= w.stale {
		now, _ := w.vault.ReadContent(path)
		w.vault.WriteFile(path, now+"\nother window\n")
	}
	return w.vault.Write(ctx, path, content, baseline)
}

func setup(t *testing.T, note string) (*hosttest.Vault, *edit.Editor, *recordingStore) {
	t.Helper()
	vault := hosttest.New("notes")
	vault.WriteFile("Home.md", note)
	store := &recordingStore{}
	return vault, edit.New(vault, vault, store, ports.NewWriteLock()), store
}

func content(t *testing.T, vault *hosttest.Vault) string {
	t.Helper()
	text, ok := vault.ReadContent("Home.md")
	if !ok {
		t.Fatal("Home.md is gone")
	}
	return text
}

func TestAddAppendsAnOpenLine(t *testing.T) {
	cases := []struct {
		name string
		note string
		text string
		want string
		line int
	}{
		{"after a final line break", "# Home\n- [ ] one\n", "two +work due:2026-11-20", "# Home\n- [ ] one\n- [ ] two +work due:2026-11-20\n", 3},
		{"after a last line with no break", "- [ ] one", " two ", "- [ ] one\n- [ ] two\n", 2},
		{"in an empty note", "", "first", "- [ ] first\n", 1},
		{"after frontmatter", "---\ntitle: x\n---\n", "first", "---\ntitle: x\n---\n- [ ] first\n", 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			vault, editor, store := setup(t, c.note)

			task, err := editor.Add(t.Context(), "Home.md", c.text)

			if err != nil {
				t.Fatal(err)
			}
			if got := content(t, vault); got != c.want {
				t.Errorf("note = %q, want %q", got, c.want)
			}
			if task.Line != c.line || task.Path != "Home.md" || task.Status != "open" {
				t.Errorf("task = %+v, want an open task at line %d", task, c.line)
			}
			if len(store.paths) != 1 {
				t.Errorf("stored %v, want the note stored once", store.paths)
			}
		})
	}
}

func TestAddRefusesWhatIsNotOneLineOrNoNote(t *testing.T) {
	cases := []struct {
		name string
		path string
		text string
		note string
		want error
	}{
		{"empty text", "Home.md", "  ", "", edit.ErrInvalidText},
		{"a line break", "Home.md", "a\nb", "", edit.ErrInvalidText},
		{"a missing note", "Gone.md", "a", "", edit.ErrNoNote},
		{"a note with a frontmatter never closed", "Home.md", "a", "---\ntitle: x\n", edit.ErrUnreadable},
		{"a note with a code fence never closed", "Home.md", "a", "```\ncode\n", edit.ErrNotTask},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			vault, editor, _ := setup(t, c.note)

			_, err := editor.Add(t.Context(), c.path, c.text)

			if !errors.Is(err, c.want) {
				t.Errorf("err = %v, want %v", err, c.want)
			}
			if got := content(t, vault); got != c.note {
				t.Errorf("note = %q, want it unchanged", got)
			}
		})
	}
}

func TestEditReplacesTheTextAfterTheCheckbox(t *testing.T) {
	cases := []struct {
		name string
		note string
		line int
		text string
		want string
	}{
		{"an open task", "# a\n- [ ] old due:2026-11-01\nend\n", 2, "new due:2026-12-01", "# a\n- [ ] new due:2026-12-01\nend\n"},
		{"a done task keeps its box", "- [x] old done:2026-11-01\n", 1, "new done:2026-11-01", "- [x] new done:2026-11-01\n"},
		{"indentation and CRLF", "- [ ] a\r\n  - [ ] sub\r\n", 2, "child", "- [ ] a\r\n  - [ ] child\r\n"},
		{"a quoted task", "> [!todo]\n> - [ ] call\n", 2, "email", "> [!todo]\n> - [ ] email\n"},
		{"a last line with no break", "- [ ] a\n- [ ] b", 2, "c", "- [ ] a\n- [ ] c"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			vault, editor, _ := setup(t, c.note)

			found := taskAt(t, c.note, c.line)

			task, err := editor.Edit(t.Context(), "Home.md", c.line, found.Text, c.text)

			if err != nil {
				t.Fatal(err)
			}
			if got := content(t, vault); got != c.want {
				t.Errorf("note = %q, want %q", got, c.want)
			}
			if task.Line != c.line || task.Text == "" {
				t.Errorf("task = %+v, want the task at line %d", task, c.line)
			}
		})
	}
}

func TestEditAndRemoveRefuseALineThatIsNotTheTaskExpected(t *testing.T) {
	note := "# Title\n- [ ] one\n- plain\n- [ ] two\n"
	cases := []struct {
		name     string
		line     int
		expected string
		want     error
	}{
		{"a heading", 1, "one", edit.ErrNotTask},
		{"a list item", 3, "one", edit.ErrNotTask},
		{"beyond the note", 9, "one", edit.ErrNotTask},
		{"no expected text", 2, "", edit.ErrNoExpected},
		{"other text", 2, "two", edit.ErrChanged},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			vault, editor, store := setup(t, note)

			_, editErr := editor.Edit(t.Context(), "Home.md", c.line, c.expected, "x")
			_, removeErr := editor.Remove(t.Context(), "Home.md", c.line, c.expected)

			if !errors.Is(editErr, c.want) || !errors.Is(removeErr, c.want) {
				t.Errorf("errors = %v and %v, want %v", editErr, removeErr, c.want)
			}
			if got := content(t, vault); got != note || len(store.paths) != 0 {
				t.Errorf("note = %q, stored %v, want nothing written", got, store.paths)
			}
		})
	}
}

func TestEditRefusesATextThatIsNotATask(t *testing.T) {
	vault, editor, _ := setup(t, "- [ ] one\n")

	_, err := editor.Edit(t.Context(), "Home.md", 1, "one", "  \t")

	if !errors.Is(err, edit.ErrInvalidText) || content(t, vault) != "- [ ] one\n" {
		t.Errorf("err = %v, want ErrInvalidText with the note unchanged", err)
	}
}

func TestRemoveDeletesTheLineAndKeepsEveryOtherByte(t *testing.T) {
	cases := []struct {
		name string
		note string
		line int
		want string
	}{
		{"a middle line", "a\n- [ ] one\nb\n", 2, "a\nb\n"},
		{"the last line with no break", "a\n- [ ] one", 2, "a\n"},
		{"a CRLF line", "a\r\n- [ ] one\r\nb\r\n", 2, "a\r\nb\r\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			vault, editor, _ := setup(t, c.note)

			removed, err := editor.Remove(t.Context(), "Home.md", c.line, "one")

			if err != nil {
				t.Fatal(err)
			}
			if got := content(t, vault); got != c.want {
				t.Errorf("note = %q, want %q", got, c.want)
			}
			if removed.Text != "one" || removed.Line != c.line {
				t.Errorf("removed = %+v, want the task one", removed)
			}
		})
	}
}

func TestAWriteOverANoteChangedMeanwhileIsRetriedOnceThenRefused(t *testing.T) {
	cases := []struct {
		name       string
		stale      int
		wantErr    error
		wantWrites int
	}{
		{"one change is retried", 1, nil, 2},
		{"two changes are refused", 2, edit.ErrStale, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			vault := hosttest.New("notes")
			vault.WriteFile("Home.md", "- [ ] one\n")
			writer := &racingWriter{vault: vault, stale: c.stale}
			editor := edit.New(vault, writer, &recordingStore{}, ports.NewWriteLock())

			_, err := editor.Edit(t.Context(), "Home.md", 1, "one", "two")

			if !errors.Is(err, c.wantErr) || writer.writes != c.wantWrites {
				t.Errorf("err = %v after %d writes, want %v after %d", err, writer.writes, c.wantErr, c.wantWrites)
			}
		})
	}
}

func TestFindReturnsTheTaskAtTheLine(t *testing.T) {
	_, editor, _ := setup(t, "# a\n- [ ] one +x\n")

	task, err := editor.Find(t.Context(), "Home.md", 2, "one")
	_, missing := editor.Find(t.Context(), "Gone.md", 1, "one")
	_, stale := editor.Find(t.Context(), "Home.md", 2, "other")
	_, empty := editor.Find(t.Context(), "Home.md", 2, "")

	if err != nil || task.Raw != "- [ ] one +x" || task.Path != "Home.md" {
		t.Errorf("Find() = %+v, %v, want the task at line 2", task, err)
	}
	if !errors.Is(missing, edit.ErrNoNote) {
		t.Errorf("Find() of a missing note = %v, want ErrNoNote", missing)
	}
	if !errors.Is(stale, edit.ErrChanged) {
		t.Errorf("Find() of other text = %v, want ErrChanged", stale)
	}
	if !errors.Is(empty, edit.ErrNoExpected) {
		t.Errorf("Find() with no expected text = %v, want ErrNoExpected", empty)
	}
}

func taskAt(t *testing.T, note string, line int) todo.Task {
	t.Helper()
	tasks, err := todo.ParseNote([]byte(note))
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range tasks {
		if task.Line == line {
			return task
		}
	}
	t.Fatalf("no task at line %d of %q", line, note)
	return todo.Task{}
}
