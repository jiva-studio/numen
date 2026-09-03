package onnxruntime

import (
	"strings"
	"testing"
	"time"

	ort "github.com/getcharzp/onnxruntime_purego"
)

// asked is what a reading or a transcription looks for its runtime by, with
// nothing fetched.
func asked() Settings { return Settings{Section: "recognition"} }

// A file that is there is not a library that loads, and every way of getting one
// ends in the same question: does it open.
func TestTheRuntimeThisMachineHoldsOpens(t *testing.T) {
	engine, at, refused := load(candidates(asked()))
	if engine == nil {
		t.Skip("this machine holds none:", refused)
	}
	defer engine.Destroy()
	t.Log("opened", at, engine.GetVersion())
}

func TestTheRuntimeFetchedForThisPlatformOpens(t *testing.T) {
	found, err := findRelease(asked())
	if err != nil {
		t.Skip(err)
	}
	archive, err := Fetched(t.Context(), asked(), found.address())
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
	first, at, err := Open(t.Context(), asked())
	if err != nil {
		t.Skipf("no onnx runtime on this machine: %v", err)
	}
	second, again, err := Open(t.Context(), asked())
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

// Here is asked by the tool that starts a reading, right after it has started
// one. It answers while that reading is opening the runtime.
func TestHereAnswersWhileTheRuntimeIsBeingOpened(t *testing.T) {
	held.Lock()
	defer held.Unlock()

	said := make(chan bool, 1)
	go func() { said <- Here(asked()) }()

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
