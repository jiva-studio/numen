package source

import (
	"context"
	"errors"
	"io/fs"

	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// RemoveCopy takes the copy of a video off this disk and leaves the note pointing at the address.
type RemoveCopy struct {
	texts    sourceTexts
	location GetCopyLocation
	writers  port.VaultWriters
}

func (u RemoveCopy) Execute(ctx context.Context, v domain.Vault, path string, at domain.URL) error {
	if !u.texts.isReady() {
		return ErrNotReady
	}
	if location := u.location.Execute(ctx, v, path, at); location.Exists && location.VaultPath != "" {
		writer, err := u.writers.Open(v)
		if err != nil {
			return err
		}
		if err := writer.Remove(ctx, location.VaultPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
	}
	store, err := u.texts.derived.Open(v)
	if err != nil {
		return err
	}
	name := domain.NewCopyName(text.Fingerprint([]byte(string(at))))
	if err := store.RemoveArtifact(ctx, name); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	return nil
}
