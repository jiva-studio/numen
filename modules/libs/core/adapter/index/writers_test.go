package index

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/chunking"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// writing saves one note of a title of its own through the opening given.
func writing(t *testing.T, db *DB, v domain.Vault, path, title string) error {
	t.Helper()
	return db.Notes().Cut(chunking.Sizes{}).Save(t.Context(), string(v.ID), []domain.Note{{
		Fingerprint: domain.Fingerprint{Path: path, Kind: domain.KindNote, Size: int64(len(title)), ModTime: 1},
		Title:       title,
		Type:        domain.TypeNote,
	}})
}

// What one opening writes is what another one reads, while both hold the index
// open.
func TestASecondOpeningReadsWhatTheFirstWrote(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.db")
	one := openedAt(t, path)

	two, err := Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { two.Close() })

	if err := writing(t, one, first, "notes/entropy.md", "Entropy"); err != nil {
		t.Fatal(err)
	}

	found, err := two.NoteQueries().Notes(t.Context(), string(first.ID), []string{"notes/entropy.md"})
	if err != nil {
		t.Fatal(err)
	}
	if found["notes/entropy.md"].Title != "Entropy" {
		t.Errorf("the second opening reads %+v", found["notes/entropy.md"])
	}
}

// Two writers over one index file lose nothing. Both write at once and every
// note either lands or is refused loudly.
func TestTwoWritersOverOneIndexLoseNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.db")
	one := openedAt(t, path)

	two, err := Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { two.Close() })

	const each = 20
	var wg sync.WaitGroup
	failed := make([]error, 0, 2*each)
	var mu sync.Mutex

	for who, db := range map[string]*DB{"one": one, "two": two} {
		for i := range each {
			wg.Add(1)
			go func() {
				defer wg.Done()
				name := fmt.Sprintf("%s-%d", who, i)
				err := writing(t, db, first, "notes/"+name+".md", name)
				if err != nil {
					mu.Lock()
					failed = append(failed, err)
					mu.Unlock()
				}
			}()
		}
	}
	wg.Wait()

	// A writer that could not have its turn says so; it does not come back
	// having written nothing.
	for _, err := range failed {
		t.Errorf("a write was refused: %v", err)
	}

	want := make([]string, 0, 2*each)
	for _, who := range []string{"one", "two"} {
		for i := range each {
			want = append(want, fmt.Sprintf("notes/%s-%d.md", who, i))
		}
	}
	found, err := one.NoteQueries().Notes(t.Context(), string(first.ID), want)
	if err != nil {
		t.Fatal(err)
	}
	missing := make([]string, 0, len(want))
	for _, path := range want {
		if found[path].Title == "" {
			missing = append(missing, path)
		}
	}
	if len(missing) != 0 {
		t.Errorf("%d writes are not in the index: %s", len(missing), strings.Join(missing, ", "))
	}
}

// Every transaction the write pool opens takes the write lock at BEGIN, so one
// that reads before it writes waits its turn.
func TestTheWritePoolBeginsItsTransactionsImmediate(t *testing.T) {
	got := writeDSN("/tmp/index.db")
	if !strings.HasSuffix(got, "&_txlock=immediate") {
		t.Errorf("the write dsn is %q", got)
	}
}
