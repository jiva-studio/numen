package recognition_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/plugin-sdk-go/host"
	"github.com/jiva-studio/numen/plugins/engines-ocr/backend/recognition"
)

func TestCreateTextRecognisesInBackground(t *testing.T) {
	s := newScene(t)
	provider := recognition.NewProvider(s.run)
	t.Cleanup(provider.Close)

	if err := provider.CreateText(t.Context(), host.TextCreation{Vault: "Physics", Path: "books/scan.pdf"}); err != nil {
		t.Fatal(err)
	}
	provider.Wait()

	if listing, _ := s.texts.ListVersions(t.Context(), "books/scan.pdf"); listing.Live == "" {
		t.Fatalf("the file has no live version after its recognition: %+v", listing)
	}
}
