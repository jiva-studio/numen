package source

import (
	"context"
	"io"

	"github.com/jiva-studio/numen/modules/libs/core/internal/text"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
)

// CopyFile is an open copy of a video; whoever opens it closes it.
type CopyFile struct {
	File io.ReadSeekCloser
	Size int64
}

// OpenCopy opens the copy of a video the application keeps in its own folder, and answers false where none stands.
type OpenCopy struct {
	texts sourceTexts
}

func (u OpenCopy) Execute(ctx context.Context, v domain.Vault, targetURL domain.URL) (CopyFile, bool) {
	if !u.texts.isReady() {
		return CopyFile{}, false
	}
	store, err := u.texts.derived.Open(v)
	if err != nil {
		return CopyFile{}, false
	}
	file, size, err := store.OpenArtifact(ctx, domain.NewCopyName(text.Fingerprint([]byte(string(targetURL)))))
	if err != nil {
		return CopyFile{}, false
	}
	return CopyFile{File: file, Size: size}, true
}
