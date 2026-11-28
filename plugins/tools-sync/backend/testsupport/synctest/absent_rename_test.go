package synctest_test

import (
	"testing"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/testsupport/synctest"
)

// The tests of this file run what the person does to a file that was listed and never downloaded: the phone
// renames, moves or deletes it and the computer must end up with the same vault.

const renamedPath = "Books/Guide v2.pdf"

func moveOnPhone(t *testing.T, w *synctest.PhoneWorld, from, to string) {
	t.Helper()
	if _, err := w.Phone.Vaults.With(synctest.PhoneVault).MoveFile(t.Context(), from, to, nil); err != nil {
		t.Fatal(err)
	}
}

func syncPhoneAfterTheRemovalDelay(t *testing.T, w *synctest.PhoneWorld) {
	t.Helper()
	w.Clock.Advance(time.Minute)
	w.SyncPhone(t)
	w.Clock.Advance(synctest.RemovalDelay)
	w.SyncPhone(t)
	w.SyncComputer(t)
}

func hasNoDeletionOnServer(t *testing.T, w *synctest.PhoneWorld) bool {
	t.Helper()
	for _, one := range w.Computer.ReadChanges(t, w.Computer.GetRemoteID(t, synctest.ComputerVault)) {
		if one.GetRevision().GetDeleted() {
			return false
		}
	}
	return true
}

func TestRenamingAnAbsentFileOnThePhoneRenamesItOnTheComputer(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	want := w.WriteGuide(t, guidePath)
	moveOnPhone(t, w, guidePath, renamedPath)

	syncPhoneAfterTheRemovalDelay(t, w)

	if !hasComputerNote(w, renamedPath, want) {
		t.Error("the computer has no file at the new name")
	}
	if _, ok := w.Computer.ReadNote(synctest.ComputerVault, guidePath); ok {
		t.Error("the computer still has the file at the old name")
	}
	if !hasNoDeletionOnServer(t, w) {
		t.Error("the server holds a deletion after a rename")
	}
	if trashed := w.Computer.Vaults.With(synctest.ComputerVault).Trashed(); len(trashed) != 0 {
		t.Errorf("the computer moved %v to the trash after a rename on the phone", trashed)
	}
}

func TestFetchFile_RenamedAbsentFile_DownloadsAtTheNewPath(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	want := w.WriteGuide(t, guidePath)
	moveOnPhone(t, w, guidePath, renamedPath)
	syncPhoneAfterTheRemovalDelay(t, w)

	if err := w.Fetch(t.Context(), renamedPath); err != nil {
		t.Fatal(err)
	}

	if !hasPhoneNote(w, renamedPath, want) {
		t.Error("the phone does not hold the file under its new name after the download")
	}
}

func TestThePhoneKeepsListingTheNewNameAfterARename(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	moveOnPhone(t, w, guidePath, renamedPath)

	syncPhoneAfterTheRemovalDelay(t, w)
	w.SyncPhone(t)

	if !w.Phone.IsAbsent(t, synctest.PhoneVault, renamedPath) {
		t.Errorf("absent files = %+v, want the file under its new name", w.Phone.ListAbsent(t, synctest.PhoneVault))
	}
}

func TestMovingAFolderOfAbsentFilesMovesThemOnTheComputer(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	first := w.Computer.WriteFileOfSize(synctest.ComputerVault, "Books/First.pdf", 5<<20)
	second := w.Computer.WriteFileOfSize(synctest.ComputerVault, "Books/Second.pdf", 5<<20+7)
	w.Clock.Advance(time.Minute)
	w.SyncComputer(t)
	w.SyncPhone(t)
	moveOnPhone(t, w, "Books", "Library")

	syncPhoneAfterTheRemovalDelay(t, w)

	if !hasComputerNote(w, "Library/First.pdf", first) || !hasComputerNote(w, "Library/Second.pdf", second) {
		t.Error("the computer does not hold both files in the new folder")
	}
	if !hasNoDeletionOnServer(t, w) {
		t.Error("the server holds a deletion after a move of a folder")
	}
}

func TestMovingAbsentFilesWithEqualContentMovesBothOnTheComputer(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	content := w.Computer.WriteFileOfSize(synctest.ComputerVault, "Books/First.pdf", 5<<20)
	w.Computer.WriteFileOfSize(synctest.ComputerVault, "Books/Second.pdf", 5<<20)
	w.Clock.Advance(time.Minute)
	w.SyncComputer(t)
	w.SyncPhone(t)
	moveOnPhone(t, w, "Books", "Library")

	syncPhoneAfterTheRemovalDelay(t, w)

	if !hasComputerNote(w, "Library/First.pdf", content) || !hasComputerNote(w, "Library/Second.pdf", content) {
		t.Error("the computer does not hold both copies in the new folder")
	}
	if !hasNoDeletionOnServer(t, w) {
		t.Error("the server holds a deletion after a move of a folder")
	}
}

func TestDeletingAnAbsentFileOnThePhoneAfterARenameRemovesItFromTheVault(t *testing.T) {
	w := synctest.NewPhoneWorld(t)
	w.WriteGuide(t, guidePath)
	moveOnPhone(t, w, guidePath, renamedPath)
	syncPhoneAfterTheRemovalDelay(t, w)
	w.Phone.Vaults.With(synctest.PhoneVault).Remove(renamedPath)

	w.Clock.Advance(time.Minute)
	w.SyncPhone(t)
	w.SyncComputer(t)

	if _, ok := w.Computer.ReadNote(synctest.ComputerVault, renamedPath); ok {
		t.Error("the computer still has a file the phone deleted after a rename")
	}
}
