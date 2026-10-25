package recognition_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/recognition"
)

func TestTheHostReadsARecognitionAPageACall(t *testing.T) {
	s := newScene(t)
	provider := recognition.NewProvider(s.run, s.store)
	t.Cleanup(provider.Close)
	if _, err := s.run.Recognise(t.Context(), recognition.Request{Path: "books/scan.pdf"}); err != nil {
		t.Fatal(err)
	}

	first, err := provider.GetText(t.Context(), host.TextRequest{Vault: "Physics", Path: "books/scan.pdf", Key: recognition.VersionKey})
	if err != nil || first.NextCursor != "1" || first.Parts[0].Place.Page != 0 {
		t.Fatalf("the first page %+v (%v)", first, err)
	}
	last, err := provider.GetText(t.Context(), host.TextRequest{Vault: "Physics", Path: "books/scan.pdf", Key: recognition.VersionKey, Cursor: "1"})
	if err != nil || last.NextCursor != "" || last.Parts[0].Place.Page != 1 {
		t.Fatalf("the last page %+v (%v)", last, err)
	}
	for name, req := range map[string]host.TextRequest{
		"another key":         {Vault: "Physics", Path: "books/scan.pdf", Key: "other"},
		"a page past the end": {Vault: "Physics", Path: "books/scan.pdf", Key: recognition.VersionKey, Cursor: "2"},
		"a file not read":     {Vault: "Physics", Path: "books/layer.pdf", Key: recognition.VersionKey},
	} {
		if _, err := provider.GetText(t.Context(), req); !errors.Is(err, host.ErrNotExist) {
			t.Errorf("%s answered %v", name, err)
		}
	}
}

func TestACreatedTextIsRecognisedBehindTheCall(t *testing.T) {
	s := newScene(t)
	provider := recognition.NewProvider(s.run, s.store)
	t.Cleanup(provider.Close)

	if err := provider.CreateText(t.Context(), host.TextCreation{Vault: "Physics", Path: "books/scan.pdf", IsUserStarted: true}); err != nil {
		t.Fatal(err)
	}
	provider.Wait()

	if listing, _ := s.texts.ListVersions(t.Context(), "books/scan.pdf"); listing.Live == "" {
		t.Fatalf("the file has no live version after its recognition: %+v", listing)
	}
}

func TestATextOfAVersionIsNoTextThePluginMakes(t *testing.T) {
	provider := recognition.NewProvider(newScene(t).run, nil)
	t.Cleanup(provider.Close)
	err := provider.CreateText(t.Context(), host.TextCreation{Vault: "Physics", Path: "books/scan.pdf", Input: "md.numen.readers.pdf/layer"})
	if !errors.Is(err, host.ErrInvalidVersion) {
		t.Fatalf("a text of a version answered %v", err)
	}
}

func TestADeletedTextIsNotThereAnyMore(t *testing.T) {
	s := newScene(t)
	provider := recognition.NewProvider(s.run, s.store)
	t.Cleanup(provider.Close)
	if _, err := s.run.Recognise(t.Context(), recognition.Request{Path: "books/scan.pdf"}); err != nil {
		t.Fatal(err)
	}

	if err := provider.DeleteText(t.Context(), "Physics", "books/scan.pdf", recognition.VersionKey); err != nil {
		t.Fatal(err)
	}
	_, err := provider.GetText(t.Context(), host.TextRequest{Vault: "Physics", Path: "books/scan.pdf", Key: recognition.VersionKey})
	if !errors.Is(err, host.ErrNotExist) {
		t.Fatalf("a deleted text answered %v", err)
	}
	if files, _ := s.store.List(t.Context()); len(files) != 0 {
		t.Errorf("the folder still keeps %+v", files)
	}
}
