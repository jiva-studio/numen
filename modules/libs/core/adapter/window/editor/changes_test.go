package editor_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1/numenv1connect"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/window/editor"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// openVault is a vault with a window's worth of machinery behind it — the index,
// the first scan, the watcher — and a client talking to it the way the window
// does.
//
// What is asked here is the wire: that a change reaches a client over the
// stream, in the shape the schema describes. What a change means is asked of
// the use case, where no server is needed to ask it.
func openVault(t *testing.T, notes map[string]string) (questions, string) {
	t.Helper()
	client, _, root, _ := openVaultWithWindow(t, notes)
	return client, root
}

// openVaultWithWindow is that same vault, with the window's half of it as well, for a test
// asking what something the window does reaches the client as.
func openVaultWithWindow(t *testing.T, notes map[string]string) (
	questions, numenv1connect.WindowServiceClient, string, *editor.Installation,
) {
	t.Helper()
	root := t.TempDir()
	for name, body := range notes {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := filesystem.Initialize(root, filesystem.DefaultServiceDir, time.Now()); err != nil {
		t.Fatal(err)
	}

	settings := container.Config{
		IndexPath:    filepath.Join(t.TempDir(), "index.db"),
		RegistryPath: filepath.Join(t.TempDir(), "vaults.json"),
	}
	registry, err := settings.Registry()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := (vaults.Add{
		Registry: registry,
		Identity: settings.VaultIdentity(),
		Now:      time.Now,
	}).Execute(root, "watched"); err != nil {
		t.Fatal(err)
	}

	opened, err := editor.Open(t.Context(), editor.NewAssembly(t, settings), "", os.Stderr)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { opened.Close() })

	drawn, itself := numenv1connect.NewWindowServiceHandler(opened.API.Window)
	mux := http.NewServeMux()
	answers(mux, opened.API)
	mux.Handle(drawn, itself)
	server := httptest.NewUnstartedServer(mux)
	server.EnableHTTP2 = true
	server.StartTLS()
	t.Cleanup(server.CloseClientConnections)
	t.Cleanup(server.Close)

	client := asks(server.Client(), server.URL)
	watching := numenv1connect.NewWindowServiceClient(server.Client(), server.URL)

	// The window opens on what the first scan stored; the watcher reports only
	// what happens after it.
	for range 200 {
		state, err := client.GetVaultState(t.Context(), connect.NewRequest(&v1.GetVaultStateRequest{}))
		if err != nil {
			t.Fatal(err)
		}
		if state.Msg.GetScan().GetIsReady() {
			return client, watching, root, opened
		}
		if reason := state.Msg.GetScan().GetError(); reason != "" {
			t.Fatalf("the first scan failed: %s", reason)
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("the first scan did not finish")
	return questions{}, nil, "", nil
}

// TestAnEditReachesAListener is the whole path: a file on disk, the watcher,
// the index, and the stream a window listens to.
func TestAnEditReachesAListener(t *testing.T) {
	client, root := openVault(t, map[string]string{
		"Note.md":  "---\ntitle: Note\n---\n\n# Note\n",
		"Other.md": "---\ntitle: Other\n---\n\n# Other\n",
	})

	// Its own context, closed before the server is: a stream is an open request,
	// and a test server waits for those.
	listening, hangUp := context.WithCancel(t.Context())
	defer hangUp()

	changes, err := client.WatchVaultChanges(listening, connect.NewRequest(&v1.WatchVaultChangesRequest{}))
	if err != nil {
		t.Fatal(err)
	}
	defer changes.Close()

	// The first message says the stream is open and nothing has changed yet.
	if !changes.Receive() {
		t.Fatalf("the stream never opened: %v", changes.Err())
	}
	if paths := changes.Msg().GetPaths(); len(paths) != 0 {
		t.Fatalf("the stream opened by reporting %v", paths)
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

	if err := os.WriteFile(filepath.Join(root, "Note.md"),
		[]byte("---\ntitle: Renamed\n---\n\n# Renamed\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	select {
	case paths := <-reported:
		if !slices.Equal(paths, []string{"Note.md"}) {
			t.Errorf("reported %v", paths)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the edit never reached the listener")
	}

	// And the index is level with the disk by the time it is announced.
	shown, err := client.GetNeighbourhood(t.Context(),
		connect.NewRequest(&v1.GetNeighbourhoodRequest{Path: "Note.md"}))
	if err != nil {
		t.Fatal(err)
	}
	if title := shown.Msg.GetFocus().GetTitle(); title != "Renamed" {
		t.Errorf("the index still says %q", title)
	}
}

// TestABookDroppedInReachesAListener. A client draws every file the vault
// holds, and a book is one of them.
func TestABookDroppedInReachesAListener(t *testing.T) {
	client, root := openVault(t, map[string]string{
		"Note.md": "---\ntitle: Note\n---\n\n# Note\n",
	})

	listening, hangUp := context.WithCancel(t.Context())
	defer hangUp()

	changes, err := client.WatchVaultChanges(listening, connect.NewRequest(&v1.WatchVaultChangesRequest{}))
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

	testsupport.WriteBook(t, root, "library/A Book.epub")

	select {
	case paths := <-reported:
		if !slices.Contains(paths, "library/A Book.epub") {
			t.Errorf("reported %v", paths)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the book never reached the listener")
	}
}
