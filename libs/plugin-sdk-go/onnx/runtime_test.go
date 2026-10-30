package onnx

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

type fakeEngine struct{ libPath string }

func TestOpenRejectsWhenTheHostGaveNoLibrary(t *testing.T) {
	var r Runtime[*fakeEngine]
	_, _, err := r.Open(t.Context(), "", func(libPath string) (*fakeEngine, error) {
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
	var received []string
	r.Register(func(libPath string) error { received = append(received, libPath); return nil })
	load := func(libPath string) (*fakeEngine, error) {
		loads.Add(1)
		return &fakeEngine{libPath: libPath}, nil
	}

	first, libPath, err := r.Open(t.Context(), "/lib/one.so", load)
	if err != nil || libPath != "/lib/one.so" || first.libPath != "/lib/one.so" {
		t.Fatalf("Open() = %v, %q, %v", first, libPath, err)
	}
	second, libPath, err := r.Open(t.Context(), "/lib/two.so", load)
	if err != nil || second != first || libPath != "/lib/one.so" {
		t.Fatalf("second Open() = %v, %q, %v, want the first engine", second, libPath, err)
	}
	if got := loads.Load(); got != 1 {
		t.Errorf("load ran %d times, want 1", got)
	}
	if len(received) != 1 || received[0] != "/lib/one.so" {
		t.Errorf("hooks heard %v, want [/lib/one.so]", received)
	}
	if engine, loadedPath := r.GetLoaded(); engine != first || loadedPath != "/lib/one.so" {
		t.Errorf("GetLoaded() = %v, %q", engine, loadedPath)
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
	if engine, loadedPath := r.GetLoaded(); engine != nil || loadedPath != "" {
		t.Fatalf("GetLoaded() = %v, %q after a failed load", engine, loadedPath)
	}
	engine, _, err := r.Open(t.Context(), "/lib/one.so", func(libPath string) (*fakeEngine, error) {
		return &fakeEngine{libPath: libPath}, nil
	})
	if err != nil || engine == nil {
		t.Fatalf("Open() = %v, %v, want an engine", engine, err)
	}
}

func TestAWaiterPreservesOnItsOwnContextAndTheNextGetsTheEngine(t *testing.T) {
	var r Runtime[*fakeEngine]
	entered := make(chan struct{})
	release := make(chan struct{})
	var loads atomic.Int32
	load := func(libPath string) (*fakeEngine, error) {
		loads.Add(1)
		close(entered)
		<-release
		return &fakeEngine{libPath: libPath}, nil
	}

	isDone := make(chan *fakeEngine)
	go func() {
		engine, _, err := r.Open(t.Context(), "/lib/one.so", load)
		if err != nil {
			t.Errorf("first Open() = %v", err)
		}
		isDone <- engine
	}()
	<-entered

	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, err := r.Open(cancelled, "/lib/one.so", load); !errors.Is(err, context.Canceled) {
		t.Fatalf("waiting Open() = %v, want context.Canceled", err)
	}

	close(release)
	first := <-isDone
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
	libPath := filepath.Join(t.TempDir(), "libonnxruntime.so")
	if r.IsAvailable(libPath) {
		t.Error("a missing file is available")
	}
	if err := os.WriteFile(libPath, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if !r.IsAvailable(libPath) {
		t.Error("a file that is there is not available")
	}
}

func TestHasRegistrationIsTrueOnceAHookIsRegistered(t *testing.T) {
	var r Runtime[*fakeEngine]
	if r.HasRegistration() {
		t.Error("a new runtime has a registration")
	}
	r.Register(func(string) error { return nil })
	if !r.HasRegistration() {
		t.Error("a registered hook is not seen")
	}
}

func TestOpenFailsAndRecordsNothingWhenAHookFails(t *testing.T) {
	var r Runtime[*fakeEngine]
	hookErr := errors.New("bind failed")
	r.Register(func(string) error { return hookErr })
	load := func(libPath string) (*fakeEngine, error) { return &fakeEngine{libPath: libPath}, nil }

	_, _, err := r.Open(t.Context(), "lib", load)

	if !errors.Is(err, hookErr) {
		t.Fatalf("Open error = %v, want %v", err, hookErr)
	}
	if r.IsAvailable("") {
		t.Error("the runtime is available after a failed hook")
	}
	if engine, path := r.GetLoaded(); engine != nil || path != "" {
		t.Errorf("GetLoaded = %v, %q after a failed hook", engine, path)
	}
}
