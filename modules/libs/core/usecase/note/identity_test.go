package note_test

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/jiva-studio/numen/modules/libs/core/markdown"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// started are the notes a run begins from: one this application made, one it
// made and nobody has titled, one written in another editor, and one written
// there with a title of its own.
var started = []string{
	"---\nid: 01J8F3K2M9QRSTVWXYZ0123456\ntitle: Old\n---\n\nA measure.\n",
	"---\nid: 01J8F3K2M9QRSTVWXYZ0123456\n---\n\nA measure.\n",
	"# Old\n\nA measure.\n",
	"---\ntitle: Old\n---\n\nA measure.\n",
}

// titles are what a note is renamed to: names a filename carries, one it cannot
// carry, and one that is a name in another script.
var titles = []string{"Entropy", "Order", "TCP/IP", "Энтропия", "a: b"}

// folders are where a note is moved to.
var folders = []string{"", "physics", "physics/thermo"}

// writeNote writes one run's note into a folder of its own and tells the index
// about it. One vault stands for the whole check: opening an index for every
// run of a property is what makes one too slow to keep.
func writeNote(rt *rapid.T, c changing, folder, raw string) string {
	rt.Helper()
	path := folder + "/Old.md"
	on := filepath.Join(c.vault.Path, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(on), 0o755); err != nil {
		rt.Fatal(err)
	}
	if err := os.WriteFile(on, []byte(raw), 0o644); err != nil {
		rt.Fatal(err)
	}
	if err := c.index(rt.Context(), c.vault, []string{path}); err != nil {
		rt.Fatal(err)
	}
	return path
}

// getIdentifier is the identifier the note at this path stands under, and
// whether it carries one at all.
func getIdentifier(t *rapid.T, root, path string) (string, bool) {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	doc, err := markdown.Open(raw)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	return doc.Identifier()
}

// A note keeps the identifier it has under any sequence of renames and moves,
// and a move never gives one to a note without.
//
// A note's identity is not its path; the identifier it has is the one it
// keeps, an identifier is written when the application changes what is in a
// note, and a move is not a change to what is in one.
func TestANoteKeepsItsIdentifierAcrossRenameAndMove(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{"other.md": "# Other\n"})
	runs := 0
	rapid.Check(t, func(rt *rapid.T) {
		runs++
		folder := fmt.Sprintf("run%04d", runs)
		path := writeNote(rt, c, folder, rapid.SampledFrom(started).Draw(rt, "note"))
		was, stamped := getIdentifier(rt, c.vault.Path, path)

		for step := range rapid.IntRange(1, 6).Draw(rt, "steps") {
			var moved bool
			switch rapid.SampledFrom([]string{
				"move", "rename", "rename apart", "called",
			}).Draw(rt, fmt.Sprintf("step %d", step)) {
			case "move":
				// A move renames the file and changes no byte in it.
				moved = true
				to := filepath.ToSlash(filepath.Join(folder,
					rapid.SampledFrom(folders).Draw(rt, "folder"),
					fmt.Sprintf("Moved %d.md", step)))
				out, err := c.move().Execute(rt.Context(), c.vault, path, to)
				if err != nil {
					rt.Fatalf("move %s to %s: %v", path, to, err)
				}
				if out.Landed {
					path = out.To
				}
			case "rename", "rename apart":
				renaming := c.rename()
				if rapid.Bool().Draw(rt, "apart") {
					renaming = c.apart()
				}
				out, err := renaming.Execute(rt.Context(), c.vault, path,
					rapid.SampledFrom(titles).Draw(rt, "title"))
				if err != nil && !errors.Is(err, note.ErrUnnameable) {
					rt.Fatalf("rename %s: %v", path, err)
				}
				if out.Path != "" {
					path = out.Path
				}
			case "called":
				// The name the file carries written into the note, which is
				// what a rename made outside the application settles into.
				if err := c.move().WriteFilenameAsTitle(rt.Context(), c.vault, path); err != nil {
					rt.Fatalf("call %s by its filename: %v", path, err)
				}
			}

			now, holds := getIdentifier(rt, c.vault.Path, path)
			switch {
			case stamped && (!holds || now != was):
				rt.Fatalf("a note stamped %q stands at %q after step %d, filed at %s",
					was, now, step, path)
			case moved && !stamped && holds:
				rt.Fatalf("a move stamped %q into a note that carried none, at %s",
					now, path)
			}
			if holds {
				was, stamped = now, true
			}
		}
	})
}

// A rename the filename alone carries writes nothing into the note, so a note
// without an identifier is still without one afterwards. A rename that writes
// the title is a change to what is in the note, and stamps it.
//
// A rename that writes the title stamps, and one the filename alone carries
// does not.
func TestOnlyARenameThatWritesTheTitleStamps(t *testing.T) {
	t.Parallel()
	c := changeable(t, map[string]string{"other.md": "# Other\n"})
	runs := 0
	rapid.Check(t, func(rt *rapid.T) {
		runs++
		raw := rapid.SampledFrom(started).Draw(rt, "note")
		path := writeNote(rt, c, fmt.Sprintf("run%04d", runs), raw)
		title := rapid.SampledFrom(titles).Draw(rt, "title")

		out, err := c.rename().Execute(rt.Context(), c.vault, path, title)
		if err != nil && !errors.Is(err, note.ErrUnnameable) {
			rt.Fatalf("rename: %v", err)
		}
		if err != nil {
			return
		}

		now := c.read(t, out.Path)
		_, holds := getIdentifier(rt, c.vault.Path, out.Path)
		had := strings.Contains(raw, "id: ")

		// The filename says it where the note carries no title of its own and
		// the file came to rest under the name that was asked for. Nothing is
		// written into such a note: the bytes are the bytes that went in.
		saysIt := !strings.Contains(raw, "title:") &&
			strings.TrimSuffix(filepath.Base(out.Path), ".md") == title
		switch {
		case saysIt && now != raw:
			rt.Fatalf("a rename its filename carries wrote into the note\n was %q\n now %q",
				raw, now)
		case !saysIt && holds != true && !had:
			rt.Fatalf("a rename that wrote the title left %q unstamped, filed at %s",
				now, out.Path)
		case had && !holds:
			rt.Fatalf("a rename took the identifier out of %q", now)
		}
		if out.By == note.ByFilename && now != raw {
			rt.Fatalf("a rename by the filename wrote into the note: %q", now)
		}
	})
}
