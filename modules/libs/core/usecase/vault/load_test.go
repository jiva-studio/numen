package vault_test

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/filesystem"
	"github.com/jiva-studio/numen/modules/libs/core/container"
	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/testsupport"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// The load test runs at the size the product is designed for, which no
// benchmark does: a hundred thousand notes takes minutes and gigabytes, and
// putting that in the ordinary suite would make everyone wait for something
// almost nobody needs.
//
//	NUMEN_LOAD=1 go test ./internal/core/usecase/vault/ -run TestLoad -v -timeout 40m
//	NUMEN_LOAD=1 NUMEN_LOAD_NOTES=10000 go test ...   # a smaller rehearsal
//
// It reports rather than asserts. A threshold that fails on a slower laptop
// teaches people to ignore the test; the numbers belong in docs/performance.md,
// where a human compares them.
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

	generating := time.Now()
	v := testsupport.GenerateVault(t, notes)
	t.Logf("generated %d notes in %s", notes, took(generating))

	db, err := container.Config{
		IndexPath: filepath.Join(t.TempDir(), "index.db"),
	}.OpenIndex(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	scan := usecase.Scan{
		Readers:     filesystem.VaultReaders{},
		Vaults:      db.Vaults(),
		Notes:       db.Notes(),
		Known:       db.Queries(),
		Maintenance: db.Maintenance(),
	}

	cold := time.Now()
	res, err := scan.Execute(ctx, v)
	if err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(cold)
	t.Logf("cold scan: %d notes in %s (%.2f ms/note, %.0f notes/s)",
		res.Indexed, elapsed.Round(time.Millisecond),
		float64(elapsed.Milliseconds())/float64(res.Indexed),
		float64(res.Indexed)/elapsed.Seconds())

	warm := time.Now()
	if _, err := scan.Execute(ctx, v); err != nil {
		t.Fatal(err)
	}
	t.Logf("warm scan: %s — this is what a startup pays", took(warm))

	size, err := os.Stat(db.Path())
	if err == nil {
		t.Logf("index: %.1f MB for %d notes", float64(size.Size())/(1<<20), notes)
	}

	// Two queries, because they measure different things. The common one matches
	// every note in a generated vault and therefore measures ranking a hundred
	// thousand rows; the rare one matches a hundred and is what a person
	// searching for something in particular actually does.
	t.Run("common term while a scan runs", func(t *testing.T) {
		measureUnderLoad(t, db, v, scan, "entropy observer")
	})
	t.Run("rare term while a scan runs", func(t *testing.T) {
		measureUnderLoad(t, db, v, scan, testsupport.RareTerm)
	})
}

// measureUnderLoad answers the question a mean cannot: not what a search costs
// on average while the index is being rewritten, but what the slowest one in
// twenty costs. An application feels slow at its ninety-fifth percentile, not at
// its median.
func measureUnderLoad(t *testing.T, db *container.Index, v domain.Vault, scan usecase.Scan, query string) {
	const (
		readers  = 4
		duration = 15 * time.Second
	)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var writing sync.WaitGroup
	writing.Add(1)
	go func() {
		defer writing.Done()
		for ctx.Err() == nil {
			if err := touchEverything(v.Path); err != nil {
				return
			}
			if _, err := scan.Execute(ctx, v); err != nil {
				return // cancelled
			}
		}
	}()

	var reading sync.WaitGroup
	latencies := make([][]time.Duration, readers)
	for i := range readers {
		reading.Add(1)
		go func() {
			defer reading.Done()
			queries := db.Queries()
			deadline := time.Now().Add(duration)
			for time.Now().Before(deadline) && ctx.Err() == nil {
				started := time.Now()
				if _, err := queries.Search(ctx, string(v.ID), query, 20); err != nil {
					return
				}
				latencies[i] = append(latencies[i], time.Since(started))
			}
		}()
	}
	reading.Wait()
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

func touchEverything(root string) error {
	at := time.Now()
	return filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		return os.Chtimes(p, at, at)
	})
}

func took(since time.Time) time.Duration { return time.Since(since).Round(time.Millisecond) }
