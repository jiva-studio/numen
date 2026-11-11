package vault_test

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/container"
	"github.com/jiva-studio/numen/libs/core/kernel/domain/source"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport/vaultfixture"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultreader"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/testsupport"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// TestLoad runs at the size the product is designed for: a hundred thousand
// notes takes minutes and gigabytes, so it is opt-in (NUMEN_LOAD=1, with
// NUMEN_LOAD_NOTES=10000 for a smaller rehearsal) and needs `-timeout 40m`.
// It reports measurements and does not assert thresholds; the numbers belong
// in docs/performance.md.
func TestLoad(t *testing.T) {
	t.Parallel()
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
	ctx := t.Context()

	generating := testsupport.GetMachine().Now()
	v := vaultfixture.GenerateVault(t, notes)
	t.Logf("generated %d notes in %s", notes, getElapsed(generating))

	db, err := container.OpenIndex(container.NewConfig(container.Config{
		Paths: container.Paths{IndexPath: filepath.Join(t.TempDir(), "index.db")},
	}), ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer testsupport.Close(t, db)

	scan := vaults.NewScan(
		vaultreader.Readers{},
		db.Vaults(),
		db.Notes(),
		db.NoteQueries(),
		db.Maintenance(), nil,
	)

	coldStart := testsupport.GetMachine().Now()
	res, err := scan.Execute(ctx, v)
	if err != nil {
		t.Fatal(err)
	}
	elapsed := testsupport.GetMachine().Now().Sub(coldStart)
	t.Logf("cold scan: %d notes in %s (%.2f ms/note, %.0f notes/s)",
		res.Indexed, elapsed.Round(time.Millisecond),
		float64(elapsed.Milliseconds())/float64(res.Indexed),
		float64(res.Indexed)/elapsed.Seconds())

	warmStart := testsupport.GetMachine().Now()
	if _, err := scan.Execute(ctx, v); err != nil {
		t.Fatal(err)
	}
	t.Logf("warm scan: %s — this is what a startup pays", getElapsed(warmStart))

	size, err := os.Stat(db.File)
	if err == nil {
		t.Logf("index: %.1f MB for %d notes", float64(size.Size())/(1<<20), notes)
	}

	// Two queries, because they measure different things. The common one matches
	// every note in a generated vault and therefore measures ranking a hundred
	// thousand rows; the rare one matches a hundred and measures a targeted
	// search.
	t.Run("common term while a scan runs", func(t *testing.T) {
		measureUnderLoad(t, db, v, scan, "entropy observer")
	})
	t.Run("rare term while a scan runs", func(t *testing.T) {
		measureUnderLoad(t, db, v, scan, vaultfixture.RareTerm)
	})
}

func measureUnderLoad(t *testing.T, db *container.Index, v domain.Vault, scan vaults.Scan, query string) {
	t.Helper()
	const (
		readers  = 4
		duration = 15 * time.Second
	)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var writing sync.WaitGroup
	writing.Go(func() {
		for ctx.Err() == nil {
			if err := touchEverything(v.Path); err != nil {
				return
			}
			if _, err := scan.Execute(ctx, v); err != nil {
				return // canceled
			}
		}
	})

	var readWg sync.WaitGroup
	latencies := make([][]time.Duration, readers)
	for i := range readers {
		readWg.Go(func() {
			queries := db.ChunkSearch()
			deadline := testsupport.GetMachine().Now().Add(duration)
			for testsupport.GetMachine().Now().Before(deadline) && ctx.Err() == nil {
				started := testsupport.GetMachine().Now()
				if _, err := queries.FindLexical(ctx, v.ID, source.LexicalQuery{Text: query, Kinds: getNotesOnly(), Limit: 20}); err != nil {
					return
				}
				latencies[i] = append(latencies[i], testsupport.GetMachine().Now().Sub(started))
			}
		})
	}
	readWg.Wait()
	cancel()
	writing.Wait()

	var all []time.Duration
	for _, l := range latencies {
		all = append(all, l...)
	}
	if len(all) == 0 {
		t.Fatal("no searches completed")
	}
	slices.Sort(all)
	t.Logf("%q: %d searches from %d readers while a scan wrote continuously", query, len(all), readers)
	t.Logf("  p50 %s   p95 %s   p99 %s   max %s",
		all[len(all)/2].Round(time.Millisecond),
		all[len(all)*95/100].Round(time.Millisecond),
		all[len(all)*99/100].Round(time.Millisecond),
		all[len(all)-1].Round(time.Millisecond))
}

func touchEverything(root string) (err error) {
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

func getElapsed(since time.Time) time.Duration {
	return testsupport.GetMachine().Now().Sub(since).Round(time.Millisecond)
}
