package onnxruntime

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	ort "github.com/getcharzp/onnxruntime_purego"
)

// getSettings is what a reading or a transcription looks for its runtime by,
// with nothing fetched.
func getSettings() Settings { return Settings{Section: "indexing.recognition"} }

// A file that is there is not a library that loads, and every way of getting one
// ends in the same question: does it open.
func TestTheRuntimeThisMachineHoldsOpens(t *testing.T) {
	engine, at, refused := load(candidates(getSettings()))
	if engine == nil {
		t.Skip("this machine holds none:", refused)
	}
	defer engine.Destroy()
	t.Log("opened", at, engine.GetVersion())
}

func TestTheRuntimeFetchedForThisPlatformOpens(t *testing.T) {
	found, err := findRelease(getSettings())
	if err != nil {
		t.Skip(err)
	}
	archive, err := Fetch(t.Context(), getSettings(), found.address())
	if err != nil {
		t.Skip("nothing fetched on this machine:", err)
	}
	at, err := unpack(archive, runtimeName(), found)
	if err != nil {
		t.Fatal(err)
	}
	support()
	engine, err := ort.NewEngine(at)
	if err != nil {
		t.Fatal(err)
	}
	defer engine.Destroy()
	t.Log("opened", at, engine.GetVersion())
}

// The runtime is opened once for the life of the process: every tensor is made
// through the memory it holds, whichever reading or transcription made it.
func TestTheRuntimeIsOpenedOnceForTheProcess(t *testing.T) {
	first, at, err := Open(t.Context(), getSettings())
	if err != nil {
		t.Skipf("no onnx runtime on this machine: %v", err)
	}
	second, again, err := Open(t.Context(), getSettings())
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Error("a second runtime was opened")
	}
	if at != again {
		t.Errorf("opened from %s and then from %s", at, again)
	}
}

// halfOpen leaves the runtime as it stands while one caller is fetching one:
// nothing settled, and an opening in flight. It hands back what puts the process
// back as it found it, runtime and all, so a test earlier in this package having
// opened one settles nothing here.
func halfOpen(t *testing.T) func() {
	t.Helper()
	held.mu.Lock()
	engine, at := held.engine, held.at
	stand := make(chan struct{})
	held.engine, held.at, held.opening = nil, "", stand
	held.mu.Unlock()
	return func() {
		held.mu.Lock()
		held.engine, held.at, held.opening = engine, at, nil
		held.mu.Unlock()
		close(stand)
	}
}

// A caller that gave up waiting is answered. Opening the runtime is a download
// of minutes, and a caller left standing in a mutex would notice its own context
// only once that download had ended for somebody else.
func TestACallerGivesUpWhileAnotherOpensTheRuntime(t *testing.T) {
	defer halfOpen(t)()

	ctx, stop := context.WithCancel(t.Context())
	gave := make(chan error, 1)
	go func() {
		_, _, err := Open(ctx, getSettings())
		gave <- err
	}()
	stop()

	select {
	case err := <-gave:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("a caller that gave up was told %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("a caller that gave up waited for another caller's runtime")
	}
}

// Here is asked by the tool that starts a reading, right after it has started
// one. It answers while that reading is opening the runtime.
func TestHereAnswersWhileTheRuntimeIsBeingOpened(t *testing.T) {
	defer halfOpen(t)()

	said := make(chan bool, 1)
	go func() { said <- IsHere(getSettings()) }()

	select {
	case <-said:
	case <-time.After(2 * time.Second):
		t.Fatal("asking what is on this machine waited for the runtime to be opened")
	}
}

// Every platform the runtime is published for names an archive and the two sums
// it is checked by, and each archive is the release the binding asks for.
func TestEveryPlatformNamesAnArchiveAndItsSums(t *testing.T) {
	for platform, found := range releases {
		if found.archive == "" || found.sum == "" || found.library == "" {
			t.Errorf("%s: %+v", platform, found)
		}
		if !strings.Contains(found.archive, runtimeVersion) {
			t.Errorf("%s is published as %s, and %s is asked for", platform, found.archive, runtimeVersion)
		}
	}
}

// Only https is somewhere to fetch from.
func TestOnlyAnHttpsAddressIsFetchedFrom(t *testing.T) {
	for _, one := range []struct {
		name string
		is   bool
	}{
		{"https://huggingface.co/model.onnx", true},
		{"http://huggingface.co/model.onnx", false},
		{"file:///tmp/model.onnx", false},
		{"/tmp/model.onnx", false},
	} {
		if IsAddress(one.name) != one.is {
			t.Errorf("%s is fetched from: %v", one.name, IsAddress(one.name))
		}
	}
}
