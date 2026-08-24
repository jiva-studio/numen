package webui_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

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

// TestANoteIsMadeCarryingTheLinkThatSeatsIt. A note made in a seat of another
// says so itself: the link is in the note that was made, so one write leaves it
// joined.
func TestANoteIsMadeCarryingTheLinkThatSeatsIt(t *testing.T) {
	f := quitting(t, nil, map[string]string{
		"Ontology.md": "---\ntitle: Ontology\n---\n\n# Ontology\n",
	})

	answer, err := f.client.Create(t.Context(), connect.NewRequest(&v1.CreateRequest{
		Title: "Entropy",
		Links: []*v1.NewLink{{To: "Ontology.md", Seat: v1.Seat_SEAT_PARENT}},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the note was refused: %v", refusal)
	}
	if path := answer.Msg.GetPath(); path != "Entropy.md" {
		t.Fatalf("the note was filed at %q", path)
	}

	made := fileAt(t, f.root, "Entropy.md")
	if !strings.Contains(made, "to: Ontology") {
		t.Errorf("the note carries no link to the note it was made from:\n%s", made)
	}
	if !strings.Contains(made, "role: parent") {
		t.Errorf("the link does not seat the note it was made from as its parent:\n%s", made)
	}
	if !strings.Contains(made, "id: ") {
		t.Errorf("a note the application made carries no identifier:\n%s", made)
	}
}

// TestANoteMadeInTheChildSeatIsTheParentsChild. The seat travels as the link the
// new note writes, so the role is the one that puts each of the two where the
// person put them.
func TestANoteMadeInTheChildSeatIsTheParentsChild(t *testing.T) {
	f := quitting(t, nil, map[string]string{
		"Ontology.md": "---\ntitle: Ontology\n---\n\n# Ontology\n",
	})

	answer, err := f.client.Create(t.Context(), connect.NewRequest(&v1.CreateRequest{
		Title: "Entropy",
		Links: []*v1.NewLink{{To: "Ontology.md", Seat: v1.Seat_SEAT_CHILD, Label: "follows from"}},
	}))
	if err != nil {
		t.Fatal(err)
	}

	made := fileAt(t, f.root, answer.Msg.GetPath())
	if !strings.Contains(made, "role: child") {
		t.Errorf("the link does not seat the note it was made from as its child:\n%s", made)
	}
	if !strings.Contains(made, "label: follows from") {
		t.Errorf("what the person wrote on the link is not in the note:\n%s", made)
	}
}

// TestANoteIsMadeInTheFolderItWasAskedFor. The person's arrangement of their own
// folders is followed and never altered.
func TestANoteIsMadeInTheFolderItWasAskedFor(t *testing.T) {
	f := quitting(t, nil, map[string]string{
		"physics/Ontology.md": "---\ntitle: Ontology\n---\n\n# Ontology\n",
	})

	answer, err := f.client.Create(t.Context(), connect.NewRequest(&v1.CreateRequest{
		Title:  "Entropy",
		Folder: "physics",
		Links:  []*v1.NewLink{{To: "physics/Ontology.md", Seat: v1.Seat_SEAT_PARENT}},
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
func TestANoteMadeIsInTheIndexBeforeTheAnswerComesBack(t *testing.T) {
	f := quitting(t, nil, map[string]string{
		"Ontology.md": "---\ntitle: Ontology\n---\n\n# Ontology\n",
	})

	if _, err := f.client.Create(t.Context(), connect.NewRequest(&v1.CreateRequest{
		Title: "Entropy",
		Links: []*v1.NewLink{{To: "Ontology.md", Seat: v1.Seat_SEAT_PARENT}},
	})); err != nil {
		t.Fatal(err)
	}

	found, err := f.opened.Index.Queries().Notes(t.Context(), f.opened.Vault.ID, []string{"Entropy.md"})
	if err != nil {
		t.Fatal(err)
	}
	if found["Entropy.md"].Title != "Entropy" {
		t.Errorf("the index answers %q for a note that was just made", found["Entropy.md"].Title)
	}
}

// TestANoteMadeWhereOneAlreadyIsIsRefused. Whether the path was free is the
// filesystem's to answer at the moment the file is made, and what is there is
// left as it stands.
func TestANoteMadeWhereOneAlreadyIsIsRefused(t *testing.T) {
	const held = "---\ntitle: Ontology\n---\n\n# Ontology\n"
	f := quitting(t, nil, map[string]string{"Ontology.md": held})

	answer, err := f.client.Create(t.Context(), connect.NewRequest(&v1.CreateRequest{
		Title: "Ontology",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_OCCUPIED {
		t.Errorf("a name already taken was answered with %v", refusal)
	}
	if answer.Msg.GetPath() != "" {
		t.Errorf("a note that was not made was filed at %q", answer.Msg.GetPath())
	}
	if now := fileAt(t, f.root, "Ontology.md"); now != held {
		t.Errorf("the note that was already there was written:\n%s", now)
	}
}

// TestASeatNoLinkWritesLeavesNoNote. A sibling is another child of a shared
// parent, so nothing writes one, and the refusal comes before the file does.
func TestASeatNoLinkWritesLeavesNoNote(t *testing.T) {
	f := quitting(t, nil, map[string]string{
		"Ontology.md": "---\ntitle: Ontology\n---\n\n# Ontology\n",
	})

	_, err := f.client.Create(t.Context(), connect.NewRequest(&v1.CreateRequest{
		Title: "Entropy",
		Links: []*v1.NewLink{{To: "Ontology.md", Seat: v1.Seat_SEAT_SIBLING}},
	}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("a sibling was answered with %v", err)
	}
	if _, err := os.Stat(filepath.Join(f.root, "Entropy.md")); err == nil {
		t.Error("a note was made for a link that cannot be written")
	}
}

// TestTwoNotesAreJoinedFromTheOneTheLinkIsWrittenIn. A link is one end's
// account of a relationship: the other note is not touched.
func TestTwoNotesAreJoinedFromTheOneTheLinkIsWrittenIn(t *testing.T) {
	const other = "---\ntitle: Entropy\n---\n\n# Entropy\n"
	f := quitting(t, nil, map[string]string{
		"Ontology.md": "---\ntitle: Ontology\ntags:\n  - draft\n---\n\n# Ontology\n",
		"Entropy.md":  other,
	})

	answer, err := f.client.Join(t.Context(), connect.NewRequest(&v1.JoinRequest{
		Path: "Ontology.md",
		Link: &v1.NewLink{To: "Entropy.md", Seat: v1.Seat_SEAT_CHILD},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if refusal := answer.Msg.GetRefusal(); refusal != v1.Refusal_REFUSAL_UNSPECIFIED {
		t.Fatalf("the link was refused: %v", refusal)
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
func TestANoteIsMadeUnderTheNoteItWasMadeFromAndNotItsNamesake(t *testing.T) {
	f := quitting(t, nil, nil)
	// Both are made through the window, which is what puts them in the index:
	// the vault is scanned at startup, and a test is not started.
	for _, folder := range []string{"", "physics"} {
		if _, err := f.client.Create(t.Context(), connect.NewRequest(&v1.CreateRequest{
			Title: "Ontology", Folder: folder,
		})); err != nil {
			t.Fatal(err)
		}
	}

	answer, err := f.client.Create(t.Context(), connect.NewRequest(&v1.CreateRequest{
		Title:  "Entropy",
		Folder: "physics",
		Links:  []*v1.NewLink{{To: "physics/Ontology.md", Seat: v1.Seat_SEAT_PARENT}},
	}))
	if err != nil {
		t.Fatal(err)
	}

	if made := fileAt(t, f.root, answer.Msg.GetPath()); !strings.Contains(made, "to: physics/Ontology") {
		t.Errorf("the link was written by a name two notes answer to:\n%s", made)
	}

	around, err := f.client.Neighbourhood(t.Context(),
		connect.NewRequest(&v1.NeighbourhoodRequest{Path: "physics/Ontology.md"}))
	if err != nil {
		t.Fatal(err)
	}
	var children []string
	for _, related := range around.Msg.GetRelated() {
		if related.GetSeat() == v1.Seat_SEAT_CHILD {
			children = append(children, related.GetNote().GetPath())
		}
	}
	if len(children) != 1 || children[0] != "physics/Entropy.md" {
		t.Errorf("the note it was made from has children %v, want the note just made", children)
	}
}
