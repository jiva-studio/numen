package keychain

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func countOpenFiles() (int, bool) {
	entries, err := os.ReadDir("/proc/self/fd")
	if err != nil {
		return 0, false
	}
	return len(entries), true
}

func holdLock(t *testing.T, dir string) {
	t.Helper()
	release, err := NewStore(dir).locks.lock(t.Context(), filepath.Join(dir, credentialsFile+lockSuffix))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := release(); err != nil {
			t.Error(err)
		}
	})
}

// Cancelled waits for the lock that another process holds leave a bounded number of goroutines and
// open files, however many calls are cancelled.
func TestCancelledWaitsForTheCredentialsLockAreBounded(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	store := NewStore(dir)
	if err := store.SetKey(t.Context(), "secret/a", "value-a"); err != nil {
		t.Fatal(err)
	}
	holdLock(t, dir)

	goroutinesBefore := runtime.NumGoroutine()
	filesBefore, canCountFiles := countOpenFiles()
	const calls, allowance = 200, 10
	for range calls {
		ctx, cancel := context.WithTimeout(t.Context(), time.Millisecond)
		err := store.SetKey(ctx, "secret/b", "value-b")
		cancel()
		if err == nil {
			t.Fatal("SetKey took the lock another process holds")
		}
	}

	if got := runtime.NumGoroutine(); got > goroutinesBefore+allowance {
		t.Errorf("%d goroutines after %d cancelled waits, %d before: want at most %d more", got, calls, goroutinesBefore, allowance)
	}
	if canCountFiles {
		if got, _ := countOpenFiles(); got > filesBefore+allowance {
			t.Errorf("%d open files after %d cancelled waits, %d before: want at most %d more", got, calls, filesBefore, allowance)
		}
	}
}

func waitForLockWaiter(t *testing.T) {
	t.Helper()
	timeout := time.After(5 * time.Second)
	buffer := make([]byte, 1<<20)
	for {
		select {
		case <-timeout:
			t.Fatal("no goroutine waits for the lock of the credentials file")
		default:
		}
		if strings.Contains(string(buffer[:runtime.Stack(buffer, true)]), "keychain.lockOpenFile") {
			return
		}
		runtime.Gosched()
	}
}

// A store is not locked while a write waits for the file lock: reads of the same store and of another
// store on the same folder go on.
func TestAWriteWaitingForTheCredentialsLockDoesNotHoldUpReads(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	store, other := NewStore(dir), NewStore(dir)
	if err := store.SetKey(t.Context(), "secret/a", "value-a"); err != nil {
		t.Fatal(err)
	}
	holdLock(t, dir)

	ctx, cancel := context.WithCancel(t.Context())
	writing := make(chan error, 1)
	go func() { writing <- store.SetKey(ctx, "secret/b", "value-b") }()
	t.Cleanup(func() {
		cancel()
		if err := <-writing; err != nil && !errors.Is(err, context.Canceled) {
			t.Error(err)
		}
	})
	waitForLockWaiter(t)

	for name, reader := range map[string]*Store{"the same store": store, "another store": other} {
		got := make(chan string, 1)
		go func() {
			value, err := reader.GetKey(t.Context(), "secret/a")
			if err != nil {
				value = err.Error()
			}
			got <- value
		}()
		select {
		case value := <-got:
			if value != "value-a" {
				t.Errorf("GetKey of %s = %q while a write waits for the lock, want the value", name, value)
			}
		case <-time.After(time.Second):
			t.Errorf("GetKey of %s is held up by a write that waits for the lock", name)
		}
	}
}
