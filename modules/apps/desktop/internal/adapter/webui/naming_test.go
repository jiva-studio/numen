package webui_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
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
	for range 500 {
		if f.opened.API.Ready.Load() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the vault was not read")
}

// TestRenamingWritesTheNoteAndMovesTheFile. A note is shown by its title, so
// the title is what is written, and the file follows it.
func TestRenamingWritesTheNoteAndMovesTheFile(t *testing.T) {
	f := quitting(t, nil, map[string]string{
		"Old.md": "---\ntitle: Old\n---\n\n# Old\n",
	})

	answer, err := f.client.Rename(t.Context(), connect.NewRequest(&v1.RenameRequest{
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

// TestRenamingSaysWhichLinksReachSomethingElseNow. Two notes under one name is
// the person's to settle, and the window can only say which link it is.
func TestRenamingSaysWhichLinksReachSomethingElseNow(t *testing.T) {
	f := quitting(t, nil, map[string]string{
		"Entropy.md":         "# Entropy\n",
		"physics/Entropy.md": "# Entropy\n",
		"Heat.md":            "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n\n# Heat\n",
	})
	scanned(t, f)

	answer, err := f.client.Rename(t.Context(), connect.NewRequest(&v1.RenameRequest{
		Path: "Entropy.md", Title: "Thermodynamics",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the rename was refused: %v", refusal)
	}

	retargeted := answer.Msg.GetMoved().GetRetargeted()
	if len(retargeted) != 1 {
		t.Fatalf("want the one link that means something else, got %+v", retargeted)
	}
	if in := retargeted[0].GetIn(); in != "Heat.md" {
		t.Errorf("the link is written in %q", in)
	}
	if target := retargeted[0].GetTarget(); target != "Entropy" {
		t.Errorf("the link is written by %q", target)
	}
	if now := retargeted[0].GetNow(); now != "physics/Entropy.md" {
		t.Errorf("the link reaches %q", now)
	}
}

// TestRenamingOntoATakenNameSaysWhatTheNoteIsCalled. The note is brought into
// line before the file is, so a refused move leaves a note that says what it is
// called under a filename that does not.
func TestRenamingOntoATakenNameSaysWhatTheNoteIsCalled(t *testing.T) {
	f := quitting(t, nil, map[string]string{
		"Old.md":     "# Old\n",
		"Entropy.md": "# Entropy\n",
	})

	answer, err := f.client.Rename(t.Context(), connect.NewRequest(&v1.RenameRequest{
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
	if by := answer.Msg.GetBy(); by != v1.Naming_NAMING_HEADING {
		t.Errorf("the heading named the note and the answer says %v", by)
	}
	if now := fileAt(t, f.root, "Entropy.md"); now != "# Entropy\n" {
		t.Errorf("the note already there was written:\n%s", now)
	}
}

// TestRenamingRefusesATitleNoFileCanBeNamedAfter. The note is not opened for a
// title that leaves nothing to name it.
func TestRenamingRefusesATitleNoFileCanBeNamedAfter(t *testing.T) {
	const held = "# Old\n"
	f := quitting(t, nil, map[string]string{"Old.md": held})

	for name, title := range map[string]string{
		"nothing at all": "",
		"only spaces":    "   ",
		"only dots":      "...",
		"a line break":   "one\ntwo",
	} {
		t.Run(name, func(t *testing.T) {
			answer, err := f.client.Rename(t.Context(), connect.NewRequest(&v1.RenameRequest{
				Path: "Old.md", Title: title,
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
		})
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

			answer, err := f.client.Rename(t.Context(), connect.NewRequest(&v1.RenameRequest{
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

	answer, err := f.client.Remove(t.Context(), connect.NewRequest(&v1.RemoveRequest{
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

	answer, err := f.client.Remove(t.Context(), connect.NewRequest(&v1.RemoveRequest{
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

	if _, err := f.client.Rename(context.Background(), connect.NewRequest(&v1.RenameRequest{
		Path: "Entropy.md", Title: "Thermodynamics",
	})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("a rename after the door was shut was answered with %v", err)
	}
	if _, err := f.client.Remove(context.Background(), connect.NewRequest(&v1.RemoveRequest{
		Path: "Entropy.md",
	})); connect.CodeOf(err) != connect.CodeUnavailable {
		t.Errorf("a remove after the door was shut was answered with %v", err)
	}
	if now := fileAt(t, f.root, "Entropy.md"); now != "# Entropy\n" {
		t.Errorf("the note was written after the door was shut:\n%s", now)
	}
}

// TestRenamingSaysWhatANoteCannotBeCalled. A title the vault cannot show the
// note under is refused, and which of the three would have to say it decides
// whether it can.
func TestRenamingSaysWhatANoteCannotBeCalled(t *testing.T) {
	for name, c := range map[string]struct {
		held  string
		title string
		want  v1.Refusal
		file  string
	}{
		"a heading closes on the hash the title ends with": {
			held:  "# Old\n",
			title: "C#",
			want:  v1.Refusal_REFUSAL_UNNAMEABLE,
		},
		"a filename cannot carry it and a heading says it as something else": {
			held:  "A measure.\n",
			title: "C#",
			want:  v1.Refusal_REFUSAL_UNNAMEABLE,
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
		"a frontmatter block that is never closed": {
			held:  "---\ntitle: Old\n# Old\n",
			title: "Entropy",
			want:  v1.Refusal_REFUSAL_UNREADABLE,
		},
	} {
		t.Run(name, func(t *testing.T) {
			f := quitting(t, nil, map[string]string{"Old.md": c.held})

			answer, err := f.client.Rename(t.Context(), connect.NewRequest(&v1.RenameRequest{
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

// TestARenamedNoteIsStillLinkedTo. A rename that filed a note under a name no
// link can be written by would break every link pointing at it, and nothing
// could repair them.
func TestARenamedNoteIsStillLinkedTo(t *testing.T) {
	const pointing = "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n\n# Heat\n"
	for name, title := range map[string]string{
		"a title in double brackets": "Notes [[draft]]",
		"a title carrying a hash":    "Issue #42",
		"a title carrying a pipe":    "Either|Or",
	} {
		t.Run(name, func(t *testing.T) {
			f := quitting(t, nil, map[string]string{
				"Entropy.md": "# Entropy\n",
				"Heat.md":    pointing,
			})
			scanned(t, f)

			answer, err := f.client.Rename(t.Context(), connect.NewRequest(&v1.RenameRequest{
				Path: "Entropy.md", Title: title,
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
		})
	}
}
