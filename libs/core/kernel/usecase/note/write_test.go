package note_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
)

func TestWrite_EnforcesSizeLimit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name           string
		size, maxBound int
		wantErr        bool
	}{
		{"no bound holds to nothing", 5, 0, false},
		{"a negative bound holds to nothing", 5, -1, false},
		{"a bound of one holds", 5, 1, true},
		{"exactly the bound is allowed", 5, 5, false},
		{"one over the bound fails", 6, 5, true},
		{"under the bound is allowed", 4, 5, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := edit.CheckSize("Entropy.md", tc.size, tc.maxBound)
			if got := errors.Is(err, edit.ErrTooLarge); got != tc.wantErr {
				t.Errorf("size %d at bound %d answered %v", tc.size, tc.maxBound, err)
			}
			if err != nil && !strings.Contains(err.Error(), "Entropy.md") {
				t.Errorf("the rejection does not name the note: %v", err)
			}
		})
	}
}

func TestAWriteOfABodyThatOpensWithFrontmatterIsRefused(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Entropy.md": "# Entropy\n"})

	_, err := c.newWrite().Execute(t.Context(), c.vault, "Entropy.md", "---\nid: x\n---\nMine.\n", domain.FileStat{})
	if !errors.Is(err, edit.ErrBodyUnwritable) {
		t.Fatalf("want ErrBodyUnwritable, got %v", err)
	}
	if body := c.read(t, "Entropy.md"); body != "# Entropy\n" {
		t.Errorf("a failed write changed the note:\n%s", body)
	}
}

func TestASaveOfABodyThatOpensWithFrontmatterIsRefused(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Entropy.md": "# Entropy\n"})

	_, err := c.newWrite().Save(t.Context(), c.vault, "Entropy.md", "---\nid: x\n---\nMine.\n", nil)
	if !errors.Is(err, edit.ErrBodyUnwritable) {
		t.Fatalf("want ErrBodyUnwritable, got %v", err)
	}
	if body := c.read(t, "Entropy.md"); body != "# Entropy\n" {
		t.Errorf("a rejected save changed the note:\n%s", body)
	}
}

func TestWrite_EnforcesWriterLimit(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Mammals.md": "# Mammals\n"})
	body := strings.Repeat("x", edit.MaxBytes+1)

	_, err := c.newWrite().Execute(t.Context(), c.vault, "Mammals.md", body, domain.FileStat{})
	if !errors.Is(err, edit.ErrTooLarge) {
		t.Fatalf("want ErrTooLarge, got %v", err)
	}
	if got := c.read(t, "Mammals.md"); got != "# Mammals\n" {
		t.Error("a failed write changed the note")
	}

	wideWrite := c.newWrite()
	wideWrite.Bound = 2 * edit.MaxBytes
	if _, err := wideWrite.Execute(t.Context(), c.vault, "Mammals.md", body, domain.FileStat{}); err != nil {
		t.Fatal(err)
	}
	if got := len(c.read(t, "Mammals.md")); got <= edit.MaxBytes {
		t.Errorf("the note holds %d bytes, and the wider bound allowed more", got)
	}
}

func (c changeFixture) newDrawingWrite(callback *[]domain.Edit) note.Write {
	w := c.newWrite()
	w.Drawing = func(_ context.Context, response domain.Edit) error {
		*callback = append(*callback, response)
		return nil
	}
	return w
}

func TestAWriteTellsTheStretchThatChanged(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, was, now string
		want           domain.Edit
	}{
		{
			name: "a stretch replaced", was: "alpha beta\n", now: "alpha delta\n",
			want: domain.Edit{From: 6, To: 10, Text: "delta"},
		},
		{
			name: "text only added", was: "alpha\n", now: "alpha\nbeta\n",
			want: domain.Edit{From: 6, To: 6, Text: "beta\n"},
		},
		{
			name: "text only removed", was: "alpha\nbeta\n", now: "alpha\n",
			want: domain.Edit{From: 6, To: 11, Text: ""},
		},
		{
			name: "positions count UTF-16 units", was: "é 😀 end\n", now: "é 😀 fin\n",
			want: domain.Edit{From: 5, To: 8, Text: "fin"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			c := openChangeFixture(t, map[string]string{"Entropy.md": tc.was})
			var notified []domain.Edit

			if _, err := c.newDrawingWrite(&notified).Execute(t.Context(), c.vault, "Entropy.md", tc.now, domain.FileStat{}); err != nil {
				t.Fatal(err)
			}
			if len(notified) != 2 {
				t.Fatalf("told %+v, want the change and its end", notified)
			}
			begin, end := notified[0], notified[1]
			if begin.Path != "Entropy.md" || begin.From != tc.want.From || begin.To != tc.want.To || begin.Text != tc.want.Text {
				t.Errorf("told %+v, want %+v", begin, tc.want)
			}
			if begin.IsDone || begin.Change == "" {
				t.Errorf("the change began as %+v", begin)
			}
			if !end.IsDone || end.Change != begin.Change || end.Path != "Entropy.md" {
				t.Errorf("the change ended as %+v", end)
			}
		})
	}
}

func TestAWriteOfTheSameProseTellsNothing(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Entropy.md": "alpha beta\n"})
	var notified []domain.Edit

	if _, err := c.newDrawingWrite(&notified).Execute(t.Context(), c.vault, "Entropy.md", "alpha beta\n", domain.FileStat{}); err != nil {
		t.Fatal(err)
	}
	if len(notified) != 0 {
		t.Errorf("told %+v of a note that did not change", notified)
	}
}

func TestARefusedWriteTellsNothing(t *testing.T) {
	t.Parallel()
	half := "---\nid: 01K5QF7T4ZPWY6X0N3EV8HMJRC\ntitle: Heat death\n"
	c := openChangeFixture(t, map[string]string{"Heat.md": half})
	var notified []domain.Edit

	_, err := c.newDrawingWrite(&notified).Execute(t.Context(), c.vault, "Heat.md", "# Heat death\n\nMine.\n", domain.FileStat{})
	if !errors.Is(err, edit.ErrUnterminated) {
		t.Fatalf("want ErrUnterminated, got %v", err)
	}
	if len(notified) != 0 {
		t.Errorf("told %+v of a change that failed", notified)
	}
}

func TestWrite_SmallerWriterLimitRejectsDefaultAllowed(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Mammals.md": "# Mammals\n"})
	small := c.newWrite()
	small.Bound = 100

	_, err := small.Execute(t.Context(), c.vault, "Mammals.md", strings.Repeat("x", 200), domain.FileStat{})
	if !errors.Is(err, edit.ErrTooLarge) {
		t.Fatalf("want ErrTooLarge, got %v", err)
	}
}

func TestAWriteUsesThePortsItIsGiven(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Entropy.md": "# Entropy\n"})
	updated := 0
	w := note.NewWrite(
		vaultreader.Readers{}, vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()},
		func(context.Context, domain.Vault, []string) error {
			updated++
			return nil
		}, testsupport.NewClock(),
		testsupport.OpenGate{},
	)

	if _, err := w.Execute(t.Context(), c.vault, "Entropy.md", "# Entropy\n\nMine.\n", domain.FileStat{}); err != nil {
		t.Fatal(err)
	}
	if body := c.read(t, "Entropy.md"); !strings.Contains(body, "Mine.") {
		t.Errorf("the note was not written:\n%s", body)
	}
	if updated != 1 {
		t.Errorf("the index was called %d times, want 1", updated)
	}
}

// An observer that cannot be notified of the change stops the write, and the note stays as it was.
func TestAWriteWhoseObserverCannotBeNotifiedIsRefused(t *testing.T) {
	t.Parallel()
	const was = "alpha beta\n"
	c := openChangeFixture(t, map[string]string{"Entropy.md": was})
	observerError := errors.New("the observer cannot be reached")
	writing := c.newWrite()
	writing.Drawing = func(context.Context, domain.Edit) error { return observerError }

	_, err := writing.Execute(t.Context(), c.vault, "Entropy.md", "alpha gamma\n", domain.FileStat{})

	if !errors.Is(err, observerError) {
		t.Fatalf("want the error of the observer, got %v", err)
	}
	if body := c.read(t, "Entropy.md"); body != was {
		t.Errorf("the refused write landed anyway:\n%s", body)
	}
}
