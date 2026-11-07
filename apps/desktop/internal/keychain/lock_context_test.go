package keychain

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

// The wait for the lock of the credentials file ends with the context, and a read of another
// entry is not held up by it.
func TestTheWaitForTheCredentialsLockEndsWithTheContext(t *testing.T) {
	useNoKeychain(t)
	dir := t.TempDir()
	store := NewStore(dir)
	if err := store.SetKey(t.Context(), "secret/a", "value-a"); err != nil {
		t.Fatal(err)
	}
	release, err := NewStore(dir).locks.lock(t.Context(), filepath.Join(dir, credentialsFile+lockSuffix))
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := release(); err != nil {
			t.Error(err)
		}
	}()

	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
	defer cancel()
	result := make(chan error, 1)
	go func() { result <- store.SetKey(ctx, "secret/b", "value-b") }()

	select {
	case err := <-result:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("SetKey returned %v, want the deadline", err)
		}
	case <-time.After(time.Second):
		t.Fatal("SetKey still waits for the lock 1s after its 300ms context")
	}

	readCtx, stop := context.WithTimeout(t.Context(), time.Second)
	defer stop()
	if got, err := store.GetKey(readCtx, "secret/a"); err != nil || got != "value-a" {
		t.Errorf("GetKey = %q, %v while another process holds the lock, want the value", got, err)
	}
}
