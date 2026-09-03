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

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

func (c changing) saving() note.Write {
	return note.Write{
		Readers: filesystem.Readers{}, Writers: filesystem.Writers{}, Index: c.index,
	}
}

// opened is a tab that has just read a note: the prose it was given, and the
// file it came out of.
func (c changing) opened(t *testing.T, path string) *note.Seen {
	t.Helper()
	found, err := (note.Read{Readers: filesystem.Readers{}}).Execute(t.Context(), c.vault, path)
	if err != nil {
		t.Fatal(err)
	}
	return &note.Seen{Prose: found.Body, At: found.Ref}
}

// A flow sequence is the commonest frontmatter line there is, and an identifier
// stamped into a note cannot be spliced past one. The person's own save writes
// no identifier, so such a note is saveable.
func TestSavingANoteWhoseFrontmatterIsWrittenOnOneLine(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{
		"Ideas.md": "---\ntags: [draft, idea]\n---\n# Ideas\n",
	})

	seen := c.opened(t, "Ideas.md")
	if _, err := c.saving().Save(t.Context(), c.vault, "Ideas.md", "# Ideas\n\nMine.\n", seen); err != nil {
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
	t.Parallel()
	c := changeable(t, map[string]string{"Entropy.md": "# Entropy\n"})

	if _, err := c.saving().Save(t.Context(), c.vault, "physics/Heat.md", "# Heat\n", nil); err != nil {
		t.Fatal(err)
	}

	if body := c.read(t, "physics/Heat.md"); body != "# Heat\n" {
		t.Errorf("a note made by a save carries the person's text and nothing else:\n%q", body)
	}
}

// A note that is not there holds no prose anybody could have missed, so the
// tab that was reading it puts it back where it opened it.
func TestASaveRemakesANoteDeletedUnderIt(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{"Entropy.md": "# Entropy\n"})
	seen := c.opened(t, "Entropy.md")
	if err := os.Remove(filepath.Join(c.vault.Path, "Entropy.md")); err != nil {
		t.Fatal(err)
	}

	if _, err := c.saving().Save(t.Context(), c.vault, "Entropy.md", "# Entropy\n\nMine.\n", seen); err != nil {
		t.Fatalf("a note that is not there was not made: %v", err)
	}
	if body := c.read(t, "Entropy.md"); body != "# Entropy\n\nMine.\n" {
		t.Errorf("a note made by a save carries the person's text and nothing else:\n%q", body)
	}
}

// The prose the tab read is gone from the file, so the save stops and the
// person is asked. Nothing is written.
func TestASaveOverProseTheTabNeverReadIsStopped(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{"Entropy.md": "# Entropy\n"})
	seen := c.opened(t, "Entropy.md")

	// Somebody else gets there first, between the tab's read and its save.
	theirs := "# Entropy\n\nTheirs.\n"
	if err := os.WriteFile(filepath.Join(c.vault.Path, "Entropy.md"), []byte(theirs), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := c.saving().Save(t.Context(), c.vault, "Entropy.md", "# Entropy\n\nMine.\n", seen)
	if !errors.Is(err, port.ErrChanged) {
		t.Fatalf("want ErrChanged, got %v", err)
	}
	if body := c.read(t, "Entropy.md"); body != theirs {
		t.Errorf("a save that was stopped wrote anyway:\n%s", body)
	}
}

// The person answered the question with *keep*, which is a save holding itself
// against nothing.
func TestASaveThatComparesNothingLands(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{"Entropy.md": "# Entropy\n"})
	if err := os.WriteFile(
		filepath.Join(c.vault.Path, "Entropy.md"), []byte("# Entropy\n\nTheirs.\n"), 0o644,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := c.saving().Save(t.Context(), c.vault, "Entropy.md", "# Entropy\n\nMine.\n", nil); err != nil {
		t.Fatalf("a save holding itself against nothing was stopped: %v", err)
	}
	if body := c.read(t, "Entropy.md"); !strings.Contains(body, "Mine.") {
		t.Errorf("the save did not land:\n%s", body)
	}
}

// A save puts down the text a person typed and adds nothing to it, so a note
// given a new level-one heading at the keyboard stays in the file it is in.
func TestSavingANewHeadingLeavesTheFileWhereItIs(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{"Entropy.md": "# Entropy\n"})
	seen := c.opened(t, "Entropy.md")

	if _, err := c.saving().Save(t.Context(), c.vault, "Entropy.md", "# Disorder\n", seen); err != nil {
		t.Fatal(err)
	}
	if body := c.read(t, "Entropy.md"); !strings.Contains(body, "# Disorder") {
		t.Errorf("the save did not land:\n%s", body)
	}
	if !gone(t, c, "Disorder.md") {
		t.Error("the file is at Disorder.md")
	}
	// A heading is prose. The note is shown by its `title` key, else by the file
	// it is filed under, and neither of those is what the person typed.
	if got := c.title(t, "Entropy.md"); got != "Entropy" {
		t.Errorf("the vault shows it as %q", got)
	}
}

// A synchroniser, a checkout and a touch all move a file's time over text that
// did not change. This is the rule that keeps the question off the screen of a
// person whose vault is in a synchronised folder.
func TestTheSameBytesWrittenAgainAreNotAChange(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{"Entropy.md": "# Entropy\n"})
	seen := c.opened(t, "Entropy.md")
	on := filepath.Join(c.vault.Path, "Entropy.md")

	raw, err := os.ReadFile(on)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(on, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(time.Hour)
	if err := os.Chtimes(on, later, later); err != nil {
		t.Fatal(err)
	}

	if _, err := c.saving().Save(t.Context(), c.vault, "Entropy.md", "# Entropy\n\nMine.\n", seen); err != nil {
		t.Fatalf("a save was stopped by a file nobody edited: %v", err)
	}
	if body := c.read(t, "Entropy.md"); !strings.Contains(body, "Mine.") {
		t.Errorf("the save did not land:\n%s", body)
	}
}

// A tab that saved and did not read again holds the file its own write
// produced. Holding the one it read would leave every sitting with one save in
// it.
func TestASaveFollowsASaveWithNoReadBetween(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{"Entropy.md": "# Entropy\n"})
	saving := c.saving()
	seen := c.opened(t, "Entropy.md")

	at, err := saving.Save(t.Context(), c.vault, "Entropy.md", "# Entropy\n\nOne.\n", seen)
	if err != nil {
		t.Fatal(err)
	}
	if at == (domain.Fingerprint{}) {
		t.Fatal("the save answered with no fingerprint")
	}

	if _, err := saving.Save(t.Context(), c.vault, "Entropy.md", "# Entropy\n\nTwo.\n",
		&note.Seen{Prose: seen.Prose, At: at}); err != nil {
		t.Fatalf("the save after a save was stopped: %v", err)
	}
	if body := c.read(t, "Entropy.md"); !strings.Contains(body, "Two.") {
		t.Errorf("the second save did not land:\n%s", body)
	}
}

// The frontmatter is not what the tab holds, so what an agent writes into it is
// not prose the tab has missed. The link is carried across by the save that
// follows it.
func TestALinkAnAgentAddsBetweenTwoSavesIsNotAChange(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{
		"Entropy.md": "# Entropy\n",
		"Heat.md":    "# Heat\n",
	})
	saving := c.saving()
	seen := c.opened(t, "Heat.md")

	mine := "# Heat\n\nMine.\n"
	at, err := saving.Save(t.Context(), c.vault, "Heat.md", mine, seen)
	if err != nil {
		t.Fatal(err)
	}

	if err := c.linking().Add(t.Context(), c.vault, "Heat.md", domain.Link{
		Target: domain.Address{Scheme: domain.SchemeName, Value: "Entropy"},
		Role:   domain.RoleParent,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := saving.Save(t.Context(), c.vault, "Heat.md", "# Heat\n\nMine, and more.\n",
		&note.Seen{Prose: mine, At: at}); err != nil {
		t.Fatalf("a link written into the frontmatter stopped a save: %v", err)
	}

	body := c.read(t, "Heat.md")
	if !strings.Contains(body, "to: Entropy") {
		t.Errorf("the agent's link is gone:\n%s", body)
	}
	if !strings.Contains(body, "Mine, and more.") {
		t.Errorf("the save did not land:\n%s", body)
	}
}

// Two vaults holding the same path and no words in common. A save answers for
// the vault it was given and reaches no other.
func TestTwoVaultsHoldTheirOwnSaves(t *testing.T) {
	t.Parallel()
	first := changeable(t, map[string]string{"Note.md": "# Note\n\nthermodynamics\n"})
	second := changeable(t, map[string]string{"Note.md": "# Note\n\nredshift\n"})

	if _, err := first.saving().Save(
		t.Context(), first.vault, "Note.md", "# Note\n\nentropy\n", first.opened(t, "Note.md"),
	); err != nil {
		t.Fatal(err)
	}

	if body := first.read(t, "Note.md"); !strings.Contains(body, "entropy") {
		t.Errorf("the save did not land:\n%s", body)
	}
	if body := second.read(t, "Note.md"); !strings.Contains(body, "redshift") {
		t.Errorf("the second vault was written by the first vault's save:\n%s", body)
	}

	if _, err := second.saving().Save(
		t.Context(), second.vault, "Note.md", "# Note\n\npulsar\n", second.opened(t, "Note.md"),
	); err != nil {
		t.Fatalf("the second vault was stopped by what the first vault holds: %v", err)
	}
	if body := second.read(t, "Note.md"); !strings.Contains(body, "pulsar") {
		t.Errorf("the second vault's save did not land:\n%s", body)
	}
	if body := first.read(t, "Note.md"); !strings.Contains(body, "entropy") {
		t.Errorf("the first vault was written by the second vault's save:\n%s", body)
	}
}

// The write lock covers the read, so the frontmatter a save splices its body
// into is the frontmatter on disk when the save lands. The agent's link is
// written while the save is reading, and it is in the file afterwards.
func TestALinkWrittenWhileASaveIsReadingSurvivesIt(t *testing.T) {
	t.Parallel()
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
		_, err := saving.Save(context.Background(), c.vault, "Heat.md", "# Heat\n\nWhat the person typed.\n", nil)
		saved <- err
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
	t.Parallel()
	// A note its filename names is called by the one it lands under, so the link
	// written as the path it had is the link the move mends.
	c := changeable(t, map[string]string{
		"physics/Entropy.md": "A measure of disorder.\n",
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
		Sources: c.db.Sources(),
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
		_, err := saving.Save(context.Background(), c.vault, "physics/Heat.md",
			"# Heat\n\nWhat the person typed.\n", nil)
		saved <- err
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
	t.Parallel()
	c := changeable(t, map[string]string{"notes/Real.md": "# Entropy\n"})
	link := filepath.Join(c.vault.Path, "Entropy.md")
	if err := os.Symlink(filepath.Join("notes", "Real.md"), link); err != nil {
		t.Fatal(err)
	}

	if _, err := c.saving().Save(t.Context(), c.vault, "Entropy.md", "# Entropy\n\nMine.\n", nil); err != nil {
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
	t.Parallel()
	c := changeable(t, map[string]string{"Entropy.md": "# Entropy\n"})

	_, err := c.saving().Save(t.Context(), c.vault, "Entropy.md", strings.Repeat("x", note.MaxBytes+1), nil)
	if !errors.Is(err, note.ErrTooLarge) {
		t.Fatalf("want ErrTooLarge, got %v", err)
	}
	if body := c.read(t, "Entropy.md"); body != "# Entropy\n" {
		t.Errorf("a refused save wrote anyway:\n%s", body)
	}
}

// The save and the read hold the note to one bound, so a note the save
// accepted is a note the read opens. The frontmatter is part of the file.
func TestANoteIsWrittenNoLargerThanItCanBeRead(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{
		"Entropy.md": "---\ntags: [physics, thermodynamics]\n---\n# Entropy\n",
	})

	body := strings.Repeat("x", note.MaxBytes-16)
	seen := c.opened(t, "Entropy.md")
	_, err := c.saving().Save(t.Context(), c.vault, "Entropy.md", body, seen)
	if err != nil && !errors.Is(err, note.ErrTooLarge) {
		t.Fatal(err)
	}

	found, err := (note.Read{Readers: filesystem.Readers{}}).Execute(t.Context(), c.vault, "Entropy.md")
	if err != nil {
		t.Fatal(err)
	}
	if found.Outcome != note.Ok {
		t.Errorf("the save left a note the read answers %q to, of %d bytes", found.Outcome, found.Ref.Size)
	}
}

// A deck is a note file read at a bound of its own, and the writer it goes to
// disk through carries that bound. One that carries none is held to MaxBytes.
func TestAWriterCarriesTheBoundItsFilesAreReadAt(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{"Mammals.md": "# Mammals\n"})

	body := strings.Repeat("x", note.MaxBytes+1)

	// A writer that names no bound is a writer of notes.
	_, err := c.saving().Save(t.Context(), c.vault, "Mammals.md", body, nil)
	if !errors.Is(err, note.ErrTooLarge) {
		t.Fatalf("want ErrTooLarge, got %v", err)
	}

	writing := c.saving()
	writing.Bound = 2 * note.MaxBytes
	if _, err := writing.Save(t.Context(), c.vault, "Mammals.md", body, nil); err != nil {
		t.Fatal(err)
	}
	if got := len(c.read(t, "Mammals.md")); got <= note.MaxBytes {
		t.Errorf("the file is %d bytes, and the body written was larger", got)
	}
}

// An agent writes through Execute, and is held to the bound the person's own
// save is held to.
func TestWritingMoreTextThanANoteHolds(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{"Entropy.md": "# Entropy\n"})

	_, err := c.saving().Execute(
		t.Context(), c.vault, "Entropy.md", strings.Repeat("x", note.MaxBytes+1), domain.Fingerprint{})
	if !errors.Is(err, note.ErrTooLarge) {
		t.Fatalf("want ErrTooLarge, got %v", err)
	}
	if body := c.read(t, "Entropy.md"); body != "# Entropy\n" {
		t.Errorf("a refused write wrote anyway:\n%s", body)
	}
}

// Create writes a whole note in one go, and is held to the same bound.
func TestCreatingMoreTextThanANoteHolds(t *testing.T) {
	t.Parallel()
	c := changeable(t, nil)

	made, err := c.create().Execute(t.Context(), c.vault, note.NewNote{
		Title: "Entropy", Body: strings.Repeat("x", note.MaxBytes+1),
	})
	if !errors.Is(err, note.ErrTooLarge) {
		t.Fatalf("want ErrTooLarge, got %v", err)
	}
	if made.Path != "" {
		t.Errorf("a refused creation reported a note at %s", made.Path)
	}
	if _, err := os.Stat(filepath.Join(c.vault.Path, "Entropy.md")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a refused creation left a file: %v", err)
	}
}

// A file the vault leaves alone and a path with nothing at it are two answers,
// and a refresh acts on them differently. Neither is indexed and both leave the
// index; only the second is a note somebody may be looking at.
func TestARefreshTellsAFileTheVaultLeavesAloneFromANoteThatVanished(t *testing.T) {
	t.Parallel()
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

func (w watchedReader) Stat(ctx context.Context, path string) (domain.Fingerprint, error) {
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
