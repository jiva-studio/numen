package note_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
)

func (c changing) saving() note.Write {
	return note.Write{
		Readers: filesystem.Readers{}, Writers: filesystem.Writers{}, Index: c.index,
	}
}

// A flow sequence is the commonest frontmatter line there is, and an identifier
// stamped into a note cannot be spliced past one. The person's own save writes
// no identifier, so such a note is saveable.
func TestSavingANoteWhoseFrontmatterIsWrittenOnOneLine(t *testing.T) {
	c := changeable(t, map[string]string{
		"Ideas.md": "---\ntags: [draft, idea]\n---\n# Ideas\n",
	})

	if err := c.saving().Save(t.Context(), c.vault, "Ideas.md", "# Ideas\n\nMine.\n"); err != nil {
		t.Fatal(err)
	}

	body := c.read(t, "Ideas.md")
	if !strings.Contains(body, "tags: [draft, idea]") {
		t.Errorf("the frontmatter did not survive:\n%s", body)
	}
	if !strings.Contains(body, "Mine.") {
		t.Errorf("the prose was not written:\n%s", body)
	}
	if strings.Contains(body, "id:") {
		t.Errorf("a save stamped an identifier:\n%s", body)
	}
}

// A note renamed or removed under an open tab is put back where it was opened.
func TestSavingANoteThatIsNotThereMakesIt(t *testing.T) {
	c := changeable(t, map[string]string{"Entropy.md": "# Entropy\n"})

	if err := c.saving().Save(t.Context(), c.vault, "physics/Heat.md", "# Heat\n"); err != nil {
		t.Fatal(err)
	}

	if body := c.read(t, "physics/Heat.md"); body != "# Heat\n" {
		t.Errorf("a note made by a save carries the person's text and nothing else:\n%q", body)
	}
}

// The refusal is for a caller that read, thought and arrived late. A save is
// not one, and is told nothing to be held to.
func TestASaveLandsOnAChangeItNeverSaw(t *testing.T) {
	c := changeable(t, map[string]string{"Entropy.md": "# Entropy\n"})
	saving := c.saving()

	// Somebody else gets there first, between the tab's read and its save.
	if err := saving.Save(t.Context(), c.vault, "Entropy.md", "# Entropy\n\nTheirs.\n"); err != nil {
		t.Fatal(err)
	}
	if err := saving.Save(t.Context(), c.vault, "Entropy.md", "# Entropy\n\nMine.\n"); err != nil {
		t.Fatalf("a save was refused: %v", err)
	}

	if body := c.read(t, "Entropy.md"); !strings.Contains(body, "Mine.") {
		t.Errorf("the save did not land:\n%s", body)
	}
}

// Nobody asked for a comparison, so nothing the file does between the read and
// the rename refuses the write. What changes it here is a writer outside the
// application, which the vault's lock does not reach.
func TestASaveIsNotRefusedByACheckNobodyAskedFor(t *testing.T) {
	c := changeable(t, map[string]string{"Entropy.md": "# Entropy\n"})
	on := filepath.Join(c.vault.Path, "Entropy.md")

	saving := note.Write{
		Readers: watchedReaders{inner: filesystem.Readers{}, stat: func(path string) {
			if err := os.WriteFile(on, []byte("# Entropy\n\nSomebody else's, longer.\n"), 0o644); err != nil {
				t.Error(err)
			}
		}},
		Writers: filesystem.Writers{},
		Index:   c.index,
	}

	if err := saving.Save(t.Context(), c.vault, "Entropy.md", "# Entropy\n\nMine.\n"); err != nil {
		t.Fatalf("a save was refused: %v", err)
	}
	if body := c.read(t, "Entropy.md"); !strings.Contains(body, "Mine.") {
		t.Errorf("the save did not land:\n%s", body)
	}
}

// The write lock covers the read, so the frontmatter a save splices its body
// into is the frontmatter on disk when the save lands. The agent's link is
// written while the save is reading, and it is in the file afterwards.
func TestALinkWrittenWhileASaveIsReadingSurvivesIt(t *testing.T) {
	c := changeable(t, map[string]string{
		"Entropy.md": "# Entropy\n",
		"Heat.md":    "# Heat\n\nWhat was there.\n",
	})

	agentRead := make(chan struct{})
	agentMayWrite := make(chan struct{})
	agent := note.Linking{
		Readers: watchedReaders{inner: filesystem.Readers{}, read: func(path string) {
			if path == "Heat.md" {
				close(agentRead)
				<-agentMayWrite
			}
		}},
		Writers: filesystem.Writers{},
		Index:   c.index,
	}

	saveAsking := make(chan struct{})
	saveRead := make(chan struct{})
	saveMayWrite := make(chan struct{})
	saving := note.Write{
		Readers: watchedReaders{inner: filesystem.Readers{}, read: func(path string) {
			if path == "Heat.md" {
				close(saveRead)
				<-saveMayWrite
			}
		}},
		Writers: watchedWriters{inner: filesystem.Writers{}, hold: func() { close(saveAsking) }},
		Index:   c.index,
	}

	added := make(chan error, 1)
	go func() {
		added <- agent.Add(context.Background(), c.vault, "Heat.md", domain.Link{
			Target: domain.Address{Scheme: domain.SchemeName, Value: "Entropy"},
			Role:   domain.RoleParent,
		})
	}()
	<-agentRead

	saved := make(chan error, 1)
	go func() {
		saved <- saving.Save(context.Background(), c.vault, "Heat.md", "# Heat\n\nWhat the person typed.\n")
	}()

	// The save has reached the point where it takes the lock, or — with
	// nothing serialising the two — has already read the note the agent is
	// about to change.
	select {
	case <-saveAsking:
	case <-saveRead:
	case <-time.After(10 * time.Second):
		t.Fatal("the save neither asked for the lock nor read the note")
	}
	close(agentMayWrite)
	if err := <-added; err != nil {
		t.Fatal(err)
	}

	// The save writes last, so what it read decides what the note ends up
	// holding.
	close(saveMayWrite)
	if err := <-saved; err != nil {
		t.Fatal(err)
	}

	body := c.read(t, "Heat.md")
	if !strings.Contains(body, "to: Entropy") {
		t.Errorf("the link the save read nothing about is gone:\n%s", body)
	}
	if !strings.Contains(body, "What the person typed.") {
		t.Errorf("the save did not land:\n%s", body)
	}
}

// Mending a link is a read and a write over a note the person may be typing
// in, and the lock covers both. The repair lands while the save is waiting for
// it, and the save carries it through.
func TestALinkMendedWhileASaveIsReadingSurvivesIt(t *testing.T) {
	c := changeable(t, map[string]string{
		"physics/Entropy.md": "# Entropy\n",
		"physics/Heat.md": "---\nlinks:\n  - to: physics/Entropy.md\n    role: parent\n---\n" +
			"# Heat\n\nWhat was there.\n",
	})

	var reading sync.Once
	mendRead := make(chan struct{})
	mendMayWrite := make(chan struct{})
	moving := note.Move{
		Readers: watchedReaders{inner: filesystem.Readers{}, read: func(path string) {
			if path != "physics/Heat.md" {
				return
			}
			reading.Do(func() { close(mendRead) })
			<-mendMayWrite
		}},
		Writers: filesystem.Writers{},
		Links:   c.db.Links(),
		Index:   c.index,
	}

	saveAsking := make(chan struct{})
	saveRead := make(chan struct{})
	saveMayWrite := make(chan struct{})
	saving := note.Write{
		Readers: watchedReaders{inner: filesystem.Readers{}, read: func(path string) {
			if path == "physics/Heat.md" {
				close(saveRead)
				<-saveMayWrite
			}
		}},
		Writers: watchedWriters{inner: filesystem.Writers{}, hold: func() { close(saveAsking) }},
		Index:   c.index,
	}

	moved := make(chan error, 1)
	go func() {
		_, err := moving.Execute(context.Background(), c.vault,
			"physics/Entropy.md", "archive/Thermodynamics.md")
		moved <- err
	}()
	<-mendRead

	saved := make(chan error, 1)
	go func() {
		saved <- saving.Save(context.Background(), c.vault, "physics/Heat.md",
			"# Heat\n\nWhat the person typed.\n")
	}()

	select {
	case <-saveAsking:
	case <-saveRead:
	case <-time.After(10 * time.Second):
		t.Fatal("the save neither asked for the lock nor read the note")
	}
	close(mendMayWrite)
	if err := <-moved; err != nil {
		t.Fatal(err)
	}

	close(saveMayWrite)
	if err := <-saved; err != nil {
		t.Fatal(err)
	}

	body := c.read(t, "physics/Heat.md")
	if !strings.Contains(body, "to: Thermodynamics") {
		t.Errorf("the mended link is gone:\n%s", body)
	}
	if !strings.Contains(body, "What the person typed.") {
		t.Errorf("the save did not land:\n%s", body)
	}
}

// A note kept as a link to a file elsewhere in the vault is a link afterwards,
// and the file at the other end is what changed.
func TestSavingThroughALinkLeavesTheLink(t *testing.T) {
	c := changeable(t, map[string]string{"notes/Real.md": "# Entropy\n"})
	link := filepath.Join(c.vault.Path, "Entropy.md")
	if err := os.Symlink(filepath.Join("notes", "Real.md"), link); err != nil {
		t.Fatal(err)
	}

	if err := c.saving().Save(t.Context(), c.vault, "Entropy.md", "# Entropy\n\nMine.\n"); err != nil {
		t.Fatal(err)
	}

	at, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if at.Mode()&os.ModeSymlink == 0 {
		t.Error("the link was replaced by a file of its own")
	}
	if body := c.read(t, "notes/Real.md"); !strings.Contains(body, "Mine.") {
		t.Errorf("the note the link points at was not written:\n%s", body)
	}
}

// The bound is on what is being written, so a note that grew on disk is still
// saveable from a tab that holds none of that growth.
func TestSavingMoreTextThanANoteHolds(t *testing.T) {
	c := changeable(t, map[string]string{"Entropy.md": "# Entropy\n"})

	err := c.saving().Save(t.Context(), c.vault, "Entropy.md", strings.Repeat("x", note.MaxBytes+1))
	if !errors.Is(err, note.ErrTooLarge) {
		t.Fatalf("want ErrTooLarge, got %v", err)
	}
	if body := c.read(t, "Entropy.md"); body != "# Entropy\n" {
		t.Errorf("a refused save wrote anyway:\n%s", body)
	}
}

// A file the vault leaves alone and a path with nothing at it are two answers,
// and a refresh acts on them differently. Neither is indexed and both leave the
// index; only the second is a note somebody may be looking at.
func TestARefreshTellsAFileTheVaultLeavesAloneFromANoteThatVanished(t *testing.T) {
	c := changeable(t, map[string]string{"Entropy.md": "# Entropy\n"})
	if err := os.WriteFile(filepath.Join(c.vault.Path, "photo.png"), []byte("not a note"), 0o644); err != nil {
		t.Fatal(err)
	}

	refresh := usecase.Refresh{Readers: filesystem.Readers{}, Notes: c.db.Notes()}
	res, err := refresh.Execute(t.Context(), c.vault, []string{"photo.png", "Gone.md", "Entropy.md"})
	if err != nil {
		t.Fatal(err)
	}

	if len(res.LeftAlone) != 1 || res.LeftAlone[0] != "photo.png" {
		t.Errorf("want the file the vault leaves alone, got %v", res.LeftAlone)
	}
	if len(res.Removed) != 1 || res.Removed[0] != "Gone.md" {
		t.Errorf("want the note that is not there, got %v", res.Removed)
	}
	if len(res.Indexed) != 1 || res.Indexed[0] != "Entropy.md" {
		t.Errorf("want the note that is, got %v", res.Indexed)
	}
	if len(res.Unreadable) != 0 {
		t.Errorf("nothing here could not be read: %v", res.Unreadable)
	}
	if slices.Contains(res.Changed(), "photo.png") {
		t.Error("a file that was never a note is reported as something to look at again")
	}
}

// watchedReaders says when a note was read and when it was looked at, from
// inside the use case doing it, so that what happens in between is the test's
// to decide.
type watchedReaders struct {
	inner port.VaultReaders
	read  func(path string)
	stat  func(path string)
}

func (w watchedReaders) Open(v domain.Vault) (port.VaultReader, error) {
	reader, err := w.inner.Open(v)
	if err != nil {
		return nil, err
	}
	return watchedReader{VaultReader: reader, read: w.read, stat: w.stat}, nil
}

type watchedReader struct {
	port.VaultReader
	read func(path string)
	stat func(path string)
}

func (w watchedReader) Read(ctx context.Context, path string) ([]byte, error) {
	raw, err := w.VaultReader.Read(ctx, path)
	if w.read != nil {
		w.read(path)
	}
	return raw, err
}

func (w watchedReader) Stat(ctx context.Context, path string) (domain.FileRef, error) {
	ref, err := w.VaultReader.Stat(ctx, path)
	if w.stat != nil {
		w.stat(path)
	}
	return ref, err
}

// watchedWriters says when the write lock was asked for, before it is waited
// for.
type watchedWriters struct {
	inner port.VaultWriters
	hold  func()
}

func (w watchedWriters) Open(v domain.Vault) (port.VaultWriter, error) {
	return w.inner.Open(v)
}

func (w watchedWriters) Hold(ctx context.Context, v domain.Vault) (func(), error) {
	w.hold()
	return w.inner.Hold(ctx, v)
}
