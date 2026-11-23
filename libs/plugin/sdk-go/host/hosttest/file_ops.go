package hosttest

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

var (
	_ host.FileMover       = (*Vault)(nil)
	_ host.FileRangeReader = (*Vault)(nil)
	_ host.FilePlacer      = (*Vault)(nil)

	errNotImplemented = errors.New("not implemented")
)

func (v *Vault) MoveFile(ctx context.Context, from, to string, baseline *domain.FileStat) (host.MoveFileResult, error) {
	return host.MoveFileResult{}, errNotImplemented
}

func (v *Vault) RemoveFile(ctx context.Context, filePath string, baseline *domain.FileStat) (host.RemoveFileResult, error) {
	return host.RemoveFileResult{}, errNotImplemented
}

func (v *Vault) ReadBytes(ctx context.Context, filePath string, start, length int64) (host.ByteRange, error) {
	return host.ByteRange{}, errNotImplemented
}

func (v *Vault) PlaceFile(ctx context.Context, name, filePath, expectedHash string, baseline *domain.FileStat) (domain.FileStat, error) {
	return domain.FileStat{}, errNotImplemented
}
