package textversion

import (
	"context"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// GetContentHash is domain.HashContent of the file at the path as it is now.
func (u *Versions) GetContentHash(ctx context.Context, v domain.Vault, path string) (string, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return "", err
	}
	raw, err := reader.Read(ctx, path)
	if err != nil {
		return "", err
	}
	return domain.HashContent(raw), nil
}
