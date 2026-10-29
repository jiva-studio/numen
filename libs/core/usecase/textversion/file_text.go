package textversion

import (
	"context"
	"errors"
	"fmt"

	"github.com/jiva-studio/numen/libs/notes/domain"

	"github.com/jiva-studio/numen/libs/core/port/textport"
	"github.com/jiva-studio/numen/libs/core/port/vaultport"
)

// FileText is the live text of a vault file and the user's edits of it, for the
// views of a window. Which version is live is determined for the file as it is
// now.
type FileText struct {
	versions *Versions
	readers  vaultport.VaultReaders
}

// NewFileText is the text the versions give the files the readers open.
func NewFileText(versions *Versions, readers vaultport.VaultReaders) (*FileText, error) {
	if versions == nil || readers == nil {
		return nil, errors.New("a file text needs text versions and vault readers")
	}
	return &FileText{versions: versions, readers: readers}, nil
}

// ReadLiveText is the live version of a file and its text, and false where
// none is live.
func (u *FileText) ReadLiveText(ctx context.Context, v domain.Vault, path string) (domain.VersionText, bool, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return domain.VersionText{}, false, err
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return domain.VersionText{}, false, fmt.Errorf("read %s: %w", path, err)
	}
	text, err := u.versions.GetLiveText(ctx, v, path, domain.HashContent(raw))
	if errors.Is(err, textport.ErrNoText) {
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
