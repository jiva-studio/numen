package fetchfile_test

import (
	"context"
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/fetchfile"
)

func TestFetchFile_GetProgress_AnotherVaultWithTheSamePath_IsNotRunning(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	deps := w.Phone.FetchDeps()
	gate := &gatedBlobs{BlobReader: deps.Blobs, allowed: 1, reached: make(chan struct{}), release: make(chan struct{})}
	deps.Blobs = gate
	use := fetchfile.NewFetchFile(deps)
	finished := make(chan error, 1)
	go func() { finished <- use.Execute(t.Context(), synctest.PhoneVault, guidePath) }()
	<-gate.reached

	_, isOtherRunning := use.GetProgress(domain.VaultID("another vault"), guidePath)
	_, isOwnRunning := use.GetProgress(synctest.PhoneVault, guidePath)

	close(gate.release)
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	if isOtherRunning {
		t.Error("the download of one vault shows as the download of another vault with the same path")
	}
	if !isOwnRunning {
		t.Error("the download does not show for its own vault")
	}
}

func TestFetchFile_Close_RunningDownload_EndsItBeforeReturning(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	deps := w.Phone.FetchDeps()
	gate := &gatedBlobs{BlobReader: deps.Blobs, allowed: 1, reached: make(chan struct{}), release: make(chan struct{})}
	deps.Blobs = gate
	use := fetchfile.NewFetchFile(deps)
	finished := make(chan error, 1)
	go func() { finished <- use.Execute(t.Context(), synctest.PhoneVault, guidePath) }()
	<-gate.reached

	use.Close()

	if _, isRunning := use.GetProgress(synctest.PhoneVault, guidePath); isRunning {
		t.Error("a download still runs after Close returned")
	}
	if err := <-finished; !errors.Is(err, context.Canceled) {
		t.Errorf("the call of the stopped download returned %v, want context.Canceled", err)
	}
	if !w.Phone.IsAbsent(t, synctest.PhoneVault, guidePath) {
		t.Error("a stopped download placed the file")
	}
}

func TestFetchFile_Close_NewCall_ReturnsErrClosed(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	use := fetchfile.NewFetchFile(w.Phone.FetchDeps())
	use.Close()

	err := use.Execute(t.Context(), synctest.PhoneVault, guidePath)

	if !errors.Is(err, fetchfile.ErrClosed) {
		t.Errorf("error = %v, want ErrClosed", err)
	}
}
