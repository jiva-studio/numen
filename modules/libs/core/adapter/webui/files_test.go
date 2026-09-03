package webui_test

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
)

// The window draws the vault as the folders and files a person filed it under,
// and moves them about. What the vault answers is what the window has to be
// able to say about either.

// drawn is what one folder of the vault holds, as the window asks for it.
func drawn(t *testing.T, f *going, at string) []*v1.Entry {
	t.Helper()
	answer, err := f.client.List(t.Context(), connect.NewRequest(&v1.ListRequest{Folder: at}))
	if err != nil {
		t.Fatal(err)
	}
	return answer.Msg.GetEntries()
}

// named is the entries by name, in the order they arrived.
func named(entries []*v1.Entry) []string {
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
	f := quitting(t, nil, map[string]string{
		"Zeta.md":            "# Zeta\n",
		"alpha.md":           "# Alpha\n",
		"Notes.txt":          "a list\n",
		"physics/Heat.md":    "# Heat\n",
		"physics/Entropy.md": "# Entropy\n",
	})
	testsupport.WriteBook(t, f.root, "library/A Book.epub")

	root := drawn(t, f, "")
	want := []string{"library", "physics", "alpha.md", "Notes.txt", "Zeta.md"}
	if got := named(root); !slices.Equal(got, want) {
		t.Fatalf("the root is drawn as %v", got)
	}
	held := map[string]*v1.Entry{}
	for _, entry := range root {
		held[entry.GetName()] = entry
	}
	if entry := held["physics"]; !entry.GetFolder() || entry.GetPath() != "physics" {
		t.Errorf("the folder came back as %+v", entry)
	}
	if kind := held["alpha.md"].GetKind(); kind != v1.SourceKind_SOURCE_KIND_NOTE {
		t.Errorf("a note is held as %v", kind)
	}
	if kind := held["Notes.txt"].GetKind(); kind != v1.SourceKind_SOURCE_KIND_UNSPECIFIED {
		t.Errorf("a file the vault holds no source for is held as %v", kind)
	}

	under := drawn(t, f, "physics")
	if got := named(under); !slices.Equal(got, []string{"Entropy.md", "Heat.md"}) {
		t.Fatalf("the folder is drawn as %v", got)
	}
	if path := under[0].GetPath(); path != "physics/Entropy.md" {
		t.Errorf("the note under a folder is called %q", path)
	}

	shelf := drawn(t, f, "library")
	if len(shelf) != 1 || shelf[0].GetKind() != v1.SourceKind_SOURCE_KIND_BOOK {
		t.Errorf("the library is drawn as %+v", shelf)
	}
}

// TestAListingLeavesOutWhatTheVaultLeavesAlone. A name beginning with a dot
// belongs to a tool, and the folder this application keeps for itself is its
// own.
func TestAListingLeavesOutWhatTheVaultLeavesAlone(t *testing.T) {
	f := quitting(t, nil, map[string]string{
		"Entropy.md":      "# Entropy\n",
		".secret.md":      "# Secret\n",
		".hidden/Kept.md": "# Kept\n",
	})
	if gone(t, f.root, ".secret.md") || !folder(t, f.root, ".hidden") ||
		!folder(t, f.root, filesystem.DefaultServiceDir) {
		t.Fatal("the vault does not hold what the listing has to leave out")
	}

	if got := named(drawn(t, f, "")); !slices.Equal(got, []string{"Entropy.md"}) {
		t.Errorf("the root is drawn as %v", got)
	}
}

// TestAFolderThatIsNotThereIsNotAnEmptyOne. A listing carries no refusal, so
// the two are told apart by the answer itself.
func TestAFolderThatIsNotThereIsNotAnEmptyOne(t *testing.T) {
	f := quitting(t, nil, map[string]string{"Entropy.md": "# Entropy\n"})

	_, err := f.client.List(t.Context(), connect.NewRequest(&v1.ListRequest{Folder: "physics"}))
	if code := connect.CodeOf(err); code != connect.CodeNotFound {
		t.Errorf("a folder that is not there was answered with %v", code)
	}
	_, err = f.client.List(t.Context(), connect.NewRequest(&v1.ListRequest{Folder: "../elsewhere"}))
	if code := connect.CodeOf(err); code != connect.CodeInvalidArgument {
		t.Errorf("a folder outside the vault was answered with %v", code)
	}
}

// TestEverythingTheVaultHoldsMoves. A file of any kind travels, whether
// anything reads it or not.
func TestEverythingTheVaultHoldsMoves(t *testing.T) {
	const picture = "\x89PNG\r\n\x1a\n"
	f := quitting(t, nil, map[string]string{"Diagram.png": picture})
	testsupport.WriteBook(t, f.root, "A Book.epub")
	scanned(t, f)

	for _, one := range []struct{ from, to string }{
		{from: "A Book.epub", to: "library/A Book.epub"},
		{from: "Diagram.png", to: "pictures/Diagram.png"},
	} {
		answer, err := f.client.Move(t.Context(), connect.NewRequest(&v1.MoveRequest{
			From: one.from, To: one.to,
		}))
		if err != nil {
			t.Fatal(err)
		}
		if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
			t.Fatalf("moving %s was refused: %v", one.from, refusal)
		}
		moved := answer.Msg.GetMoved()
		if moved.GetFrom() != one.from || moved.GetTo() != one.to {
			t.Errorf("what the file did came back as %+v", moved)
		}
		if !gone(t, f.root, one.from) {
			t.Errorf("%s is still where it was", one.from)
		}
	}
	if now := fileAt(t, f.root, "pictures/Diagram.png"); now != picture {
		t.Errorf("the picture holds %q", now)
	}
	if len(drawn(t, f, "library")) != 1 {
		t.Error("the book is not in the folder it was sent to")
	}
}

// TestAFolderOfNotesMovesAndIsStillLinkedTo. A link written by name finds its
// note wherever it is filed, so a folder moves without a link being rewritten.
func TestAFolderOfNotesMovesAndIsStillLinkedTo(t *testing.T) {
	const pointing = "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n\n# Heat\n"
	f := quitting(t, nil, map[string]string{
		"physics/Entropy.md": "# Entropy\n",
		"Heat.md":            pointing,
	})
	scanned(t, f)

	answer, err := f.client.Move(t.Context(), connect.NewRequest(&v1.MoveRequest{
		From: "physics", To: "science/physics",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the folder was refused: %v", refusal)
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

	around, err := f.client.Neighbourhood(t.Context(), connect.NewRequest(&v1.NeighbourhoodRequest{
		Path: "Heat.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	related := around.Msg.GetRelated()
	if len(related) != 1 {
		t.Fatalf("want the one note the link reaches, got %+v", related)
	}
	if path := related[0].GetNote().GetPath(); path != "science/physics/Entropy.md" {
		t.Errorf("the link reaches %q", path)
	}
}

// TestAMoveOntoATakenNameLeavesBothWhereTheyAre. Two files arriving at one path
// is a question only the person can answer.
func TestAMoveOntoATakenNameLeavesBothWhereTheyAre(t *testing.T) {
	const held = "# Entropy in physics\n"
	f := quitting(t, nil, map[string]string{
		"Entropy.md":         "# Entropy\n",
		"physics/Entropy.md": held,
	})
	scanned(t, f)

	answer, err := f.client.Move(t.Context(), connect.NewRequest(&v1.MoveRequest{
		From: "Entropy.md", To: "physics/Entropy.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_OCCUPIED {
		t.Errorf("a name already taken was answered with %v", refusal)
	}
	if moved := answer.Msg.GetMoved(); moved != nil {
		t.Errorf("a refused move said the file did %+v", moved)
	}
	if now := fileAt(t, f.root, "Entropy.md"); now != "# Entropy\n" {
		t.Errorf("the note that was to move holds:\n%s", now)
	}
	if now := fileAt(t, f.root, "physics/Entropy.md"); now != held {
		t.Errorf("the note already there was written:\n%s", now)
	}
}

// TestAMoveOfAPathWithNothingAtItLeavesNothingBehind, and says so as a
// refusal: the core is what names a path as missing.
func TestAMoveOfAPathWithNothingAtItLeavesNothingBehind(t *testing.T) {
	f := quitting(t, nil, map[string]string{"Entropy.md": "# Entropy\n"})

	answer, err := f.client.Move(t.Context(), connect.NewRequest(&v1.MoveRequest{
		From: "physics", To: "science",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_MISSING {
		t.Errorf("a path with nothing at it was answered with %v", refusal)
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
func TestARemovedFolderGoesToTheTrashWithEverythingUnderIt(t *testing.T) {
	const pointing = "---\nlinks:\n  - to: Entropy\n    role: parent\n---\n\n# Heat\n"
	f := quitting(t, nil, map[string]string{
		"physics/Entropy.md": "# Entropy\n",
		"Heat.md":            pointing,
	})
	scanned(t, f)

	answer, err := f.client.Remove(t.Context(), connect.NewRequest(&v1.RemoveRequest{
		Path: "physics",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the folder was refused: %v", refusal)
	}
	if trashed := answer.Msg.GetTrashed(); trashed != ".trash/physics" {
		t.Errorf("the folder sits at %q", trashed)
	}
	dangling := answer.Msg.GetDangling()
	if len(dangling) != 1 || dangling[0] != "Heat.md" {
		t.Errorf("what now reaches nothing came back as %v", dangling)
	}
	if !gone(t, f.root, "physics/Entropy.md") {
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
	f := quitting(t, nil, map[string]string{"Entropy.md": "# Entropy\n"})

	answer, err := f.client.MakeFolder(t.Context(), connect.NewRequest(&v1.MakeFolderRequest{
		Path: "science/physics",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the folder was refused: %v", refusal)
	}
	if !folder(t, f.root, "science/physics") {
		t.Fatal("the folder is not on the disk")
	}
	if got := named(drawn(t, f, "science")); !slices.Equal(got, []string{"physics"}) {
		t.Errorf("the folder above is drawn as %v", got)
	}
	if len(drawn(t, f, "science/physics")) != 0 {
		t.Error("the folder that was made holds something")
	}

	// Making it again is the outcome that was asked for.
	again, err := f.client.MakeFolder(t.Context(), connect.NewRequest(&v1.MakeFolderRequest{
		Path: "science/physics",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := again.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Errorf("a folder that is already there was answered with %v", refusal)
	}
}

// TestAFolderIsNotMadeWhereAFileIsFiled. Nothing is written over the file.
func TestAFolderIsNotMadeWhereAFileIsFiled(t *testing.T) {
	const held = "# Entropy\n"
	f := quitting(t, nil, map[string]string{"Entropy.md": held})

	answer, err := f.client.MakeFolder(t.Context(), connect.NewRequest(&v1.MakeFolderRequest{
		Path: "Entropy.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_OCCUPIED {
		t.Errorf("a path a file already holds was answered with %v", refusal)
	}
	if now := fileAt(t, f.root, "Entropy.md"); now != held {
		t.Errorf("the file was written:\n%s", now)
	}
}

// TestAListingSaysWhichOfFourEachNoteIs. The folder says which of its entries
// are notes and the index says what each of those notes is, so a tree draws a
// deck as a deck without opening it.
func TestAListingSaysWhichOfFourEachNoteIs(t *testing.T) {
	f := quitting(t, nil, map[string]string{
		"Entropy.md":  "# Entropy\n",
		"Animal.md":   "---\ntype: stencil\nfields:\n  - Height\n---\n\n## Recognise\n",
		"Animals.md":  "---\ntype: deck\n---\n\n## Llama\n\n[[Animal]]\n",
		"Sanskrit.md": "---\ntype: preset\nminutes_a_day: 20\n---\n\n# Sanskrit\n",
		"Notes.txt":   "a list\n",
	})
	f.read(t)

	held := map[string]*v1.Entry{}
	for _, entry := range drawn(t, f, "") {
		held[entry.GetName()] = entry
	}

	want := map[string]v1.NoteType{
		"Animals.md":  v1.NoteType_NOTE_TYPE_DECK,
		"Animal.md":   v1.NoteType_NOTE_TYPE_STENCIL,
		"Sanskrit.md": v1.NoteType_NOTE_TYPE_PRESET,
		"Entropy.md":  v1.NoteType_NOTE_TYPE_UNSPECIFIED,
		"Notes.txt":   v1.NoteType_NOTE_TYPE_UNSPECIFIED,
	}
	for name, one := range want {
		if got := held[name].GetType(); got != one {
			t.Errorf("%s is drawn as %v, want %v", name, got, one)
		}
	}
}

// stands is what the vault says it holds at each of those paths, by path.
func stands(t *testing.T, f *going, paths ...string) map[string]*v1.Standing {
	t.Helper()
	answer, err := f.client.Standing(t.Context(),
		connect.NewRequest(&v1.StandingRequest{Paths: paths}))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]*v1.Standing{}
	for _, one := range answer.Msg.GetFound() {
		out[one.GetPath()] = one
	}
	return out
}

// TestAPathSaysWhatStandsThere. A window holding a path and nothing else opens
// what stands there in the editor made for it, and this is the one question
// that says which that is.
func TestAPathSaysWhatStandsThere(t *testing.T) {
	f := quitting(t, nil, map[string]string{
		"Entropy.md":   "# Entropy\n",
		"Animal.md":    "---\ntype: stencil\nfields:\n  - Height\n---\n\n## Recognise\n",
		"Animals.md":   "---\ntype: deck\n---\n\n## Llama\n\n[[Animal]]\n",
		"Sanskrit.md":  "---\ntype: preset\nminutes_a_day: 20\n---\n\n# Sanskrit\n",
		"Physics.epub": "an epub\n",
		"Notes.txt":    "a list\n",
	})
	f.read(t)

	held := stands(t, f, "Animals.md", "Animal.md", "Sanskrit.md",
		"Entropy.md", "Physics.epub", "Notes.txt", "Gone.md")

	want := map[string]struct {
		kind v1.SourceKind
		is   v1.NoteType
	}{
		"Animals.md":   {v1.SourceKind_SOURCE_KIND_NOTE, v1.NoteType_NOTE_TYPE_DECK},
		"Animal.md":    {v1.SourceKind_SOURCE_KIND_NOTE, v1.NoteType_NOTE_TYPE_STENCIL},
		"Sanskrit.md":  {v1.SourceKind_SOURCE_KIND_NOTE, v1.NoteType_NOTE_TYPE_PRESET},
		"Entropy.md":   {v1.SourceKind_SOURCE_KIND_NOTE, v1.NoteType_NOTE_TYPE_UNSPECIFIED},
		"Physics.epub": {v1.SourceKind_SOURCE_KIND_BOOK, v1.NoteType_NOTE_TYPE_UNSPECIFIED},
		"Notes.txt":    {v1.SourceKind_SOURCE_KIND_UNSPECIFIED, v1.NoteType_NOTE_TYPE_UNSPECIFIED},
	}
	for path, one := range want {
		stood, answered := held[path]
		if !answered {
			t.Errorf("%s was not answered about at all", path)
			continue
		}
		if stood.GetKind() != one.kind || stood.GetType() != one.is {
			t.Errorf("%s stands as %v %v, want %v %v",
				path, stood.GetKind(), stood.GetType(), one.kind, one.is)
		}
	}

	// A path with nothing at it is not a file the vault leaves alone, and the
	// two are told apart by one being answered about and the other not.
	if _, answered := held["Gone.md"]; answered {
		t.Error("a path with nothing at it was answered about anyway")
	}
}

// TestABookStandsThereBeforeAnythingHasReadIt. The index holds no note for a
// book and never will, so an answer drawn from it alone leaves every book out
// and a window reads that absence as an ordinary note.
func TestABookStandsThereBeforeAnythingHasReadIt(t *testing.T) {
	f := quitting(t, nil, map[string]string{"Entropy.md": "# Entropy\n"})

	// Written after the vault was opened and asked about at once, so nothing
	// has had the chance to read it.
	if err := os.WriteFile(filepath.Join(f.root, "Physics.epub"), []byte("an epub\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	stood := stands(t, f, "Physics.epub")["Physics.epub"]
	if stood.GetKind() != v1.SourceKind_SOURCE_KIND_BOOK {
		t.Errorf("a book nothing has read stands as %v", stood.GetKind())
	}
}
