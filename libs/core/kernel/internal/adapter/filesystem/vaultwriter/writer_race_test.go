package vaultwriter_test

import (
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaulttest"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapter/filesystem/vaultwriter"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
)

// Writers that start from the same stored file are racing for it: the one that
// replaces it first makes the baseline of the others stale.
func TestWritersWithOneBaselineHaveOneWinner(t *testing.T) {
	const racers = 8
	writers := vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}
	vault := vaulttest.NewVault(t, nil)
	first, err := writers.Open(vault)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := first.Write(t.Context(), "n.md", []byte("start"), domain.FileStat{})
	if err != nil {
		t.Fatal(err)
	}

	opened := make([]*vaultwriter.VaultWriter, racers)
	for i := range opened {
		if opened[i], err = writers.Open(vault); err != nil {
			t.Fatal(err)
		}
	}
	errs := make([]error, racers)
	var ready, done sync.WaitGroup
	ready.Add(racers)
	begin := make(chan struct{})
	for i := range racers {
		done.Go(func() {
			content := []byte(strings.Repeat("x", i+10))
			ready.Done()
			<-begin
			_, errs[i] = opened[i].Write(t.Context(), "n.md", content, baseline.Stat)
		})
	}
	ready.Wait()
	close(begin)
	done.Wait()

	won := 0
	for i, err := range errs {
		switch {
		case err == nil:
			won++
		case !errors.Is(err, edit.ErrStale):
			t.Errorf("writer %d: %v", i, err)
		}
	}
	if won != 1 {
		t.Fatalf("%d of %d writers with one baseline replaced the file, want 1", won, racers)
	}
}

// The version of a vault says how many writes it took. A refused write took
// none, and another vault counts on its own.
func TestVersionRisesOnEveryWriteAndNotOnARefusedOne(t *testing.T) {
	writers := vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}
	vault := vaulttest.NewVault(t, nil)
	elsewhere := vaulttest.NewVault(t, nil)
	w, err := writers.Open(vault)
	if err != nil {
		t.Fatal(err)
	}
	other, err := writers.Open(elsewhere)
	if err != nil {
		t.Fatal(err)
	}

	one, err := w.Write(t.Context(), "a.md", []byte("1"), domain.FileStat{})
	if err != nil {
		t.Fatal(err)
	}
	two, err := w.Write(t.Context(), "b.md", []byte("2"), domain.FileStat{})
	if err != nil {
		t.Fatal(err)
	}
	if one.Version != 1 || two.Version != 2 {
		t.Fatalf("versions of two writes = %d, %d, want 1, 2", one.Version, two.Version)
	}

	stale := domain.FileStat{Size: 99, ModTime: one.Stat.ModTime}
	if _, err := w.Write(t.Context(), "a.md", []byte("refused"), stale); !errors.Is(err, edit.ErrStale) {
		t.Fatalf("a write from a stale baseline: %v", err)
	}
	latest, err := w.Write(t.Context(), "c.md", []byte("4"), domain.FileStat{})
	if err != nil {
		t.Fatal(err)
	}
	if latest.Version != 3 {
		t.Errorf("the write after the refused one is at version %d, want 3", latest.Version)
	}

	fresh, err := other.Write(t.Context(), "a.md", []byte("1"), domain.FileStat{})
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Version != 1 {
		t.Errorf("the first write of another vault is at version %d, want 1", fresh.Version)
	}
}

// A caller that holds the vault lock through VaultWriters.Lock writes without
// waiting for itself.
func TestWriteSucceedsWithTheVaultLockTaken(t *testing.T) {
	writers := vaultwriter.VaultWriters{Locks: vaultwriter.NewLocks()}
	vault := vaulttest.NewVault(t, nil)
	w, err := writers.Open(vault)
	if err != nil {
		t.Fatal(err)
	}
	release, err := writers.Lock(t.Context(), vault)
	if err != nil {
		t.Fatal(err)
	}
	defer release()

	finished := make(chan error, 1)
	go func() {
		_, err := w.Write(t.Context(), "n.md", []byte("held"), domain.FileStat{})
		finished <- err
	}()
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
}
