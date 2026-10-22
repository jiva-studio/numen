package index

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"testing"

	notecase "github.com/jiva-studio/numen/modules/libs/core/usecase/note"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// The types of note these tests keep in the way a plugin declares: a deck is found by its sections and its
// cards, a stencil by its title, and neither is cut.
const (
	deckType    domain.NoteType = "deck"
	stencilType domain.NoteType = "stencil"
	presetType  domain.NoteType = "preset"
)

var profiles = map[domain.NoteType]domain.NoteProfile{
	deckType:    {HeadingLevels: []int{1, 2}, MarkedLevels: []int{2}, Mark: regexp.MustCompile("^[0-9a-hjkmnp-tv-z]{10}$")},
	stencilType: {HeadingLevels: []int{}},
}

// saveCut stores notes cut at the default sizes, the way a scan hands them over.
func saveCut(ctx context.Context, db *DB, vaultID domain.VaultID, notes []domain.Note) error {
	cutNotes := notecase.CutNotes{Profiles: func(t domain.NoteType) (domain.NoteProfile, bool) {
		profile, declared := profiles[t]
		return profile, declared
	}}
	return db.Notes().Save(ctx, vaultID, cutNotes.Execute(notes))
}

// saveNote saves one note of a title of its own through the opening given.
func saveNote(t *testing.T, db *DB, v domain.Vault, path, title string) error {
	t.Helper()
	return saveCut(t.Context(), db, v.ID, []domain.Note{{
		Fingerprint: domain.Fingerprint{Path: path, Kind: domain.KindNote, Size: int64(len(title)), ModTime: walked},
		Title:       title,
		Type:        domain.TypeNote,
	}})
}

// What one opening writes is what another one reads, while both hold the index
// open.
func TestASecondOpeningReadsWhatTheFirstWrote(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.db")
	one := openDBAt(t, path)

	two, err := Open(t.Context(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { two.Close() })

	if err := saveNote(t, one, first, "notes/entropy.md", "Entropy"); err != nil {
		t.Fatal(err)
	}

	found, err := two.NoteQueries().GetNotes(t.Context(), first.ID, []string{"notes/entropy.md"})
	if err != nil {
		t.Fatal(err)
	}
	if found["notes/entropy.md"].Title != "Entropy" {
		t.Errorf("the second opening reads %+v", found["notes/entropy.md"])
	}
}

// Two writers over one index file lose nothing. Both write at once and every
// note either lands or fails loudly.
func TestTwoWritersOverOneIndexLoseNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "index.db")
	one := openDBAt(t, path)

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
				err := saveNote(t, db, first, "notes/"+name+".md", name)
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
		t.Errorf("a write failed: %v", err)
	}

	want := make([]string, 0, 2*each)
	for _, who := range []string{"one", "two"} {
		for i := range each {
			want = append(want, fmt.Sprintf("notes/%s-%d.md", who, i))
		}
	}
	found, err := one.NoteQueries().GetNotes(t.Context(), first.ID, want)
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
	got := writeDSN("/tmp/index.db", settings{synchronous: shipped})
	if !strings.HasSuffix(got, "&_txlock=immediate") {
		t.Errorf("the write dsn is %q", got)
	}
}
