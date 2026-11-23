package localchanges_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/note"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/transfer"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/internal/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/localchanges"
)

const guidePath = "Books/Guide.pdf"

func scanPhone(t *testing.T, w *synctest.PhoneWorld, content string) []transfer.Change {
	t.Helper()
	w.Clock.Advance(time.Minute)
	in := transfer.ScanInput{VaultID: synctest.PhoneVault, Bases: map[string]string{guidePath: note.Hash([]byte(content))}}
	found, err := localchanges.NewScanLocalChanges(w.Phone.LocalChangesDeps()).Execute(t.Context(), in)
	if err != nil {
		t.Fatal(err)
	}
	return found
}

func TestScanLocalChanges_ReportsNothingForAFileThatIsAbsent(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	content := w.WriteGuide(t, guidePath)

	if found := scanPhone(t, w, content); len(found) != 0 {
		t.Errorf("found = %+v, want an absent file to count as its base", found)
	}
}

func TestScanLocalChanges_ReportsNoDeleteForAFileWhoseDownloadWasRemoved(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	content := w.WriteGuide(t, guidePath)
	if err := w.Fetch(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}
	scanPhone(t, w, content)
	if err := w.DropDownload(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}

	if found := scanPhone(t, w, content); len(found) != 0 {
		t.Errorf("found = %+v, want no change after the download was removed", found)
	}
}

func TestScanLocalChanges_ReportsAFileChangedAfterItsDownload(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	content := w.WriteGuide(t, guidePath)
	if err := w.Fetch(t.Context(), guidePath); err != nil {
		t.Fatal(err)
	}
	scanPhone(t, w, content)
	w.Phone.WriteNote(synctest.PhoneVault, guidePath, content+" and my notes")

	found := scanPhone(t, w, content)

	if len(found) != 1 || found[0].Path != guidePath {
		t.Errorf("found = %+v, want the changed file", found)
	}
}
