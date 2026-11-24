package synctest_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/shared/syncprotocol"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/lifecycle"
)

// The tests of this file run the scenarios of features/mobile/sync on real pieces: a computer and a phone with
// their use cases over the fake sync server, the test host of the SDK and a clock set by hand. The network
// stream is the one thing a test replaces with a fake.

const (
	guidePath = "Books/Guide.pdf"
	listPath  = "Shopping list.md"
)

func newLifecycle(w *synctest.PhoneWorld, s *synctest.Stream) *lifecycle.Lifecycle {
	return lifecycle.NewLifecycle(w.Phone.LifecycleDeps(30*time.Second, s, make(chan domain.VaultID)))
}

func suspendAfter(t *testing.T, life *lifecycle.Lifecycle) {
	t.Helper()
	t.Cleanup(func() {
		if err := life.Suspend(t.Context()); err != nil {
			t.Error(err)
		}
	})
}

func hasPhoneNote(w *synctest.PhoneWorld, path, text string) bool {
	got, ok := w.Phone.ReadNote(synctest.PhoneVault, path)
	return ok && got == text
}

func TestAPhoneGetsANoteWithItsText(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.Computer.WriteNote(synctest.ComputerVault, listPath, "milk")
	w.Clock.Advance(time.Minute)
	w.ComputerSync(t)

	w.PhoneSync(t)

	if !hasPhoneNote(w, listPath, "milk") {
		t.Error("the phone has no note with the text milk")
	}
}

func TestAPhoneListsALargeFileAndDoesNotDownloadIt(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)

	if !w.Phone.IsAbsent(t, synctest.PhoneVault, guidePath) {
		t.Errorf("absent files = %+v, want the guide", w.Phone.ListAbsent(t, synctest.PhoneVault))
	}
	if _, ok := w.Phone.ReadNote(synctest.PhoneVault, guidePath); ok {
		t.Error("the phone holds the bytes of a file on tap")
	}
}

func TestAPhoneDropsAFileDeletedOnTheComputerFromItsList(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	w.Computer.Vaults.With(synctest.ComputerVault).Remove(guidePath)
	w.Clock.Advance(time.Minute)
	w.ComputerSync(t)

	w.PhoneSync(t)

	if w.Phone.IsAbsent(t, synctest.PhoneVault, guidePath) {
		t.Error("the phone still lists a deleted file")
	}
}

func TestAnOpenedFileIsDownloaded(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	want := w.WriteGuide(t, guidePath)

	if err := w.Fetch(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}

	if !hasPhoneNote(w, guidePath, want) {
		t.Error("the phone does not hold the guide after it was opened")
	}
}

func TestAFileIsNotDownloadedWhileTheServerIsAway(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	w.Server.SetDown(true)

	err := w.Fetch(t.Context(), guidePath)

	if !errors.Is(err, syncprotocol.ErrUnreachable) {
		t.Errorf("error = %v, want ErrUnreachable", err)
	}
	if _, ok := w.Phone.ReadNote(synctest.PhoneVault, guidePath); ok {
		t.Error("the phone holds a file the server did not send")
	}
}

func TestAPhoneStillListsAFileAfterItsDownloadIsRemoved(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	if err := w.Fetch(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}

	if err := w.DropDownload(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}

	if !w.Phone.IsAbsent(t, synctest.PhoneVault, guidePath) {
		t.Error("the phone does not list the file")
	}
	if _, ok := w.Phone.ReadNote(synctest.PhoneVault, guidePath); ok {
		t.Error("the phone holds the bytes of a removed download")
	}
}

func TestTheVaultKeepsAFileAfterItsDownloadIsRemovedOnThePhone(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	want := w.WriteGuide(t, guidePath)
	if err := w.Fetch(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}
	if err := w.DropDownload(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}
	w.Clock.Advance(time.Minute)

	w.PhoneSync(t)
	w.ComputerSync(t)

	if !hasComputerNote(w, guidePath, want) {
		t.Error("the computer lost the file after the phone removed its download")
	}
	for _, one := range w.Computer.ReadChanges(t, w.Computer.GetRemoteID(t, synctest.ComputerVault)) {
		if one.GetRevision().GetDeleted() {
			t.Error("the server holds a deletion after a removed download")
		}
	}
}

func hasComputerNote(w *synctest.PhoneWorld, path, text string) bool {
	got, ok := w.Computer.ReadNote(synctest.ComputerVault, path)
	return ok && got == text
}

func TestAFileDownloadsAgainAfterItsDownloadIsRemoved(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	want := w.WriteGuide(t, guidePath)
	if err := w.Fetch(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}
	if err := w.DropDownload(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}

	if err := w.Fetch(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}

	if !hasPhoneNote(w, guidePath, want) {
		t.Error("the phone does not hold the file again")
	}
}

func TestDeletingAFileOnThePhoneRemovesItFromTheVault(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	if err := w.Fetch(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}
	w.Phone.Vaults.With(synctest.PhoneVault).Remove(guidePath)
	w.Clock.Advance(time.Minute)

	w.PhoneSync(t)
	w.ComputerSync(t)

	if _, ok := w.Computer.ReadNote(synctest.ComputerVault, guidePath); ok {
		t.Error("the computer still has a file the phone deleted")
	}
}

func TestDeletingAnAbsentFileOnThePhoneRemovesItFromTheVault(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	w.Phone.Vaults.With(synctest.PhoneVault).Remove(guidePath)
	w.Clock.Advance(time.Minute)

	w.PhoneSync(t)
	w.ComputerSync(t)

	if _, ok := w.Computer.ReadNote(synctest.ComputerVault, guidePath); ok {
		t.Error("the computer still has a file the phone deleted")
	}
	if w.Phone.IsAbsent(t, synctest.PhoneVault, guidePath) {
		t.Error("the phone still lists a file the person deleted")
	}
}

func TestANoteCreatedOnTheComputerArrivesWhileTheApplicationIsOpenOnThePhone(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	s := &synctest.Stream{}
	life := newLifecycle(w, s)
	if err := life.Resume(t.Context()); err != nil {
		t.Fatal(err)
	}
	suspendAfter(t, life)
	synctest.Eventually(t, "an open stream", func() bool { return s.CountOpen() == 1 })
	w.Computer.WriteNote(synctest.ComputerVault, listPath, "milk")
	w.Clock.Advance(time.Minute)
	w.ComputerSync(t)

	s.Hint(synctest.PhoneVault)

	synctest.Eventually(t, "the note on the phone", func() bool { return hasPhoneNote(w, listPath, "milk") })
}

func TestANoteCreatedWhileTheApplicationIsClosedArrivesWhenItOpens(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.Computer.WriteNote(synctest.ComputerVault, listPath, "milk")
	w.Clock.Advance(time.Minute)
	w.ComputerSync(t)
	life := newLifecycle(w, &synctest.Stream{})

	if err := life.Resume(t.Context()); err != nil {
		t.Fatal(err)
	}
	suspendAfter(t, life)

	synctest.Eventually(t, "the note on the phone", func() bool { return hasPhoneNote(w, listPath, "milk") })
}

func TestNumenAsksForNothingWhenItIsOffTheScreen(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	s := &synctest.Stream{}
	life := newLifecycle(w, s)
	if err := life.Resume(t.Context()); err != nil {
		t.Fatal(err)
	}
	synctest.Eventually(t, "an open stream", func() bool { return s.CountOpen() == 1 })

	if err := life.Suspend(t.Context()); err != nil {
		t.Fatal(err)
	}

	if s.CountOpen() != 0 {
		t.Errorf("%d streams are open after Numen left the screen, want 0", s.CountOpen())
	}
	asked := w.Server.CountRequests()
	w.Clock.Advance(time.Minute)
	if got := w.Server.CountRequests(); got != asked {
		t.Errorf("the server took %d requests from a phone that left the screen", got-asked)
	}
}

func TestNumenAsksAgainWhenItReturnsToTheScreen(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	s := &synctest.Stream{}
	life := newLifecycle(w, s)
	if err := life.Resume(t.Context()); err != nil {
		t.Fatal(err)
	}
	if err := life.Suspend(t.Context()); err != nil {
		t.Fatal(err)
	}

	if err := life.Resume(t.Context()); err != nil {
		t.Fatal(err)
	}
	suspendAfter(t, life)

	synctest.Eventually(t, "an open stream", func() bool { return s.CountOpen() == 1 })
}

func TestANoteIsSentWhenTheApplicationIsSuspended(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	life := newLifecycle(w, &synctest.Stream{})
	if err := life.Resume(t.Context()); err != nil {
		t.Fatal(err)
	}
	w.Phone.WriteNote(synctest.PhoneVault, listPath, "bread")
	w.Clock.Advance(time.Minute)

	if err := life.Suspend(t.Context()); err != nil {
		t.Fatal(err)
	}

	w.ComputerSync(t)
	if !hasComputerNote(w, listPath, "bread") {
		t.Error("the computer has no note with the text bread")
	}
}

func TestANoteCreatedWithoutTheNetworkIsSentLater(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.Server.SetDown(true)
	w.Phone.WriteNote(synctest.PhoneVault, listPath, "eggs")
	w.Clock.Advance(time.Minute)
	if err := w.Phone.Sync(t.Context(), synctest.PhoneVault); err == nil {
		t.Fatal("a pass succeeded with the server down")
	}
	w.Server.SetDown(false)

	w.PhoneSync(t)
	w.ComputerSync(t)

	if !hasComputerNote(w, listPath, "eggs") {
		t.Error("the computer has no note with the text eggs")
	}
}
