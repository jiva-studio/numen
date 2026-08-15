package vault_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/container"
	usecase "github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/testsupport"
)

// The budgets these measure against are in ADR-0002: a scan of an unchanged
// vault fast enough to run at startup, an incremental update inside 100 ms, and
// a full rebuild of 100k notes in single-digit minutes.
//
//	go test ./internal/core/usecase/vault/ -run XXX -bench . -benchtime 1x

func openIndexFor(b *testing.B) *container.Index {
	b.Helper()
	db, err := container.Config{
		IndexPath: filepath.Join(b.TempDir(), "index.db"),
	}.OpenIndex(b.Context())
	if err != nil {
		b.Fatal(err)
	}
	b.Cleanup(func() { db.Close() })
	return db
}

func scanFor(db *container.Index) usecase.Scan {
	return usecase.Scan{
		Readers: filesystem.Readers{},
		Vaults:  db.Vaults(),
		Notes:   db.Notes(),
		Known:   db.Queries(),
	}
}

// BenchmarkColdScan is the full rebuild: an empty index and every note read,
// parsed and written.
func BenchmarkColdScan(b *testing.B) {
	for _, notes := range []int{1_000, 10_000} {
		b.Run(fmt.Sprint(notes), func(b *testing.B) {
			v := testsupport.GenerateVault(b, notes)
			b.ResetTimer()
			for range b.N {
				b.StopTimer()
				db := openIndexFor(b)
				b.StartTimer()

				res, err := scanFor(db).Execute(b.Context(), v)
				if err != nil {
					b.Fatal(err)
				}
				if res.Indexed != notes {
					b.Fatalf("indexed %d of %d", res.Indexed, notes)
				}
			}
		})
	}
}

// BenchmarkWarmScan is what happens at startup: nothing has changed, so no file
// is opened and the whole cost is the walk and one query.
func BenchmarkWarmScan(b *testing.B) {
	for _, notes := range []int{1_000, 10_000} {
		b.Run(fmt.Sprint(notes), func(b *testing.B) {
			v := testsupport.GenerateVault(b, notes)
			db := openIndexFor(b)
			scan := scanFor(db)
			if _, err := scan.Execute(b.Context(), v); err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()
			for range b.N {
				res, err := scan.Execute(b.Context(), v)
				if err != nil {
					b.Fatal(err)
				}
				if res.Indexed != 0 {
					b.Fatalf("a warm scan reindexed %d notes", res.Indexed)
				}
			}
		})
	}
}

// BenchmarkIncrementalScan is one edited note in a vault that is otherwise
// untouched — the path a file watcher takes.
func BenchmarkIncrementalScan(b *testing.B) {
	for _, notes := range []int{1_000, 10_000} {
		b.Run(fmt.Sprint(notes), func(b *testing.B) {
			v := testsupport.GenerateVault(b, notes)
			db := openIndexFor(b)
			scan := scanFor(db)
			if _, err := scan.Execute(b.Context(), v); err != nil {
				b.Fatal(err)
			}
			edited := filepath.Join(v.Path, "00", "note-000000.md")

			b.ResetTimer()
			for i := range b.N {
				b.StopTimer()
				touch(b, edited, i)
				b.StartTimer()

				res, err := scan.Execute(b.Context(), v)
				if err != nil {
					b.Fatal(err)
				}
				if res.Indexed != 1 {
					b.Fatalf("reindexed %d notes, want 1", res.Indexed)
				}
			}
		})
	}
}

func BenchmarkSearch(b *testing.B) {
	for _, notes := range []int{1_000, 10_000} {
		b.Run(fmt.Sprint(notes), func(b *testing.B) {
			v := testsupport.GenerateVault(b, notes)
			db := openIndexFor(b)
			if _, err := scanFor(db).Execute(b.Context(), v); err != nil {
				b.Fatal(err)
			}
			queries := db.Queries()

			b.ResetTimer()
			for range b.N {
				if _, err := queries.Search(b.Context(), v.ID, "entropy observer", 20); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func touch(b *testing.B, path string, seed int) {
	b.Helper()
	body := fmt.Sprintf("# Edited\n\nrevision %d of this note\n", seed)
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		b.Fatal(err)
	}
	// Modification time is the invalidation key, and a benchmark writes faster
	// than the clock ticks.
	at := time.Now().Add(time.Duration(seed+1) * time.Second)
	if err := os.Chtimes(path, at, at); err != nil {
		b.Fatal(err)
	}
}
