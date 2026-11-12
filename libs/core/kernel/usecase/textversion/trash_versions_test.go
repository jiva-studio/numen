package textversion_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion"
)

const trashedScanPath = ".trash/" + scanPath

func (s *scene) trashScan(t *testing.T) domain.TextVersion {
	t.Helper()
	s.requestText(t)
	s.runEntry(t, ocr)
	version, ok := s.findVersion(t, s.getReaderVersion())
	if !ok {
		t.Fatal("the scan has no version")
	}
	version.Path = trashedScanPath
	writer, err := s.repo.OpenVersionWriter(s.vault)
	if err != nil {
		t.Fatal(err)
	}
	if err := writer.SaveVersion(t.Context(), version); err != nil {
		t.Fatal(err)
	}
	return version
}

func (s *scene) getTrashedVersions(t *testing.T) []domain.TextVersion {
	t.Helper()
	reader, err := s.repo.OpenVersionReader(s.vault)
	if err != nil {
		t.Fatal(err)
	}
	found, err := reader.GetVersionsUnder(t.Context(), ".trash")
	if err != nil {
		t.Fatal(err)
	}
	return found
}

func TestDeleteTrashedRemovesRecordsAndFilesOfTrashedVersions(t *testing.T) {
	s := newScene(t)
	version := s.trashScan(t)
	trash := textversion.NewTrashVersions(s.repo, s.folders)

	if err := trash.DeleteUnder(t.Context(), s.vault, ".trash", nil); err != nil {
		t.Fatal(err)
	}

	if got := s.getTrashedVersions(t); len(got) != 0 {
		t.Errorf("the trash still has versions: %v", got)
	}
	if s.folders.has(version.PluginID + "/" + version.File) {
		t.Errorf("the file %s of the trashed version is still in the folder of %s", version.File, version.PluginID)
	}
}

func TestDeleteTrashedKeepsVersionsOfOtherFiles(t *testing.T) {
	s := newScene(t)
	s.trashScan(t)
	trash := textversion.NewTrashVersions(s.repo, s.folders)

	if err := trash.DeleteUnder(t.Context(), s.vault, ".trash", []string{".trash/books"}); err != nil {
		t.Fatal(err)
	}

	if got := s.getTrashedVersions(t); len(got) != 1 {
		t.Errorf("the versions of a file that stays were deleted: %v", got)
	}
}
