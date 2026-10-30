package note_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

const (
	heatNote    = "---\ntitle: Heat\nid: 01K5QF7T4ZPWY6X0N3EV8HMJRC\n---\n# Heat\n"
	entropyNote = "---\ntitle: Entropy\nid: 01K5QF7T4ZPWY6X0N3EV8HMJRD\n---\n# Entropy\n"
)

func (c changing) addresses() *note.GetLinkAddress {
	return note.NewGetLinkAddress(c.db.Queries())
}

func (c changing) saveConnection() note.SaveConnection {
	return note.NewSaveConnection(
		vaultreader.Readers{}, vaultwriter.VaultWriters{}, c.index, time.Now, c.db.Links(), c.addresses())
}

func (c changing) saveConnectionWith(writers vaultport.VaultWriters) note.SaveConnection {
	return note.NewSaveConnection(
		vaultreader.Readers{}, writers, c.index, time.Now, c.db.Links(), c.addresses())
}

func (c changing) removeConnection() note.RemoveConnection {
	return note.NewRemoveConnection(
		vaultreader.Readers{}, vaultwriter.VaultWriters{}, c.index, time.Now, c.db.Links(), c.addresses())
}

func (c changing) removeSide() note.RemoveSide {
	return note.NewRemoveSide(
		vaultreader.Readers{}, vaultwriter.VaultWriters{}, c.index, time.Now, c.db.Links(), c.addresses())
}

func (c changing) join() note.Join {
	return note.NewJoin(
		vaultreader.Readers{}, vaultwriter.VaultWriters{}, c.index, time.Now, c.db.Links(), c.addresses())
}

func (c changing) connection(t *testing.T, a, b string) domain.Connection {
	t.Helper()
	got, err := note.NewGetConnection(c.db.Links()).Execute(t.Context(), c.vault, a, b, "")
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestSavingAConnectionWritesBothSidesWithTheirOwnLabelAndMarker(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": heatNote, "Entropy.md": entropyNote})

	err := c.saveConnection().Execute(t.Context(), c.vault, domain.Connection{
		A: "Heat.md", B: "Entropy.md",
		AB: &domain.Side{Role: domain.RoleParent, Label: "measured by", Marker: domain.MarkerArrow},
		BA: &domain.Side{Role: domain.RoleChild, Label: "measures"},
	})
	if err != nil {
		t.Fatal(err)
	}

	got := c.connection(t, "Heat.md", "Entropy.md")
	if got.AB == nil || got.AB.Label != "measured by" || got.AB.Marker != domain.MarkerArrow || got.AB.Role != domain.RoleParent {
		t.Errorf("A to B = %+v", got.AB)
	}
	if got.BA == nil || got.BA.Label != "measures" || got.BA.Marker != domain.MarkerNone || got.BA.Role != domain.RoleChild {
		t.Errorf("B to A = %+v", got.BA)
	}
}

func TestASideWithAnEmptyLabelAndAnArrowIsSaved(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": heatNote, "Entropy.md": entropyNote})

	err := c.saveConnection().Execute(t.Context(), c.vault, domain.Connection{
		A: "Heat.md", B: "Entropy.md",
		AB: &domain.Side{Role: domain.RoleJump, Marker: domain.MarkerArrow},
	})
	if err != nil {
		t.Fatal(err)
	}

	got := c.connection(t, "Heat.md", "Entropy.md")
	if got.AB == nil || got.AB.Label != "" || got.AB.Marker != domain.MarkerArrow {
		t.Errorf("the arrow of a side with no label was lost: %+v", got.AB)
	}
	if got.BA != nil {
		t.Errorf("Entropy wrote a record it was not given: %+v", got.BA)
	}
	if body := c.read(t, "Entropy.md"); strings.Contains(body, "links:") {
		t.Errorf("Entropy was written to:\n%s", body)
	}
}

func TestSavingAConnectionWithoutASideRemovesThatRecord(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": heatNote, "Entropy.md": entropyNote})
	both := domain.Connection{
		A: "Heat.md", B: "Entropy.md",
		AB: &domain.Side{Role: domain.RoleJump}, BA: &domain.Side{Role: domain.RoleJump},
	}
	if err := c.saveConnection().Execute(t.Context(), c.vault, both); err != nil {
		t.Fatal(err)
	}

	both.BA = nil
	if err := c.saveConnection().Execute(t.Context(), c.vault, both); err != nil {
		t.Fatal(err)
	}

	got := c.connection(t, "Heat.md", "Entropy.md")
	if got.AB == nil || got.BA != nil {
		t.Errorf("connection = %+v", got)
	}
}

func TestSavingAConnectionKeepsTheTypeAndTheReasonItIsGiven(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": heatNote, "Entropy.md": entropyNote})
	side := &domain.Side{Role: domain.RoleParent, Type: "requires", Why: "only eigenvectors"}
	if err := c.saveConnection().Execute(t.Context(), c.vault,
		domain.Connection{A: "Heat.md", B: "Entropy.md", AB: side}); err != nil {
		t.Fatal(err)
	}

	got, err := note.NewGetConnection(c.db.Links()).Execute(t.Context(), c.vault, "Heat.md", "Entropy.md", "requires")
	if err != nil {
		t.Fatal(err)
	}
	if got.AB == nil || got.AB.Type != "requires" || got.AB.Why != "only eigenvectors" {
		t.Errorf("A to B = %+v", got.AB)
	}
	if untyped := c.connection(t, "Heat.md", "Entropy.md"); untyped.AB != nil {
		t.Errorf("a typed record answers for the untyped connection: %+v", untyped.AB)
	}
}

func TestSavingAConnectionReplacesTheRecordOfAnotherAddress(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{
		"Heat.md":    "---\nid: 01K5QF7T4ZPWY6X0N3EV8HMJRC\nlinks:\n  - to: \"[[Entropy]]\"\n    role: parent\n    label: old\n---\n# Heat\n",
		"Entropy.md": entropyNote,
	})

	if err := c.saveConnection().Execute(t.Context(), c.vault, domain.Connection{
		A: "Heat.md", B: "Entropy.md", AB: &domain.Side{Role: domain.RoleJump, Label: "new"},
	}); err != nil {
		t.Fatal(err)
	}

	if body := c.read(t, "Heat.md"); strings.Count(body, "to:") != 1 || !strings.Contains(body, "label: new") {
		t.Errorf("the record was duplicated or not replaced:\n%s", body)
	}
}

func TestSavingAConnectionWritesNothingWhenOneNoteCannotBeEdited(t *testing.T) {
	t.Parallel()
	inline := "---\nid: 01K5QF7T4ZPWY6X0N3EV8HMJRD\nlinks: [Heat]\n---\n# Entropy\n"
	c := openChanging(t, map[string]string{"Heat.md": heatNote, "Entropy.md": inline})

	err := c.saveConnection().Execute(t.Context(), c.vault, domain.Connection{
		A: "Heat.md", B: "Entropy.md",
		AB: &domain.Side{Role: domain.RoleJump}, BA: &domain.Side{Role: domain.RoleJump},
	})

	if err == nil {
		t.Fatal("a connection was saved into a note that cannot hold it")
	}
	if got := c.read(t, "Heat.md"); got != heatNote {
		t.Errorf("Heat was written although Entropy was not:\n%s", got)
	}
}

// pathFailingWriters fail the write of one path.
type pathFailingWriters struct {
	vaultport.VaultWriters
	path string
}

type pathFailingWriter struct {
	vaultport.VaultWriter
	path string
}

var errPathDisk = errors.New("the disk refused the write")

func (w pathFailingWriters) Open(v domain.Vault) (vaultport.VaultWriter, error) {
	inner, err := w.VaultWriters.Open(v)
	if err != nil {
		return nil, err
	}
	return pathFailingWriter{VaultWriter: inner, path: w.path}, nil
}

func (w pathFailingWriter) Write(
	ctx context.Context, path string, content []byte, stat domain.FileStat,
) (domain.FileStat, error) {
	if path == w.path {
		return domain.FileStat{}, errPathDisk
	}
	return w.VaultWriter.Write(ctx, path, content, stat)
}

func TestAWriteThatFailsAfterTheFirstNoteLandedRestoresTheFirstNote(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": heatNote, "Entropy.md": entropyNote})
	writers := pathFailingWriters{VaultWriters: vaultwriter.VaultWriters{}, path: "Entropy.md"}

	err := c.saveConnectionWith(writers).Execute(t.Context(), c.vault, domain.Connection{
		A: "Heat.md", B: "Entropy.md",
		AB: &domain.Side{Role: domain.RoleJump}, BA: &domain.Side{Role: domain.RoleJump},
	})

	if !errors.Is(err, errPathDisk) {
		t.Fatalf("err = %v, want the failed write", err)
	}
	if got := c.read(t, "Heat.md"); got != heatNote {
		t.Errorf("Heat kept half a connection:\n%s", got)
	}
	if got := c.connection(t, "Heat.md", "Entropy.md"); got.AB != nil || got.BA != nil {
		t.Errorf("the index kept half a connection: %+v", got)
	}
}

func TestAConnectionThatClosesTheHierarchyIsRefusedAndWritesNothing(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{
		"A.md": "---\nid: 01K5QF7T4ZPWY6X0N3EV8HMJR1\n---\n# A\n",
		"B.md": "---\nid: 01K5QF7T4ZPWY6X0N3EV8HMJR2\nlinks:\n  - to: A\n    role: parent\n---\n# B\n",
		"C.md": "---\nid: 01K5QF7T4ZPWY6X0N3EV8HMJR3\nlinks:\n  - to: B\n    role: parent\n---\n# C\n",
	})
	before := c.read(t, "A.md")

	// C is below B, which is below A. Making C the parent of A closes the loop.
	err := c.saveConnection().Execute(t.Context(), c.vault, domain.Connection{
		A: "A.md", B: "C.md", AB: &domain.Side{Role: domain.RoleParent},
	})

	if !errors.Is(err, domain.ErrHierarchyCycle) {
		t.Fatalf("err = %v, want ErrHierarchyCycle", err)
	}
	if got := c.read(t, "A.md"); got != before {
		t.Errorf("A was written:\n%s", got)
	}
}

func TestReplacingTheEdgeBetweenThePairIsNotACycle(t *testing.T) {
	t.Parallel()
	// A writes `child: B`, so B is below A. Making A the child of B closes a loop.
	c := openChanging(t, map[string]string{
		"A.md": "---\nid: 01K5QF7T4ZPWY6X0N3EV8HMJR1\nlinks:\n  - to: B\n    role: child\n---\n# A\n",
		"B.md": "---\nid: 01K5QF7T4ZPWY6X0N3EV8HMJR2\n---\n# B\n",
	})

	err := c.join().Execute(t.Context(), c.vault, "A.md", "B.md", domain.Side{Role: domain.RoleParent})
	if err != nil {
		t.Fatalf("replacing the edge between the pair is not a cycle: %v", err)
	}
	err = c.join().Execute(t.Context(), c.vault, "B.md", "A.md", domain.Side{Role: domain.RoleChild})
	if err != nil {
		t.Fatalf("err = %v", err)
	}
}

func TestAParentOfItsOwnAncestorIsRefused(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{
		"A.md": "---\nid: 01K5QF7T4ZPWY6X0N3EV8HMJR1\nlinks:\n  - to: B\n    role: child\n---\n# A\n",
		"B.md": "---\nid: 01K5QF7T4ZPWY6X0N3EV8HMJR2\nlinks:\n  - to: C\n    role: child\n---\n# B\n",
		"C.md": "---\nid: 01K5QF7T4ZPWY6X0N3EV8HMJR3\n---\n# C\n",
	})

	// A is above B, B above C. C becoming A's parent is a cycle.
	err := c.join().Execute(t.Context(), c.vault, "C.md", "A.md", domain.Side{Role: domain.RoleParent})
	if err != nil {
		t.Fatalf("C below A is no cycle: %v", err)
	}
	err = c.join().Execute(t.Context(), c.vault, "C.md", "A.md", domain.Side{Role: domain.RoleChild})
	if !errors.Is(err, domain.ErrHierarchyCycle) {
		t.Fatalf("err = %v, want ErrHierarchyCycle", err)
	}
}

func TestJoinWritesOneSideAndKeepsTheOther(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": heatNote, "Entropy.md": entropyNote})
	if err := c.join().Execute(t.Context(), c.vault, "Entropy.md", "Heat.md",
		domain.Side{Role: domain.RoleChild, Label: "measures"}); err != nil {
		t.Fatal(err)
	}

	if err := c.join().Execute(t.Context(), c.vault, "Heat.md", "Entropy.md",
		domain.Side{Role: domain.RoleParent, Marker: domain.MarkerArrow}); err != nil {
		t.Fatal(err)
	}

	got := c.connection(t, "Heat.md", "Entropy.md")
	if got.AB == nil || got.AB.Marker != domain.MarkerArrow {
		t.Errorf("A to B = %+v", got.AB)
	}
	if got.BA == nil || got.BA.Label != "measures" {
		t.Errorf("Join changed the record it was not asked to write: %+v", got.BA)
	}
}

func TestJoinRefusesARoleThatIsNotTheInverseOfTheOtherRecord(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": heatNote, "Entropy.md": entropyNote})
	if err := c.join().Execute(t.Context(), c.vault, "Entropy.md", "Heat.md",
		domain.Side{Role: domain.RoleChild}); err != nil {
		t.Fatal(err)
	}

	err := c.join().Execute(t.Context(), c.vault, "Heat.md", "Entropy.md", domain.Side{Role: domain.RoleJump})

	if !errors.Is(err, domain.ErrInvalidConnection) {
		t.Errorf("err = %v, want ErrInvalidConnection", err)
	}
}

func TestRemovingASideDoesNotWriteTheOtherNote(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": heatNote, "Entropy.md": entropyNote})
	if err := c.saveConnection().Execute(t.Context(), c.vault, domain.Connection{
		A: "Heat.md", B: "Entropy.md",
		AB: &domain.Side{Role: domain.RoleJump}, BA: &domain.Side{Role: domain.RoleJump},
	}); err != nil {
		t.Fatal(err)
	}
	entropy := c.read(t, "Entropy.md")

	if err := c.removeSide().Execute(t.Context(), c.vault, "Heat.md", "Entropy.md", ""); err != nil {
		t.Fatal(err)
	}

	got := c.connection(t, "Heat.md", "Entropy.md")
	if got.AB != nil || got.BA == nil {
		t.Errorf("connection = %+v", got)
	}
	if c.read(t, "Entropy.md") != entropy {
		t.Error("removing a side wrote the other note")
	}
}

func TestRemovingAConnectionRemovesBothSidesButNotOtherTypes(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{
		"Heat.md": "---\nid: 01K5QF7T4ZPWY6X0N3EV8HMJRC\nlinks:\n  - to: Entropy\n    role: jump\n" +
			"  - to: Entropy\n    role: ref\n    type: preset\n---\n# Heat\n",
		"Entropy.md": "---\nid: 01K5QF7T4ZPWY6X0N3EV8HMJRD\nlinks:\n  - to: Heat\n    role: jump\n---\n# Entropy\n",
	})

	if err := c.removeConnection().Execute(t.Context(), c.vault, "Heat.md", "Entropy.md", ""); err != nil {
		t.Fatal(err)
	}

	if got := c.connection(t, "Heat.md", "Entropy.md"); got.AB != nil || got.BA != nil {
		t.Errorf("connection = %+v", got)
	}
	if body := c.read(t, "Heat.md"); !strings.Contains(body, "type: preset") {
		t.Errorf("a record of another type was removed:\n%s", body)
	}
}

func TestReplacingLinksKeepsTheTypeAndTheReasonOfTheRecord(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{
		"Heat.md": "---\nid: 01K5QF7T4ZPWY6X0N3EV8HMJRC\nlinks:\n  - to: Entropy\n    role: parent\n" +
			"    type: requires\n    note: only eigenvectors\n---\n# Heat\n",
		"Entropy.md": entropyNote,
	})

	err := c.newReplaceLinks().Execute(t.Context(), c.vault, "Heat.md", "Entropy.md", []note.PairLink{
		{From: "Heat.md", To: "Entropy.md", Role: domain.RoleJump, Type: "requires", Label: "renamed"},
	})
	if err != nil {
		t.Fatal(err)
	}

	body := c.read(t, "Heat.md")
	for _, want := range []string{"type: requires", "note: only eigenvectors", "label: renamed", "role: jump"} {
		if !strings.Contains(body, want) {
			t.Errorf("Heat is missing %q:\n%s", want, body)
		}
	}
}

func TestRemovingALinkAddressesATypeAndKeepsOtherTypes(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{
		"Heat.md": "---\nid: 01K5QF7T4ZPWY6X0N3EV8HMJRC\nlinks:\n  - to: Entropy\n    role: parent\n" +
			"  - to: Entropy\n    role: ref\n    type: preset\n---\n# Heat\n",
		"Entropy.md": entropyNote,
	})
	target := domain.Address{Scheme: domain.SchemeName, Value: "Entropy"}

	if _, err := c.newEditLinks().Remove(t.Context(), c.vault, "Heat.md",
		domain.LinkKey{Target: target}, domain.FileStat{}); err != nil {
		t.Fatal(err)
	}

	body := c.read(t, "Heat.md")
	if strings.Contains(body, "role: parent") || !strings.Contains(body, "type: preset") {
		t.Errorf("the untyped record should go and the preset stay:\n%s", body)
	}
}

func TestCreatingInARelationWritesTheInverseRoleAndFilesInTheFolderOfTheSource(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"area/Heat.md": heatNote})
	u := note.NewCreateInRelation(c.create(), c.addresses())

	made, err := u.Execute(t.Context(), c.vault, domain.RelatedNote{
		From: "area/Heat.md", Role: domain.RoleParent, Type: "series",
	})
	if err != nil {
		t.Fatal(err)
	}

	if made.Path != "area/Untitled note.md" {
		t.Errorf("path = %s", made.Path)
	}
	body := c.read(t, made.Path)
	for _, want := range []string{"to: Heat", "role: child", "type: series"} {
		if !strings.Contains(body, want) {
			t.Errorf("the new note is missing %q:\n%s", want, body)
		}
	}
}

func TestCreatingInARelationNumbersUntitledNotesAndRetriesOnConflict(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{
		"Heat.md":             heatNote,
		"Untitled note.md":    "# one\n",
		"Untitled note 2.md":  "# two\n",
		"Untitled note 10.md": "# ten\n",
	})
	u := note.NewCreateInRelation(c.create(), c.addresses())

	made, err := u.Execute(t.Context(), c.vault, domain.RelatedNote{From: "Heat.md", Role: domain.RoleChild})
	if err != nil {
		t.Fatal(err)
	}

	if made.Path != "Untitled note 3.md" {
		t.Errorf("path = %s", made.Path)
	}
	if body := c.read(t, made.Path); !strings.Contains(body, "role: parent") {
		t.Errorf("a child is written with the parent role:\n%s", body)
	}
}

func TestCreatingInARelationFailsAfterTheAttemptsAreSpent(t *testing.T) {
	t.Parallel()
	notes := map[string]string{"Heat.md": heatNote, "Untitled note.md": "# n\n"}
	for i := 2; i <= note.MaxUntitledAttempts; i++ {
		notes["Untitled note "+itoa(i)+".md"] = "# n\n"
	}
	c := openChanging(t, notes)
	u := note.NewCreateInRelation(c.create(), c.addresses())

	_, err := u.Execute(t.Context(), c.vault, domain.RelatedNote{From: "Heat.md", Role: domain.RoleJump})

	if !errors.Is(err, domain.ErrNoFreeName) {
		t.Errorf("err = %v, want ErrNoFreeName", err)
	}
}

func itoa(n int) string {
	digits := ""
	for ; n > 0; n /= 10 {
		digits = string(rune('0'+n%10)) + digits
	}
	return digits
}

func TestCreatingInARelationWithATitleDoesNotRetry(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": heatNote, "Work.md": "# Work\n"})
	u := note.NewCreateInRelation(c.create(), c.addresses())

	_, err := u.Execute(t.Context(), c.vault, domain.RelatedNote{
		From: "Heat.md", Role: domain.RoleJump, Title: "Work",
	})

	if !errors.Is(err, vaultport.ErrAlreadyExists) {
		t.Errorf("err = %v, want ErrAlreadyExists", err)
	}
}

func TestCreatingInARelationRefusesARoleANoteCannotBeCreatedIn(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{"Heat.md": heatNote})
	u := note.NewCreateInRelation(c.create(), c.addresses())

	for _, role := range []domain.LinkRole{domain.RoleRef, domain.RoleAttachment, "sibling"} {
		_, err := u.Execute(t.Context(), c.vault, domain.RelatedNote{From: "Heat.md", Role: role})
		if !errors.Is(err, domain.ErrNotCreatable) {
			t.Errorf("%s: err = %v, want ErrNotCreatable", role, err)
		}
	}
}

func TestBacklinksCarryTheirRoleAndTheRoleFromTheNoteAskedAbout(t *testing.T) {
	t.Parallel()
	c := openChanging(t, map[string]string{
		"Area.md": "---\ntitle: Area\n---\n# Area\n",
		"Idea.md": "---\ntitle: Idea\nlinks:\n  - to: \"[[Area]]\"\n    role: parent\n    marker: arrow\n---\n# Idea\n",
	})

	got, err := note.NewGetBacklinks(c.db.Links()).Execute(t.Context(), c.vault, "Area.md")
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 {
		t.Fatalf("backlinks = %+v", got)
	}
	b := got[0]
	if b.From != "Idea.md" || b.Role != domain.RoleParent || b.RoleFromHere != domain.RoleChild || b.Marker != domain.MarkerArrow {
		t.Errorf("backlink = %+v", b)
	}
}

func TestTheNeighbourhoodCarriesTheMarkerOfEachRecord(t *testing.T) {
	t.Parallel()
	db, v := newIndexedVault(t, map[string]string{
		"Heat.md":    "---\ntitle: Heat\nlinks:\n  - to: Entropy\n    role: jump\n    marker: arrow\n---\n# Heat\n",
		"Entropy.md": "---\ntitle: Entropy\nlinks:\n  - to: Heat\n    role: jump\n---\n# Entropy\n",
	})

	got, err := note.NewShowNeighbourhood(db.Links(), db.Queries()).Execute(t.Context(), v, "Heat.md")
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Related) != 1 {
		t.Fatalf("related = %+v", got.Related)
	}
	r := got.Related[0]
	if r.Marker != domain.MarkerArrow || r.ReverseMarker != domain.MarkerNone || !r.IsMutual {
		t.Errorf("neighbour = %+v", r)
	}
}
