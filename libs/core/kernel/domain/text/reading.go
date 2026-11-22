package text

import (
	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// VersionListener is notified that the versions of a file changed: one was added,
// removed or moved, so the live version may differ.
type VersionListener interface {
	ReportVersionsChanged(v domain.Vault, path string)
}
