package editor_test

import (
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

func TestAConnectionRoleFillsTheRecordsThatCarryNone(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Heat.md": windowHeat, "Entropy.md": windowEntropy})
	waitForScan(t, f)

	saved, err := f.client.SaveConnection(t.Context(), connect.NewRequest(&v1.SaveConnectionRequest{
		Connection: &v1.Connection{
			First: "Heat.md", Second: "Entropy.md", Role: v1.Role_ROLE_PARENT,
			FirstSide:  &v1.ConnectionSide{Marker: v1.Marker_MARKER_ARROW},
			SecondSide: &v1.ConnectionSide{Label: "measures"},
		},
	}))
	if err != nil || saved.Msg.Error != nil || saved.Msg.GetIsInvalid() {
		t.Fatalf("answer = %v (%v)", saved.Msg, err)
	}

	if heat := fileAt(t, f.root, "Heat.md"); !strings.Contains(heat, "role: parent") {
		t.Errorf("Heat holds:\n%s", heat)
	}
	if entropy := fileAt(t, f.root, "Entropy.md"); !strings.Contains(entropy, "role: child") {
		t.Errorf("Entropy holds:\n%s", entropy)
	}
	read, err := f.client.GetConnection(t.Context(), connect.NewRequest(&v1.GetConnectionRequest{
		First: "Entropy.md", Second: "Heat.md",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if read.Msg.GetConnection().GetRole() != v1.Role_ROLE_CHILD {
		t.Errorf("role = %v, want child", read.Msg.GetConnection().GetRole())
	}
}

func TestANoteWithoutATitleIsNumberedFromTheLowestFreeUntitledName(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Untitled note.md": windowHeat})
	waitForScan(t, f)

	made, err := f.client.CreateNote(t.Context(), connect.NewRequest(&v1.CreateNoteRequest{}))
	if err != nil {
		t.Fatal(err)
	}

	if made.Msg.GetPath() != "Untitled note 2.md" {
		t.Errorf("answer = %v", made.Msg)
	}
}

func TestANoteCreatedInARelationWritesTheMarkerItIsGiven(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Heat.md": windowHeat})
	waitForScan(t, f)

	made, err := f.client.CreateRelatedNote(t.Context(), connect.NewRequest(&v1.CreateRelatedNoteRequest{
		From: "Heat.md", Role: v1.Role_ROLE_JUMP, Marker: v1.Marker_MARKER_ARROW,
	}))
	if err != nil {
		t.Fatal(err)
	}

	if body := fileAt(t, f.root, made.Msg.GetPath()); !strings.Contains(body, "marker: arrow") {
		t.Errorf("the new note holds:\n%s", body)
	}
}
