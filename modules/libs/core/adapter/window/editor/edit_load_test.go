package editor

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// TestEditLoad is what a person waits for between a save and the vault being
// level again, at the size the product is designed for.
//
// It measures the path the application owns: the write, the watcher noticing
// it, the refresh, and the change reaching a client. What a webview does either
// side of that — a keystroke becoming an RPC, a picture being painted — is a
// browser's, and is not here.
//
//	NUMEN_LOAD=1 go test ./adapter/window/editor/ -run TestEditLoad -v -timeout 20m
//	NUMEN_LOAD=1 NUMEN_LOAD_NOTES=10000 go test ...   # a smaller rehearsal
//
// It reports what it measured and asserts nothing.
func TestEditLoad(t *testing.T) {
	if os.Getenv("NUMEN_LOAD") == "" {
		t.Skip("set NUMEN_LOAD=1 to run the load test")
	}
	notes := 100_000
	if s := os.Getenv("NUMEN_LOAD_NOTES"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil {
			t.Fatalf("NUMEN_LOAD_NOTES: %v", err)
		}
		notes = n
	}

	v := testsupport.GenerateVault(t, notes)
	cfg := container.Config{IndexPath: filepath.Join(t.TempDir(), "index.db")}
	db, err := cfg.OpenIndex(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })

	writing := note.NewWrite(
		filesystem.VaultReaders{}, filesystem.VaultWriters{}, levelNothing, time.Now)
	read := note.NewRead(filesystem.VaultReaders{})
	api := createServices(&State{
		ChangeEvents: newChangeBroadcaster(),
		Places:       newPlaceBroadcaster(),
	})
	bindNotes(api, db)
	api.Note.read, api.Note.write = &read, &writing
	api.state.Indexing.Progress = db.Progress()
	api.state.setVault(v)

	reading := time.Now()
	made := container.NewEditorAssembly(cfg, nil, db)
	worker := newIndexWorker(made, api.state, nil, time.Hour, io.Discard)
	_, wait := startIndexing(t.Context(), v, made, api.state, false, worker, worker.NewSession(), io.Discard)
	t.Cleanup(wait)
	for !api.state.Ready.Load() && api.state.Error.Why() == "" {
		time.Sleep(50 * time.Millisecond)
	}
	if why := api.state.Error.Why(); why != "" {
		t.Fatalf("the vault could not be read: %s", why)
	}
	t.Logf("read %d notes in %s", notes, time.Since(reading).Round(time.Millisecond))

	line, isDone := api.state.ChangeEvents.subscribe()
	t.Cleanup(isDone)

	// One save, the way a quiet interval makes it, and then the wait until the
	// index answers with what was written and the window is told.
	const path = "01/note-000001.md"
	body := "# Heat\n\nA line nobody wrote before, at " + strconv.FormatInt(time.Now().UnixNano(), 10) + ".\n"

	saving := time.Now()
	if _, err := api.Note.write.Save(t.Context(), v, path, body, nil); err != nil {
		t.Fatal(err)
	}
	wrote := time.Since(saving)

	notified := make(chan time.Duration, 1)
	go func() {
		for what := range line {
			for _, p := range what.paths {
				if p == path {
					notified <- time.Since(saving)
					return
				}
			}
			if what.shouldReload {
				notified <- time.Since(saving)
				return
			}
		}
	}()

	select {
	case reached := <-notified:
		t.Logf("save %s, the window told after %s",
			wrote.Round(time.Millisecond), reached.Round(time.Millisecond))
	case <-time.After(2 * time.Minute):
		t.Fatalf("the change never reached a client; the save itself took %s", wrote)
	}
}
