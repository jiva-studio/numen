package fetchfile_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/chaincourier"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/fetchfile"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/pull"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/rotatevaultkeys"
	"github.com/jiva-studio/numen/tools/syncserver/fakeserver"
)

const guidePath = "Books/Guide.pdf"

func countRanges(t *testing.T, w *synctest.PhoneWorld) int {
	t.Helper()
	stored := w.Phone.ReadChanges(t, w.Phone.GetRemoteID(t, synctest.PhoneVault))
	if len(stored) != 1 {
		t.Fatalf("the server holds %d files, want the guide only", len(stored))
	}
	return int((int64(stored[0].GetRevision().GetBlobSize()) + fetchfile.RangeSize - 1) / fetchfile.RangeSize)
}

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

func TestFetchFile_ProducesNoChangeForThePassToSend(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	before := len(w.Phone.ReadChanges(t, w.Phone.GetRemoteID(t, synctest.PhoneVault)))
	if err := w.Fetch(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}
	w.Clock.Advance(time.Minute)

	w.SyncPhone(t)

	after := w.Phone.ReadChanges(t, w.Phone.GetRemoteID(t, synctest.PhoneVault))
	if len(after) != before {
		t.Errorf("the server holds %d revisions after the pass, want the %d it held: a fill is no change", len(after), before)
	}
}

func TestFetchFile_AsksForTheFileInRanges(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	before := w.Server.CountBlobGets()
	want := countRanges(t, w)

	if err := w.Fetch(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}

	if got := w.Server.CountBlobGets() - before; got != want {
		t.Errorf("the server took %d blob requests, want %d, one for each range of %d bytes", got, want, fetchfile.RangeSize)
	}
}

func TestFetchFile_ResumesWhereAnInterruptedDownloadStopped(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	want := countRanges(t, w)
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

	if got := w.Server.CountBlobGets() - before; got != want-2 {
		t.Errorf("the second try took %d blob requests, want the %d ranges the first did not get", got, want-2)
	}
	if _, ok := w.Phone.ReadNote(synctest.PhoneVault, guidePath); !ok {
		t.Error("the file is not in the vault after the resumed download")
	}
}

func TestFetchFile_PlacesNothingWhenTheBytesAreDamaged(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	w.Server.SetBlobCorruption(true)

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
	want := countRanges(t, w)
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
	if got := w.Server.CountBlobGets() - before; got != want {
		t.Errorf("the server took %d blob requests for two taps, want the %d of one download", got, want)
	}
}

func TestFetchFile_ReturnsAnErrorForAFileThatIsNotListedAsAbsent(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.Phone.WriteNote(synctest.PhoneVault, "Shopping list.md", "milk")

	if err := w.Fetch(t.Context(), "Shopping list.md"); err == nil {
		t.Error("a file that is held was downloaded again")
	}
}

func listFetchFiles(t *testing.T, w *synctest.PhoneWorld) []string {
	t.Helper()
	files, err := w.Phone.Host.Store(synctest.PhoneVault).List(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, file := range files {
		if strings.HasPrefix(file.Name, "fetch-") {
			names = append(names, file.Name)
		}
	}
	return names
}

func TestFetchFile_PlacedFile_StoreHasNoFetchFiles(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)

	if err := w.Fetch(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}

	if left := listFetchFiles(t, w); len(left) != 0 {
		t.Errorf("the store keeps %v after the placement", left)
	}
}

func TestFetchFile_DamagedBytes_StoreHasNoFetchFiles(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	w.Server.SetBlobCorruption(true)

	if err := w.Fetch(t.Context(), guidePath); err == nil {
		t.Fatal("damaged bytes were placed")
	}

	if left := listFetchFiles(t, w); len(left) != 0 {
		t.Errorf("the store keeps %v after the damaged download", left)
	}
}

func TestFetchFile_PlacesAFileSealedAtALaterEpochOfTheVault(t *testing.T) {
	clock := synctest.NewClock(synctest.GetStart())
	server := synctest.NewServer(t, clock, fakeserver.Config{})
	words := synctest.NewWords(t)
	computer := synctest.NewFirstComputer(t, server, clock, synctest.ComputerVault, words)
	computer.Connect(t, synctest.ComputerVault)
	if _, err := chaincourier.NewAdopt(computer.ChainDeps()).Execute(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := computer.Fake.AdvanceEpoch(); err != nil {
		t.Fatal(err)
	}
	if published, err := chaincourier.NewPublish(computer.ChainDeps()).Execute(t.Context()); err != nil || !published {
		t.Fatalf("published = %v, error = %v, want the chain of the epoch published", published, err)
	}
	if rotated, err := rotatevaultkeys.NewRotateVaultKeys(computer.RotateDeps()).Execute(t.Context()); err != nil || rotated != 1 {
		t.Fatalf("rotated = %d, error = %v, want the vault rotated", rotated, err)
	}
	want := computer.WriteFileOfSize(synctest.ComputerVault, guidePath, 5<<20)
	clock.Advance(time.Minute)
	if err := computer.Sync(t.Context(), synctest.ComputerVault); err != nil {
		t.Fatal(err)
	}
	phone := synctest.NewPhone(t, server, clock, synctest.PhoneVault, words)
	phone.ConnectExisting(t, synctest.PhoneVault, computer.Link(t, synctest.ComputerVault).RemoteVaultID)
	if err := phone.Sync(t.Context(), synctest.PhoneVault); err != nil {
		t.Fatal(err)
	}

	if err := fetchfile.NewFetchFile(phone.FetchDeps()).Execute(t.Context(), synctest.PhoneVault, guidePath); err != nil {
		t.Fatal(err)
	}

	if got, ok := phone.ReadNote(synctest.PhoneVault, guidePath); !ok || got != want {
		t.Errorf("file of %d bytes (there: %v), want the 5 MiB of the computer", len(got), ok)
	}
}

func TestFetchFile_RefusesAHeadWithABadSignature(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	deps := w.Phone.FetchDeps()
	deps.Revisions = synctest.ForgedSignatures{Revisions: deps.Revisions}

	err := fetchfile.NewFetchFile(deps).Execute(t.Context(), synctest.PhoneVault, guidePath)

	if !errors.Is(err, note.ErrUnverified) {
		t.Errorf("error = %v, want ErrUnverified", err)
	}
	if _, ok := w.Phone.ReadNote(synctest.PhoneVault, guidePath); ok {
		t.Error("a file was placed from a revision with a bad signature")
	}
}

func TestFetchFile_SecondTapStillGetsTheFileWhenTheFirstTapIsCancelled(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	want := w.WriteGuide(t, guidePath)
	before := w.Server.CountBlobGets()
	first, cancelFirst := context.WithCancel(t.Context())
	t.Cleanup(cancelFirst)
	var wg sync.WaitGroup
	wg.Go(func() { _ = w.Fetch(first, guidePath) })
	for w.Server.CountBlobGets() == before {
		time.Sleep(time.Millisecond)
	}
	second := make(chan error, 1)
	wg.Go(func() { second <- w.Fetch(t.Context(), guidePath) })
	cancelFirst()
	wg.Wait()

	if err := <-second; err != nil {
		t.Errorf("the second tap failed with %v, want the file", err)
	}
	if got, ok := w.Phone.ReadNote(synctest.PhoneVault, guidePath); !ok || got != want {
		t.Errorf("file of %d bytes (there: %v), want the 5 MiB of the computer", len(got), ok)
	}
}

func TestFetchFile_EndsTheDownloadWhenNoCallWaitsForIt(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	gone, cancel := context.WithCancel(t.Context())
	cancel()

	err := w.Fetch(gone, guidePath)

	if !errors.Is(err, context.Canceled) {
		t.Errorf("error = %v, want context.Canceled", err)
	}
	if err := w.Fetch(t.Context(), guidePath); err != nil {
		t.Errorf("a download after a cancelled one failed: %v", err)
	}
}
