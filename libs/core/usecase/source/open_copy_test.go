package source

import (
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/text"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

func TestACopyInTheStoreIsOpened(t *testing.T) {
	s := newTextScene(t)

	if _, found := s.texts.OpenCopy.Execute(t.Context(), s.vault, linkedURL); found {
		t.Fatal("a copy opened before one was fetched")
	}
	s.write(t, domain.NewCopyName(text.Fingerprint([]byte(string(linkedURL)))), "video")

	copied, found := s.texts.OpenCopy.Execute(t.Context(), s.vault, linkedURL)
	if !found || copied.Size != 5 {
		t.Fatalf("opened %d bytes, found %v", copied.Size, found)
	}
	if err := copied.File.Close(); err != nil {
		t.Fatal(err)
	}
}
