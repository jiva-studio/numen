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
	texts liveFileTexts
	edits versionEdits
}

// liveFileTexts reads the content hash of a file and the text of its live version.
type liveFileTexts interface {
	GetContentHash(ctx context.Context, v domain.Vault, path string) (string, error)
	GetLiveText(ctx context.Context, v domain.Vault, path, contentHash string) (domain.VersionText, error)
}

// versionEdits records the edit of a user as a version.
type versionEdits interface {
	CreateEdit(ctx context.Context, v domain.Vault, edit textdomain.Edit) (domain.TextVersion, error)
}

// NewFileText returns the text of the vault files from their text versions, and records the edits of
// the user as versions.
func NewFileText(texts *Texts, versions *Versions) (*FileText, error) {
	if texts == nil || versions == nil {
		return nil, errors.New("file text: texts and text versions are required")
	}
	return &FileText{texts: texts, edits: versions}, nil
}

// ReadLiveText is the live version of a file and its text, and false where
// none is live.
func (u *FileText) ReadLiveText(ctx context.Context, v domain.Vault, path string) (domain.VersionText, bool, error) {
	contentHash, err := u.texts.GetContentHash(ctx, v, path)
	if err != nil {
		return domain.VersionText{}, false, fmt.Errorf("read %s: %w", path, err)
	}
	text, err := u.texts.GetLiveText(ctx, v, path, contentHash)
	if errors.Is(err, textdomain.ErrNoText) {
		return domain.VersionText{}, false, nil
	}
	if err != nil {
		return domain.VersionText{}, false, err
	}
	return text, true, nil
}

// CreateEdit records a user's edit as a new version created from its source.
func (u *FileText) CreateEdit(ctx context.Context, v domain.Vault, edit textdomain.Edit) (domain.TextVersion, error) {
	return u.edits.CreateEdit(ctx, v, edit)
}
