package file

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// ReadFile returns the whole content of the file at a path, with what the vault reports about the
// file as it was before the bytes were read, the hash of the bytes returned and the version of the vault
// taken before the read.
//
// The version is taken first: a write that lands during the read raises the version after this answer
// took it, so the answer is never newer than the version it carries.
type ReadFile struct {
	readers  vaultport.VaultReaders
	versions vaultport.VaultVersions
}

func NewReadFile(readers vaultport.VaultReaders, versions vaultport.VaultVersions) *ReadFile {
	return &ReadFile{readers: readers, versions: versions}
}

func (u *ReadFile) Execute(ctx context.Context, v domain.Vault, path string) ([]byte, domain.FileStat, error) {
	version := u.versions.GetVersion(v)
	reader, err := u.readers.Open(v)
	if err != nil {
		return nil, domain.FileStat{}, err
	}
	stat, err := reader.StatFile(ctx, path)
	if err != nil {
		return nil, domain.FileStat{}, err
	}
	content, err := reader.Read(ctx, path)
	if err != nil {
		return nil, domain.FileStat{}, err
	}
	stat.Hash = domain.HashContent(content)
	stat.Version = version
	return content, stat, nil
}
