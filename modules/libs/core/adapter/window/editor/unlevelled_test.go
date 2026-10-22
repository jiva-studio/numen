package editor_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// Two processes write the index, so a write that reached the vault may find the
// the index is behind: the file is on disk and search does not hold what it now
// says.

// refuseToLevel is an index that will not come level with what was written.
func refuseToLevel(context.Context, domain.Vault, []string) error {
	return errors.New("the index is held open elsewhere")
}

// unwritten is a vault with those notes in it and an index that will not come level with anything
// written to it, asked the way the editor's window does.
func unwritten(t *testing.T, notes map[string]string) *going {
	t.Helper()

	f := openWindow(t, nil, notes)
	waitForScan(t, f)
	*f.installation.Services.NoteWrite() = f.installation.Services.NoteWrite().WithIndex(refuseToLevel)
	return f
}

func TestASaveTheIndexWouldNotComeLevelWithIsAnswered(t *testing.T) {
	f := unwritten(t, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})

	answer, err := f.client.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
		Path: "Note.md",
		Body: "what the person typed\n",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetError() != v1.ErrorCode_ERROR_CODE_UNSPECIFIED {
		t.Fatalf("saving answered %+v", answer.Msg)
	}
	// Without it the next save has nothing to present and is answered as a note
	// somebody else touched.
	if answer.Msg.GetAt() == nil {
		t.Error("the save landed and the client was handed no fingerprint")
	}
	if !answer.Msg.GetIsUnlevelled() {
		t.Error("the save is answered as findable and search does not hold it")
	}
	if target := onDisk(t, f.root, "Note.md"); !strings.Contains(target, "what the person typed") {
		t.Errorf("the note on disk is %q", target)
	}
}

// TestASaveTheIndexCameLevelWithSaysNothingIsBehind is the control. A flag
// every save carries says nothing, and a client shown a warning after every
// save learns to ignore it.
func TestASaveTheIndexCameLevelWithReturnsNothingIsPrior(t *testing.T) {
	f := openWindow(t, nil, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})
	waitForScan(t, f)

	answer, err := f.client.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
		Path: "Note.md",
		Body: "what the person typed\n",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if answer.Msg.GetIsUnlevelled() {
		t.Error("a save the index came level with is answered as behind")
	}
	if pos := findLevellingTask(t, f); pos != nil {
		t.Errorf("a save that went through is in the list as %+v", pos)
	}
}

// TestASaveTheIndexWouldNotComeLevelWithStandsInTheList. A person saving a
// second note has no reason to look at what the first one answered, and the
// list of what is being done is where they are.
func TestASaveTheIndexWouldNotComeLevelWithExistsInTheList(t *testing.T) {
	f := unwritten(t, map[string]string{"Note.md": "---\ntitle: Note\n---\n\n# Note\n"})

	save := func() {
		t.Helper()
		if _, err := f.client.WriteNote(t.Context(), connect.NewRequest(&v1.WriteNoteRequest{
			Path: "Note.md",
			Body: "what the person typed\n",
		})); err != nil {
			t.Fatal(err)
		}
	}

	save()
	pos := findLevellingTask(t, f)
	if pos == nil || pos.GetError() == "" {
		t.Fatalf("what a person is shown after a save search cannot see is %+v", pos)
	}

	// The index is behind until something puts it right, and the save that does
	// is what says so. A list nothing leaves is a list nobody reads.
	*f.installation.Services.NoteWrite() = f.installation.Services.NoteWrite().WithIndex(func(context.Context, domain.Vault, []string) error { return nil })
	save()
	if offset := findLevellingTask(t, f); offset != nil {
		t.Errorf("the index came level and a person is still shown %+v", offset)
	}
}

// findLevellingTask is what the list of what is being done says about the index, asked
// for the way the window asks for it. The first message of the stream is the
// list currently, so nothing here waits on anything.
func findLevellingTask(t *testing.T, f *going) *v1.Task {
	t.Helper()

	listening, cancel := context.WithCancel(t.Context())
	defer cancel()
	tasks, err := f.canvas.WatchTasks(listening, connect.NewRequest(&v1.WatchTasksRequest{
		Window: wire.Editor,
	}))
	if err != nil {
		t.Fatal(err)
	}
	defer tasks.Close()
	if !tasks.Receive() {
		t.Fatalf("the tasks stream never opened: %v", tasks.Err())
	}
	for _, item := range tasks.Msg().GetTasks() {
		if item.GetId() == "levelling the index" {
			return item
		}
	}
	return nil
}
