package vault_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// groupedWrites stands in for the index and remembers how it was called: not
// what was stored, but in how many pieces.
type groupedWrites struct {
	groups [][]string
	fail   error
}

func (g *groupedWrites) Save(_ context.Context, _ domain.VaultID, notes []domain.Note) error {
	if g.fail != nil {
		return g.fail
	}
	paths := make([]string, len(notes))
	for i, n := range notes {
		paths[i] = n.Fingerprint.Path
	}
	g.groups = append(g.groups, paths)
	return nil
}

func (g *groupedWrites) Remove(context.Context, domain.VaultID, []string) error { return nil }

type countedMeasurements struct{ n int }

func (c *countedMeasurements) ReportChanges(context.Context) error { c.n++; return nil }

// TestNotesAreWrittenInGroups. Nothing about a scan's result says how many
// writes it took, so that is what is asserted.
func TestNotesAreWrittenInGroups(t *testing.T) {
	t.Parallel()
	const notes = 600
	v := testsupport.GenerateVault(t, notes)
	written := &groupedWrites{}
	db := openIndex(t)

	scan := vaults.Scan{
		Readers:     filesystem.VaultReaders{},
		Vaults:      db.Vaults(),
		Notes:       written,
		Known:       db.Queries(),
		Maintenance: &countedMeasurements{},
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

// TestALongNoteClosesTheGroupEarly covers the size bound. No other test or
// benchmark has a vault of long files, so nothing else reaches it.
func TestALongNoteClosesTheGroupEarly(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	const notes, each = 12, 1 << 20
	body := strings.Repeat("entropy thermodynamics observer ", each/32)
	for i := range notes {
		name := filepath.Join(root, fmt.Sprintf("transcript-%02d.md", i))
		if err := os.WriteFile(name, []byte("# Transcript\n\n"+body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := filesystem.Initialize(root, filesystem.DefaultServiceDir, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	v := domain.Vault{ID: domain.VaultID(cfg.ID), Name: "transcripts", Path: root}

	written := &groupedWrites{}
	db := openIndex(t)
	scan := vaults.Scan{
		Readers:     filesystem.VaultReaders{},
		Vaults:      db.Vaults(),
		Notes:       written,
		Known:       db.Queries(),
		Maintenance: &countedMeasurements{},
	}
	if _, err := scan.Execute(t.Context(), v); err != nil {
		t.Fatal(err)
	}

	if len(written.groups) < 2 {
		t.Fatalf("%d notes of a megabyte each were written in %d group(s): the size bound never closed one",
			notes, len(written.groups))
	}
	if len(written.groups[0]) > 9 {
		t.Errorf("the first group held %d megabytes before it was written", len(written.groups[0]))
	}
}

// TestAFailedWriteCountsNothing: a group that did not reach the index is not
// indexed, however many notes were parsed into it.
func TestAFailedWriteCountsNothing(t *testing.T) {
	t.Parallel()
	v := testsupport.GenerateVault(t, 10)
	refused := errors.New("disk full")
	db := openIndex(t)

	scan := vaults.Scan{
		Readers:     filesystem.VaultReaders{},
		Vaults:      db.Vaults(),
		Notes:       &groupedWrites{fail: refused},
		Known:       db.Queries(),
		Maintenance: &countedMeasurements{},
	}
	res, err := scan.Execute(t.Context(), v)
	if !errors.Is(err, refused) {
		t.Fatalf("error = %v, want the one the index gave", err)
	}
	if res.Indexed != 0 {
		t.Errorf("counted %d notes as indexed that were never written", res.Indexed)
	}
}

// TestTheIndexIsMeasuredWhenItChanges, and only then: an unchanged vault is
// scanned at every startup and must not pay for it.
func TestTheIndexIsMeasuredWhenItChanges(t *testing.T) {
	t.Parallel()
	v, readers := vaultAt(t, testsupport.VaultDir(t))
	db := openIndex(t)
	measured := &countedMeasurements{}

	scan := vaults.Scan{
		Readers:     readers,
		Vaults:      db.Vaults(),
		Notes:       db.Notes(),
		Known:       db.Queries(),
		Maintenance: measured,
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
