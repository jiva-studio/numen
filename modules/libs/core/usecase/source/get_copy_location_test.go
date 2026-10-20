package source

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

func TestACopyInTheStoreIsLocatedByItsAddress(t *testing.T) {
	s := newTextScene(t)

	if s.texts.GetCopyLocation.Execute(t.Context(), s.vault, linkPath, linkedURL).Exists {
		t.Fatal("a copy exists before one was fetched")
	}
	s.write(t, domain.NewCopyName(text.Fingerprint([]byte(string(linkedURL)))), "video")

	where := s.texts.GetCopyLocation.Execute(t.Context(), s.vault, linkPath, linkedURL)
	if !where.Exists || where.VaultPath != "" || where.Size != 5 {
		t.Errorf("location is %+v, want 5 bytes in the store", where)
	}
}
