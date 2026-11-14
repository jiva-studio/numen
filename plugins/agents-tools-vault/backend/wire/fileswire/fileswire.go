// Package fileswire calls the files use case with the messages of the file tools.
package fileswire

import (
	"context"

	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/api"
	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/usecase/files"
)

// ReadFile answers file_read.
func ReadFile(ctx context.Context, u *files.UseCase, in api.FileReadInput) (api.FileReadOutput, error) {
	read, err := u.Read(ctx, in.Path, in.Start, in.Length)
	if err != nil {
		return api.FileReadOutput{}, err
	}
	return api.FileReadOutput{
		Path:        read.Path,
		Text:        read.Text,
		Start:       read.Start,
		Length:      len(read.Text),
		Size:        read.Size,
		Fingerprint: read.Fingerprint,
	}, nil
}

// CreateFile answers file_create.
func CreateFile(ctx context.Context, u *files.UseCase, in api.FileCreateInput) (api.FileCreateOutput, error) {
	size, err := u.Create(ctx, in.Path, in.Text)
	if err != nil {
		return api.FileCreateOutput{}, err
	}
	return api.FileCreateOutput{Path: in.Path, Size: size}, nil
}
