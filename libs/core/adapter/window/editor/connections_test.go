package editor_test

import (
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

const (
	windowHeat    = "---\ntitle: Heat\n---\n\n# Heat\n"
	windowEntropy = "---\ntitle: Entropy\n---\n\n# Entropy\n"
)

// TestAConnectionIsSavedWithAnArrowAndNoLabelAndReadBack. A side that carries a
// marker and no label is a record, and both notes hold their own.
func TestAConnectionIsSavedWithAnArrowAndNoLabelAndReadBack(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Heat.md": windowHeat, "Entropy.md": windowEntropy})
	waitForScan(t, f)

	saved, err := f.client.SaveConnection(t.Context(), connect.NewRequest(&v1.SaveConnectionRequest{
		Connection: &v1.Connection{
			First: "Heat.md", Second: "Entropy.md",
			FirstSide:  &v1.ConnectionSide{Role: v1.Role_ROLE_JUMP, Marker: v1.Marker_MARKER_ARROW},
			SecondSide: &v1.ConnectionSide{Role: v1.Role_ROLE_JUMP, Label: "back"},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if saved.Msg.Error != nil || saved.Msg.GetIsInvalid() || saved.Msg.GetIsCycle() {
		t.Fatalf("the connection was refused: %v", saved.Msg)
	}

	if heat := fileAt(t, f.root, "Heat.md"); !strings.Contains(heat, "marker: arrow") || strings.Contains(heat, "label:") {
		t.Errorf("Heat holds:\n%s", heat)
	}
	if entropy := fileAt(t, f.root, "Entropy.md"); !strings.Contains(entropy, "label: back") || strings.Contains(entropy, "marker") {
		t.Errorf("Entropy holds:\n%s", entropy)
	}
	read, err := f.client.GetConnection(t.Context(), connect.NewRequest(&v1.GetConnectionRequest{
		First: "Heat.md", Second: "Entropy.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	got := read.Msg.GetConnection()
	if got.GetFirstSide().GetMarker() != v1.Marker_MARKER_ARROW || got.GetSecondSide().GetLabel() != "back" {
		t.Errorf("the connection read back as %v", got)
	}

	neighbourhood, err := f.client.GetNeighbourhood(t.Context(), connect.NewRequest(&v1.GetNeighbourhoodRequest{
		Path: "Heat.md",
	}))
	if err != nil || len(neighbourhood.Msg.GetRelated()) != 1 {
		t.Fatalf("neighbourhood %v (%v)", neighbourhood.Msg, err)
	}
	related := neighbourhood.Msg.GetRelated()[0]
	if related.GetMarker() != v1.Marker_MARKER_ARROW || related.GetReverseMarker() != v1.Marker_MARKER_UNSPECIFIED {
		t.Errorf("the neighbour carries marker %v and reverse %v", related.GetMarker(), related.GetReverseMarker())
	}
}

// TestAConnectionThatClosesTheHierarchyIsAnsweredAsACycle. The answer is a
// successful one with the flag set, and nothing is written.
func TestAConnectionThatClosesTheHierarchyIsAnsweredAsACycle(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"A.md": "---\ntitle: A\n---\n\n# A\n",
		"B.md": "---\ntitle: B\nlinks:\n  - to: A\n    role: parent\n---\n\n# B\n",
		"C.md": "---\ntitle: C\nlinks:\n  - to: B\n    role: parent\n---\n\n# C\n",
	})
	waitForScan(t, f)

	saved, err := f.client.SaveConnection(t.Context(), connect.NewRequest(&v1.SaveConnectionRequest{
		Connection: &v1.Connection{
			First: "A.md", Second: "C.md", FirstSide: &v1.ConnectionSide{Role: v1.Role_ROLE_PARENT},
		},
	}))

	if err != nil {
		t.Fatalf("a cycle came back as a failed call: %v", err)
	}
	if !saved.Msg.GetIsCycle() {
		t.Errorf("answer = %v, want a cycle", saved.Msg)
	}
	if a := fileAt(t, f.root, "A.md"); a != "---\ntitle: A\n---\n\n# A\n" {
		t.Errorf("A was written:\n%s", a)
	}
}

func TestAnUnrelatedRoleIsAnsweredAsInvalid(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Heat.md": windowHeat, "Entropy.md": windowEntropy})
	waitForScan(t, f)

	saved, err := f.client.SaveConnection(t.Context(), connect.NewRequest(&v1.SaveConnectionRequest{
		Connection: &v1.Connection{
			First: "Heat.md", Second: "Entropy.md", FirstSide: &v1.ConnectionSide{Role: v1.Role_ROLE_REF},
		},
	}))

	if err != nil || !saved.Msg.GetIsInvalid() {
		t.Errorf("answer = %v (%v), want an invalid connection", saved.Msg, err)
	}
}

func TestANoteIsCreatedInARelationInTheFolderOfTheNoteItIsRelatedTo(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"area/Heat.md": windowHeat})
	waitForScan(t, f)

	made, err := f.client.CreateRelatedNote(t.Context(), connect.NewRequest(&v1.CreateRelatedNoteRequest{
		From: "area/Heat.md", Role: v1.Role_ROLE_PARENT,
	}))
	if err != nil {
		t.Fatal(err)
	}

	if made.Msg.GetPath() != "area/Untitled note.md" {
		t.Fatalf("answer = %v", made.Msg)
	}
	if body := fileAt(t, f.root, made.Msg.GetPath()); !strings.Contains(body, "role: child") || !strings.Contains(body, "to: Heat") {
		t.Errorf("the new note holds:\n%s", body)
	}
}

func TestBacklinksAreListedWithTheirRoles(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"Area.md": "---\ntitle: Area\n---\n\n# Area\n",
		"Idea.md": "---\ntitle: Idea\nlinks:\n  - to: Area\n    role: parent\n    marker: arrow\n---\n\n# Idea\n",
	})
	waitForScan(t, f)

	listed, err := f.client.ListBacklinks(t.Context(), connect.NewRequest(&v1.ListBacklinksRequest{Path: "Area.md"}))
	if err != nil || len(listed.Msg.GetBacklinks()) != 1 {
		t.Fatalf("answer = %v (%v)", listed.Msg, err)
	}
	b := listed.Msg.GetBacklinks()[0]
	if b.GetFrom().GetPath() != "Idea.md" || b.GetRole() != v1.Role_ROLE_PARENT ||
		b.GetRoleFromHere() != v1.Role_ROLE_CHILD || b.GetMarker() != v1.Marker_MARKER_ARROW {
		t.Errorf("backlink = %v", b)
	}
}
