package webui_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// The window renames a note and takes one out of the vault. What the vault
// answers is what the window has to be able to say about either.

// gone says whether a file is off the disk.
func gone(t *testing.T, root, path string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(path)))
	return os.IsNotExist(err)
}

// scanned waits for the first read of the vault, which is what puts the notes
// already on disk into the index and the links they carry with them.
func scanned(t *testing.T, f *going) {
	t.Helper()
	waitFor(t, f.opened.API.Ready.Load)
}

// TestRenamingWritesTheNoteAndMovesTheFile. A note is shown by its title, so
// the title is what is written, and the file follows it.
func TestRenamingWritesTheNoteAndMovesTheFile(t *testing.T) {
	f := quitting(t, nil, map[string]string{
		"Old.md": "---\ntitle: Old\n---\n\n# Old\n",
	})

	answer, err := f.client.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
		Path: "Old.md", Title: "Entropy",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the rename was refused: %v", refusal)
	}
	if path := answer.Msg.GetPath(); path != "Entropy.md" {
		t.Fatalf("the note is filed at %q", path)
	}
	if title := answer.Msg.GetTitle(); title != "Entropy" {
		t.Errorf("the note is called %q", title)
	}
	if by := answer.Msg.GetBy(); by != v1.Naming_NAMING_FRONTMATTER {
		t.Errorf("the frontmatter named the note and the answer says %v", by)
	}
	moved := answer.Msg.GetMoved()
	if moved.GetFrom() != "Old.md" || moved.GetTo() != "Entropy.md" {
		t.Errorf("what the file did came back as %+v", moved)
	}
	if !gone(t, f.root, "Old.md") {
		t.Error("the note is still filed under the name it had")
	}
	if now := fileAt(t, f.root, "Entropy.md"); !strings.Contains(now, "title: Entropy") {
		t.Errorf("the title was not written:\n%s", now)
	}
}

// TestTheWindowRenamesTheWayTheSettingsSay. The palette and the plex menu both
// ask for a rename by title, and the file tree asks for a move. The window
// carries the one setting to all three.
func TestTheWindowRenamesTheWayTheSettingsSay(t *testing.T) {
	for name, c := range map[string]struct {
		sync note.SyncTitleAndFilename
		// at is where the renamed note is filed, and called what the moved one
		// is called.
		at     string
		called string
	}{
		"one name":   {sync: true, at: "Disorder.md", called: "Warmth"},
		"told apart": {sync: false, at: "Entropy.md", called: "Heat"},
	} {
		t.Run(name, func(t *testing.T) {
			f := opening(t, nil, map[string]string{
				"Entropy.md": "---\ntitle: Entropy\n---\nA measure.\n",
				"Heat.md":    "---\ntitle: Heat\n---\nA measure.\n",
			}, c.sync)
			scanned(t, f)

			// The palette and the plex menu: a title, and the file follows it
			// where the two are one name.
			renamed, err := f.client.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
				Path: "Entropy.md", Title: "Disorder",
			}))
			if err != nil {
				t.Fatal(err)
			}
			if got := renamed.Msg.GetPath(); got != c.at {
				t.Errorf("the note is filed at %q", got)
			}

			// The file tree: a name, and the note is called by it where the two
			// are one name.
			if _, err := f.client.MoveFile(t.Context(), connect.NewRequest(&v1.MoveFileRequest{
				From: "Heat.md", To: "Warmth.md",
			})); err != nil {
				t.Fatal(err)
			}
			if !gone(t, f.root, "Heat.md") {
				t.Error("the file is still filed under the name it had")
			}
			want := "title: " + c.called
			if now := fileAt(t, f.root, "Warmth.md"); !strings.Contains(now, want) {
				t.Errorf("want %q in\n%s", want, now)
			}
		})
	}
}

// oneName writes into the settings whether a note's title and the name of its
// file are kept as one name.
func oneName(t *testing.T, f *going, kept bool) error {
	t.Helper()
	_, err := f.configuring.WriteSettings(t.Context(), connect.NewRequest(&v1.WriteSettingsRequest{
		Settings: []*v1.Setting{
			{At: []string{"naming", "sync_title_and_filename"}, Value: fmt.Sprint(kept)},
		},
	}))
	return err
}

// TestTurningTheSettingIsAnsweredByTheNextRename. The palette turns it, the
// file is written, and the rename after it reads what was written. Nothing is
// launched again in between.
func TestTurningTheSettingIsAnsweredByTheNextRename(t *testing.T) {
	f := opening(t, nil, map[string]string{
		"Entropy.md": "---\ntitle: Entropy\n---\nA measure.\n",
		"Heat.md":    "---\ntitle: Heat\n---\nA measure.\n",
	}, true)
	scanned(t, f)

	if setting(t, f, "naming", "sync_title_and_filename") != true {
		t.Fatal("an installation nobody has configured tells the two apart")
	}

	if err := oneName(t, f, false); err != nil {
		t.Fatal(err)
	}

	// Read back through the same window, and then acted on by a rename.
	if setting(t, f, "naming", "sync_title_and_filename") != false {
		t.Error("the setting was turned and the window still says one name")
	}

	renamed, err := f.client.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
		Path: "Entropy.md", Title: "Disorder",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got := renamed.Msg.GetPath(); got != "Entropy.md" {
		t.Errorf("the file moved to %q under a setting that was turned off", got)
	}
	if _, err := f.client.MoveFile(t.Context(), connect.NewRequest(&v1.MoveFileRequest{
		From: "Heat.md", To: "Warmth.md",
	})); err != nil {
		t.Fatal(err)
	}
	if now := fileAt(t, f.root, "Warmth.md"); !strings.Contains(now, "title: Heat") {
		t.Errorf("the note was called by its file under a setting that was turned off:\n%s", now)
	}
}

// The setting a person turns is written where they will read it, and every
// other byte of the file is left as they typed it.
func TestTurningTheSettingLeavesTheRestOfTheFileAlone(t *testing.T) {
	f := opening(t, nil, nil, true)
	path := filepath.Join(filepath.Dir(f.settings), "numen.json")
	if err := os.WriteFile(path, []byte("{\n  \"appearance\": {\"text_scale\": 1.5}\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := oneName(t, f, false); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"text_scale": 1.5`) {
		t.Errorf("what the person typed was rewritten:\n%s", raw)
	}
	if !strings.Contains(string(raw), `"sync_title_and_filename": false`) {
		t.Errorf("the setting is not in the file:\n%s", raw)
	}
}

// TestRenamingLeavesALinkThatMeansAnotherNoteNow. Two notes under one name is
// the person's to settle, and a link that resolves is not repaired.
func TestRenamingLeavesALinkThatMeansAnotherNoteNow(t *testing.T) {
	const heat = "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n\n# Heat\n"
	f := quitting(t, nil, map[string]string{
		"Entropy.md":         "# Entropy\n",
		"physics/Entropy.md": "# Entropy\n",
		"Heat.md":            heat,
	})
	scanned(t, f)

	answer, err := f.client.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
		Path: "Entropy.md", Title: "Thermodynamics",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the rename was refused: %v", refusal)
	}
	if repaired := answer.Msg.GetMoved().GetRepaired(); len(repaired) != 0 {
		t.Errorf("a link that resolves is not repaired: %v", repaired)
	}
	if now := fileAt(t, f.root, "Heat.md"); now != heat {
		t.Errorf("the note that pointed at it was rewritten:\n%s", now)
	}
}

// TestRenamingOntoATakenNameSaysWhatTheNoteIsCalled, and where it still is.
func TestRenamingOntoATakenNameSaysWhatTheNoteIsCalled(t *testing.T) {
	f := quitting(t, nil, map[string]string{
		"Old.md":     "# Old\n",
		"Entropy.md": "# Entropy\n",
	})

	answer, err := f.client.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
		Path: "Old.md", Title: "Entropy",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_OCCUPIED {
		t.Errorf("a name already taken was answered with %v", refusal)
	}
	if path := answer.Msg.GetPath(); path != "Old.md" {
		t.Errorf("the note is filed at %q", path)
	}
	if by := answer.Msg.GetBy(); by != v1.Naming_NAMING_FILENAME {
		t.Errorf("the filename names the note and the answer says %v", by)
	}
	if now := fileAt(t, f.root, "Entropy.md"); now != "# Entropy\n" {
		t.Errorf("the note already there was written:\n%s", now)
	}
}

// TestRenamingRefusesATitleNoFileCanBeNamedAfter. The note is not opened for a
// title that leaves nothing to name it. The titles that exercise the rule are
// the core's, in TestRenamingRefusesATitleNoNoteCanBeGiven.
func TestRenamingRefusesATitleNoFileCanBeNamedAfter(t *testing.T) {
	const held = "# Old\n"
	f := quitting(t, nil, map[string]string{"Old.md": held})

	answer, err := f.client.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
		Path: "Old.md", Title: "   ",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNNAMEABLE {
		t.Errorf("a title with no filename in it was answered with %v", refusal)
	}
	if now := fileAt(t, f.root, "Old.md"); now != held {
		t.Errorf("the refused rename wrote to the note:\n%s", now)
	}
}

// TestRenamingSaysWhatItCouldNotName. Each of these is something the person can
// see for themselves once they are told which it is.
func TestRenamingSaysWhatItCouldNotName(t *testing.T) {
	for name, c := range map[string]struct {
		path string
		want v1.Refusal
	}{
		"a note that is not there": {
			path: "Missing.md",
			want: v1.Refusal_REFUSAL_MISSING,
		},
		"a file the vault does not hold as a note": {
			path: "Reading.txt",
			want: v1.Refusal_REFUSAL_NOT_A_NOTE,
		},
		"a note whose frontmatter cannot be read": {
			path: "Broken.md",
			want: v1.Refusal_REFUSAL_UNREADABLE,
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := quitting(t, nil, map[string]string{
				"Reading.txt": "a list\n",
				"Broken.md":   "---\nid: [unterminated\n---\n# Broken\n",
			})

			answer, err := f.client.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
				Path: c.path, Title: "Entropy",
			}))
			if err != nil {
				t.Fatal(err)
			}
			if refusal := answer.Msg.GetRefusal(); refusal != c.want {
				t.Errorf("want %v, got %v", c.want, refusal)
			}
			if !gone(t, f.root, "Entropy.md") {
				t.Error("a refused rename left a file under the name it was given")
			}
		})
	}
}

// TestARemovedNoteGoesToTheTrashAndSaysWhatNowReachesNothing. A link is not
// wrong because the note it names is gone, so it is reported and not repaired.
func TestARemovedNoteGoesToTheTrashAndSaysWhatNowReachesNothing(t *testing.T) {
	const pointing = "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n\n# Heat\n"
	f := quitting(t, nil, map[string]string{
		"Entropy.md": "# Entropy\n",
		"Heat.md":    pointing,
	})
	scanned(t, f)

	answer, err := f.client.RemoveFile(t.Context(), connect.NewRequest(&v1.RemoveFileRequest{
		Path: "Entropy.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the note was refused: %v", refusal)
	}
	if trashed := answer.Msg.GetTrashed(); trashed != ".trash/Entropy.md" {
		t.Errorf("the note sits at %q", trashed)
	}
	dangling := answer.Msg.GetDangling()
	if len(dangling) != 1 || dangling[0] != "Heat.md" {
		t.Errorf("what now reaches nothing came back as %v", dangling)
	}
	if !gone(t, f.root, "Entropy.md") {
		t.Error("the note is still where it was")
	}
	if now := fileAt(t, f.root, ".trash/Entropy.md"); now != "# Entropy\n" {
		t.Errorf("the note in the trash holds:\n%s", now)
	}
	if now := fileAt(t, f.root, "Heat.md"); now != pointing {
		t.Errorf("the note whose link reaches nothing was written:\n%s", now)
	}
}

// TestADestroyedNoteLeavesNothingBehind. Nothing brings it back, and the answer
// names no place in the trash to look.
func TestADestroyedNoteLeavesNothingBehind(t *testing.T) {
	f := quitting(t, nil, map[string]string{"Entropy.md": "# Entropy\n"})

	answer, err := f.client.RemoveFile(t.Context(), connect.NewRequest(&v1.RemoveFileRequest{
		Path: "Entropy.md", Destroy: true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the note was refused: %v", refusal)
	}
	if trashed := answer.Msg.GetTrashed(); trashed != "" {
		t.Errorf("a destroyed note was said to sit at %q", trashed)
	}
	if !gone(t, f.root, "Entropy.md") || !gone(t, f.root, ".trash/Entropy.md") {
		t.Error("the file is still on the disk")
	}
}

// TestARemovedFileIsReportedTheFirstTimeItIsAskedFor. The watcher reports only
// the paths the vault holds a source for, and a picture is not one of them.
func TestARemovedFileIsReportedTheFirstTimeItIsAskedFor(t *testing.T) {
	client, root := opened(t, map[string]string{
		"Note.md":            "---\ntitle: Note\n---\n\n# Note\n",
		"assets/diagram.png": "a picture, near enough\n",
	})

	// Its own context, closed before the server is: a stream is an open request,
	// and a test server waits for those.
	listening, hangUp := context.WithCancel(t.Context())
	defer hangUp()

	changes, err := client.WatchVaultChanges(listening, connect.NewRequest(&v1.WatchVaultChangesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer changes.Close()
	if !changes.Receive() {
		t.Fatalf("the stream never opened: %v", changes.Err())
	}

	reported := make(chan []string, 1)
	go func() {
		for changes.Receive() {
			if paths := changes.Msg().GetPaths(); len(paths) > 0 {
				reported <- paths
				return
			}
		}
	}()

	answer, err := client.RemoveFile(t.Context(), connect.NewRequest(&v1.RemoveFileRequest{
		Path: "assets/diagram.png",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the file was refused: %v", refusal)
	}
	if !gone(t, root, "assets/diagram.png") {
		t.Fatal("the file is still where it was")
	}

	select {
	case paths := <-reported:
		if !slices.Contains(paths, "assets/diagram.png") {
			t.Errorf("the removal was reported as %v", paths)
		}
	case <-time.After(5 * time.Second):
		t.Error("the removal reached nobody, and the tree still draws the row")
	}
}

// TestNeitherARenameNorARemoveIsTakenWhileTheWindowIsGoing. The door is shut on
// what would reach the vault after the writes already taken have landed.
func TestNeitherARenameNorARemoveIsTakenWhileTheWindowIsGoing(t *testing.T) {
	f := quitting(t, nil, map[string]string{"Entropy.md": "# Entropy\n"})
	t.Cleanup(func() { f.opened.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if !f.opened.Settle(ctx) {
		t.Fatal("the vault did not settle with nobody holding anything")
	}

	if _, err := f.client.RenameNote(context.Background(), connect.NewRequest(&v1.RenameNoteRequest{
		Path: "Entropy.md", Title: "Thermodynamics",
	})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("a rename after the door was shut was answered with %v", err)
	}
	if _, err := f.client.RemoveFile(context.Background(), connect.NewRequest(&v1.RemoveFileRequest{
		Path: "Entropy.md",
	})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("a remove after the door was shut was answered with %v", err)
	}
	if now := fileAt(t, f.root, "Entropy.md"); now != "# Entropy\n" {
		t.Errorf("the note was written after the door was shut:\n%s", now)
	}
}

// TestRenamingSaysWhatANoteCannotBeCalled. A title the vault cannot show the
// note under is refused, and a title the `title` key can carry is written there
// whatever the note held before.
func TestRenamingSaysWhatANoteCannotBeCalled(t *testing.T) {
	for name, c := range map[string]struct {
		held  string
		title string
		want  v1.Refusal
		file  string
	}{
		"a note carrying a heading takes the key, the heading naming nothing": {
			held:  "# Old\n",
			title: "C#",
			file:  "---\ntitle: C#\n",
		},
		"a filename that cannot carry it hands the title to the key": {
			held:  "A measure.\n",
			title: "C#",
			file:  "---\ntitle: C#\n",
		},
		"a title over more than one line": {
			held:  "---\ntitle: Old\n---\nbody\n",
			title: "one\ntwo",
			want:  v1.Refusal_REFUSAL_UNNAMEABLE,
		},
		"the key carries the hash, so the note is renamed": {
			held:  "---\ntitle: Old\n---\nbody\n",
			title: "C#",
			file:  "---\ntitle: C#\n",
		},
		"a frontmatter written on one line cannot be changed a key at a time": {
			held:  "---\n{title: Old, id: 01J8}\n---\n# Old\n",
			title: "Entropy",
			want:  v1.Refusal_REFUSAL_UNREADABLE,
		},
		// Nothing closes the block, so nothing in the file is frontmatter and
		// the filename names the note. The file moves and is not written to.
		"a frontmatter block that is never closed": {
			held:  "---\ntitle: Old\n# Old\n",
			title: "Entropy",
			file:  "---\ntitle: Old\n# Old\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := quitting(t, nil, map[string]string{"Old.md": c.held})

			answer, err := f.client.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
				Path: "Old.md", Title: c.title,
			}))
			if err != nil {
				t.Fatalf("want an answer the window can read, got %v", err)
			}
			if refusal := answer.Msg.GetRefusal(); refusal != c.want {
				t.Errorf("want %v, got %v", c.want, refusal)
			}
			if c.want != v1.Refusal_REFUSAL_UNSPECIFIED {
				if now := fileAt(t, f.root, "Old.md"); now != c.held {
					t.Errorf("the refused rename wrote to the note:\n%s", now)
				}
				return
			}
			if now := fileAt(t, f.root, answer.Msg.GetPath()); !strings.HasPrefix(now, c.file) {
				t.Errorf("the note holds:\n%s", now)
			}
		})
	}
}

// TestARenamedNoteIsStillLinkedTo. A rename files a note under a name a link
// can be written by, and the repair crosses to the window. The titles that
// exercise the rule are the core's, in
// TestARenamedNoteIsStillReachedByTheLinksThatNameIt.
func TestARenamedNoteIsStillLinkedTo(t *testing.T) {
	f := quitting(t, nil, map[string]string{
		"Entropy.md": "# Entropy\n",
		"Heat.md":    "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n\n# Heat\n",
	})
	scanned(t, f)

	answer, err := f.client.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
		Path: "Entropy.md", Title: "Notes [[draft]]",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the rename was refused: %v", refusal)
	}
	repaired := answer.Msg.GetMoved().GetRepaired()
	if len(repaired) != 1 || repaired[0] != "Heat.md" {
		t.Fatalf("want the note whose link broke repaired, got %v", repaired)
	}
	name := strings.TrimSuffix(answer.Msg.GetPath(), ".md")
	if now := fileAt(t, f.root, "Heat.md"); !strings.Contains(now, "to: "+name) &&
		!strings.Contains(now, "to: '"+name+"'") && !strings.Contains(now, `to: "`+name+`"`) {
		t.Errorf("the link does not reach %q:\n%s", name, now)
	}
}

// overtaking is the vault's writers, with the write refused the way one is when
// something outside this process wrote the file in the meantime.
type overtaking struct{ port.VaultWriters }

func (o overtaking) Open(v domain.Vault) (port.VaultWriter, error) {
	writer, err := o.VaultWriters.Open(v)
	if err != nil {
		return nil, err
	}
	return overtaken{VaultWriter: writer}, nil
}

type overtaken struct{ port.VaultWriter }

func (overtaken) Write(
	context.Context, string, []byte, domain.Fingerprint,
) (domain.Fingerprint, error) {
	return domain.Fingerprint{}, port.ErrChanged
}

// TestRenamingANoteWrittenElsewhereIsAQuestion. A note holding prose nobody
// here has read is something the person settles, the way a save and a join
// already put it to them.
func TestRenamingANoteWrittenElsewhereIsAQuestion(t *testing.T) {
	f := quitting(t, nil, map[string]string{
		"Old.md": "---\ntitle: Old\n---\n\n# Old\n",
	})
	scanned(t, f)
	f.opened.API.Notes.Rename.Writers = overtaking{VaultWriters: f.opened.API.Notes.Rename.Writers}

	out, err := f.opened.API.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
		Path:  "Old.md",
		Title: "New",
	}))
	if err != nil {
		t.Fatalf("a note written elsewhere came back as an error: %v", err)
	}
	if refusal := out.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_STALE {
		t.Errorf("a note that changed was answered %v", refusal)
	}
	if gone(t, f.opened.API.Showing().Path, "Old.md") {
		t.Error("the file moved for a rename that wrote nothing")
	}
}
