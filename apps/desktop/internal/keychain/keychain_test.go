package keychain

import (
	"context"
	"errors"
	"runtime"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
	"github.com/zalando/go-keyring"
)

// A key stored is read back, replaced by the next one stored under its name,
// and gone once deleted.
func TestAKeyIsStoredReplacedAndDeleted(t *testing.T) {
	keyring.MockInit()
	store := Store{}
	ctx := t.Context()

	if err := store.SetKey(ctx, "md.numen.agent/apiKey", "value-one"); err != nil {
		t.Fatal(err)
	}
	if got, err := store.GetKey(ctx, "md.numen.agent/apiKey"); err != nil || got != "value-one" {
		t.Fatalf("read %q, %v", got, err)
	}
	if err := store.SetKey(ctx, "md.numen.agent/apiKey", "value-two"); err != nil {
		t.Fatal(err)
	}
	if got, err := store.GetKey(ctx, "md.numen.agent/apiKey"); err != nil || got != "value-two" {
		t.Errorf("read %q after a replace, want value-two", got)
	}
	if err := store.DeleteKey(ctx, "md.numen.agent/apiKey"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetKey(ctx, "md.numen.agent/apiKey"); !errors.Is(err, plugin.ErrNoKey) {
		t.Errorf("a deleted key read as %v, want port.ErrNoKey", err)
	}
}

// A key that was never stored is port.ErrNoKey, to read and to delete.
func TestAKeyNeverStoredIsNoKey(t *testing.T) {
	keyring.MockInit()
	store := Store{}
	if _, err := store.GetKey(t.Context(), "absent"); !errors.Is(err, plugin.ErrNoKey) {
		t.Errorf("read: error = %v, want port.ErrNoKey", err)
	}
	if err := store.DeleteKey(t.Context(), "absent"); !errors.Is(err, plugin.ErrNoKey) {
		t.Errorf("delete: error = %v, want port.ErrNoKey", err)
	}
}

// Keys under two names stay apart.
func TestKeysBelowTwoNamesStayApart(t *testing.T) {
	keyring.MockInit()
	store := Store{}
	for name, key := range map[string]string{"a/apiKey": "value-a", "b/apiKey": "value-b"} {
		if err := store.SetKey(t.Context(), name, key); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := store.GetKey(t.Context(), "a/apiKey"); err != nil || got != "value-a" {
		t.Errorf("a/apiKey reads %q", got)
	}
}

// A machine whose keychain cannot be reached fails every call with
// port.ErrNoKeyStore and reports why; the key is stored nowhere else.
func TestAMachineWithoutAKeychainFailsWithANamedError(t *testing.T) {
	reason := errors.New("the secret service is not running")
	keyring.MockInitWithError(reason)
	store := Store{}
	ctx := t.Context()

	for name, call := range map[string]func() error{
		"set": func() error { return store.SetKey(ctx, "n", "value-secret") },
		"get": func() error { _, err := store.GetKey(ctx, "n"); return err },
		"del": func() error { return store.DeleteKey(ctx, "n") },
	} {
		err := call()
		if !errors.Is(err, plugin.ErrNoKeyStore) || !errors.Is(err, reason) {
			t.Errorf("%s: error = %v, want port.ErrNoKeyStore carrying %v", name, err, reason)
		}
	}

	keyring.MockInit()
	if _, err := store.GetKey(ctx, "n"); !errors.Is(err, plugin.ErrNoKey) {
		t.Errorf("a key set while the keychain was down was kept: %v", err)
	}
}

// A name is required, and a canceled context touches nothing.
func TestAKeyNeedsANameAndALiveContext(t *testing.T) {
	keyring.MockInit()
	store := Store{}
	if err := store.SetKey(t.Context(), "", "value"); !errors.Is(err, errEmptyName) {
		t.Errorf("a key without a name was stored: %v", err)
	}
	if _, err := store.GetKey(t.Context(), ""); !errors.Is(err, errEmptyName) {
		t.Errorf("a key without a name was read: %v", err)
	}
	if err := store.DeleteKey(t.Context(), ""); !errors.Is(err, errEmptyName) {
		t.Errorf("a key without a name was deleted: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := checkName(ctx, "n"); !errors.Is(err, context.Canceled) {
		t.Errorf("checkName with canceled context = %v, want context.Canceled", err)
	}
	if err := store.SetKey(ctx, "", "value"); !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
	if err := store.SetKey(ctx, "n", "value"); !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
	if _, err := store.GetKey(t.Context(), "n"); !errors.Is(err, plugin.ErrNoKey) {
		t.Errorf("a canceled set stored a key: %v", err)
	}
}

// A call the context ends on returns at once with the context's error, though
// the keychain has not answered.
func TestACanceledContextEndsAStalledCall(t *testing.T) {
	release := make(chan struct{})
	ctx, cancel := context.WithCancel(t.Context())
	baseline := runtime.NumGoroutine()
	returned := make(chan error, 1)
	go func() {
		_, err := await(ctx, func() (string, error) {
			<-release
			return "", nil
		})
		returned <- err
	}()
	cancel()
	if err := <-returned; !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
	close(release)
	for range 400 {
		if runtime.NumGoroutine() <= baseline {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got := runtime.NumGoroutine(); got > baseline {
		t.Errorf("%d goroutines after the stalled call answered, %d before it", got, baseline)
	}
}
