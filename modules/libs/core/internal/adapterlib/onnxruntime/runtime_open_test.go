package onnxruntime

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// gate is a transport that answers a download only when told to, and counts the
// downloads started.
type gate struct {
	started chan struct{}
	release chan struct{}
	calls   atomic.Int32
	failure error
}

func newGate(failure error) *gate {
	return &gate{started: make(chan struct{}, 8), release: make(chan struct{}), failure: failure}
}

func (g *gate) RoundTrip(request *http.Request) (*http.Response, error) {
	g.calls.Add(1)
	g.started <- struct{}{}
	select {
	case <-g.release:
		return nil, g.failure
	case <-request.Context().Done():
		return nil, request.Context().Err()
	}
}

// useGate sends every download to the gate and leaves the process with no
// runtime settled.
func useGate(t *testing.T, g *gate) {
	t.Helper()
	isolateCache(t)
	t.Setenv("LD_LIBRARY_PATH", t.TempDir())
	if engine, _, _ := loadFirstLibrary(getLibraryCandidates(getSettings())); engine != nil {
		engine.Destroy()
		t.Skip("this machine holds a runtime, and nothing is fetched")
	}
	previous := httpsClient.Transport
	httpsClient.Transport = g
	t.Cleanup(func() { httpsClient.Transport = previous })

	runtimeState.mu.Lock()
	engine, at, opening := runtimeState.engine, runtimeState.at, runtimeState.opening
	runtimeState.engine, runtimeState.at, runtimeState.opening = nil, "", nil
	runtimeState.mu.Unlock()
	t.Cleanup(func() {
		runtimeState.mu.Lock()
		runtimeState.engine, runtimeState.at, runtimeState.opening = engine, at, opening
		runtimeState.mu.Unlock()
	})
}

func downloadSettings(t *testing.T) Settings {
	t.Helper()
	return Settings{Section: "indexing.recognition", Dir: t.TempDir(), ShouldDownload: true}
}

func waitFor(t *testing.T, signal <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for", what)
	}
}

// A machine holding no runtime fetches one, and a download that fails is the
// error of the call, with its cause kept.
func TestOpenReportsAFailedDownloadAndKeepsItsCause(t *testing.T) {
	cause := errors.New("network down")
	g := newGate(cause)
	useGate(t, g)
	close(g.release)

	engine, at, err := Open(t.Context(), downloadSettings(t))
	if engine != nil || at != "" {
		t.Errorf("Open = %v, %q; want nothing", engine, at)
	}
	if !errors.Is(err, cause) {
		t.Errorf("error = %v, want one wrapping %v", err, cause)
	}
	if g.calls.Load() != 1 {
		t.Errorf("%d downloads, want 1", g.calls.Load())
	}
}

// A runtime named in the settings is opened as named, and a name that does not
// open is refused with the name and the cause.
func TestOpenRefusesANamedRuntimeThatDoesNotOpen(t *testing.T) {
	isolateCache(t)
	named := filepath.Join(t.TempDir(), "libonnxruntime-missing")

	engine, at, err := Open(t.Context(), Settings{Section: "indexing.recognition", Runtime: named})
	if engine != nil || at != "" || err == nil {
		t.Fatalf("Open = %v, %q, %v; want an error", engine, at, err)
	}
	if want := "the onnx runtime " + named; !strings.HasPrefix(err.Error(), want) {
		t.Errorf("error = %q, want it to begin %q", err, want)
	}
	if errors.Unwrap(err) == nil {
		t.Error("the cause of the refusal was not kept")
	}
}

// One caller opens the runtime and the rest wait on it, each against its own
// context; when the opener is done the waiters are released.
func TestCallersWaitOnTheOneOpeningTheRuntime(t *testing.T) {
	cause := errors.New("network down")
	g := newGate(cause)
	useGate(t, g)

	opener := make(chan error, 1)
	go func() {
		_, _, err := Open(t.Context(), downloadSettings(t))
		opener <- err
	}()
	waitFor(t, g.started, "the first download")

	impatient, stop := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer stop()
	if _, _, err := Open(impatient, downloadSettings(t)); !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("a caller that gave up was told %v", err)
	}
	if g.calls.Load() != 1 {
		t.Fatalf("%d downloads while one caller was opening, want 1", g.calls.Load())
	}

	var waiter sync.WaitGroup
	waited := make(chan error, 1)
	waiter.Add(1)
	go func() {
		defer waiter.Done()
		ctx, stopWaiting := context.WithTimeout(t.Context(), 20*time.Second)
		defer stopWaiting()
		_, _, err := Open(ctx, downloadSettings(t))
		waited <- err
	}()
	time.Sleep(100 * time.Millisecond)
	if g.calls.Load() != 1 {
		t.Fatalf("%d downloads while one caller was opening, want 1", g.calls.Load())
	}

	close(g.release)
	select {
	case err := <-opener:
		if !errors.Is(err, cause) {
			t.Errorf("the opener was told %v, want %v", err, cause)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("the opener did not return")
	}
	select {
	case err := <-waited:
		if !errors.Is(err, cause) {
			t.Errorf("the waiter was told %v, want its own attempt's %v", err, cause)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("a waiter stayed held after the opener was done")
	}
	waiter.Wait()
	if g.calls.Load() != 2 {
		t.Errorf("%d downloads, want 1 by the opener and 1 by the waiter", g.calls.Load())
	}
}

// Every library that makes an engine of its own is registered, and is handed the
// library when the runtime is settled.
func TestKeepHandsTheRuntimeToEveryRegisteredLibrary(t *testing.T) {
	runtimeState.mu.Lock()
	registered := len(runtimeState.also)
	engine, at, stood := runtimeState.engine, runtimeState.at, runtimeState.also
	runtimeState.mu.Unlock()
	wasHere := isLoaded.Load()
	t.Cleanup(func() {
		runtimeState.mu.Lock()
		runtimeState.engine, runtimeState.at, runtimeState.also = engine, at, stood
		runtimeState.mu.Unlock()
		isLoaded.Store(wasHere)
	})
	if registered == 0 {
		t.Error("the graph optimization setting is not registered to be made on the runtime")
	}

	var got []string
	Register(func(at string) { got = append(got, at) })
	runtimeState.mu.Lock()
	recordRuntime(nil, "/runtime/libonnxruntime")
	runtimeState.mu.Unlock()

	if len(got) != 1 || got[0] != "/runtime/libonnxruntime" {
		t.Errorf("registered library was handed %v", got)
	}
	if !isLoaded.Load() {
		t.Error("a settled runtime is not here")
	}
}

func TestIsHereKnowsWhatTheMachineHolds(t *testing.T) {
	isolateCache(t)
	libraries := t.TempDir()
	t.Setenv("LD_LIBRARY_PATH", libraries)
	settings := Settings{Section: "indexing.recognition", Dir: t.TempDir()}
	wasHere := isLoaded.Swap(false)
	t.Cleanup(func() { isLoaded.Store(wasHere) })

	for _, at := range getLibraryCandidates(settings) {
		if filepath.IsAbs(at) {
			t.Skip("this machine holds a runtime file")
		}
	}
	if IsAvailable(settings) {
		t.Error("a machine with no runtime file is here")
	}

	if err := os.WriteFile(filepath.Join(libraries, runtimeName()), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !IsAvailable(settings) {
		t.Error("a machine holding a runtime file is not here")
	}
}

func TestIsHereChecksTheFileTheSettingsName(t *testing.T) {
	isolateCache(t)
	wasHere := isLoaded.Swap(false)
	t.Cleanup(func() { isLoaded.Store(wasHere) })
	named := filepath.Join(t.TempDir(), "libonnxruntime-named")

	if IsAvailable(Settings{Runtime: named}) {
		t.Error("a name with no file is here")
	}
	if err := os.WriteFile(named, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if !IsAvailable(Settings{Runtime: named}) {
		t.Error("a name with a file is not here")
	}
}

func TestIsHereIsTrueOnceTheProcessHasItsRuntime(t *testing.T) {
	isolateCache(t)
	t.Setenv("LD_LIBRARY_PATH", t.TempDir())
	wasHere := isLoaded.Swap(true)
	t.Cleanup(func() { isLoaded.Store(wasHere) })

	if !IsAvailable(Settings{Runtime: filepath.Join(t.TempDir(), "nothing")}) {
		t.Error("a process holding its runtime is not here")
	}
}

func TestAReleaseIsFetchedFromTheAddressItsArchiveIsPublishedAt(t *testing.T) {
	for platform, found := range releases {
		address := found.address()
		if !IsAddress(address) || address != from+found.archive {
			t.Errorf("%s is fetched from %s", platform, address)
		}
		if filepath.Base(address) != found.archive {
			t.Errorf("%s: %s does not end in %s", platform, address, found.archive)
		}
	}
}
