package file

import (
	"context"
	"errors"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// HashFile returns what StatFile returns, with the hash of the content and the version of the vault
// taken before the content is read. It reads the whole file, so it is the call for a caller that
// compares content; StatFile is the call that opens nothing.
type HashFile struct {
	readers  vaultport.VaultReaders
	versions VaultVersions
}

func NewHashFile(readers vaultport.VaultReaders, versions VaultVersions) *HashFile {
	return &HashFile{readers: readers, versions: versions}
}

func (u *HashFile) Execute(ctx context.Context, v domain.Vault, path string) (_ domain.FileStat, err error) {
	version := u.versions.GetVersion(v)
	reader, err := u.readers.Open(v)
	if err != nil {
		return domain.FileStat{}, err
	}
	stat, err := reader.StatFile(ctx, path)
	if err != nil {
		return domain.FileStat{}, err
	}
	file, err := reader.Open(ctx, path)
	if err != nil {
		return domain.FileStat{}, err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	hash, err := domain.HashReader(file)
	if err != nil {
		return domain.FileStat{}, err
	}
	stat.Hash = hash
	stat.Version = version
	return stat, nil
}
