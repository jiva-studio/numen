package vault_test

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
)

// What these measure against: a scan of an unchanged vault fast enough to run
// at startup, an update to a handful of notes inside 100 ms, and a full rebuild
// of 100k notes in single-digit minutes. The numbers are recorded with the
// measurements in docs/performance.md, which defines what fast enough means.
//
//	go test ./usecase/vault/ -run XXX -bench . -benchtime 1x

func openIndexFor(b *testing.B) *container.Index {
	b.Helper()
	db, err := container.OpenIndex(container.NewConfig(container.Config{
		Paths: container.Paths{IndexPath: filepath.Join(b.TempDir(), "index.db")},
	}), b.Context())
	if err != nil {
		b.Fatal(err)
	}
	testsupport.CloseAfter(b, db)
	return db
}

func scanFor(db *container.Index) vaults.Scan {
	return vaults.NewScan(
		vaultreader.Readers{},
		db.Vaults(),
		db.Notes(),
		db.NoteQueries(),
		db.Maintenance(),
	)
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

// BenchmarkIncrementalScan is one edited note in a vault with no other change:
// a whole walk, and a file stat of every note, for one file. It is the cost of
// detecting a change without notification, which is what a scan at startup
// pays; a change the watcher reports goes through Refresh and touches only that
// path.
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
			queries := db.ChunkSearch()

			b.ResetTimer()
			for range b.N {
				if _, err := queries.FindLexical(
					b.Context(), v.ID, source.LexicalQuery{Text: "entropy observer", Kinds: getNotesOnly(), Limit: 20}); err != nil {
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
	pos := testsupport.GetMachine().Now().Add(time.Duration(seed+1) * time.Second)
	if err := os.Chtimes(path, pos, pos); err != nil {
		b.Fatal(err)
	}
}

// BenchmarkSearchDuringScan measures what a search costs while the index is
// being written.
// A scan runs in the background over a vault big enough to take seconds, and
// searches run against the same database throughout. WAL lets a reader never
// wait for the writer, and the write pool is one connection so writers queue.
func BenchmarkSearchDuringScan(b *testing.B) {
	const notes = 10_000
	v := testsupport.GenerateVault(b, notes)
	db := openIndexFor(b)

	ctx, cancel := context.WithCancel(b.Context())
	defer cancel()

	// Fill the index first. Searching an empty one is fast, and a benchmark
	// that starts there sizes itself against a measurement of nothing.
	if _, err := scanFor(db).Execute(ctx, v); err != nil {
		b.Fatal(err)
	}

	scanning := make(chan struct{})
	go func() {
		defer close(scanning)
		for {
			if _, err := scanFor(db).Execute(ctx, v); err != nil {
				return // canceled at the end of the benchmark
			}
			if ctx.Err() != nil {
				return
			}
			// Keep writing for as long as the benchmark reads: reindex from
			// scratch and not measuring against an index that is finished.
			if err := touchAll(v.Path); err != nil {
				return
			}
		}
	}()

	queries := db.ChunkSearch()
	b.ResetTimer()
	for range b.N {
		if _, err := queries.FindLexical(ctx, v.ID, source.LexicalQuery{Text: "entropy observer", Kinds: getNotesOnly(), Limit: 20}); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()
	cancel()
	<-scanning
}

func touchAll(root string) (err error) {
	pos := testsupport.GetMachine().Now()
	scope, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, scope.Close()) }()
	return fs.WalkDir(scope.FS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		return scope.Chtimes(p, pos, pos)
	})
}

// BenchmarkLinks measures resolution, which dominates the cost of links:
// storing them is a handful of rows per note, but every link is resolved again
// on each request, because resolution is a query and not a stored fact.
func BenchmarkLinks(b *testing.B) {
	for _, notes := range []int{1_000, 10_000} {
		b.Run(fmt.Sprint(notes), func(b *testing.B) {
			v := testsupport.GenerateVault(b, notes)
			db := openIndexFor(b)
			if _, err := scanFor(db).Execute(b.Context(), v); err != nil {
				b.Fatal(err)
			}
			links := db.NoteQueries()
			const of = "00/note-000000.md"

			b.ResetTimer()
			for range b.N {
				resolved, err := links.GetLinks(b.Context(), v.ID, of)
				if err != nil {
					b.Fatal(err)
				}
				if len(resolved) == 0 {
					b.Fatal("the generated vault has no links to resolve")
				}
			}
		})
	}
}

// BenchmarkBacklinks measures the other direction, which is the one that has to
// look at every link in the vault and not at one note's worth.
func BenchmarkBacklinks(b *testing.B) {
	for _, notes := range []int{1_000, 10_000} {
		b.Run(fmt.Sprint(notes), func(b *testing.B) {
			v := testsupport.GenerateVault(b, notes)
			db := openIndexFor(b)
			if _, err := scanFor(db).Execute(b.Context(), v); err != nil {
				b.Fatal(err)
			}
			links := db.NoteQueries()
			const to = "00/note-000000.md"

			b.ResetTimer()
			for range b.N {
				if _, err := links.GetBacklinks(b.Context(), v.ID, to); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
