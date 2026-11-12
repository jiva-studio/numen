package textversion_test

import (
	"errors"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"

	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
)

func (s *scene) findVersion(t *testing.T, id domain.TextVersionID) (domain.TextVersion, bool) {
	t.Helper()
	listing, err := s.versions.Texts.ListVersions(t.Context(), s.vault, scanPath, s.getContentHash())
	if err != nil {
		t.Fatal(err)
	}
	index := slices.IndexFunc(listing.Versions, func(one domain.TextVersion) bool { return one.GetID() == id })
	if index < 0 {
		return domain.TextVersion{}, false
	}
	return listing.Versions[index], true
}

func TestDeleteVersionRemovesTheRecordAndTheFileOfTheVersionAndReportsTheFile(t *testing.T) {
	s := newScene(t)
	s.requestText(t)
	s.runEntry(t, ocr)
	version, _ := s.findVersion(t, s.getReaderVersion())
	s.listener.take()

	if err := s.versions.DeleteVersion(t.Context(), s.vault, scanPath, s.getReaderVersion()); err != nil {
		t.Fatal(err)
	}

	if _, isListed := s.findVersion(t, s.getReaderVersion()); isListed {
		t.Error("the deleted version is still listed")
	}
	if s.folders.has(pdfReader + "/" + version.File) {
		t.Errorf("the file %s of the deleted version is still in the folder of %s", version.File, pdfReader)
	}
	if got := s.listener.take(); !slices.Equal(got, []string{scanPath}) {
		t.Errorf("reported %v, want %s", got, scanPath)
	}
}

func TestDeleteVersionDeletesAnEditOfAPluginWithNoBackend(t *testing.T) {
	s := newScene(t)
	s.requestText(t)
	edit, err := s.versions.CreateEdit(t.Context(), s.vault, textdomain.Edit{
		PluginID: "org.example.viewer", Path: scanPath, Source: s.getReaderVersion(),
		Segments: []domain.TextSegment{{Text: "edited"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := s.versions.DeleteVersion(t.Context(), s.vault, scanPath, edit.GetID()); err != nil {
		t.Fatal(err)
	}

	if _, isListed := s.findVersion(t, edit.GetID()); isListed || s.folders.has("org.example.viewer/"+edit.File) {
		t.Error("the edit is still there")
	}
}

func TestDeleteVersionRejectsAVersionTheFileDoesNotHave(t *testing.T) {
	s := newScene(t)
	s.requestText(t)

	err := s.versions.DeleteVersion(t.Context(), s.vault, scanPath, "org.example.scans/none")

	if !errors.Is(err, textdomain.ErrInvalidVersion) {
		t.Errorf("got %v, want ErrInvalidVersion", err)
	}
}

func TestTheDeleteItemsAreEveryVersionOfTheFile(t *testing.T) {
	s := newScene(t)
	menu, _ := newMenu(t, s)
	s.requestText(t)
	s.runEntry(t, ocr)

	items, err := menu.ListEveryVersion(t.Context(), s.vault, scanPath)
	if err != nil {
		t.Fatal(err)
	}

	if len(items) != 2 || items[0].Kind == "" || items[0].IsLive == items[1].IsLive {
		t.Errorf("items = %+v, want both versions, one of them live", items)
	}
}
