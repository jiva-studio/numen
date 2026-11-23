package fetchfile_test

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/fetchfile"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/pull"
)

const (
	guidePath = "Books/Guide.pdf"
	rangeLen  = 5
)

func TestFetchFile_PlacesTheWholeFileAndListsItNoMore(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	want := w.WriteGuide(t, guidePath)

	if err := w.Fetch(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}

	if got, ok := w.Phone.ReadNote(synctest.PhoneVault, guidePath); !ok || got != want {
		t.Errorf("file of %d bytes (there: %v), want the 5 MiB of the computer", len(got), ok)
	}
	if w.Phone.IsAbsent(t, synctest.PhoneVault, guidePath) {
		t.Error("the file is still listed as absent")
	}
}

func TestFetchFile_LeavesNothingForThePassToSend(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	before := len(w.Phone.ReadChanges(t, w.Phone.GetRemoteID(t, synctest.PhoneVault)))
	if err := w.Fetch(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}
	w.Clock.Advance(time.Minute)

	w.PhoneSync(t)

	after := w.Phone.ReadChanges(t, w.Phone.GetRemoteID(t, synctest.PhoneVault))
	if len(after) != before {
		t.Errorf("the server holds %d revisions after the pass, want the %d it held: a fill is no change", len(after), before)
	}
}

func TestFetchFile_AsksForTheFileInRanges(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	before := w.Server.CountBlobGets()

	if err := w.Fetch(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}

	if got := w.Server.CountBlobGets() - before; got < rangeLen {
		t.Errorf("the server took %d blob requests, want at least %d, one for each range of %d bytes", got, rangeLen, fetchfile.RangeSize)
	}
}

func TestFetchFile_ResumesWhereAnInterruptedDownloadStopped(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	w.Server.FailBlobGetsAfter(2)
	if err := w.Fetch(t.Context(), guidePath); err == nil {
		t.Fatal("a download that lost the server succeeded")
	}
	if !w.Phone.IsAbsent(t, synctest.PhoneVault, guidePath) {
		t.Fatal("a partial download is a file in the vault")
	}
	w.Server.FailBlobGetsAfter(-1)
	before := w.Server.CountBlobGets()

	if err := w.Fetch(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}

	if got := w.Server.CountBlobGets() - before; got != rangeLen-2 {
		t.Errorf("the second try took %d blob requests, want the %d ranges the first did not get", got, rangeLen-2)
	}
	if _, ok := w.Phone.ReadNote(synctest.PhoneVault, guidePath); !ok {
		t.Error("the file is not in the vault after the resumed download")
	}
}

func TestFetchFile_PlacesNothingWhenTheBytesAreDamaged(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	w.Server.CorruptBlobs(true)

	err := w.Fetch(t.Context(), guidePath)

	if !errors.Is(err, pull.ErrCorruptBlob) {
		t.Errorf("error = %v, want ErrCorruptBlob", err)
	}
	if _, ok := w.Phone.ReadNote(synctest.PhoneVault, guidePath); ok {
		t.Error("a file was placed from damaged bytes")
	}
	if !w.Phone.IsAbsent(t, synctest.PhoneVault, guidePath) {
		t.Error("the file stopped being listed after a damaged download")
	}
}

func TestFetchFile_ReturnsUnreachableWhenTheServerIsAway(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	w.Server.SetDown(true)

	err := w.Fetch(t.Context(), guidePath)

	if !errors.Is(err, syncprotocol.ErrUnreachable) {
		t.Errorf("error = %v, want ErrUnreachable", err)
	}
	if _, ok := w.Phone.ReadNote(synctest.PhoneVault, guidePath); ok {
		t.Error("a file was placed while the server was away")
	}
}

func TestFetchFile_RunsOneDownloadForOnePathAtATime(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	before := w.Server.CountBlobGets()
	var wg sync.WaitGroup
	failures := make(chan error, 2)
	for range 2 {
		wg.Go(func() { failures <- w.Fetch(t.Context(), guidePath) })
	}
	wg.Wait()
	close(failures)

	for err := range failures {
		if err != nil {
			t.Errorf("a download failed: %v", err)
		}
	}
	if got := w.Server.CountBlobGets() - before; got > rangeLen+1 {
		t.Errorf("the server took %d blob requests for two taps, want the %d of one download", got, rangeLen)
	}
}

func TestFetchFile_ReturnsAnErrorForAFileThatIsNotListedAsAbsent(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.Phone.WriteNote(synctest.PhoneVault, "Shopping list.md", "milk")

	if err := w.Fetch(t.Context(), "Shopping list.md"); err == nil {
		t.Error("a file that is held was downloaded again")
	}
}
