package note_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/search"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
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
	refresh := usecase.Refresh{Readers: filesystem.Readers{}, Notes: db.Notes()}
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
	return search.New(c.db.Passages(), filesystem.Readers{}, nil)
}

func (c changing) create() note.Create {
	return note.Create{
		Writers: filesystem.Writers{}, Names: c.db.Queries(), Index: c.index,
	}
}

func (c changing) move() note.Move {
	return note.Move{
		Readers: filesystem.Readers{}, Writers: filesystem.Writers{},
		Links: c.db.Links(), Index: c.index,
	}
}

func (c changing) remove() note.Remove {
	return note.Remove{
		Readers: filesystem.Readers{}, Writers: filesystem.Writers{},
		Links: c.db.Links(), Index: c.index,
	}
}

func (c changing) linking() note.Linking {
	return note.Linking{
		Readers: filesystem.Readers{}, Writers: filesystem.Writers{}, Index: c.index,
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
	if created.Identifier == "" {
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

// A title that cannot be a filename still names the note, from the heading.
func TestATitleThatCannotBeAFilenameBecomesAHeading(t *testing.T) {
	c := changeable(t, nil)

	created, err := c.create().Execute(t.Context(), c.vault, note.NewNote{Title: "TCP/IP"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(created.Path, "TCP/IP") {
		t.Errorf("the slash made a folder: %s", created.Path)
	}
	if body := c.read(t, created.Path); !strings.Contains(body, "# TCP/IP") {
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
	c := changeable(t, map[string]string{"Entropy.md": "# Entropy\n"})

	_, err := c.create().Execute(t.Context(), c.vault, note.NewNote{Title: "Entropy"})
	if !errors.Is(err, port.ErrOccupied) {
		t.Fatalf("want ErrOccupied, got %v", err)
	}
}

// The whole point of resolving a name at the moment it is asked: a note moves
// and the links written by its name follow it, untouched.
func TestMovingLeavesLinksWrittenByNameAlone(t *testing.T) {
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

// A link written as a path is the one that can break. Renaming the note changes
// its filename too, so neither the path nor the name finds it any more — and
// the repair writes the new name, which survives every later move.
func TestMovingRepairsALinkThatStoppedResolving(t *testing.T) {
	c := changeable(t, map[string]string{
		"physics/Entropy.md": "# Entropy\n",
		"physics/Heat.md":    "---\nlinks:\n  - to: physics/Entropy.md\n    role: parent\n---\n# Heat\n",
	})

	moved, err := c.move().Execute(t.Context(), c.vault, "physics/Entropy.md", "archive/Thermodynamics.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(moved.Repaired) != 1 || moved.Repaired[0] != "physics/Heat.md" {
		t.Fatalf("want the note whose link broke, got %+v", moved)
	}
	if len(moved.Retargeted) != 0 {
		t.Errorf("nothing was retargeted: %+v", moved.Retargeted)
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
// reported rather than rewritten.
func TestMovingReportsALinkThatNowMeansAnotherNote(t *testing.T) {
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
	if len(moved.Retargeted) != 1 || moved.Retargeted[0].Now != "chemistry/Entropy.md" {
		t.Fatalf("want the note it reaches now, got %+v", moved.Retargeted)
	}
	if got := c.read(t, "chemistry/Heat.md"); got != before {
		t.Errorf("a link that was not broken was rewritten")
	}
}

func TestRemovingPutsTheNoteInTheTrashAndOutOfTheIndex(t *testing.T) {
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

func TestLinkingWritesTheIdentifierTheNoteDidNotHave(t *testing.T) {
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

func TestRemovingALinkLeavesTheOtherNoteAlone(t *testing.T) {
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
	c := changeable(t, map[string]string{"Entropy.md": "# Entropy\n"})
	writing := note.Write{
		Readers: filesystem.Readers{}, Writers: filesystem.Writers{}, Index: c.index,
	}

	reader, err := (filesystem.Readers{}).Open(c.vault)
	if err != nil {
		t.Fatal(err)
	}
	stale, err := reader.Stat(t.Context(), "Entropy.md")
	if err != nil {
		t.Fatal(err)
	}

	// Somebody else gets there first.
	if _, err := writing.Execute(t.Context(), c.vault, "Entropy.md", "# Entropy\n\nTheirs.\n", domain.FileRef{}); err != nil {
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
	c := changeable(t, map[string]string{"Entropy.md": "# Entropy\n"})
	writing := note.Write{
		Readers: filesystem.Readers{}, Writers: filesystem.Writers{}, Index: c.index,
	}

	reader, err := (filesystem.Readers{}).Open(c.vault)
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
	if at == (domain.FileRef{}) {
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
