package dropdownload_test

import (
	"errors"
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/dropdownload"
)

const guidePath = "Books/Guide.pdf"

func newDownloadedWorld(t *testing.T) (*synctest.PhoneWorld, string) {
	t.Helper()
	w := synctest.NewPhoneWorld(t)
	content := w.WriteGuide(t, guidePath)
	if err := w.Fetch(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}
	return w, content
}

func TestDropDownload_ListsTheFileAndFreesItsBytes(t *testing.T) {
	w, _ := newDownloadedWorld(t)

	if err := w.DropDownload(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}

	if _, ok := w.Phone.ReadNote(synctest.PhoneVault, guidePath); ok {
		t.Error("the bytes are still on the phone")
	}
	if !w.Phone.IsAbsent(t, synctest.PhoneVault, guidePath) {
		t.Error("the phone does not list the file after its download was removed")
	}
}

func TestDropDownload_IsNoDeleteForThePass(t *testing.T) {
	w, _ := newDownloadedWorld(t)
	remote := w.Phone.GetRemoteID(t, synctest.PhoneVault)
	before := len(w.Phone.ReadChanges(t, remote))
	if err := w.DropDownload(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}
	w.Clock.Advance(time.Minute)

	w.PhoneSync(t)

	changes := w.Phone.ReadChanges(t, remote)
	if len(changes) != before {
		t.Errorf("the server holds %d revisions, want the %d it held: no upload, no deletion", len(changes), before)
	}
	for _, one := range changes {
		if one.GetRevision().GetDeleted() {
			t.Error("the server holds a deletion of the file after a removed download")
		}
	}
	w.ComputerSync(t)
	if _, ok := w.Computer.ReadNote(synctest.ComputerVault, guidePath); !ok {
		t.Error("the computer lost the file after the phone removed its download")
	}
}

func TestDropDownload_RefusesAFileWithChanges(t *testing.T) {
	w, content := newDownloadedWorld(t)
	w.Phone.WriteNote(synctest.PhoneVault, guidePath, content+" and my notes")

	err := w.DropDownload(t.Context(), guidePath)

	if !errors.Is(err, dropdownload.ErrChanged) {
		t.Errorf("error = %v, want ErrChanged", err)
	}
	if got, ok := w.Phone.ReadNote(synctest.PhoneVault, guidePath); !ok || got != content+" and my notes" {
		t.Error("a file with changes lost its bytes")
	}
}

func TestDropDownload_LetsTheFileBeDownloadedAgain(t *testing.T) {
	w, content := newDownloadedWorld(t)
	if err := w.DropDownload(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}

	if err := w.Fetch(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}

	if got, ok := w.Phone.ReadNote(synctest.PhoneVault, guidePath); !ok || got != content {
		t.Errorf("file of %d bytes (there: %v), want the 5 MiB again", len(got), ok)
	}
}
