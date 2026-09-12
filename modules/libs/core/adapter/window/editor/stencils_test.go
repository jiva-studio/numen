package editor_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// What the window puts into a stencil, and what a client asking for a file the
// vault does not hold is told.

// stencil is what the window reads a stencil as.
func stencil(t *testing.T, f *cutting, path string) *v1.ReadStencilResponse {
	t.Helper()
	answer, err := f.client.ReadStencil(t.Context(), connect.NewRequest(&v1.ReadStencilRequest{
		Path: path,
	}))
	if err != nil {
		t.Fatal(err)
	}
	return answer.Msg
}

// TestAStencilIsWrittenWithItsFacesAndItsFields. The faces are the body and the
// fields are one key of the frontmatter, and both are the one file, so what
// comes back is the file holding both.
func TestAStencilIsWrittenWithItsFacesAndItsFields(t *testing.T) {
	f := dealing(t, map[string]string{"cards/Animal.md": animal})

	read := stencil(t, f, "cards/Animal.md")
	faces := read.GetStencil().GetFaces()
	faces[0].Back = "{{Height}} at the shoulder"

	answer, err := f.client.WriteStencil(t.Context(), connect.NewRequest(&v1.WriteStencilRequest{
		Path:     "cards/Animal.md",
		Fields:   read.GetStencil().GetFields(),
		Faces:    faces,
		Preamble: read.GetStencil().GetPreamble(),
		Tail:     read.GetStencil().GetTail(),
		Seen:     read.GetAt(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetError() != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("writing a stencil answered %+v", answer.Msg)
	}

	held := onDisk(t, f.root, "cards/Animal.md")
	if !strings.Contains(held, "{{Height}} at the shoulder") {
		t.Errorf("the face is not the one written: %q", held)
	}
	if !strings.Contains(held, "fields:\n  - Name\n  - Height\n") {
		t.Errorf("the stencil declares something else: %q", held)
	}
	// The fingerprint that comes back is the file on disk, so the write that
	// follows this one lands.
	if answer.Msg.GetAt().GetSize() != int64(len(held)) {
		t.Errorf("the fingerprint is not the file: %+v", answer.Msg.GetAt())
	}
}

// TestAStencilIsWrittenWithTheFieldsItNowDeclares. A person adding a field
// writes the stencil's frontmatter and its faces in the one act.
func TestAStencilIsWrittenWithTheFieldsItNowDeclares(t *testing.T) {
	f := dealing(t, map[string]string{"cards/Animal.md": animal})

	read := stencil(t, f, "cards/Animal.md")
	answer, err := f.client.WriteStencil(t.Context(), connect.NewRequest(&v1.WriteStencilRequest{
		Path:     "cards/Animal.md",
		Fields:   []string{"Name", "Height", "Life span"},
		Faces:    read.GetStencil().GetFaces(),
		Preamble: read.GetStencil().GetPreamble(),
		Tail:     read.GetStencil().GetTail(),
		Seen:     read.GetAt(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetError() != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("writing a stencil answered %+v", answer.Msg)
	}

	after := stencil(t, f, "cards/Animal.md")
	if fields := after.GetStencil().GetFields(); len(fields) != 3 || fields[2] != "Life span" {
		t.Errorf("the stencil asks for %v", fields)
	}
	if answer.Msg.GetAt().GetSize() != int64(len(onDisk(t, f.root, "cards/Animal.md"))) {
		t.Errorf("the fingerprint is not the file: %+v", answer.Msg.GetAt())
	}
}

// TestWritingAStencilThatChangedSinceItWasReadWritesNothing. Changed says
// nothing was written, so the faces are not on disk either and what the person
// wrote is the file.
func TestWritingAStencilThatChangedSinceItWasReadWritesNothing(t *testing.T) {
	f := dealing(t, map[string]string{"cards/Animal.md": animal})

	read := stencil(t, f, "cards/Animal.md")
	// The person writes their own stencil while the client is thinking about
	// what it read.
	theirs := animal + "\n## Name it\n\n### Front\n\n{{Height}}\n\n### Back\n\n{{Name}}\n"
	if err := os.WriteFile(
		filepath.Join(f.root, "cards", "Animal.md"), []byte(theirs), 0o644,
	); err != nil {
		t.Fatal(err)
	}

	faces := read.GetStencil().GetFaces()
	faces[0].Front = "{{Species}}"
	answer, err := f.client.WriteStencil(t.Context(), connect.NewRequest(&v1.WriteStencilRequest{
		Path:     "cards/Animal.md",
		Fields:   []string{"Species", "Height"},
		Faces:    faces,
		Preamble: read.GetStencil().GetPreamble(),
		Tail:     read.GetStencil().GetTail(),
		Seen:     read.GetAt(),
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetError() != v1.ErrorCode_ERROR_CODE_STALE {
		t.Fatalf("a write over a stencil the person had edited answered %+v", answer.Msg)
	}
	if held := onDisk(t, f.root, "cards/Animal.md"); held != theirs {
		t.Errorf("the stencil on disk is now %q", held)
	}
}

// TestWritingAStencilWhereTheVaultHoldsNoNoteIsRefused. A write puts fields and
// faces into a stencil that is there, and a stencil is put in the vault by name.
func TestWritingAStencilWhereTheVaultHoldsNoNoteIsRefused(t *testing.T) {
	f := dealing(t, map[string]string{"cards/Animal.md": animal})

	answer, err := f.client.WriteStencil(t.Context(), connect.NewRequest(&v1.WriteStencilRequest{
		Path:   "cards/Nowhere.md",
		Fields: []string{"Name"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_MISSING {
		t.Errorf("writing a stencil where the vault holds no note answered %v", code)
	}
	if _, err := os.Stat(filepath.Join(f.root, "cards", "Nowhere.md")); err == nil {
		t.Error("a file was made where the write was refused")
	}
}

// TestAStencilMadeOnANameAlreadyTakenIsRefused. Nothing is written over: the
// person is told the name is taken and picks another.
func TestAStencilMadeOnANameAlreadyTakenIsRefused(t *testing.T) {
	f := dealing(t, map[string]string{"cards/Animal.md": animal})

	answer, err := f.client.CreateStencil(t.Context(), connect.NewRequest(&v1.CreateStencilRequest{
		Title: "Animal", Path: "cards", Fields: []string{"Species"},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_OCCUPIED {
		t.Errorf("making a stencil on a name already taken answered %v", code)
	}
	if held := onDisk(t, f.root, "cards/Animal.md"); held != animal {
		t.Errorf("the stencil that was there is now %q", held)
	}
}

// TestADeckMadeOnANameAlreadyTakenIsRefused. What holds for a stencil holds for
// a deck.
func TestADeckMadeOnANameAlreadyTakenIsRefused(t *testing.T) {
	f := dealing(t, map[string]string{"cards/Animal.md": animal})

	answer, err := f.client.CreateDeck(t.Context(), connect.NewRequest(&v1.CreateDeckRequest{
		Title: "Animal", Path: "cards",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_OCCUPIED {
		t.Errorf("making a deck on a name already taken answered %v", code)
	}
	if held := onDisk(t, f.root, "cards/Animal.md"); held != animal {
		t.Errorf("the file that was there is now %q", held)
	}
}

// TestAPathThatLeavesTheVaultIsTheClientsToCorrect. Nothing escapes the vault
// either way; what is being said is whose mistake it was, and a path the client
// built is the client's.
func TestAPathThatLeavesTheVaultIsTheClientsToCorrect(t *testing.T) {
	f := dealing(t, map[string]string{"cards/Animal.md": animal})

	_, err := f.client.ReadDeck(t.Context(), connect.NewRequest(&v1.ReadDeckRequest{
		Path: "../../etc/passwd",
	}))
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Errorf("reading a deck outside the vault answered %v (%v)", got, err)
	}

	_, err = f.client.ReadStencil(t.Context(), connect.NewRequest(&v1.ReadStencilRequest{
		Path: "../../etc/passwd",
	}))
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Errorf("reading a stencil outside the vault answered %v (%v)", got, err)
	}

	_, err = f.client.CreateDeck(t.Context(), connect.NewRequest(&v1.CreateDeckRequest{
		Title: "Elsewhere", Path: "../elsewhere",
	}))
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Errorf("making a deck outside the vault answered %v (%v)", got, err)
	}
}

// TestARenameOntoANameTheStencilDeclaresIsTheClientsToCorrect. A field a
// stencil already declares is a name the client is holding wrongly, and it is
// told so.
func TestARenameOntoANameTheStencilDeclaresIsTheClientsToCorrect(t *testing.T) {
	f := dealing(t, map[string]string{"cards/Animal.md": animal})

	_, err := f.client.RenameStencilField(t.Context(),
		connect.NewRequest(&v1.RenameStencilFieldRequest{
			Path: "cards/Animal.md", From: "Height", To: "Name",
		}))
	if got := connect.CodeOf(err); got != connect.CodeInvalidArgument {
		t.Errorf("renaming a field onto a declared name answered %v (%v)", got, err)
	}
	if held := onDisk(t, f.root, "cards/Animal.md"); held != animal {
		t.Errorf("the stencil is now %q", held)
	}
}

// TestAFolderThatIsAFileIsAnAnswerAPersonCanActOn. A file stands where the
// folder would be, so nothing goes in it and the person is told the name is
// taken.
func TestAFolderThatIsAFileIsAnAnswerAPersonCanActOn(t *testing.T) {
	f := dealing(t, map[string]string{"Entropy.md": "# Entropy\n"})

	answer, err := f.client.CreateDeck(t.Context(), connect.NewRequest(&v1.CreateDeckRequest{
		Title: "Camelids", Path: "Entropy.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if code := answer.Msg.GetError(); code != v1.ErrorCode_ERROR_CODE_OCCUPIED {
		t.Errorf("making a deck under a file answered %v", code)
	}
	if held := onDisk(t, f.root, "Entropy.md"); held != "# Entropy\n" {
		t.Errorf("the file in the way is now %q", held)
	}
}
