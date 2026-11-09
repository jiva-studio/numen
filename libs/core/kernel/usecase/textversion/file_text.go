package textversion

import (
	"context"
	"errors"
	"fmt"

	textdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/text"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// FileText is the live text of a vault file and the user's edits of it, for the
// views of a window. Which version is live is determined for the file as it is
// now.
type FileText struct {
	versions *Versions
}

// NewFileText returns the text of the vault files from their text versions.
func NewFileText(versions *Versions) (*FileText, error) {
	if versions == nil {
		return nil, errors.New("file text: text versions are required")
	}
	return &FileText{versions: versions}, nil
}

// ReadLiveText is the live version of a file and its text, and false where
// none is live.
func (u *FileText) ReadLiveText(ctx context.Context, v domain.Vault, path string) (domain.VersionText, bool, error) {
	contentHash, err := u.versions.GetContentHash(ctx, v, path)
	if err != nil {
		return domain.VersionText{}, false, fmt.Errorf("read %s: %w", path, err)
	}
	text, err := u.versions.GetLiveText(ctx, v, path, contentHash)
	if errors.Is(err, textdomain.ErrNoText) {
		return domain.VersionText{}, false, nil
	}
	if err != nil {
		return domain.VersionText{}, false, err
	}
	return text, true, nil
}

// CreateEdit records a user's edit as a new version created from its source.
func (u *FileText) CreateEdit(ctx context.Context, v domain.Vault, edit TextEdit) (domain.TextVersion, error) {
	return u.versions.CreateEdit(ctx, v, edit)
}
