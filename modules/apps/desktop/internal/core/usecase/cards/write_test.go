package cards_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/cards"
)

// counted is a writer that says how many times a file was replaced, so a test
// about one write can say it was one.
type counted struct {
	port.VaultWriters
	writes int
}

func (w *counted) Open(v domain.Vault) (port.VaultWriter, error) {
	writer, err := w.VaultWriters.Open(v)
	if err != nil {
		return nil, err
	}
	return &countedWriter{VaultWriter: writer, on: w}, nil
}

type countedWriter struct {
	port.VaultWriter
	on *counted
}

func (w *countedWriter) Write(
	ctx context.Context, path string, content []byte, fingerprint domain.FileRef,
) (domain.FileRef, error) {
	w.on.writes++
	return w.VaultWriter.Write(ctx, path, content, fingerprint)
}

// The faces are the body and the fields are one key of the frontmatter, and
// both are the one file, so one write carries both. Two writes leave the file
// in a shape nobody asked for whenever the second of them does not land.
func TestAStencilsFacesAndItsFieldsAreOneWrite(t *testing.T) {
	vs := indexed(t)
	writers := &counted{VaultWriters: filesystem.Writers{}}
	u := cards.Write{Readers: filesystem.Readers{}, Writers: writers}

	body := "\n## Recognise\n\n### Front\n\n{{Name}}\n\n### Back\n\n{{Wingspan}}\n"
	at, err := u.Stencil(
		t.Context(), vs.first, "Animal.md", body,
		[]string{"Name", "Wingspan"}, domain.FileRef{})
	if err != nil {
		t.Fatalf("write: %v", err)
	}
	if writers.writes != 1 {
		t.Errorf("the stencil was written %d times", writers.writes)
	}

	held := read(t, vs.first, "Animal.md")
	if !strings.Contains(held, "fields:\n  - Name\n  - Wingspan\n") {
		t.Errorf("the stencil declares something else: %q", held)
	}
	if !strings.Contains(held, "{{Wingspan}}") {
		t.Errorf("the faces are not the ones written: %q", held)
	}
	if !strings.Contains(held, "mine: keep me verbatim") {
		t.Errorf("a key nobody owns was rewritten: %q", held)
	}
	// The fingerprint that comes back is the file on disk, so the next write of
	// it lands.
	if at.Size != int64(len(held)) {
		t.Errorf("the fingerprint is not the file: %+v", at)
	}
}

// A stencil already declaring exactly these fields, in this order, is not
// written a second time, so what the person wrote around them stands.
func TestAStencilAlreadyDeclaringTheseFieldsKeepsWhatStandsAroundThem(t *testing.T) {
	vs := indexed(t)
	write(t, vs.first, "Kept.md", "---\ntype: stencil\nfields:\n"+
		"  # the one the card is named by\n  - Name\n  - Height\n---\n"+
		"\n## Recognise\n\n### Front\n\n{{Name}}\n")

	u := cards.Write{Readers: filesystem.Readers{}, Writers: filesystem.Writers{}}
	if _, err := u.Stencil(
		t.Context(), vs.first, "Kept.md",
		"\n## Recognise\n\n### Front\n\n{{Height}}\n",
		[]string{"Name", "Height"}, domain.FileRef{},
	); err != nil {
		t.Fatalf("write: %v", err)
	}

	held := read(t, vs.first, "Kept.md")
	if !strings.Contains(held, "  # the one the card is named by\n") {
		t.Errorf("the fields were written over: %q", held)
	}
	if !strings.Contains(held, "{{Height}}") {
		t.Errorf("the faces are not the ones written: %q", held)
	}
}
