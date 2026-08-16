package index

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestEveryConnectionGetsThePragmas(t *testing.T) {
	// The bug this guards against is invisible to an ordinary test: executing
	// `PRAGMA foreign_keys = ON` after opening configures one connection of the
	// pool, and a sequential test keeps being handed that same connection. Only
	// holding several at once shows the rest were never configured.
	ctx := t.Context()
	db, err := Open(ctx, filepath.Join(t.TempDir(), "index.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	const connections = 8
	db.read.SetMaxOpenConns(connections)

	var wg sync.WaitGroup
	foreignKeys := make([]int, connections)
	busyTimeout := make([]int, connections)
	synchronous := make([]int, connections)
	release := make(chan struct{})

	for i := range connections {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn, err := db.read.Conn(ctx)
			if err != nil {
				t.Error(err)
				return
			}
			defer conn.Close()
			if err := conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys[i]); err != nil {
				t.Error(err)
				return
			}
			if err := conn.QueryRowContext(ctx, "PRAGMA busy_timeout").Scan(&busyTimeout[i]); err != nil {
				t.Error(err)
				return
			}
			if err := conn.QueryRowContext(ctx, "PRAGMA synchronous").Scan(&synchronous[i]); err != nil {
				t.Error(err)
				return
			}
			// Hold the connection so the next goroutine is forced to open a new
			// one instead of reusing this one.
			<-release
		}()
	}
	// Give every goroutine time to take a distinct connection.
	for range connections {
		release <- struct{}{}
	}
	wg.Wait()

	for i := range connections {
		if foreignKeys[i] != 1 {
			t.Errorf("connection %d has foreign_keys = %d, want 1", i, foreignKeys[i])
		}
		if busyTimeout[i] != 5000 {
			t.Errorf("connection %d has busy_timeout = %d, want 5000", i, busyTimeout[i])
		}
		// NORMAL is 1. FULL asks the disk to settle at every commit, which a
		// rebuild does thousands of times, and nothing else in the suite
		// notices which one is set.
		if synchronous[i] != 1 {
			t.Errorf("connection %d has synchronous = %d, want 1 (NORMAL)", i, synchronous[i])
		}
	}
}

func TestDSNCarriesEveryPragma(t *testing.T) {
	got := dsn("/tmp/index.db")
	for _, want := range []string{"foreign_keys%281%29", "busy_timeout%285000%29", "journal_mode%28WAL%29", "synchronous%28NORMAL%29"} {
		if !strings.Contains(got, want) {
			t.Errorf("dsn %q is missing %s", got, want)
		}
	}
	if !strings.HasPrefix(got, "/tmp/index.db?") {
		t.Errorf("dsn %q does not start with the path", got)
	}
}
