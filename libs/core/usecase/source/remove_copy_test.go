package source

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/core/internal/text"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

func TestACopyInTheStoreIsRemoved(t *testing.T) {
	s := newTextScene(t)
	name := domain.NewCopyName(text.Fingerprint([]byte(string(linkedURL))))
	s.write(t, name, "video")

	if err := s.texts.RemoveCopy.Execute(t.Context(), s.vault, linkPath, linkedURL); err != nil {
		t.Fatal(err)
	}
	if _, err := s.shelf.ReadArtifact(t.Context(), name); err == nil {
		t.Error("the copy is still in the store")
	}
}

func TestACopyIsNotRemovedFromAVaultWhoseIndexIsNotUp(t *testing.T) {
	texts := NewTexts(vaults{first.ID: newLibrary()}, nil, nil, nil)

	if err := texts.RemoveCopy.Execute(t.Context(), first, linkPath, linkedURL); !errors.Is(err, ErrNotReady) {
		t.Errorf("error is %v, want ErrNotReady", err)
	}
}
