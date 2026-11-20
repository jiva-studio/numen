package note

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// VersionMover re-files the text versions of what moved.
type VersionMover interface {
	// MoveVersions files the versions of what was at one path, and of everything
	// under it, at the path it now has. The versions of whatever was at the
	// destination are removed.
	MoveVersions(ctx context.Context, vaultID domain.VaultID, from, to string) error
}
