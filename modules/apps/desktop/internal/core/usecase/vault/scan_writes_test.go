package vault_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/testsupport"
)

// groupedWrites stands in for the index and remembers how it was called, which
// is the thing under test: not what was stored, but in how many pieces.
type groupedWrites struct {
	groups [][]string
	fail   error
}

func (g *groupedWrites) Save(_ context.Context, _ string, notes []domain.Note) error {
	if g.fail != nil {
		return g.fail
	}
	paths := make([]string, len(notes))
	for i, n := range notes {
		paths[i] = n.Ref.Path
	}
	g.groups = append(g.groups, paths)
	return nil
}

func (g *groupedWrites) Remove(context.Context, string, []string) error { return nil }

type countedMeasurements struct{ n int }

func (c *countedMeasurements) Update(context.Context) error { c.n++; return nil }

// TestNotesAreWrittenInGroups holds the scan to writing in batches. Note by note
// is correct and three times slower, and nothing about the result says which one
// happened — so the number of writes is what has to be asserted.
func TestNotesAreWrittenInGroups(t *testing.T) {
	const notes = 600
	v := testsupport.GenerateVault(t, notes)
	written := &groupedWrites{}
	db := openIndex(t)

	scan := usecase.Scan{
		Readers:    filesystem.Readers{},
		Vaults:     db.Vaults(),
		Notes:      written,
		Known:      db.Queries(),
		Statistics: &countedMeasurements{},
	}
	res, err := scan.Execute(t.Context(), v)
	if err != nil {
		t.Fatal(err)
	}

	if res.Indexed != notes {
		t.Fatalf("indexed %d of %d", res.Indexed, notes)
	}
	if len(written.groups) != 2 {
		t.Errorf("wrote %d groups, want 2 — %d notes should not be %d transactions",
			len(written.groups), notes, len(written.groups))
	}
	seen := 0
	for _, group := range written.groups {
		if len(group) > 500 {
			t.Errorf("a group held %d notes: a vault of long files would be read into memory whole", len(group))
		}
		seen += len(group)
	}
	if seen != notes {
		t.Errorf("%d notes reached the index, want %d", seen, notes)
	}
}

// TestAFailedWriteCountsNothing: a group that did not reach the index is not
// indexed, however many notes were parsed into it. Counting them would report a
// scan as complete and leave the next one believing the files are up to date.
func TestAFailedWriteCountsNothing(t *testing.T) {
	v := testsupport.GenerateVault(t, 10)
	refused := errors.New("disk full")
	db := openIndex(t)

	scan := usecase.Scan{
		Readers:    filesystem.Readers{},
		Vaults:     db.Vaults(),
		Notes:      &groupedWrites{fail: refused},
		Known:      db.Queries(),
		Statistics: &countedMeasurements{},
	}
	res, err := scan.Execute(t.Context(), v)
	if !errors.Is(err, refused) {
		t.Fatalf("error = %v, want the one the index gave", err)
	}
	if res.Indexed != 0 {
		t.Errorf("counted %d notes as indexed that were never written", res.Indexed)
	}
}

// TestTheIndexIsMeasuredWhenItChanges. The measurement is what makes the
// difference between answering "what points here" through an index and reading
// every link in the vault. Nothing about a scan's result shows whether it
// happened, so it is asserted directly — and a scan that changed nothing must
// not pay for it, because an unchanged vault is scanned at every startup.
func TestTheIndexIsMeasuredWhenItChanges(t *testing.T) {
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)
	measured := &countedMeasurements{}

	scan := usecase.Scan{
		Readers:    readers,
		Vaults:     db.Vaults(),
		Notes:      db.Notes(),
		Known:      db.Queries(),
		Statistics: measured,
	}
	if _, err := scan.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	if measured.n != 1 {
		t.Fatalf("a scan that filled the index measured it %d times, want 1", measured.n)
	}

	if _, err := scan.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}
	if measured.n != 1 {
		t.Errorf("a scan that changed nothing measured the index again")
	}
}
