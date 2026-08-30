package index

import (
	"path/filepath"
	"slices"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// reading is the index at the path given, opened for asking alone.
func reading(t *testing.T, path string) *Reading {
	t.Helper()
	r, err := OpenToRead(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	return r
}

// nothing is the index of a machine where nothing has scanned.
func nothing(t *testing.T) *Reading {
	t.Helper()
	r, err := OpenNothing(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { r.Close() })
	return r
}

// What one process built is what another one reads, and it reads it while the
// first still holds the index open.
func TestAReadingAnswersOverWhatTheWriteSideBuilt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.db")
	db := openedAt(t, path)
	book(t, db, first, "library/entropy.epub", 0xfe)
	book(t, db, second, "library/quasar.epub", 0xff)

	r := reading(t, path)

	// One database holds every vault, and both of them file a book under the
	// same folder name. Their words share nothing, so a row from the wrong one
	// is recognisable.
	for _, held := range []struct {
		vault domain.Vault
		word  string
		path  string
	}{
		{first, "entropy", "library/entropy.epub"},
		{second, "quasar", "library/quasar.epub"},
	} {
		found, err := r.ChunkQueries().Lexical(t.Context(), held.vault.ID, held.word, nil, 10, false)
		if err != nil {
			t.Fatal(err)
		}
		if len(found) == 0 {
			t.Fatalf("%q found nothing in the %s vault", held.word, held.vault.Name)
		}
		for _, p := range found {
			if p.Source != held.path {
				t.Errorf("the %s vault answered with %q", held.vault.Name, p.Source)
			}
		}

		sources, err := r.SourcesKnown().Under(t.Context(), held.vault.ID, "library")
		if err != nil {
			t.Fatal(err)
		}
		paths := make([]string, 0, len(sources))
		for _, ref := range sources {
			paths = append(paths, ref.Path)
		}
		if want := []string{held.path}; !slices.Equal(paths, want) {
			t.Errorf("the %s vault holds %v, want %v", held.vault.Name, paths, want)
		}
	}
}

// The index opened for asking hands out queries and nothing that writes.
func TestAReadingHandsOutNoWritePort(t *testing.T) {
	sources := nothing(t).SourcesKnown()

	var _ port.SourceQueries = sources
	var _ port.VectorQueries = sources

	held := any(sources)
	if _, is := held.(port.SourceRepository); is {
		t.Errorf("%T records sources", held)
	}
	if _, is := held.(port.VectorRepository); is {
		t.Errorf("%T records vectors", held)
	}
}

// A machine where nothing has scanned has an index with nothing in it, and it
// answers every question with nothing.
func TestAnIndexNothingHasScannedAnswersEmpty(t *testing.T) {
	r := nothing(t)
	ctx := t.Context()

	if found, err := r.ChunkQueries().Lexical(ctx, first.ID, "entropy", nil, 10, false); err != nil || len(found) != 0 {
		t.Errorf("the words answered with %d passages: %v", len(found), err)
	}
	if found, err := r.ChunkQueries().Named(ctx, first.ID, "entropy", nil, 10, false); err != nil || len(found) != 0 {
		t.Errorf("the names answered with %d passages: %v", len(found), err)
	}
	if found, err := r.SourcesKnown().Under(ctx, first.ID, "library"); err != nil || len(found) != 0 {
		t.Errorf("the vault holds %d sources: %v", len(found), err)
	}
	if found, err := r.SourcesKnown().Fingerprints(ctx, first.ID, domain.KindNote); err != nil || len(found) != 0 {
		t.Errorf("the index believes something about %d files: %v", len(found), err)
	}
	if found, err := r.SourcesKnown().Recognised(ctx, first.ID, domain.KindBook); err != nil || len(found) != 0 {
		t.Errorf("%d sources had their text made for them: %v", len(found), err)
	}
	model := port.EmbeddingModel{Name: "model", Dimensions: 1024}
	if found, err := r.SourcesKnown().Unembedded(ctx, first.ID, model, 0, 10); err != nil || len(found) != 0 {
		t.Errorf("%d chunks owe a vector: %v", len(found), err)
	}
}

// Every connection a reading opens is one that cannot write. The pragma is
// per-connection state, and a sequential test is handed the same connection
// every time, so the rest are only seen by holding several at once.
func TestEveryConnectionOfAReadingIsQueryOnly(t *testing.T) {
	ctx := t.Context()
	path := filepath.Join(t.TempDir(), "index.db")
	openedAt(t, path)
	r := reading(t, path)

	const conns = 8
	r.read.SetMaxOpenConns(conns)

	var wg sync.WaitGroup
	queryOnly := make([]int, conns)
	busyTimeout := make([]int, conns)
	refused := make([]error, conns)
	release := make(chan struct{})

	for i := range conns {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := r.read.Conn(ctx)
			if err != nil {
				t.Error(err)
				return
			}
			defer conn.Close()
			if err := conn.QueryRowContext(ctx, "PRAGMA query_only").Scan(&queryOnly[i]); err != nil {
				t.Error(err)
				return
			}
			if err := conn.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout[i]); err != nil {
				t.Error(err)
				return
			}
			_, refused[i] = conn.ExecContext(ctx, `DELETE FROM vaults`)
			// Hold the connection so the next goroutine is forced to open a new
			// one instead of reusing this one.
			<-release
		}()
	}
	for range conns {
		release <- struct{}{}
	}
	wg.Wait()

	for i := range conns {
		if queryOnly[i] != 1 {
			t.Errorf("connection %d has query_only = %d, want 1", i, queryOnly[i])
		}
		if busyTimeout[i] != 5000 {
			t.Errorf("connection %d has busy_timeout = %d, want 5000", i, busyTimeout[i])
		}
		if refused[i] == nil {
			t.Errorf("connection %d emptied a table", i)
		}
	}
}
