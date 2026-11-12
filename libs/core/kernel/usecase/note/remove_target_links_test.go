package note_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// heatWithEntropyLinks is a note with three records to Entropy, of different types and roles, and
// one record to Order.
const heatWithEntropyLinks = "---\nid: 01K5QF7T4ZPWY6X0N3EV8HMJRC\nlinks:\n" +
	"  - to: Entropy\n    role: parent\n    type: series\n" +
	"  - to: Order\n    role: ref\n" +
	"  - to: Entropy\n    role: ref\n" +
	"  - to: Entropy\n    role: parent\n    type: topic\n" +
	"---\n# Heat\n"

func getEntropyAddress() domain.Address {
	return domain.Address{Scheme: domain.SchemeName, Value: "Entropy"}
}

// countingWriters opens writers that count the files they write.
type countingWriters struct {
	vaultwriter.VaultWriters
	writes *int
}

func (w countingWriters) OpenNoteWriter(v domain.Vault) (vaultport.NoteWriter, error) {
	inner, err := w.VaultWriters.OpenNoteWriter(v)
	if err != nil {
		return nil, err
	}
	return countingWriter{NoteWriter: inner, writes: w.writes}, nil
}

type countingWriter struct {
	vaultport.NoteWriter
	writes *int
}

func (w countingWriter) Write(ctx context.Context, path string, content []byte, stat domain.FileStat) (domain.FileStat, error) {
	*w.writes++
	return w.NoteWriter.Write(ctx, path, content, stat)
}

func TestRemoveTargetLinksRemovesEveryRecordToTheTargetInOneWrite(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Heat.md": heatWithEntropyLinks})
	writes := 0
	linking := c.newEditLinks().WithWriters(countingWriters{VaultWriters: vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}, writes: &writes})

	stat, err := linking.RemoveTargetLinks(t.Context(), c.vault, "Heat.md", getEntropyAddress(), "", domain.FileStat{})
	if err != nil {
		t.Fatalf("the removal failed: %v", err)
	}

	body := c.read(t, "Heat.md")
	if strings.Contains(body, "Entropy") {
		t.Errorf("a record to Entropy is still written:\n%s", body)
	}
	if !strings.Contains(body, "  - to: Order\n    role: ref\n") || !strings.Contains(body, "# Heat\n") {
		t.Errorf("the record to Order or the body was lost:\n%s", body)
	}
	if writes != 1 {
		t.Errorf("the removal wrote the note %d times, want 1", writes)
	}
	if stat.Size != int64(len(body)) {
		t.Errorf("the removal returned the file stat %+v, want the size of the written file, %d", stat, len(body))
	}
}

func TestRemoveTargetLinksWithARoleRemovesOnlyThatRole(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Heat.md": heatWithEntropyLinks})

	if _, err := c.newEditLinks().RemoveTargetLinks(t.Context(), c.vault, "Heat.md", getEntropyAddress(), domain.RoleParent, domain.FileStat{}); err != nil {
		t.Fatalf("the removal failed: %v", err)
	}

	body := c.read(t, "Heat.md")
	if n := strings.Count(body, "to: Entropy"); n != 1 {
		t.Errorf("%d records to Entropy are written, want the one of role ref:\n%s", n, body)
	}
	if !strings.Contains(body, "  - to: Entropy\n    role: ref\n") || !strings.Contains(body, "  - to: Order\n    role: ref\n") {
		t.Errorf("the records of role ref are not both kept:\n%s", body)
	}
	if strings.Contains(body, "type: series") || strings.Contains(body, "type: topic") {
		t.Errorf("a record of role parent to Entropy is still written:\n%s", body)
	}
}

func TestRemoveTargetLinksOfUnlinkedTargetIsRejected(t *testing.T) {
	t.Parallel()
	for name, role := range map[string]domain.LinkRole{"any role": "", "a role no record has": domain.RoleChild} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			c := openChangeFixture(t, map[string]string{"Heat.md": heatWithEntropyLinks})
			to := domain.Address{Scheme: domain.SchemeName, Value: "Chaos"}
			if role != "" {
				to = getEntropyAddress()
			}

			_, err := c.newEditLinks().RemoveTargetLinks(t.Context(), c.vault, "Heat.md", to, role, domain.FileStat{})

			if err == nil || !strings.Contains(err.Error(), "no link to") || !strings.Contains(err.Error(), to.Value) ||
				!strings.Contains(err.Error(), "is written here") {
				t.Errorf("want the error no link to %s is written here, got %v", to.Value, err)
			}
			if body := c.read(t, "Heat.md"); body != heatWithEntropyLinks {
				t.Errorf("the note was written anyway:\n%s", body)
			}
		})
	}
}

func TestRemoveTargetLinksOfANoteThatChangedSinceItWasReadIsStale(t *testing.T) {
	t.Parallel()
	c := openChangeFixture(t, map[string]string{"Heat.md": heatWithEntropyLinks})
	read := c.openNote(t, "Heat.md").FileStat
	theirs := heatWithEntropyLinks + "\nSaved in the meantime.\n"
	c.overwrite(t, "Heat.md", theirs)

	_, err := c.newEditLinks().RemoveTargetLinks(t.Context(), c.vault, "Heat.md", getEntropyAddress(), "", read)

	if !errors.Is(err, edit.ErrStale) {
		t.Errorf("want ErrStale, got %v", err)
	}
	if body := c.read(t, "Heat.md"); body != theirs {
		t.Errorf("the removal wrote over a note that had changed:\n%s", body)
	}
}
