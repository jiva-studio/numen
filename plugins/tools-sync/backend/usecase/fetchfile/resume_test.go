package fetchfile_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
)

var errRefused = errors.New("the vault refuses the file")

func TestFetchFile_ResumesAfterAPassRanBetweenTheBreakAndTheTry(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	want := countRanges(t, w)
	w.Server.FailBlobGetsAfter(2)
	if err := w.Fetch(t.Context(), guidePath); err == nil {
		t.Fatal("a download that lost the server succeeded")
	}
	w.Server.FailBlobGetsAfter(-1)
	w.Clock.Advance(time.Minute)
	w.SyncPhone(t)
	before := w.Server.CountBlobGets()

	if err := w.Fetch(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}

	if got := w.Server.CountBlobGets() - before; got != want-2 {
		t.Errorf("the try after a pass took %d blob requests, want the %d ranges the first did not get", got, want-2)
	}
}

func TestFetchFile_KeepsTheBytesWhenTheVaultRefusesToPlaceTheFile(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	want := w.WriteGuide(t, guidePath)
	vault := w.Phone.Vaults.With(synctest.PhoneVault)
	vault.Refuse(guidePath, errRefused)
	if err := w.Fetch(t.Context(), guidePath); !errors.Is(err, errRefused) {
		t.Fatalf("error = %v, want the refusal of the vault", err)
	}
	vault.Refuse(guidePath, nil)
	before := w.Server.CountBlobGets()

	if err := w.Fetch(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}

	if got := w.Server.CountBlobGets() - before; got != 0 {
		t.Errorf("the second try asked the server for %d ranges, want none: the bytes were kept", got)
	}
	if got, ok := w.Phone.ReadNote(synctest.PhoneVault, guidePath); !ok || got != want {
		t.Errorf("file of %d bytes (there: %v), want the 5 MiB of the computer", len(got), ok)
	}
}

func TestFetchFile_APassDeletesWhatNoFileWaitsFor(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	store := w.Phone.Host.Store(synctest.PhoneVault)
	for _, name := range []string{"fetch-0123456789abcdef.bin", "fetch-0123456789abcdef.part"} {
		if err := store.Write(t.Context(), name, []byte("left by a crash")); err != nil {
			t.Fatal(err)
		}
	}
	w.Clock.Advance(time.Minute)

	w.SyncPhone(t)

	if left := listFetchFiles(t, w); len(left) != 0 {
		t.Errorf("the store keeps %v after a pass, want nothing: no listed file waits for them", left)
	}
}

func TestFetchFile_APassKeepsThePartOfAListedFile(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	w.Server.FailBlobGetsAfter(1)
	if err := w.Fetch(t.Context(), guidePath); err == nil {
		t.Fatal("a download that lost the server succeeded")
	}
	w.Clock.Advance(time.Minute)

	w.SyncPhone(t)

	if left := listFetchFiles(t, w); len(left) != 1 {
		t.Errorf("the store keeps %v after a pass, want the part of the listed file", left)
	}
}

func TestFetchFile_APassDeletesThePartOfAFileThatIsNotListedAnyMore(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	w.Server.FailBlobGetsAfter(1)
	if err := w.Fetch(t.Context(), guidePath); err == nil {
		t.Fatal("a download that lost the server succeeded")
	}
	w.Phone.Vaults.With(synctest.PhoneVault).Remove(guidePath)
	w.Clock.Advance(time.Minute)

	w.SyncPhone(t)

	if left := listFetchFiles(t, w); len(left) != 0 {
		t.Errorf("the store keeps %v after the file was deleted, want nothing", left)
	}
}
