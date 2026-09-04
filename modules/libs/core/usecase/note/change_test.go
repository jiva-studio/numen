package note_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/search"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// changing is a vault that can be both asked about and written to, with the
// index kept level as a real caller would keep it.
type changing struct {
	db    *container.Index
	vault domain.Vault
	index func(ctx context.Context, v domain.Vault, paths []string) error
}

func changeable(t *testing.T, notes map[string]string) changing {
	t.Helper()
	db, v := indexed(t, notes)
	refresh := vaults.Refresh{Readers: filesystem.VaultReaders{}, Notes: db.Notes()}
	return changing{
		db:    db,
		vault: v,
		index: func(ctx context.Context, v domain.Vault, paths []string) error {
			_, err := refresh.Execute(ctx, v, paths)
			return err
		},
	}
}

// search is the one search, with no embedder: the words half answers alone.
func (c changing) search() search.Search {
	return search.New(c.db.Passages(), filesystem.VaultReaders{}, nil, nil, nil, 0, nil)
}

func (c changing) create() note.Create {
	return note.Create{
		Writers: filesystem.VaultWriters{}, Names: c.db.Queries(), Index: c.index,
	}
}

func (c changing) move() note.Move {
	return note.Move{
		Readers: filesystem.VaultReaders{}, Writers: filesystem.VaultWriters{},
		Links: c.db.Links(), Names: c.db.Queries(),
		Sources: c.db.Sources(), Index: c.index,
	}
}

func (c changing) remove() note.Remove {
	return note.Remove{
		Writers: filesystem.VaultWriters{},
		Links:   c.db.Links(), Known: c.db.SourcesKnown(), Index: c.index,
	}
}

func (c changing) linking() note.EditLinks {
	return note.EditLinks{
		Readers: filesystem.VaultReaders{}, Writers: filesystem.VaultWriters{}, Index: c.index,
	}
}

func (c changing) read(t *testing.T, path string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(c.vault.Path, filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}

func TestACreatedNoteIsNamedAfterItsTitleAndFoundByIt(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{"other.md": "# Other\n"})

	created, err := c.create().Execute(t.Context(), c.vault, note.NewNote{
		Title: "Entropy", Body: "A measure of disorder.\n", Folder: "physics",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Path != "physics/Entropy.md" {
		t.Errorf("want physics/Entropy.md, got %s", created.Path)
	}
	if created.ID == "" {
		t.Error("a note the application made carries an identifier")
	}
	if len(created.Shares) != 0 {
		t.Errorf("nothing else answers to this name: %v", created.Shares)
	}

	// The index is level before the caller is told, so this finds it.
	found, err := c.search().Execute(t.Context(), c.vault, "disorder", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].Source != created.Path {
		t.Errorf("the new note is not searchable yet: %v", found)
	}
}

// A title that cannot be a filename still names the note, from the `title` key.
func TestATitleThatCannotBeAFilenameGoesIntoTheKey(t *testing.T) {
	t.Parallel()
	c := changeable(t, nil)

	created, err := c.create().Execute(t.Context(), c.vault, note.NewNote{Title: "TCP/IP"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(created.Path, "TCP/IP") {
		t.Errorf("the slash made a folder: %s", created.Path)
	}
	if body := c.read(t, created.Path); !strings.Contains(body, "title: TCP/IP") {
		t.Errorf("the exact title is not in the note:\n%s", body)
	}

	shown, err := c.db.Queries().Notes(t.Context(), c.vault.ID, []string{created.Path})
	if err != nil {
		t.Fatal(err)
	}
	if shown[created.Path].Title != "TCP/IP" {
		t.Errorf("want the title back, got %q", shown[created.Path].Title)
	}
}

func TestCreatingSaysWhenTheNameIsAlreadyTaken(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{"archive/Entropy.md": "# Entropy\n"})

	created, err := c.create().Execute(t.Context(), c.vault, note.NewNote{Title: "Entropy"})
	if err != nil {
		t.Fatal(err)
	}
	if len(created.Shares) != 1 || created.Shares[0] != "archive/Entropy.md" {
		t.Errorf("want the note that already answers to the name, got %v", created.Shares)
	}
}

func TestCreatingRefusesToLandOnAnExistingNote(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{"Entropy.md": "# Entropy\n"})

	_, err := c.create().Execute(t.Context(), c.vault, note.NewNote{Title: "Entropy"})
	if !errors.Is(err, port.ErrOccupied) {
		t.Fatalf("want ErrOccupied, got %v", err)
	}
}

// The whole point of resolving a name at the moment it is asked: a note moves
// and the links written by its name follow it, untouched.
func TestMovingLeavesLinksWrittenByNameAlone(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{
		"Entropy.md": "# Entropy\n",
		"Heat.md":    "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n# Heat\n",
	})
	before := c.read(t, "Heat.md")

	moved, err := c.move().Execute(t.Context(), c.vault, "Entropy.md", "physics/Entropy.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(moved.Repaired) != 0 {
		t.Errorf("nothing needed repairing: %v", moved.Repaired)
	}
	if got := c.read(t, "Heat.md"); got != before {
		t.Errorf("the note that pointed at it was rewritten\n want %q\n  got %q", before, got)
	}

	found := links(t, c.db, c.vault, "Heat.md")
	if len(found.Links) != 1 || found.Links[0].To != "physics/Entropy.md" {
		t.Errorf("the link did not follow the note: %+v", found.Links)
	}
}

// A note its title names is called that wherever its file goes, so the name a
// link is written by still finds it after the file is renamed.
func TestRenamingTheFileOfATitledNoteLeavesTheNameLinksFindItBy(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{
		"physics/Old.md": "---\ntitle: Entropy\n---\nA measure of disorder.\n",
		"Heat.md":        "---\nlinks:\n  - to: Old\n    role: parent\n---\n# Heat\n",
	})
	before := c.read(t, "Heat.md")

	moved, err := c.move().Execute(t.Context(), c.vault, "physics/Old.md", "physics/Thermodynamics.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(moved.Repaired) != 0 {
		t.Errorf("the link still reaches the note, so nothing is repaired: %v", moved.Repaired)
	}
	if got := c.read(t, "Heat.md"); got != before {
		t.Errorf("the note that pointed at it was rewritten:\n%s", got)
	}

	found := links(t, c.db, c.vault, "Heat.md")
	if len(found.Links) != 1 || found.Links[0].To != "physics/Thermodynamics.md" {
		t.Errorf("the link written by the name reaches %+v", found.Links)
	}
}

// A `title` key names the note whatever the filename says, so a file renamed
// under it leaves the note called what the file says it is called.
func TestRenamingTheFileOfATitledNoteLeavesTheTitleAlone(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{
		"physics/Entropy.md": "---\ntitle: Entropy\n---\nA measure of disorder.\n",
	})

	if _, err := c.move().Execute(t.Context(), c.vault, "physics/Entropy.md", "physics/Old.md"); err != nil {
		t.Fatal(err)
	}

	if got := c.title(t, "physics/Old.md"); got != "Entropy" {
		t.Errorf("the vault shows the note as %q, and the file says Entropy", got)
	}
	named, err := c.db.Queries().Named(t.Context(), c.vault.ID, "Entropy")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(named, []string{"physics/Old.md"}) {
		t.Errorf("the vault files %v under the name the note carries", named)
	}
}

// A note with neither a title nor a heading is called by its filename, so
// renaming the file renames the note, and a link written by the old name is
// written again by the new one.
func TestRenamingTheFileOfAnUntitledNoteNamesItByItsNewFilename(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{
		"physics/Old.md": "A measure of disorder.\n",
		"Heat.md":        "---\nlinks:\n  - to: Old\n    role: parent\n---\n# Heat\n",
	})

	moved, err := c.move().Execute(t.Context(), c.vault, "physics/Old.md", "physics/Entropy.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(moved.Repaired) != 1 || moved.Repaired[0] != "Heat.md" {
		t.Fatalf("want the note whose link stopped resolving, got %+v", moved)
	}
	if body := c.read(t, "Heat.md"); !strings.Contains(body, "to: Entropy") {
		t.Errorf("the repair writes the new name:\n%s", body)
	}

	found := links(t, c.db, c.vault, "Heat.md")
	if len(found.Links) != 1 || found.Links[0].To != "physics/Entropy.md" {
		t.Errorf("the repaired link reaches %+v", found.Links)
	}

	// The note answers to the name it is filed under now, wherever a link
	// naming it is written.
	named, err := c.db.Queries().Named(t.Context(), c.vault.ID, "Entropy")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(named, "physics/Entropy.md") {
		t.Errorf("the vault files %v under the name", named)
	}
}

// A link written as a path is the one that can break. A note its filename names
// is called by the filename it lands under, so neither the path nor the name
// finds it any more — and the repair writes the new name, which survives every
// later move.
func TestMovingRepairsALinkThatStoppedResolving(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{
		"physics/Entropy.md": "A measure of disorder.\n",
		"physics/Heat.md":    "---\nlinks:\n  - to: physics/Entropy.md\n    role: parent\n---\n# Heat\n",
	})

	moved, err := c.move().Execute(t.Context(), c.vault, "physics/Entropy.md", "archive/Thermodynamics.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(moved.Repaired) != 1 || moved.Repaired[0] != "physics/Heat.md" {
		t.Fatalf("want the note whose link broke, got %+v", moved)
	}

	body := c.read(t, "physics/Heat.md")
	if strings.Contains(body, "physics/Entropy.md") {
		t.Errorf("the broken path is still written:\n%s", body)
	}
	if !strings.Contains(body, "to: Thermodynamics") {
		t.Errorf("the repair writes the new name:\n%s", body)
	}

	found := links(t, c.db, c.vault, "physics/Heat.md")
	if len(found.Links) != 1 || found.Links[0].To != "archive/Thermodynamics.md" {
		t.Errorf("the repaired link does not reach the note: %+v", found.Links)
	}
}

// A link that reaches a different note of the same name is not broken, so it is
// left alone: which of two notes under one name it means is the person's.
func TestMovingLeavesALinkThatNowMeansAnotherNote(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{
		"physics/Entropy.md":   "# Entropy\n",
		"chemistry/Entropy.md": "# Entropy\n",
		"chemistry/Heat.md":    "---\nlinks:\n  - to: physics/Entropy.md\n    role: parent\n---\n# Heat\n",
	})
	before := c.read(t, "chemistry/Heat.md")

	moved, err := c.move().Execute(t.Context(), c.vault, "physics/Entropy.md", "archive/Entropy.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(moved.Repaired) != 0 {
		t.Errorf("nothing was broken, so nothing is repaired: %+v", moved.Repaired)
	}
	if got := c.read(t, "chemistry/Heat.md"); got != before {
		t.Errorf("a link that was not broken was rewritten")
	}
	if to := links(t, c.db, c.vault, "chemistry/Heat.md").Links[0].To; to != "chemistry/Entropy.md" {
		t.Errorf("the link reaches %q", to)
	}
}

// The note lands under a name another note is already filed under, and a name
// is read back as an exact path from the root before anything else. The repair
// writes the path, which is what reaches the note that moved.
func TestRepairingALinkWritesThePathWhereTheNameIsShared(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{
		"Entropy.md":      "# Entropy\n",
		"physics/Heat.md": "# Heat\n",
		"Ref.md":          "---\nlinks:\n  - to: physics/Heat.md\n    role: parent\n---\n# Ref\n",
	})

	moved, err := c.move().Execute(t.Context(), c.vault, "physics/Heat.md", "archive/Entropy.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(moved.Repaired) != 1 || moved.Repaired[0] != "Ref.md" {
		t.Fatalf("want the note whose link broke, got %+v", moved)
	}
	if to := links(t, c.db, c.vault, "Ref.md").Links[0].To; to != "archive/Entropy.md" {
		t.Errorf("the repaired link reaches %q", to)
	}
}

func TestRemovingPutsTheNoteInTheTrashAndOutOfTheIndex(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{
		"Entropy.md": "# Entropy\n\nA measure of disorder.\n",
		"Heat.md":    "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n# Heat\n",
	})

	removed, err := c.remove().Execute(t.Context(), c.vault, "Entropy.md")
	if err != nil {
		t.Fatal(err)
	}
	if removed.Trashed != ".trash/Entropy.md" {
		t.Errorf("want the note kept under its own path, got %q", removed.Trashed)
	}
	if len(removed.Dangling) != 1 || removed.Dangling[0] != "Heat.md" {
		t.Errorf("want the note left pointing at nothing, got %v", removed.Dangling)
	}
	if _, err := os.Stat(filepath.Join(c.vault.Path, ".trash", "Entropy.md")); err != nil {
		t.Errorf("the note is not in the trash: %v", err)
	}

	found, err := c.search().Execute(t.Context(), c.vault, "disorder", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Errorf("a removed note is still in the index: %v", found)
	}
}

// A folder goes to the trash whole, in one move, and the notes that pointed
// into it are named. A note that went with the folder points at nothing from
// the trash, so it is not among them.
func TestRemovingAFolderTakesWhatIsUnderItAndNamesTheLinksLeft(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{
		"physics/Entropy.md": "# Entropy\n\nA measure of disorder.\n",
		"physics/Heat.md":    "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n# Heat\n",
		"Outside.md":         "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n# Outside\n",
	})

	removed, err := c.remove().Execute(t.Context(), c.vault, "physics")
	if err != nil {
		t.Fatal(err)
	}
	if removed.Trashed != ".trash/physics" {
		t.Errorf("want the folder kept under its own path, got %q", removed.Trashed)
	}
	if len(removed.Dangling) != 1 || removed.Dangling[0] != "Outside.md" {
		t.Errorf("want the note left pointing at nothing, got %v", removed.Dangling)
	}
	for _, path := range []string{"Entropy.md", "Heat.md"} {
		if _, err := os.Stat(filepath.Join(c.vault.Path, ".trash", "physics", path)); err != nil {
			t.Errorf("%s is not in the trash: %v", path, err)
		}
	}

	found, err := c.search().Execute(t.Context(), c.vault, "disorder", search.Parameters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 0 {
		t.Errorf("a note under the removed folder is still in the index: %v", found)
	}
}

func TestLinkingWritesTheIdentifierTheNoteDidNotHave(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{
		"Entropy.md": "# Entropy\n",
		"Heat.md":    "# Heat\n",
	})

	if err := c.linking().Add(t.Context(), c.vault, "Heat.md", domain.Link{
		Target: domain.Address{Scheme: domain.SchemeName, Value: "Entropy"},
		Role:   domain.RoleParent,
		Label:  "follows from",
	}); err != nil {
		t.Fatal(err)
	}

	body := c.read(t, "Heat.md")
	if !strings.Contains(body, "to: Entropy") || !strings.Contains(body, "role: parent") {
		t.Errorf("the link is not written:\n%s", body)
	}
	if !strings.Contains(body, "id: ") {
		t.Errorf("editing a note is when it gets an identifier:\n%s", body)
	}
	if !strings.Contains(body, "# Heat") {
		t.Errorf("the prose was lost:\n%s", body)
	}

	found := links(t, c.db, c.vault, "Heat.md")
	if len(found.Links) != 1 || found.Links[0].To != "Entropy.md" {
		t.Errorf("the link does not reach the note: %+v", found.Links)
	}
}

func TestPointingANoteAtAPlaceUnderATypeReplacesTheEntryItHad(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{
		"Sanskrit.md": "# Sanskrit\n",
		"Slow.md":     "# Slow going\n",
		"Roots.md": "---\ntype: deck\nlinks:\n  - to: Sanskrit\n    role: ref\n    type: preset\n" +
			"  - to: Grammar\n    role: parent\n---\n# Roots\n",
	})

	at, err := c.linking().PointAt(t.Context(), c.vault, "Roots.md", "preset",
		domain.Address{Scheme: domain.SchemeName, Value: "Slow"}, domain.RoleRef, domain.Fingerprint{})
	if err != nil {
		t.Fatal(err)
	}
	if at == (domain.Fingerprint{}) {
		t.Error("the write says nothing about the file it made")
	}

	body := c.read(t, "Roots.md")
	if strings.Contains(body, "to: Sanskrit") || !strings.Contains(body, "to: Slow") {
		t.Errorf("the entry did not move:\n%s", body)
	}
	if !strings.Contains(body, "  - to: Grammar\n    role: parent\n") {
		t.Errorf("the other entry was rewritten:\n%s", body)
	}
	if !strings.Contains(body, "id: ") {
		t.Errorf("editing a note is when it gets an identifier:\n%s", body)
	}

	named := 0
	for _, link := range links(t, c.db, c.vault, "Roots.md").Links {
		if link.Type != "preset" {
			continue
		}
		named++
		if link.To != "Slow.md" {
			t.Errorf("the entry reaches %q", link.To)
		}
	}
	if named != 1 {
		t.Errorf("the note names %d places under the type", named)
	}
}

func TestRemovingALinkLeavesTheOtherNoteAlone(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{
		"Entropy.md": "# Entropy\n",
		"Heat.md": "---\nlinks:\n  - to: Entropy\n    role: parent\n" +
			"  - to: Work\n    role: jump\n---\n# Heat\n",
		"Work.md": "# Work\n",
	})
	before := c.read(t, "Entropy.md")

	if err := c.linking().Remove(t.Context(), c.vault, "Heat.md",
		domain.Address{Scheme: domain.SchemeName, Value: "Entropy"}, ""); err != nil {
		t.Fatal(err)
	}

	body := c.read(t, "Heat.md")
	if strings.Contains(body, "to: Entropy") {
		t.Errorf("the link is still there:\n%s", body)
	}
	if !strings.Contains(body, "to: Work") {
		t.Errorf("the other link was taken with it:\n%s", body)
	}
	if got := c.read(t, "Entropy.md"); got != before {
		t.Errorf("the note at the other end was written to")
	}
}

func TestAWriteRefusesToLandOnAnEditItDidNotSee(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{"Entropy.md": "# Entropy\n"})
	writing := note.Write{
		Readers: filesystem.VaultReaders{}, Writers: filesystem.VaultWriters{}, Index: c.index,
	}

	reader, err := (filesystem.VaultReaders{}).Open(c.vault)
	if err != nil {
		t.Fatal(err)
	}
	stale, err := reader.Stat(t.Context(), "Entropy.md")
	if err != nil {
		t.Fatal(err)
	}

	// Somebody else gets there first.
	if _, err := writing.Execute(t.Context(), c.vault, "Entropy.md", "# Entropy\n\nTheirs.\n", domain.Fingerprint{}); err != nil {
		t.Fatal(err)
	}

	_, err = writing.Execute(t.Context(), c.vault, "Entropy.md", "# Entropy\n\nMine.\n", stale)
	if !errors.Is(err, port.ErrChanged) {
		t.Fatalf("want ErrChanged, got %v", err)
	}
	if body := c.read(t, "Entropy.md"); !strings.Contains(body, "Theirs.") {
		t.Errorf("the refused write landed anyway:\n%s", body)
	}
}

// A caller that wrote a note and writes it again presents the fingerprint its
// own write answered with. Holding the one it read would leave every sitting
// with one write in it.
func TestAWriteFollowsAWriteWithNoReadBetween(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{"Entropy.md": "# Entropy\n"})
	writing := note.Write{
		Readers: filesystem.VaultReaders{}, Writers: filesystem.VaultWriters{}, Index: c.index,
	}

	reader, err := (filesystem.VaultReaders{}).Open(c.vault)
	if err != nil {
		t.Fatal(err)
	}
	read, err := reader.Stat(t.Context(), "Entropy.md")
	if err != nil {
		t.Fatal(err)
	}

	at, err := writing.Execute(t.Context(), c.vault, "Entropy.md", "# Entropy\n\nOne.\n", read)
	if err != nil {
		t.Fatal(err)
	}
	if at == (domain.Fingerprint{}) {
		t.Fatal("the write answered with no fingerprint")
	}

	if _, err := writing.Execute(
		t.Context(), c.vault, "Entropy.md", "# Entropy\n\nTwo.\n", at,
	); err != nil {
		t.Fatalf("the write after a write was refused: %v", err)
	}
	if body := c.read(t, "Entropy.md"); !strings.Contains(body, "Two.") {
		t.Errorf("the second write did not land:\n%s", body)
	}
}
