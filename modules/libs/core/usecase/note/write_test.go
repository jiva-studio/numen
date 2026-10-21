package note_test

import (
	"context"
	"errors"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func TestASizeIsHeldToItsBound(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name        string
		size, bound int
		refused     bool
	}{
		{"no bound holds to nothing", 5, 0, false},
		{"a negative bound holds to nothing", 5, -1, false},
		{"a bound of one holds", 5, 1, true},
		{"exactly the bound is allowed", 5, 5, false},
		{"one over the bound is refused", 6, 5, true},
		{"under the bound is allowed", 4, 5, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := note.CheckSize("Entropy.md", tc.size, tc.bound)
			if got := errors.Is(err, note.ErrTooLarge); got != tc.refused {
				t.Errorf("size %d at bound %d answered %v", tc.size, tc.bound, err)
			}
			if err != nil && !strings.Contains(err.Error(), "Entropy.md") {
				t.Errorf("the refusal does not name the note: %v", err)
			}
		})
	}
}

func TestAWriteOfABodyThatOpensWithFrontmatterIsRefused(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Entropy.md": "# Entropy\n"})

	_, err := c.newWrite().Execute(t.Context(), c.vault, "Entropy.md", "---\nid: x\n---\nMine.\n", domain.Fingerprint{})
	if !errors.Is(err, note.ErrBodyUnwritable) {
		t.Fatalf("want ErrBodyUnwritable, got %v", err)
	}
	if body := c.read(t, "Entropy.md"); body != "# Entropy\n" {
		t.Errorf("a refused write changed the note:\n%s", body)
	}
}

func TestASaveOfABodyThatOpensWithFrontmatterIsRefused(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Entropy.md": "# Entropy\n"})

	_, err := c.newWrite().Save(t.Context(), c.vault, "Entropy.md", "---\nid: x\n---\nMine.\n", nil)
	if !errors.Is(err, note.ErrBodyUnwritable) {
		t.Fatalf("want ErrBodyUnwritable, got %v", err)
	}
	if body := c.read(t, "Entropy.md"); body != "# Entropy\n" {
		t.Errorf("a refused save changed the note:\n%s", body)
	}
}

func TestAWriteIsHeldToTheBoundOfItsWriter(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Mammals.md": "# Mammals\n"})
	body := strings.Repeat("x", note.MaxBytes+1)

	_, err := c.newWrite().Execute(t.Context(), c.vault, "Mammals.md", body, domain.Fingerprint{})
	if !errors.Is(err, note.ErrTooLarge) {
		t.Fatalf("want ErrTooLarge, got %v", err)
	}
	if got := c.read(t, "Mammals.md"); got != "# Mammals\n" {
		t.Error("a refused write changed the note")
	}

	wideWrite := c.newWrite()
	wideWrite.Bound = 2 * note.MaxBytes
	if _, err := wideWrite.Execute(t.Context(), c.vault, "Mammals.md", body, domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}
	if got := len(c.read(t, "Mammals.md")); got <= note.MaxBytes {
		t.Errorf("the note holds %d bytes, and the wider bound allowed more", got)
	}
}

// newDrawingWrite is a writer that records what it is told while a note is being drawn.
func (c changing) newDrawingWrite(told *[]domain.Edit) note.Write {
	w := c.newWrite()
	w.Drawing = func(_ context.Context, said domain.Edit) { *told = append(*told, said) }
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
			c := openChanging(t, map[string]string{"Entropy.md": tc.was})
			var told []domain.Edit

			if _, err := c.newDrawingWrite(&told).Execute(t.Context(), c.vault, "Entropy.md", tc.now, domain.Fingerprint{}); err != nil {
				t.Fatal(err)
			}
			if len(told) != 2 {
				t.Fatalf("told %+v, want the change and its end", told)
			}
			begin, end := told[0], told[1]
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
	c := openChanging(t, map[string]string{"Entropy.md": "alpha beta\n"})
	var told []domain.Edit

	if _, err := c.newDrawingWrite(&told).Execute(t.Context(), c.vault, "Entropy.md", "alpha beta\n", domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}
	if len(told) != 0 {
		t.Errorf("told %+v of a note that did not change", told)
	}
}

func TestARefusedWriteTellsNothing(t *testing.T) {
	t.Parallel()
	half := "---\nid: 01K5QF7T4ZPWY6X0N3EV8HMJRC\ntitle: Heat death\n"
	c := openChanging(t, map[string]string{"Heat.md": half})
	var told []domain.Edit

	_, err := c.newDrawingWrite(&told).Execute(t.Context(), c.vault, "Heat.md", "# Heat death\n\nMine.\n", domain.Fingerprint{})
	if !errors.Is(err, note.ErrUnterminated) {
		t.Fatalf("want ErrUnterminated, got %v", err)
	}
	if len(told) != 0 {
		t.Errorf("told %+v of a change that was refused", told)
	}
}

func TestAWriterWithASmallerBoundRefusesWhatTheDefaultAllows(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Mammals.md": "# Mammals\n"})
	small := c.newWrite()
	small.Bound = 100

	_, err := small.Execute(t.Context(), c.vault, "Mammals.md", strings.Repeat("x", 200), domain.Fingerprint{})
	if !errors.Is(err, note.ErrTooLarge) {
		t.Fatalf("want ErrTooLarge, got %v", err)
	}
}

func TestAWriteThroughReplacedPortsUsesThem(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Entropy.md": "# Entropy\n"})
	levelled := 0
	w := note.NewWrite(brokenReaders{open: errDiskGone}, brokenWriters{hold: errDiskGone}, nil, time.Now).
		WithReaders(filesystem.VaultReaders{}).
		WithWriters(filesystem.VaultWriters{}).
		WithIndex(func(context.Context, domain.Vault, []string) error {
			levelled++
			return nil
		})

	if _, err := w.Execute(t.Context(), c.vault, "Entropy.md", "# Entropy\n\nMine.\n", domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}
	if body := c.read(t, "Entropy.md"); !strings.Contains(body, "Mine.") {
		t.Errorf("the note was not written:\n%s", body)
	}
	if levelled != 1 {
		t.Errorf("the replaced index was called %d times, want 1", levelled)
	}
}
