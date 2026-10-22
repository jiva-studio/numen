package editor_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// The window renames a note and takes one out of the vault. What the vault
// answers is what the window has to be able to say about either.

// isGone says whether a file is off the disk.
func isGone(t *testing.T, root, path string) bool {
	t.Helper()
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(path)))
	return os.IsNotExist(err)
}

// waitForScan waits for the first read of the vault, which is what puts the notes
// already on disk into the index and the links they carry with them.
func waitForScan(t *testing.T, f *going) {
	t.Helper()
	testsupport.WaitFor(t, f.installation.State.Ready.Load)
}

// TestRenamingWritesTheNoteAndMovesTheFile. A note is shown by its title, so
// the title is what is written, and the file follows it.
func TestRenamingWritesTheNoteAndMovesTheFile(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"Old.md": "---\ntitle: Old\n---\n\n# Old\n",
	})

	answer, err := f.client.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
		Path: "Old.md", Title: "Entropy",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("the rename failed: %v", code)
	}
	if path := answer.Msg.GetPath(); path != "Entropy.md" {
		t.Fatalf("the note is filed at %q", path)
	}
	if title := answer.Msg.GetTitle(); title != "Entropy" {
		t.Errorf("the note is called %q", title)
	}
	if delta := answer.Msg.GetNameSource(); delta != v1.NameSource_NAME_SOURCE_FRONTMATTER {
		t.Errorf("the frontmatter named the note and the answer says %v", delta)
	}
	moved := answer.Msg.GetMoved()
	if moved.GetFrom() != "Old.md" || moved.GetTo() != "Entropy.md" {
		t.Errorf("what the file did came back as %+v", moved)
	}
	if !isGone(t, f.root, "Old.md") {
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
		posVal string
		called string
	}{
		"one name":   {sync: true, posVal: "Disorder.md", called: "Warmth"},
		"told apart": {sync: false, posVal: "Entropy.md", called: "Heat"},
	} {
		t.Run(name, func(t *testing.T) {
			f := opening(t, nil, map[string]string{
				"Entropy.md": "---\ntitle: Entropy\n---\nA measure.\n",
				"Heat.md":    "---\ntitle: Heat\n---\nA measure.\n",
			}, c.sync)
			waitForScan(t, f)

			// The palette and the plex menu: a title, and the file follows it
			// where the two are one name.
			renamed, err := f.client.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
				Path: "Entropy.md", Title: "Disorder",
			}))
			if err != nil {
				t.Fatal(err)
			}
			if got := renamed.Msg.GetPath(); got != c.posVal {
				t.Errorf("the note is filed at %q", got)
			}

			// The file tree: a name, and the note is called by it where the two
			// are one name.
			if _, err := f.client.MoveFile(t.Context(), connect.NewRequest(&v1.MoveFileRequest{
				From: "Heat.md", To: "Warmth.md",
			})); err != nil {
				t.Fatal(err)
			}
			if !isGone(t, f.root, "Heat.md") {
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
func oneName(t *testing.T, f *going, preserved bool) error {
	t.Helper()
	_, err := f.configuring.WriteSettings(t.Context(), connect.NewRequest(&v1.WriteSettingsRequest{
		Settings: []*v1.Setting{
			{At: []string{"naming", "sync_title_and_filename"}, Value: fmt.Sprint(preserved)},
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
	waitForScan(t, f)

	if getSetting(t, f, "naming", "sync_title_and_filename") != true {
		t.Fatal("an installation nobody has configured tells the two apart")
	}

	if err := oneName(t, f, false); err != nil {
		t.Fatal(err)
	}

	// Read back through the same window, and then acted on by a rename.
	if getSetting(t, f, "naming", "sync_title_and_filename") != false {
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
func TestTurningTheSettingPreservesTheRestOfTheFileAlone(t *testing.T) {
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
func TestRenamingPreservesALinkThatMeansAnotherNoteNow(t *testing.T) {
	const heat = "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n\n# Heat\n"
	f := openWindow(t, nil, map[string]string{
		"Entropy.md":         "# Entropy\n",
		"physics/Entropy.md": "# Entropy\n",
		"Heat.md":            heat,
	})
	waitForScan(t, f)

	answer, err := f.client.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
		Path: "Entropy.md", Title: "Thermodynamics",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("the rename failed: %v", code)
	}
	if repaired := answer.Msg.GetMoved().GetRepaired(); len(repaired) != 0 {
		t.Errorf("a link that resolves is not repaired: %v", repaired)
	}
	if now := fileAt(t, f.root, "Heat.md"); now != heat {
		t.Errorf("the note that pointed at it was rewritten:\n%s", now)
	}
}

// TestRenamingOntoATakenNameSaysWhatTheNoteIsCalled, and where it still is.
func TestRenamingOntoATakenNameReturnsWhatTheNoteIsCalled(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"Old.md":     "# Old\n",
		"Entropy.md": "# Entropy\n",
	})

	answer, err := f.client.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
		Path: "Old.md", Title: "Entropy",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_OCCUPIED {
		t.Errorf("a name already taken was answered with %v", code)
	}
	if path := answer.Msg.GetPath(); path != "Old.md" {
		t.Errorf("the note is filed at %q", path)
	}
	if delta := answer.Msg.GetNameSource(); delta != v1.NameSource_NAME_SOURCE_FILENAME {
		t.Errorf("the filename names the note and the answer says %v", delta)
	}
	if now := fileAt(t, f.root, "Entropy.md"); now != "# Entropy\n" {
		t.Errorf("the note already there was written:\n%s", now)
	}
}

// TestRenamingRefusesATitleNoFileCanBeNamedAfter. The note is not opened for a
// title that leaves nothing to name it. The titles that exercise the rule are
// the core's, in TestRenamingRefusesATitleNoNoteCanBeGiven.
func TestRenamingRefusesATitleNoFileCanBeNamedAfter(t *testing.T) {
	const target = "# Old\n"
	f := openWindow(t, nil, map[string]string{"Old.md": target})

	answer, err := f.client.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
		Path: "Old.md", Title: "   ",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_UNNAMEABLE {
		t.Errorf("a title with no filename in it was answered with %v", code)
	}
	if now := fileAt(t, f.root, "Old.md"); now != target {
		t.Errorf("the rejected rename wrote to the note:\n%s", now)
	}
}

// TestRenamingSaysWhatItCouldNotName. Each of these is something the person can
// see for themselves once they are told which it is.
func TestRenamingReturnsWhatItCouldNotName(t *testing.T) {
	for name, c := range map[string]struct {
		path string
		want v1.ErrorCode
	}{
		"a note that is not there": {
			path: "Missing.md",
			want: v1.ErrorCode_ERROR_CODE_MISSING,
		},
		"a file the vault does not hold as a note": {
			path: "Reading.txt",
			want: v1.ErrorCode_ERROR_CODE_NOT_A_NOTE,
		},
		"a note whose frontmatter cannot be read": {
			path: "Broken.md",
			want: v1.ErrorCode_ERROR_CODE_UNREADABLE,
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := openWindow(t, nil, map[string]string{
				"Reading.txt": "a list\n",
				"Broken.md":   "---\nid: [unterminated\n---\n# Broken\n",
			})

			answer, err := f.client.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
				Path: c.path, Title: "Entropy",
			}))
			if err != nil {
				t.Fatal(err)
			}
			if code := answer.Msg.GetError(); code != c.want {
				t.Errorf("want %v, got %v", c.want, code)
			}
			if !isGone(t, f.root, "Entropy.md") {
				t.Error("a rejected rename left a file under the name it was given")
			}
		})
	}
}

// TestARemovedNoteGoesToTheTrashAndSaysWhatNowReachesNothing. A link is not
// wrong because the note it names is gone, so it is reported and not repaired.
func TestARemovedNoteGoesToTheTrashAndReturnsWhatNowReachesNothing(t *testing.T) {
	const pointing = "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n\n# Heat\n"
	f := openWindow(t, nil, map[string]string{
		"Entropy.md": "# Entropy\n",
		"Heat.md":    pointing,
	})
	waitForScan(t, f)

	answer, err := f.client.RemoveFile(t.Context(), connect.NewRequest(&v1.RemoveFileRequest{
		Path: "Entropy.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("the note failed: %v", code)
	}
	if trashed := answer.Msg.GetTrashed(); trashed != ".trash/Entropy.md" {
		t.Errorf("the note sits at %q", trashed)
	}
	dangling := answer.Msg.GetDangling()
	if len(dangling) != 1 || dangling[0] != "Heat.md" {
		t.Errorf("what now reaches nothing came back as %v", dangling)
	}
	if !isGone(t, f.root, "Entropy.md") {
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
func TestADestroyedNotePreservesNothingPrior(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Entropy.md": "# Entropy\n"})

	answer, err := f.client.RemoveFile(t.Context(), connect.NewRequest(&v1.RemoveFileRequest{
		Path: "Entropy.md", Destroy: true,
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("the note failed: %v", code)
	}
	if trashed := answer.Msg.GetTrashed(); trashed != "" {
		t.Errorf("a destroyed note was said to sit at %q", trashed)
	}
	if !isGone(t, f.root, "Entropy.md") || !isGone(t, f.root, ".trash/Entropy.md") {
		t.Error("the file is still on the disk")
	}
}

// TestRemovingARemadeNote. A note removed, made again under the same name, and
// removed a second time is put into the trash with a suffix.
func TestRemovingARemadeNote(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Entropy.md": "# Entropy\n"})

	answer1, err := f.client.RemoveFile(t.Context(), connect.NewRequest(&v1.RemoveFileRequest{
		Path: "Entropy.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer1.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("first removal failed: %v", code)
	}

	created, err := f.client.CreateNote(t.Context(), connect.NewRequest(&v1.CreateNoteRequest{
		Title: "Entropy",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := created.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("create note failed: %v", code)
	}

	answer2, err := f.client.RemoveFile(t.Context(), connect.NewRequest(&v1.RemoveFileRequest{
		Path: "Entropy.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer2.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("second removal failed: %v", code)
	}
	if trashed := answer2.Msg.GetTrashed(); trashed != ".trash/Entropy-2.md" {
		t.Errorf("second removal trashed: %q, want .trash/Entropy-2.md", trashed)
	}
}

// TestARemovedFileIsReportedTheFirstTimeItIsAskedFor. The watcher reports only
// the paths the vault holds a source for, and a picture is not one of them.
func TestARemovedFileIsReportedTheFirstTimeItIsAskedFor(t *testing.T) {
	client, root := openVault(t, map[string]string{
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
	if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("the file failed: %v", code)
	}
	if !isGone(t, root, "assets/diagram.png") {
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
	f := openWindow(t, nil, map[string]string{"Entropy.md": "# Entropy\n"})
	t.Cleanup(func() { f.installation.Close() })

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	if !f.installation.FlushPending(ctx) {
		t.Fatal("the vault did not settle with nobody holding anything")
	}

	if _, err := f.client.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
		Path: "Entropy.md", Title: "Thermodynamics",
	})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("a rename after the door was shut was answered with %v", err)
	}
	if _, err := f.client.RemoveFile(t.Context(), connect.NewRequest(&v1.RemoveFileRequest{
		Path: "Entropy.md",
	})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("a remove after the door was shut was answered with %v", err)
	}
	if now := fileAt(t, f.root, "Entropy.md"); now != "# Entropy\n" {
		t.Errorf("the note was written after the door was shut:\n%s", now)
	}
}

// TestRenamingSaysWhatANoteCannotBeCalled. A title the vault cannot show the
// note under fails, and a title the `title` key can carry is written there
// whatever the note held before.
func TestRenamingReturnsWhatANoteCannotBeCalled(t *testing.T) {
	for name, c := range map[string]struct {
		activeItem string
		title      string
		want       v1.ErrorCode
		file       string
	}{
		"a note carrying a heading takes the key, the heading naming nothing": {
			activeItem: "# Old\n",
			title:      "C#",
			file:       "---\ntitle: C#\n",
		},
		"a filename that cannot carry it hands the title to the key": {
			activeItem: "A measure.\n",
			title:      "C#",
			file:       "---\ntitle: C#\n",
		},
		"a title over more than one line": {
			activeItem: "---\ntitle: Old\n---\nbody\n",
			title:      "one\ntwo",
			want:       v1.ErrorCode_ERROR_CODE_UNNAMEABLE,
		},
		"the key carries the hash, so the note is renamed": {
			activeItem: "---\ntitle: Old\n---\nbody\n",
			title:      "C#",
			file:       "---\ntitle: C#\n",
		},
		"a frontmatter written on one line cannot be changed a key at a time": {
			activeItem: "---\n{title: Old, id: 01J8}\n---\n# Old\n",
			title:      "Entropy",
			want:       v1.ErrorCode_ERROR_CODE_UNREADABLE,
		},
		// Nothing closes the block, so nothing in the file is frontmatter and
		// the filename names the note. The file moves and is not written to.
		"a frontmatter block that is never closed": {
			activeItem: "---\ntitle: Old\n# Old\n",
			title:      "Entropy",
			file:       "---\ntitle: Old\n# Old\n",
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := openWindow(t, nil, map[string]string{"Old.md": c.activeItem})

			answer, err := f.client.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
				Path: "Old.md", Title: c.title,
			}))
			if err != nil {
				t.Fatalf("want an answer the window can read, got %v", err)
			}
			if code := answer.Msg.GetError(); code != c.want {
				t.Errorf("want %v, got %v", c.want, code)
			}
			if c.want != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
				if now := fileAt(t, f.root, "Old.md"); now != c.activeItem {
					t.Errorf("the rejected rename wrote to the note:\n%s", now)
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
	f := openWindow(t, nil, map[string]string{
		"Entropy.md": "# Entropy\n",
		"Heat.md":    "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n\n# Heat\n",
	})
	waitForScan(t, f)

	answer, err := f.client.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
		Path: "Entropy.md", Title: "Notes [[draft]]",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("the rename failed: %v", code)
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

// when something outside this process wrote the file in the meantime.
type staleWriters struct{ port.VaultWriters }

func (o staleWriters) Open(v domain.Vault) (port.VaultWriter, error) {
	writer, err := o.VaultWriters.Open(v)
	if err != nil {
		return nil, err
	}
	return staleWriter{VaultWriter: writer}, nil
}

type staleWriter struct{ port.VaultWriter }

func (staleWriter) Write(
	context.Context, string, []byte, domain.Fingerprint,
) (domain.Fingerprint, error) {
	return domain.Fingerprint{}, port.ErrStale
}

// TestRenamingANoteWrittenElsewhereIsAQuestion. A note holding prose nobody
// here has read is something the person settles, the way a save and a join
// already put it to them.
func TestRenamingANoteSavedElsewhereIsAQuestion(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"Old.md": "---\ntitle: Old\n---\n\n# Old\n",
	})
	waitForScan(t, f)
	*f.installation.Services.NoteRename() = f.installation.Services.NoteRename().WithWriters(staleWriters{VaultWriters: f.installation.Services.NoteRename().Writers()})

	out, err := f.installation.Services.Note.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
		Path:  "Old.md",
		Title: "New",
	}))
	if err != nil {
		t.Fatalf("a note written elsewhere came back as an error: %v", err)
	}
	if code := out.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_STALE {
		t.Errorf("a note that changed was answered %v", code)
	}
	if isGone(t, f.installation.State.GetCurrentVault().Path, "Old.md") {
		t.Error("the file moved for a rename that wrote nothing")
	}
}

// stuck is an index that will not file a row under the path a file moved to.
type stuck struct{ port.SourceRepository }

func (stuck) MoveSources(context.Context, domain.VaultID, string, string) error {
	return errors.New("the index would not file the row under the new path")
}

func TestAMoveThatCameApartAfterTheFileLandedReturnsWhereItWent(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"Entropy.md": "---\ntitle: Entropy\n---\n\n# Entropy\n",
	})
	m := f.installation.Services.FileMove().WithSources(stuck{f.installation.Services.FileMove().Sources()})
	f.installation.Services.SetFileMove(&m)

	answer, err := f.client.MoveFile(t.Context(), connect.NewRequest(&v1.MoveFileRequest{
		From: "Entropy.md",
		To:   "notes/Entropy.md",
	}))
	if err != nil {
		t.Fatalf("file moved to new path but client received error: %v", err)
	}
	if to := answer.Msg.GetMoved().GetTo(); to != "notes/Entropy.md" {
		t.Errorf("the file the vault holds at notes/Entropy.md is answered for as %q", to)
	}
	if target := onDisk(t, f.root, "notes/Entropy.md"); target == "" {
		t.Error("the file the client received about holds nothing")
	}
}

func TestARenameThatCameApartAfterTheFileLandedReturnsWhatTheNoteIsCalled(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"Entropy.md": "---\ntitle: Entropy\n---\n\n# Entropy\n",
	})
	waitForScan(t, f)
	*f.installation.Services.NoteRename() = f.installation.Services.NoteRename().WithSources(stuck{f.installation.Services.NoteRename().Sources()})

	answer, err := f.client.RenameNote(t.Context(), connect.NewRequest(&v1.RenameNoteRequest{
		Path:  "Entropy.md",
		Title: "Enthalpy",
	}))
	if err != nil {
		t.Fatalf("the note was written and the client received %v", err)
	}
	if title := answer.Msg.GetTitle(); title != "Enthalpy" {
		t.Errorf("the note is called %q", title)
	}
	if path := answer.Msg.GetPath(); path != "Enthalpy.md" {
		t.Errorf("the note is filed at %q", path)
	}
}

// onDisk is what the vault holds at a path.
func onDisk(t *testing.T, root, path string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

const identified = "01M02ACGM0FYMSXNDP29C90JNR"

// getResolved is where each address landed, by what was asked about.
func getResolved(t *testing.T, answer *v1.ResolveAddressesResponse) map[string]*v1.ResolvedAddress {
	t.Helper()
	delta := map[string]*v1.ResolvedAddress{}
	for _, one := range answer.GetResolved() {
		delta[one.GetWritten()] = one
	}
	return delta
}

func TestAnAddressIsAnsweredWithTheNoteItReaches(t *testing.T) {
	client, _ := openVault(t, map[string]string{
		"physics/Entropy.md": "---\ntitle: Entropy\nid: " + identified + "\n---\n\nheat\n",
		"Note.md":            "---\ntitle: Note\n---\n\nprose\n",
	})

	answer, err := client.ResolveAddresses(t.Context(), connect.NewRequest(&v1.ResolveAddressesRequest{
		From:    "Note.md",
		Written: []string{"name://Entropy", "note://" + identified, "name://Nowhere"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	landed := getResolved(t, answer.Msg)
	for _, content := range []string{"name://Entropy", "note://" + identified} {
		one := landed[content]
		if one.GetPath() != "physics/Entropy.md" {
			t.Errorf("%q lands at %q", content, one.GetPath())
		}
		if one.GetVault() == "" || one.GetIsCrossed() {
			t.Errorf("%q lands in vault %q, crossed %v", content, one.GetVault(), one.GetIsCrossed())
		}
		if one.GetIsAmbiguous() {
			t.Errorf("%q is called ambiguous", content)
		}
	}
	if one, found := landed["name://Nowhere"]; found {
		t.Errorf("a name no note answers to lands at %q", one.GetPath())
	}
}

func TestANameReachesTheNoteAdjacentTheOneItIsSavedIn(t *testing.T) {
	client, _ := openVault(t, map[string]string{
		"heat/Entropy.md":  "---\ntitle: Entropy\n---\n\nheat\n",
		"heat/Note.md":     "---\ntitle: Note\n---\n\nprose\n",
		"order/Entropy.md": "---\ntitle: Entropy\n---\n\norder\n",
		"order/Note.md":    "---\ntitle: Note\n---\n\nprose\n",
	})

	for from, want := range map[string]string{
		"heat/Note.md":  "heat/Entropy.md",
		"order/Note.md": "order/Entropy.md",
	} {
		answer, err := client.ResolveAddresses(t.Context(), connect.NewRequest(&v1.ResolveAddressesRequest{
			From: from, Written: []string{"name://Entropy"},
		}))
		if err != nil {
			t.Fatal(err)
		}

		if got := getResolved(t, answer.Msg)["name://Entropy"].GetPath(); got != want {
			t.Errorf("written in %s it reaches %q, want %q", from, got, want)
		}
	}
}

func TestANameSeveralNotesAnswerToIsReported(t *testing.T) {
	client, _ := openVault(t, map[string]string{
		"heat/Entropy.md":    "---\ntitle: Entropy\n---\n\nheat\n",
		"order/Entropy.md":   "---\ntitle: Entropy\n---\n\norder\n",
		"heat/steam/Note.md": "---\ntitle: Note\n---\n\nprose\n",
	})

	answer, err := client.ResolveAddresses(t.Context(), connect.NewRequest(&v1.ResolveAddressesRequest{
		From: "heat/steam/Note.md", Written: []string{"name://Entropy"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	one := getResolved(t, answer.Msg)["name://Entropy"]
	if one.GetPath() != "heat/Entropy.md" {
		t.Errorf("it reaches %q, want the nearest in the tree", one.GetPath())
	}
	if !one.GetIsAmbiguous() {
		t.Error("a name two notes answer to is not called ambiguous")
	}
}

func TestANameSavedInNoNoteReachesTheNoteItNames(t *testing.T) {
	const lecture = "Seminar 1.2–1.3 — Lisbon, 9 July 1973"
	client, _ := openVault(t, map[string]string{
		"notes/" + lecture + ".md": "---\ntitle: " + lecture + "\n---\n\nprose\n",
	})

	answer, err := client.ResolveAddresses(t.Context(), connect.NewRequest(&v1.ResolveAddressesRequest{
		From: "", Written: []string{"name://" + lecture},
	}))
	if err != nil {
		t.Fatal(err)
	}

	one := getResolved(t, answer.Msg)["name://"+lecture]
	if one.GetPath() != "notes/"+lecture+".md" {
		t.Errorf("it reaches %q", one.GetPath())
	}
	if one.GetIsAmbiguous() {
		t.Error("one note answers to that name and it was called ambiguous")
	}
}

func TestAnAddressAskedTwiceIsAnsweredOnce(t *testing.T) {
	client, _ := openVault(t, map[string]string{
		"Entropy.md": "---\ntitle: Entropy\n---\n\nheat\n",
	})

	answer, err := client.ResolveAddresses(t.Context(), connect.NewRequest(&v1.ResolveAddressesRequest{
		Written: []string{"name://Entropy", "name://Entropy"},
	}))
	if err != nil {
		t.Fatal(err)
	}

	if got := answer.Msg.GetResolved(); len(got) != 1 {
		t.Fatalf("reached = %+v, want the one address asked about", got)
	}
}

// The window makes a note from the picture: a name, and the one link that puts
// it where the person reached out from.

func fileAt(t *testing.T, root, path string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}

// TestANoteIsMadeCarryingTheLinkThatRelatesIt. A note made in a relation of another
// says so itself: the link is in the note that was made, so one write leaves it
// joined.
func TestANoteIsMadeCarryingTheLinkThatRelatesIt(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"Ontology.md": "---\ntitle: Ontology\n---\n\n# Ontology\n",
	})

	answer, err := f.client.CreateNote(t.Context(), connect.NewRequest(&v1.CreateNoteRequest{
		Title: "Entropy",
		Links: []*v1.Link{{To: "Ontology.md", Role: v1.Role_ROLE_PARENT}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("the note failed: %v", code)
	}
	if path := answer.Msg.GetPath(); path != "Entropy.md" {
		t.Fatalf("the note was filed at %q", path)
	}

	made := fileAt(t, f.root, "Entropy.md")
	if !strings.Contains(made, "to: Ontology") {
		t.Errorf("the note carries no link to the note it was made from:\n%s", made)
	}
	if !strings.Contains(made, "role: parent") {
		t.Errorf("the link does not relate the note it was made from as its parent:\n%s", made)
	}
	if !strings.Contains(made, "id: ") {
		t.Errorf("a note the application made carries no identifier:\n%s", made)
	}
}

// TestANoteMadeInTheChildRelationIsTheParentsChild. The relation travels as the link the
// new note writes, so the role is the one that puts each of the two where the
// person put them.
func TestANoteMadeInTheChildRelationIsTheParentsChild(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"Ontology.md": "---\ntitle: Ontology\n---\n\n# Ontology\n",
	})

	answer, err := f.client.CreateNote(t.Context(), connect.NewRequest(&v1.CreateNoteRequest{
		Title: "Entropy",
		Links: []*v1.Link{{To: "Ontology.md", Role: v1.Role_ROLE_CHILD, Label: "follows from"}},
	}))
	if err != nil {
		t.Fatal(err)
	}

	made := fileAt(t, f.root, answer.Msg.GetPath())
	if !strings.Contains(made, "role: child") {
		t.Errorf("the link does not relate the note it was made from as its child:\n%s", made)
	}
	if !strings.Contains(made, "label: follows from") {
		t.Errorf("what the person wrote on the link is not in the note:\n%s", made)
	}
}

// TestANoteIsMadeInTheFolderItWasAskedFor. The person's arrangement of their own
// folders is followed and never altered.
func TestANoteIsMadeInTheFolderItWasAskedFor(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"physics/Ontology.md": "---\ntitle: Ontology\n---\n\n# Ontology\n",
	})

	answer, err := f.client.CreateNote(t.Context(), connect.NewRequest(&v1.CreateNoteRequest{
		Title: "Entropy",
		Path:  "physics",
		Links: []*v1.Link{{To: "physics/Ontology.md", Role: v1.Role_ROLE_PARENT}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if path := answer.Msg.GetPath(); path != "physics/Entropy.md" {
		t.Fatalf("the note was filed at %q", path)
	}
	// The link is written by the other note's name, which is what carries across
	// folders.
	if made := fileAt(t, f.root, "physics/Entropy.md"); !strings.Contains(made, "to: Ontology") {
		t.Errorf("the link was not written by name:\n%s", made)
	}
}

// TestANoteMadeIsInTheIndexBeforeTheAnswerComesBack. This is what puts a note
// in the picture as soon as it exists.
func TestANoteMadeIsInTheIndexBeforeTheAnswerReachesBack(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"Ontology.md": "---\ntitle: Ontology\n---\n\n# Ontology\n",
	})

	if _, err := f.client.CreateNote(t.Context(), connect.NewRequest(&v1.CreateNoteRequest{
		Title: "Entropy",
		Links: []*v1.Link{{To: "Ontology.md", Role: v1.Role_ROLE_PARENT}},
	})); err != nil {
		t.Fatal(err)
	}

	found, err := f.installation.Queries().GetNotes(t.Context(), f.installation.GetCurrentVault().ID, []string{"Entropy.md"})
	if err != nil {
		t.Fatal(err)
	}
	if found["Entropy.md"].Title != "Entropy" {
		t.Errorf("the index answers %q for a note that was just made", found["Entropy.md"].Title)
	}
}

// TestANoteMadeWhereOneAlreadyIsIsRefused. Whether the path was free is the
// filesystem's to answer at the moment the file is made, and what is there is
// left currently.
func TestANoteMadeWhereOneAlreadyIsIsRefused(t *testing.T) {
	const target = "---\ntitle: Ontology\n---\n\n# Ontology\n"
	f := openWindow(t, nil, map[string]string{"Ontology.md": target})

	answer, err := f.client.CreateNote(t.Context(), connect.NewRequest(&v1.CreateNoteRequest{
		Title: "Ontology",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_OCCUPIED {
		t.Errorf("a name already taken was answered with %v", code)
	}
	if answer.Msg.GetPath() != "" {
		t.Errorf("a note that was not made was filed at %q", answer.Msg.GetPath())
	}
	if now := fileAt(t, f.root, "Ontology.md"); now != target {
		t.Errorf("the note that was already there was written:\n%s", now)
	}
}

// TestALinkNamingNoRoleLeavesNoNote. A link the application acts on carries one
// of the roles the schema names, and the error comes before the file does.
func TestALinkNamingNoRolePreservesNoNote(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"Ontology.md": "---\ntitle: Ontology\n---\n\n# Ontology\n",
	})

	_, err := f.client.CreateNote(t.Context(), connect.NewRequest(&v1.CreateNoteRequest{
		Title: "Entropy",
		Links: []*v1.Link{{To: "Ontology.md", Role: v1.Role_ROLE_UNSPECIFIED}},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("a link with no role was answered with %v", err)
	}
	if _, err := os.Stat(filepath.Join(f.root, "Entropy.md")); err == nil {
		t.Error("a note was made for a link that cannot be written")
	}
}

// TestTwoNotesAreJoinedFromTheOneTheLinkIsWrittenIn. A link is one end's
// account of a relationship: the other note is not touched.
func TestTwoNotesAreJoinedFromTheOneTheLinkIsSavedIn(t *testing.T) {
	const other = "---\ntitle: Entropy\n---\n\n# Entropy\n"
	f := openWindow(t, nil, map[string]string{
		"Ontology.md": "---\ntitle: Ontology\ntags:\n  - draft\n---\n\n# Ontology\n",
		"Entropy.md":  other,
	})

	answer, err := f.client.WriteLink(t.Context(), connect.NewRequest(&v1.WriteLinkRequest{
		Path: "Ontology.md",
		Link: &v1.Link{To: "Entropy.md", Role: v1.Role_ROLE_CHILD},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("the link failed: %v", code)
	}

	joined := fileAt(t, f.root, "Ontology.md")
	if !strings.Contains(joined, "to: Entropy") || !strings.Contains(joined, "role: child") {
		t.Errorf("the note does not say what it was joined to:\n%s", joined)
	}
	// The keys the person wrote are theirs, and come out as they went in.
	if !strings.Contains(joined, "tags:\n  - draft\n") {
		t.Errorf("a key the application does not own was rewritten:\n%s", joined)
	}
	if now := fileAt(t, f.root, "Entropy.md"); now != other {
		t.Errorf("the note at the other end was written:\n%s", now)
	}
}

// TestANoteIsMadeUnderTheNoteItWasMadeFromAndNotItsNamesake. A name is read
// back as an exact path from the root before it is read as a neighbour, so a
// note made from one of two notes sharing a name carries the path it was made
// from. Written by name, it would hang off the other one, and the picture the
// person is looking at would not draw it at all.
func TestANoteIsMadeBelowTheNoteItWasMadeFromAndNotItsNamesake(t *testing.T) {
	f := openWindow(t, nil, nil)
	// Both are made through the window, which is what puts them in the index:
	// the vault is scanned at startup, and a test is not started.
	for _, folder := range []string{"", "physics"} {
		if _, err := f.client.CreateNote(t.Context(), connect.NewRequest(&v1.CreateNoteRequest{
			Title: "Ontology", Path: folder,
		})); err != nil {
			t.Fatal(err)
		}
	}

	answer, err := f.client.CreateNote(t.Context(), connect.NewRequest(&v1.CreateNoteRequest{
		Title: "Entropy",
		Path:  "physics",
		Links: []*v1.Link{{To: "physics/Ontology.md", Role: v1.Role_ROLE_PARENT}},
	}))
	if err != nil {
		t.Fatal(err)
	}

	if made := fileAt(t, f.root, answer.Msg.GetPath()); !strings.Contains(made, "to: physics/Ontology") {
		t.Errorf("the link was written by a name two notes answer to:\n%s", made)
	}

	center, err := f.client.GetNeighbourhood(t.Context(),
		connect.NewRequest(&v1.GetNeighbourhoodRequest{Path: "physics/Ontology.md"}))
	if err != nil {
		t.Fatal(err)
	}
	var children []string
	for _, related := range center.Msg.GetRelated() {
		if related.GetRelation() == v1.Relation_RELATION_CHILD {
			children = append(children, related.GetNote().GetPath())
		}
	}
	if len(children) != 1 || children[0] != "physics/Entropy.md" {
		t.Errorf("the note it was made from has children %v, want the note just made", children)
	}
}

// TestEveryRoleTheSchemaNamesIsWrittenUnderThatWord. The wire and the core call
// a link's role by one word, so the role a request names is the role the note
// comes back carrying.
func TestEveryRoleTheSchemaNamesIsSavedBelowThatWord(t *testing.T) {
	for _, one := range []struct {
		role v1.Role
		word string
	}{
		{v1.Role_ROLE_PARENT, "parent"},
		{v1.Role_ROLE_CHILD, "child"},
		{v1.Role_ROLE_JUMP, "jump"},
		{v1.Role_ROLE_REF, "ref"},
		{v1.Role_ROLE_ATTACHMENT, "attachment"},
	} {
		t.Run(one.word, func(t *testing.T) {
			f := openWindow(t, nil, map[string]string{
				"Ontology.md": "---\ntitle: Ontology\n---\n\n# Ontology\n",
			})

			answer, err := f.client.CreateNote(t.Context(), connect.NewRequest(&v1.CreateNoteRequest{
				Title: "Entropy",
				Links: []*v1.Link{{To: "Ontology.md", Role: one.role}},
			}))
			if err != nil {
				t.Fatal(err)
			}

			made := fileAt(t, f.root, answer.Msg.GetPath())
			if !strings.Contains(made, "role: "+one.word) {
				t.Errorf("the link does not carry the %s role:\n%s", one.word, made)
			}
		})
	}
}

// The window draws the vault as the folders and files a person filed it under,
// and moves them about. What the vault answers is what the window has to be
// able to say about either.

// getEntries is what one folder of the vault holds, as the window asks for it.
func getEntries(t *testing.T, f *going, pos string) []*v1.Entry {
	t.Helper()
	answer, err := f.client.ListFiles(t.Context(), connect.NewRequest(&v1.ListFilesRequest{Path: pos}))
	if err != nil {
		t.Fatal(err)
	}
	return answer.Msg.GetEntries()
}

// getEntryNames is the entries by name, in the order they arrived.
func getEntryNames(entries []*v1.Entry) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.GetName())
	}
	return out
}

// folder says whether a folder is on the disk.
func folder(t *testing.T, root, path string) bool {
	t.Helper()
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(path)))
	return err == nil && info.IsDir()
}

// TestListingAFolderAnswersInTheOrderToDrawItIn. Folders come first, then
// files, each group by name with case ignored.
func TestListingAFolderAnswersInTheOrderToDrawItIn(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"Zeta.md":            "# Zeta\n",
		"alpha.md":           "# Alpha\n",
		"Notes.txt":          "a list\n",
		"physics/Heat.md":    "# Heat\n",
		"physics/Entropy.md": "# Entropy\n",
	})
	testsupport.WriteBook(t, f.root, "library/A Book.epub")

	root := getEntries(t, f, "")
	want := []string{"library", "physics", "alpha.md", "Notes.txt", "Zeta.md"}
	if got := getEntryNames(root); !slices.Equal(got, want) {
		t.Fatalf("the root is drawn as %v", got)
	}
	target := map[string]*v1.Entry{}
	for _, entry := range root {
		target[entry.GetName()] = entry
	}
	if entry := target["physics"]; !entry.GetIsFolder() || entry.GetPath() != "physics" {
		t.Errorf("the folder came back as %+v", entry)
	}
	if kind := target["alpha.md"].GetKind(); kind != v1.SourceKind_SOURCE_KIND_NOTE {
		t.Errorf("a note is held as %v", kind)
	}
	if kind := target["Notes.txt"].GetKind(); kind != v1.SourceKind_SOURCE_KIND_UNSPECIFIED {
		t.Errorf("a file the vault holds no source for is held as %v", kind)
	}

	entries := getEntries(t, f, "physics")
	if got := getEntryNames(entries); !slices.Equal(got, []string{"Entropy.md", "Heat.md"}) {
		t.Fatalf("the folder is drawn as %v", got)
	}
	if path := entries[0].GetPath(); path != "physics/Entropy.md" {
		t.Errorf("the note under a folder is called %q", path)
	}

	shelf := getEntries(t, f, "library")
	if len(shelf) != 1 || shelf[0].GetKind() != v1.SourceKind_SOURCE_KIND_BOOK {
		t.Errorf("the library is drawn as %+v", shelf)
	}
}

// TestAListingLeavesOutWhatTheVaultLeavesAlone. A name beginning with a dot
// belongs to a tool, and the folder this application keeps for itself is its
// own.
func TestAListingPreservesOutWhatTheVaultPreservesAlone(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"Entropy.md":      "# Entropy\n",
		".secret.md":      "# Secret\n",
		".hidden/Kept.md": "# Kept\n",
	})
	if isGone(t, f.root, ".secret.md") || !folder(t, f.root, ".hidden") ||
		!folder(t, f.root, filesystem.DefaultServiceDir) {
		t.Fatal("the vault does not hold what the listing has to leave out")
	}

	if got := getEntryNames(getEntries(t, f, "")); !slices.Equal(got, []string{"Entropy.md"}) {
		t.Errorf("the root is drawn as %v", got)
	}
}

// TestAFolderThatIsNotThereIsNotAnEmptyOne. A listing carries no error code, so
// the two are told apart by the answer itself.
func TestAFolderThatIsNotThereIsNotAnEmptyOne(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Entropy.md": "# Entropy\n"})

	_, err := f.client.ListFiles(t.Context(), connect.NewRequest(&v1.ListFilesRequest{Path: "physics"}))
	if code := connect.CodeOf(err); code != connect.CodeNotFound {
		t.Errorf("a folder that is not there was answered with %v", code)
	}
	_, err = f.client.ListFiles(t.Context(),
		connect.NewRequest(&v1.ListFilesRequest{Path: "../elsewhere"}))
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Errorf("a folder outside the vault was answered with %v", code)
	}
}

// TestEverythingTheVaultHoldsMoves. A file of any kind travels, whether
// anything reads it or not.
func TestEverythingTheVaultContainsMoves(t *testing.T) {
	const picture = "\x89PNG\r\n\x1a\n"
	f := openWindow(t, nil, map[string]string{"Diagram.png": picture})
	testsupport.WriteBook(t, f.root, "A Book.epub")
	waitForScan(t, f)

	for _, one := range []struct{ from, to string }{
		{from: "A Book.epub", to: "library/A Book.epub"},
		{from: "Diagram.png", to: "pictures/Diagram.png"},
	} {
		answer, err := f.client.MoveFile(t.Context(), connect.NewRequest(&v1.MoveFileRequest{
			From: one.from, To: one.to,
		}))
		if err != nil {
			t.Fatal(err)
		}
		if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
			t.Fatalf("moving %s failed: %v", one.from, code)
		}
		moved := answer.Msg.GetMoved()
		if moved.GetFrom() != one.from || moved.GetTo() != one.to {
			t.Errorf("what the file did came back as %+v", moved)
		}
		if !isGone(t, f.root, one.from) {
			t.Errorf("%s is still where it was", one.from)
		}
	}
	if now := fileAt(t, f.root, "pictures/Diagram.png"); now != picture {
		t.Errorf("the picture holds %q", now)
	}
	if len(getEntries(t, f, "library")) != 1 {
		t.Error("the book is not in the folder it was sent to")
	}
}

// TestAFolderOfNotesMovesAndIsStillLinkedTo. A link written by name finds its
// note wherever it is filed, so a folder moves without a link being rewritten.
func TestAFolderOfNotesMovesAndIsStillLinkedTo(t *testing.T) {
	const pointing = "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n\n# Heat\n"
	f := openWindow(t, nil, map[string]string{
		"physics/Entropy.md": "# Entropy\n",
		"Heat.md":            pointing,
	})
	waitForScan(t, f)

	answer, err := f.client.MoveFile(t.Context(), connect.NewRequest(&v1.MoveFileRequest{
		From: "physics", To: "science/physics",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("the folder failed: %v", code)
	}
	if repaired := answer.Msg.GetMoved().GetRepaired(); len(repaired) != 0 {
		t.Errorf("a link written by name was repaired: %v", repaired)
	}
	if now := fileAt(t, f.root, "Heat.md"); now != pointing {
		t.Errorf("the note that points at the folder was written:\n%s", now)
	}
	if now := fileAt(t, f.root, "science/physics/Entropy.md"); now != "# Entropy\n" {
		t.Errorf("the note under the folder holds:\n%s", now)
	}

	center, err := f.client.GetNeighbourhood(t.Context(), connect.NewRequest(&v1.GetNeighbourhoodRequest{
		Path: "Heat.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	related := center.Msg.GetRelated()
	if len(related) != 1 {
		t.Fatalf("want the one note the link reaches, got %+v", related)
	}
	if path := related[0].GetNote().GetPath(); path != "science/physics/Entropy.md" {
		t.Errorf("the link reaches %q", path)
	}
}

// TestAMoveOntoATakenNameLeavesBothWhereTheyAre. Two files arriving at one path
// is a question only the person can answer.
func TestAMoveOntoATakenNamePreservesBothWhereTheyAre(t *testing.T) {
	const target = "# Entropy in physics\n"
	f := openWindow(t, nil, map[string]string{
		"Entropy.md":         "# Entropy\n",
		"physics/Entropy.md": target,
	})
	waitForScan(t, f)

	answer, err := f.client.MoveFile(t.Context(), connect.NewRequest(&v1.MoveFileRequest{
		From: "Entropy.md", To: "physics/Entropy.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_OCCUPIED {
		t.Errorf("a name already taken was answered with %v", code)
	}
	if moved := answer.Msg.GetMoved(); moved != nil {
		t.Errorf("a rejected move said the file did %+v", moved)
	}
	if now := fileAt(t, f.root, "Entropy.md"); now != "# Entropy\n" {
		t.Errorf("the note that was to move holds:\n%s", now)
	}
	if now := fileAt(t, f.root, "physics/Entropy.md"); now != target {
		t.Errorf("the note already there was written:\n%s", now)
	}
}

// TestAMoveOfAPathWithNothingAtItLeavesNothingBehind, and says so as a
// an error: the core is what names a path as missing.
func TestAMoveOfAPathWithNothingAtItPreservesNothingPrior(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Entropy.md": "# Entropy\n"})

	answer, err := f.client.MoveFile(t.Context(), connect.NewRequest(&v1.MoveFileRequest{
		From: "physics", To: "science",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_MISSING {
		t.Errorf("a path with nothing at it was answered with %v", code)
	}
	if answer.Msg.GetMoved() != nil {
		t.Error("a move that did not happen came back with a file under a different name")
	}
	if folder(t, f.root, "science") {
		t.Error("a move that did not happen left a folder behind")
	}
}

// TestARemovedFolderGoesToTheTrashWithEverythingUnderIt, and what pointed into
// it is reported and not repaired.
func TestARemovedFolderGoesToTheTrashWithEverythingBelowIt(t *testing.T) {
	const pointing = "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n\n# Heat\n"
	f := openWindow(t, nil, map[string]string{
		"physics/Entropy.md": "# Entropy\n",
		"Heat.md":            pointing,
	})
	waitForScan(t, f)

	answer, err := f.client.RemoveFile(t.Context(), connect.NewRequest(&v1.RemoveFileRequest{
		Path: "physics",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("the folder failed: %v", code)
	}
	if trashed := answer.Msg.GetTrashed(); trashed != ".trash/physics" {
		t.Errorf("the folder sits at %q", trashed)
	}
	dangling := answer.Msg.GetDangling()
	if len(dangling) != 1 || dangling[0] != "Heat.md" {
		t.Errorf("what now reaches nothing came back as %v", dangling)
	}
	if !isGone(t, f.root, "physics/Entropy.md") {
		t.Error("the folder is still where it was")
	}
	if now := fileAt(t, f.root, ".trash/physics/Entropy.md"); now != "# Entropy\n" {
		t.Errorf("the note in the trash holds:\n%s", now)
	}
	if now := fileAt(t, f.root, "Heat.md"); now != pointing {
		t.Errorf("the note whose link reaches nothing was written:\n%s", now)
	}
}

// TestAFolderIsMadeWithTheFoldersAboveIt. An empty folder is a place to file
// notes in, and the window can make one before there is anything to put there.
func TestAFolderIsMadeWithTheFoldersAboveIt(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Entropy.md": "# Entropy\n"})

	answer, err := f.client.CreateFolder(t.Context(), connect.NewRequest(&v1.CreateFolderRequest{
		Path: "science/physics",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("the folder failed: %v", code)
	}
	if !folder(t, f.root, "science/physics") {
		t.Fatal("the folder is not on the disk")
	}
	if got := getEntryNames(getEntries(t, f, "science")); !slices.Equal(got, []string{"physics"}) {
		t.Errorf("the folder above is drawn as %v", got)
	}
	if len(getEntries(t, f, "science/physics")) != 0 {
		t.Error("the folder that was made holds something")
	}

	// Making it again is the outcome that was asked for.
	again, err := f.client.CreateFolder(t.Context(), connect.NewRequest(&v1.CreateFolderRequest{
		Path: "science/physics",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := again.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Errorf("a folder that is already there was answered with %v", code)
	}
}

// TestAFolderIsNotMadeWhereAFileIsFiled. Nothing is written over the file.
func TestAFolderIsNotMadeWhereAFileIsFiled(t *testing.T) {
	const target = "# Entropy\n"
	f := openWindow(t, nil, map[string]string{"Entropy.md": target})

	answer, err := f.client.CreateFolder(t.Context(), connect.NewRequest(&v1.CreateFolderRequest{
		Path: "Entropy.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_OCCUPIED {
		t.Errorf("a path a file already holds was answered with %v", code)
	}
	if now := fileAt(t, f.root, "Entropy.md"); now != target {
		t.Errorf("the file was written:\n%s", now)
	}
}

// TestAListingSaysWhichOfFourEachNoteIs. The folder says which of its entries
// are notes and the index says what each of those notes is, so a tree draws a
// deck as a deck without opening it.
func TestAListingReturnsWhichOfFourEachNoteIs(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"Entropy.md":  "# Entropy\n",
		"Animal.md":   "---\ntype: stencil\nfields:\n  - Height\n---\n\n## Recognise\n",
		"Animals.md":  "---\ntype: deck\n---\n\n## Llama\n\n[[Animal]]\n",
		"Sanskrit.md": "---\ntype: preset\nminutes_a_day: 20\n---\n\n# Sanskrit\n",
		"Notes.txt":   "a list\n",
	})
	f.read(t)

	target := map[string]*v1.Entry{}
	for _, entry := range getEntries(t, f, "") {
		target[entry.GetName()] = entry
	}

	want := map[string]string{
		"Animals.md":  "deck",
		"Animal.md":   "stencil",
		"Sanskrit.md": "preset",
		"Entropy.md":  "",
		"Notes.txt":   "",
	}
	for name, one := range want {
		if got := target[name].GetType(); got != one {
			t.Errorf("%s is drawn as %v, want %v", name, got, one)
		}
	}
}

// TestMorePathsThanStandingAnswersAtOnceAreRefused. A path with nothing at it
// is absent from the answer, so an answer cut to fit the ceiling would be one a
// caller cannot tell from a path holding nothing.
func TestMorePathsThanActiveAnswersAtOnceAreRefused(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Entropy.md": "# Entropy\n"})
	f.read(t)

	paths := make([]string, 0, 201)
	for i := range 201 {
		paths = append(paths, fmt.Sprintf("Note%03d.md", i))
	}

	_, err := f.client.ListFileKinds(t.Context(), connect.NewRequest(&v1.ListFileKindsRequest{Paths: paths}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("err = %v, want the paths named to be rejected", err)
	}
}

// getKindsByPath is what the vault says it holds at each of those paths, by
// path.
func getKindsByPath(t *testing.T, f *going, paths ...string) map[string]*v1.FileKind {
	t.Helper()
	answer, err := f.client.ListFileKinds(t.Context(),
		connect.NewRequest(&v1.ListFileKindsRequest{Paths: paths}))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*v1.FileKind{}
	for _, one := range answer.Msg.GetKinds() {
		out[one.GetPath()] = one
	}
	return out
}

// TestAPathSaysWhatStandsThere. A window holding a path and nothing else opens
// what stands there in the editor made for it, and this is the one question
// that says which that is.
func TestAPathReturnsWhatExistsThere(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"Entropy.md":   "# Entropy\n",
		"Animal.md":    "---\ntype: stencil\nfields:\n  - Height\n---\n\n## Recognise\n",
		"Animals.md":   "---\ntype: deck\n---\n\n## Llama\n\n[[Animal]]\n",
		"Sanskrit.md":  "---\ntype: preset\nminutes_a_day: 20\n---\n\n# Sanskrit\n",
		"Physics.epub": "an epub\n",
		"Scan.pdf":     "a scan\n",
		"Notes.txt":    "a list\n",
	})
	f.read(t)

	target := getKindsByPath(t, f, "Animals.md", "Animal.md", "Sanskrit.md",
		"Entropy.md", "Physics.epub", "Scan.pdf", "Notes.txt", "Gone.md")

	// each in the reader made for it.
	want := map[string]struct {
		kind   v1.SourceKind
		is     string
		format v1.BookFormat
	}{
		"Animals.md":   {v1.SourceKind_SOURCE_KIND_NOTE, "deck", v1.BookFormat_BOOK_FORMAT_UNSPECIFIED},
		"Animal.md":    {v1.SourceKind_SOURCE_KIND_NOTE, "stencil", v1.BookFormat_BOOK_FORMAT_UNSPECIFIED},
		"Sanskrit.md":  {v1.SourceKind_SOURCE_KIND_NOTE, "preset", v1.BookFormat_BOOK_FORMAT_UNSPECIFIED},
		"Entropy.md":   {v1.SourceKind_SOURCE_KIND_NOTE, "", v1.BookFormat_BOOK_FORMAT_UNSPECIFIED},
		"Physics.epub": {v1.SourceKind_SOURCE_KIND_BOOK, "", v1.BookFormat_BOOK_FORMAT_EPUB},
		"Scan.pdf":     {v1.SourceKind_SOURCE_KIND_BOOK, "", v1.BookFormat_BOOK_FORMAT_PDF},
		"Notes.txt":    {v1.SourceKind_SOURCE_KIND_UNSPECIFIED, "", v1.BookFormat_BOOK_FORMAT_UNSPECIFIED},
	}
	for path, one := range want {
		status, reply := target[path]
		if !reply {
			t.Errorf("%s was not answered about at all", path)
			continue
		}
		if status.GetKind() != one.kind || status.GetType() != one.is || status.GetFormat() != one.format {
			t.Errorf("%s stands as %v %v %v, want %v %v %v",
				path, status.GetKind(), status.GetType(), status.GetFormat(),
				one.kind, one.is, one.format)
		}
	}

	// A path with nothing at it is not a file the vault leaves alone, and the
	// two are told apart by one being answered about and the other not.
	if _, replyVal := target["Gone.md"]; replyVal {
		t.Error("a path with nothing at it was answered about anyway")
	}
}

// TestABookStandsThereBeforeAnythingHasReadIt. The index holds no note for a
// book and never will, so an answer drawn from it alone leaves every book out
// and a window reads that absence as an ordinary note.
func TestABookExistsThereBeforeAnythingHasReadIt(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Entropy.md": "# Entropy\n"})

	// Written after the vault was opened and asked about at once, so nothing
	// has had the chance to read it.
	if err := os.WriteFile(filepath.Join(f.root, "Physics.epub"), []byte("an epub\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	status := getKindsByPath(t, f, "Physics.epub")["Physics.epub"]
	if status.GetKind() != v1.SourceKind_SOURCE_KIND_BOOK {
		t.Errorf("a book nothing has read stands as %v", status.GetKind())
	}
}

// TestAFileLetGoOfOverTheTreeArrivesAndIsSaid is the path a drop takes: the
// window hands over what the person release, the vault holds it, and everyone
// drawing the vault is told where to look.
//
// A picture is what the watcher reports to nobody, so the stream is the only
// way the tree finds out about one.
func TestAFileLetGoOfOverTheTreeArrivesAndIsSaid(t *testing.T) {
	client, _, root, installation := openVaultWithWindow(t, map[string]string{
		"physics/Entropy.md": "---\ntitle: Entropy\n---\n\n# Entropy\n",
	})

	outside := t.TempDir()
	cover := filepath.Join(outside, "Cover.png")
	if err := os.WriteFile(cover, []byte("PNG"), 0o644); err != nil {
		t.Fatal(err)
	}

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

	installation.Imports(t.Context(), "physics", []string{cover})

	target, err := os.ReadFile(filepath.Join(root, "physics", "Cover.png"))
	if err != nil {
		t.Fatalf("the file did not arrive: %v", err)
	}
	if string(target) != "PNG" {
		t.Errorf("the file arrived as %q", target)
	}

	select {
	case paths := <-reported:
		if !slices.Equal(paths, []string{"physics/Cover.png"}) {
			t.Errorf("reported %v", paths)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the drop never reached the listener")
	}
}

// A file that is already there stays as it is, and what stopped the drop stands
// in the list of what the window is doing.
func TestAFileLetGoOfOverANameAlreadyThereIsSaid(t *testing.T) {
	_, rendered, root, installation := openVaultWithWindow(t, map[string]string{"Entropy.md": "# Mine\n"})

	outside := t.TempDir()
	theirs := filepath.Join(outside, "Entropy.md")
	if err := os.WriteFile(theirs, []byte("# Theirs\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	installation.Imports(t.Context(), "", []string{theirs})

	target, err := os.ReadFile(filepath.Join(root, "Entropy.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(target) != "# Mine\n" {
		t.Errorf("the note that was there is now %q", target)
	}

	answer, err := rendered.WatchTasks(t.Context(), connect.NewRequest(&v1.WatchTasksRequest{Window: wire.Editor}))
	if err != nil {
		t.Fatal(err)
	}
	defer answer.Close()
	if !answer.Receive() {
		t.Fatalf("nothing said what the window is doing: %v", answer.Err())
	}

	var response string
	for _, item := range answer.Msg().GetTasks() {
		if item.GetError() != "" {
			response = item.GetError()
		}
	}
	if response == "" {
		t.Fatal("the rejection was never said")
	}
	if !strings.Contains(response, "Entropy.md") {
		t.Errorf("the rejection says %q", response)
	}
}
