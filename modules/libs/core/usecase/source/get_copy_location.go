package source

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

type CopyLocation struct {
	VaultPath string
	Size      int64
	Exists    bool
}

// GetCopyLocation says where the copy of a video stands.
type GetCopyLocation struct {
	texts sourceTexts
}

func (u GetCopyLocation) Execute(ctx context.Context, v domain.Vault, path string, targetURL domain.URL) CopyLocation {
	if targetURL == "" {
		return CopyLocation{}
	}
	if u.texts.readers != nil {
		if reader, err := u.texts.readers.Open(v); err == nil {
			if ref, err := reader.GetFingerprint(ctx, CopyBeside(path)); err == nil {
				return CopyLocation{VaultPath: ref.Path, Size: ref.Size, Exists: true}
			}
		}
	}
	if !u.texts.isReady() {
		return CopyLocation{}
	}
	store, err := u.texts.derived.Open(v)
	if err != nil {
		return CopyLocation{}
	}
	file, size, err := store.OpenArtifact(ctx, domain.NewCopyName(text.Fingerprint([]byte(string(targetURL)))))
	if err != nil {
		return CopyLocation{}
	}
	_ = file.Close()
	return CopyLocation{Size: size, Exists: true}
}
