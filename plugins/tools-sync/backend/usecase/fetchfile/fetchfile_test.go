package fetchfile_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
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

func TestFetchFile_SecondTapStillGetsTheFileWhenTheFirstTapIsCanceled(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	want := w.WriteGuide(t, guidePath)
	before := w.Server.CountBlobGets()
	first, cancelFirst := context.WithCancel(t.Context())
	t.Cleanup(cancelFirst)
	var wg sync.WaitGroup
	wg.Go(func() {
		if err := w.Fetch(first, guidePath); err != nil && !errors.Is(err, context.Canceled) {
			t.Errorf("the first tap failed with %v, want it canceled or done", err)
		}
	})
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
		t.Errorf("a download after a canceled one failed: %v", err)
	}
}

// gatedBlobs lets a download take the given number of ranges and holds the next one until released.
type gatedBlobs struct {
	ports.BlobReader
	allowed int
	reached chan struct{}
	release chan struct{}
	mu      sync.Mutex
	opened  int
}

func (g *gatedBlobs) OpenBlob(ctx context.Context, vault, blob []byte, from int64) (io.ReadCloser, error) {
	g.mu.Lock()
	g.opened++
	isHeld := g.opened == g.allowed+1
	g.mu.Unlock()
	if isHeld {
		close(g.reached)
		select {
		case <-g.release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return g.BlobReader.OpenBlob(ctx, vault, blob, from)
}

func TestFetchFile_GetProgress_RunningDownload_ReturnsTheBytesOfTheRangesThatCameAndTheSizeOfTheFile(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	deps := w.Phone.FetchDeps()
	gate := &gatedBlobs{BlobReader: deps.Blobs, allowed: 2, reached: make(chan struct{}), release: make(chan struct{})}
	deps.Blobs = gate
	use := fetchfile.NewFetchFile(deps)
	finished := make(chan error, 1)
	go func() { finished <- use.Execute(t.Context(), synctest.PhoneVault, guidePath) }()
	<-gate.reached

	got, isRunning := use.GetProgress(guidePath)

	close(gate.release)
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	want := fetchfile.Progress{Vault: synctest.PhoneVault, Bytes: 2 * fetchfile.RangeSize, Total: 5 << 20, IsStaged: false}
	if !isRunning || got != want {
		t.Errorf("progress = %+v (running: %v), want %+v", got, isRunning, want)
	}
}

func TestFetchFile_GetProgress_NoDownloadRuns_ReturnsNotRunning(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	use := fetchfile.NewFetchFile(w.Phone.FetchDeps())

	if _, isRunning := use.GetProgress(guidePath); isRunning {
		t.Error("a download runs before any was started")
	}
	if err := use.Execute(t.Context(), synctest.PhoneVault, guidePath); err != nil {
		t.Fatal(err)
	}
	if _, isRunning := use.GetProgress(guidePath); isRunning {
		t.Error("a download runs after it ended")
	}
}

func TestFetchFile_GetProgress_ResumedDownload_StartsAtTheBytesOfThePart(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	w.Server.FailBlobGetsAfter(2)
	if err := w.Fetch(t.Context(), guidePath); err == nil {
		t.Fatal("a download that lost the server succeeded")
	}
	w.Server.FailBlobGetsAfter(-1)
	deps := w.Phone.FetchDeps()
	gate := &gatedBlobs{BlobReader: deps.Blobs, allowed: 0, reached: make(chan struct{}), release: make(chan struct{})}
	deps.Blobs = gate
	use := fetchfile.NewFetchFile(deps)
	finished := make(chan error, 1)
	go func() { finished <- use.Execute(t.Context(), synctest.PhoneVault, guidePath) }()
	<-gate.reached

	got, isRunning := use.GetProgress(guidePath)

	close(gate.release)
	if err := <-finished; err != nil {
		t.Fatal(err)
	}
	if !isRunning || got.Bytes != 2*fetchfile.RangeSize {
		t.Errorf("progress = %+v (running: %v), want the 2 ranges of the first try", got, isRunning)
	}
}
