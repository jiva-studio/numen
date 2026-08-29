package webui

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// TestEditLoad is what a person waits for between a save and the vault being
// level again, at the size the product is designed for.
//
// It measures the path the application owns: the write, the watcher noticing
// it, the refresh, and the change reaching a client. What a webview does either
// side of that — a keystroke becoming an RPC, a picture being painted — is a
// browser's, and is not here.
//
//	NUMEN_LOAD=1 go test ./internal/adapter/webui/ -run TestEditLoad -v -timeout 20m
//	NUMEN_LOAD=1 NUMEN_LOAD_NOTES=10000 go test ...   # a smaller rehearsal
//
// It reports rather than asserts, for the reason the vault load test does: a
// threshold that fails on a slower machine teaches people to ignore the test.
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

	api := &API{
		Notes:     db.Queries(),
		Links:     db.Links(),
		Listeners: following(),
		Watching:  focusing(),
		Progress:  db.Progress(),
		Reads:     &note.Read{Readers: filesystem.Readers{}},
		Saves:     &note.Write{Readers: filesystem.Readers{}, Writers: filesystem.Writers{}},
	}
	api.show(v)
	scan := usecase.Scan{
		Readers:     filesystem.Readers{},
		Vaults:      db.Vaults(),
		Notes:       db.Notes(),
		Known:       db.Queries(),
		Maintenance: db.Maintenance(),
	}
	held := &holding{NoteRepository: db.Notes()}
	follow := usecase.Follow{
		Watcher: cfg.VaultWatcher(),
		Refresh: usecase.Refresh{Readers: filesystem.Readers{}, Notes: held},
		Scan:    scan,
	}

	reading := time.Now()
	wait := begin(t.Context(), v, cfg, db, api, scan, follow, held, filesystem.Readers{}, nil, waking(time.Hour), &pending{}, io.Discard)
	t.Cleanup(wait)
	for !api.Ready.Load() && api.failure() == "" {
		time.Sleep(50 * time.Millisecond)
	}
	if why := api.failure(); why != "" {
		t.Fatalf("the vault could not be read: %s", why)
	}
	t.Logf("read %d notes in %s", notes, time.Since(reading).Round(time.Millisecond))

	line, done := api.Listeners.listen()
	t.Cleanup(done)

	// One save, the way a quiet interval makes it, and then the wait until the
	// index answers with what was written and the window is told.
	const path = "01/note-000001.md"
	body := "# Heat\n\nA line nobody wrote before, at " + strconv.FormatInt(time.Now().UnixNano(), 10) + ".\n"

	saving := time.Now()
	if _, err := api.Saves.Save(t.Context(), v, path, body, nil); err != nil {
		t.Fatal(err)
	}
	wrote := time.Since(saving)

	told := make(chan time.Duration, 1)
	go func() {
		for what := range line {
			for _, p := range what.paths {
				if p == path {
					told <- time.Since(saving)
					return
				}
			}
			if what.reload {
				told <- time.Since(saving)
				return
			}
		}
	}()

	select {
	case reached := <-told:
		t.Logf("save %s, the window told after %s",
			wrote.Round(time.Millisecond), reached.Round(time.Millisecond))
	case <-time.After(2 * time.Minute):
		t.Fatalf("the change never reached a client; the save itself took %s", wrote)
	}
}
