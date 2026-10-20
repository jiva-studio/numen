package editor_test

import (
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
)

// A file that landed and an index that would not follow has to come back with its path, or the
// person is left with a note they cannot see, cannot open and cannot make again.

// TestANoteMadeWhenTheIndexWouldNotComeLevelIsAnswered. A note is made through
// its own use case, and it says the same thing a stencil does.
func TestANoteMadeWhenTheIndexWouldNotComeLevelIsAnswered(t *testing.T) {
	f := openWindow(t, nil, nil)
	waitForScan(t, f)
	*f.installation.Services.NoteCreate() = f.installation.Services.NoteCreate().WithIndex(refuseToLevel)

	answer, err := f.client.CreateNote(t.Context(), connect.NewRequest(&v1.CreateNoteRequest{
		Title: "Entropy",
	}))
	if err != nil {
		t.Fatalf("the note is on disk and the client was told %v", err)
	}
	if path := answer.Msg.GetPath(); path != "Entropy.md" {
		t.Fatalf("the note is filed at %q", path)
	}
	if !answer.Msg.GetIsUnlevelled() {
		t.Error("the note is answered as findable and search does not hold it")
	}
}

// TestAFileRemovedWhenTheIndexWouldNotComeLevelIsAnswered. The file is in the
// trash and search still answers about it, and what its links now reach is
// reported all the same.
func TestAFileRemovedWhenTheIndexWouldNotComeLevelIsAnswered(t *testing.T) {
	f := openWindow(t, nil, map[string]string{
		"Ontology.md": "---\ntitle: Ontology\n---\n\n# Ontology\n",
		"Entropy.md": "---\ntitle: Entropy\nlinks:\n  - to: Ontology\n    role: parent\n---\n\n" +
			"# Entropy\n",
	})
	waitForScan(t, f)
	*f.installation.Services.FileRemove() = f.installation.Services.FileRemove().WithIndex(refuseToLevel)

	answer, err := f.client.RemoveFile(t.Context(), connect.NewRequest(&v1.RemoveFileRequest{
		Path: "Ontology.md",
	}))
	if err != nil {
		t.Fatalf("the note is in the trash and the client was told %v", err)
	}
	if answer.Msg.GetTrashed() == "" {
		t.Error("the note went to the trash and the client was told nothing of where")
	}
	if !answer.Msg.GetIsUnlevelled() {
		t.Error("the removal is answered as followed and search still holds the note")
	}
	if len(answer.Msg.GetDangling()) != 1 {
		t.Errorf("the links now reaching nothing are %q", answer.Msg.GetDangling())
	}
}
