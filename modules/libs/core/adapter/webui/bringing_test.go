package webui_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
)

// TestAFileLetGoOfOverTheTreeArrivesAndIsSaid is the path a drop takes: the
// window hands over what the person let go of, the vault holds it, and everyone
// drawing the vault is told where to look.
//
// A picture is what the watcher reports to nobody, so the stream is the only
// way the tree finds out about one.
func TestAFileLetGoOfOverTheTreeArrivesAndIsSaid(t *testing.T) {
	client, _, root, opened := serving(t, map[string]string{
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

	changes, err := client.Changes(listening, connect.NewRequest(&v1.ChangesRequest{}))
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

	opened.Brings(t.Context(), "physics", []string{cover})

	held, err := os.ReadFile(filepath.Join(root, "physics", "Cover.png"))
	if err != nil {
		t.Fatalf("the file did not arrive: %v", err)
	}
	if string(held) != "PNG" {
		t.Errorf("the file arrived as %q", held)
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
	_, drawn, root, opened := serving(t, map[string]string{"Entropy.md": "# Mine\n"})

	outside := t.TempDir()
	theirs := filepath.Join(outside, "Entropy.md")
	if err := os.WriteFile(theirs, []byte("# Theirs\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	opened.Brings(t.Context(), "", []string{theirs})

	held, err := os.ReadFile(filepath.Join(root, "Entropy.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(held) != "# Mine\n" {
		t.Errorf("the note that was there is now %q", held)
	}

	answer, err := drawn.Tasks(t.Context(), connect.NewRequest(&v1.TasksRequest{Window: wire.Editor}))
	if err != nil {
		t.Fatal(err)
	}
	defer answer.Close()
	if !answer.Receive() {
		t.Fatalf("nothing said what the window is doing: %v", answer.Err())
	}

	var said string
	for _, at := range answer.Msg().GetTasks() {
		if at.GetFailed() != "" {
			said = at.GetFailed()
		}
	}
	if said == "" {
		t.Fatal("the refusal was never said")
	}
	if !strings.Contains(said, "Entropy.md") {
		t.Errorf("the refusal says %q", said)
	}
}
