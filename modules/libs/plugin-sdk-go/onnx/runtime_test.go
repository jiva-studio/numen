package onnx

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

type fakeEngine struct{ at string }

func TestOpenRefusesWhenTheHostGaveNoLibrary(t *testing.T) {
	var r Runtime[*fakeEngine]
	_, _, err := r.Open(t.Context(), "", func(at string) (*fakeEngine, error) {
		t.Fatal("load ran with no library")
		return nil, nil
	})
	if !errors.Is(err, ErrNoLibrary) {
		t.Fatalf("Open() = %v, want ErrNoLibrary", err)
	}
	if r.IsAvailable("") {
		t.Error("a runtime that failed to open is available")
	}
}

func TestOpenLoadsOnceAndRunsEveryHookOnce(t *testing.T) {
	var r Runtime[*fakeEngine]
	var loads atomic.Int32
	var heard []string
	r.Register(func(at string) { heard = append(heard, at) })
	load := func(at string) (*fakeEngine, error) {
		loads.Add(1)
		return &fakeEngine{at: at}, nil
	}

	first, at, err := r.Open(t.Context(), "/lib/one.so", load)
	if err != nil || at != "/lib/one.so" || first.at != "/lib/one.so" {
		t.Fatalf("Open() = %v, %q, %v", first, at, err)
	}
	second, at, err := r.Open(t.Context(), "/lib/two.so", load)
	if err != nil || second != first || at != "/lib/one.so" {
		t.Fatalf("second Open() = %v, %q, %v, want the first engine", second, at, err)
	}
	if got := loads.Load(); got != 1 {
		t.Errorf("load ran %d times, want 1", got)
	}
	if len(heard) != 1 || heard[0] != "/lib/one.so" {
		t.Errorf("hooks heard %v, want [/lib/one.so]", heard)
	}
	if engine, at := r.GetLoaded(); engine != first || at != "/lib/one.so" {
		t.Errorf("GetLoaded() = %v, %q", engine, at)
	}
	if !r.IsAvailable("") {
		t.Error("an open runtime is not available")
	}
}

func TestOpenTriesAgainAfterALoadFails(t *testing.T) {
	var r Runtime[*fakeEngine]
	broken := errors.New("broken library")
	if _, _, err := r.Open(t.Context(), "/lib/one.so", func(string) (*fakeEngine, error) {
		return nil, broken
	}); !errors.Is(err, broken) {
		t.Fatalf("Open() = %v, want %v", err, broken)
	}
	if engine, at := r.GetLoaded(); engine != nil || at != "" {
		t.Fatalf("GetLoaded() = %v, %q after a failed load", engine, at)
	}
	engine, _, err := r.Open(t.Context(), "/lib/one.so", func(at string) (*fakeEngine, error) {
		return &fakeEngine{at: at}, nil
	})
	if err != nil || engine == nil {
		t.Fatalf("Open() = %v, %v, want an engine", engine, err)
	}
}

func TestAWaiterLeavesOnItsOwnContextAndTheNextGetsTheEngine(t *testing.T) {
	var r Runtime[*fakeEngine]
	entered := make(chan struct{})
	release := make(chan struct{})
	var loads atomic.Int32
	load := func(at string) (*fakeEngine, error) {
		loads.Add(1)
		close(entered)
		<-release
		return &fakeEngine{at: at}, nil
	}

	done := make(chan *fakeEngine)
	go func() {
		engine, _, err := r.Open(context.Background(), "/lib/one.so", load)
		if err != nil {
			t.Errorf("first Open() = %v", err)
		}
		done <- engine
	}()
	<-entered

	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, err := r.Open(cancelled, "/lib/one.so", load); !errors.Is(err, context.Canceled) {
		t.Fatalf("waiting Open() = %v, want context.Canceled", err)
	}

	close(release)
	first := <-done
	engine, _, err := r.Open(t.Context(), "/lib/one.so", load)
	if err != nil || engine != first {
		t.Fatalf("Open() = %v, %v, want the first engine", engine, err)
	}
	if got := loads.Load(); got != 1 {
		t.Errorf("load ran %d times, want 1", got)
	}
}

func TestALibraryIsAvailableWhereTheFileIs(t *testing.T) {
	var r Runtime[*fakeEngine]
	at := filepath.Join(t.TempDir(), "libonnxruntime.so")
	if r.IsAvailable(at) {
		t.Error("a missing file is available")
	}
	if err := os.WriteFile(at, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if !r.IsAvailable(at) {
		t.Error("a file that is there is not available")
	}
}

func TestHasRegistrationIsTrueOnceAHookIsRegistered(t *testing.T) {
	var r Runtime[*fakeEngine]
	if r.HasRegistration() {
		t.Error("a new runtime has a registration")
	}
	r.Register(func(string) {})
	if !r.HasRegistration() {
		t.Error("a registered hook is not seen")
	}
}
