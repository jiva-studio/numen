package downloadwire_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/wire/downloadwire"
)

const guidePath = "Books/Guide.pdf"

func newDownloads(w *synctest.PhoneWorld) *downloadwire.Downloads {
	return downloadwire.New(downloadwire.Deps{Fetch: w.Phone.FetchDeps(), Entries: w.Phone.State})
}

func TestFetchFile_PlacesTheFileThatTheRequestNames(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	want := w.WriteGuide(t, guidePath)

	_, err := newDownloads(w).FetchFile(t.Context(), api.FetchFileRequest{VaultID: string(synctest.PhoneVault), Path: guidePath})

	if err != nil {
		t.Fatal(err)
	}
	if got, ok := w.Phone.ReadNote(synctest.PhoneVault, guidePath); !ok || got != want {
		t.Error("the file is not in the vault after the request")
	}
}

func TestDropDownload_ListsTheFileAsAbsentAgain(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	downloads := newDownloads(w)
	if _, err := downloads.FetchFile(t.Context(), api.FetchFileRequest{VaultID: string(synctest.PhoneVault), Path: guidePath}); err != nil {
		t.Fatal(err)
	}

	_, err := downloads.DropDownload(t.Context(), api.DropDownloadRequest{VaultID: string(synctest.PhoneVault), Path: guidePath})

	if err != nil {
		t.Fatal(err)
	}
	if !w.Phone.IsAbsent(t, synctest.PhoneVault, guidePath) {
		t.Error("the file is not listed as absent after its download was dropped")
	}
}

func TestGetDownloads_ListsNothingWhenNoDownloadRuns(t *testing.T) {
	w := synctest.NewPhoneWorld(t)

	got, err := newDownloads(w).GetDownloads(t.Context(), api.GetDownloadsRequest{})

	if err != nil {
		t.Fatal(err)
	}
	if len(got.Downloads) != 0 {
		t.Errorf("got %d downloads, want none", len(got.Downloads))
	}
}
