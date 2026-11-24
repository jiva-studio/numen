package main

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol/fakeserver"
)

func TestTheStateFolderIsCreatedWhenItIsMissing(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state", "server")

	if err := prepareStateDir(dir); err != nil {
		t.Fatal(err)
	}

	if info, err := os.Stat(dir); err != nil || !info.IsDir() {
		t.Errorf("the state folder is %v, %v, want a folder", info, err)
	}
}

func newTestKeeper(t *testing.T, stateDir string) (*keeper, string) {
	t.Helper()
	fake := fakeserver.New(fakeserver.Config{BodyDir: filepath.Join(stateDir, bodiesFolder)})
	path := getStatePath(fake, stateDir)
	return newKeeper(fake, path, &atomic.Bool{}, make(chan struct{})), path
}

func post(t *testing.T, handler http.Handler, path string) {
	t.Helper()
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, nil))
}

func TestARequestThatChangedNothingWritesNoFile(t *testing.T) {
	keep, path := newTestKeeper(t, t.TempDir())
	post(t, keep, "/numen.sync.v1.CatalogService/GetCatalog")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("the first request left no state file: %v", err)
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}

	post(t, keep, "/numen.sync.v1.CatalogService/GetCatalog")

	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a request that changed nothing wrote the state file again: %v", err)
	}
}

func TestAStateFileOfTheOldLayoutIsIgnoredAndTheServerStartsEmpty(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, stateFile), []byte(`{"chains":{},"bodies":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	keep, _ := newTestKeeper(t, dir)

	raw, err := keep.fake.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(raw, []byte(fakeserver.SnapshotPrefix)) {
		t.Errorf("the server did not start empty on the new layout: %.60s", raw)
	}
}

func TestTheStateSurvivesARestart(t *testing.T) {
	dir := t.TempDir()
	first, _ := newTestKeeper(t, dir)
	post(t, first, "/numen.sync.v1.CatalogService/GetCatalog")
	before, err := first.fake.Snapshot()
	if err != nil {
		t.Fatal(err)
	}

	second, _ := newTestKeeper(t, dir)

	after, err := second.fake.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Errorf("state after the restart = %s, want %s", after, before)
	}
}

func TestTheSessionFolderIsRemovedWhenTheServerStops(t *testing.T) {
	folder, remove, err := newSessionFolder()
	if err != nil {
		t.Fatal(err)
	}
	keep, _ := newTestKeeper(t, folder)
	post(t, keep, "/numen.sync.v1.CatalogService/GetCatalog")

	remove()

	if _, err := os.Stat(folder); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("the session folder is still there: %v", err)
	}
}
