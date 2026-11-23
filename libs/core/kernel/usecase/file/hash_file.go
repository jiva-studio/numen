package file

import (
	"context"
	"errors"
	"sync"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// HashFile returns what StatFile returns, with the hash of the content and the version of the vault
// taken before the content is read. It reads the whole file, so it is the call for a caller that
// compares content; StatFile is the call that opens nothing.
type HashFile struct {
	readers  vaultport.VaultReaders
	versions VaultVersions
	known    knownHashes
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

// knownHashes remembers the hashes of the files a ranged read hashed, by the size and the modification time the
// file had, so a read of a big file in many ranges hashes it once while it does not change.
type knownHashes struct {
	mu     sync.Mutex
	hashes map[knownKey]string
}

type knownKey struct {
	vault domain.VaultID
	path  string
	size  int64
	mtime int64
}

const mostKnownHashes = 256

func (k *knownHashes) get(key knownKey) (string, bool) {
	k.mu.Lock()
	defer k.mu.Unlock()
	hash, ok := k.hashes[key]
	return hash, ok
}

func (k *knownHashes) put(key knownKey, hash string) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.hashes == nil || len(k.hashes) >= mostKnownHashes {
		k.hashes = make(map[knownKey]string, mostKnownHashes)
	}
	k.hashes[key] = hash
}

// ExecuteKnown returns what Execute returns, and reads the file only when the size or the modification time of the
// file is not the one a former call hashed.
func (u *HashFile) ExecuteKnown(ctx context.Context, v domain.Vault, path string) (domain.FileStat, error) {
	reader, err := u.readers.Open(v)
	if err != nil {
		return domain.FileStat{}, err
	}
	stat, err := reader.StatFile(ctx, path)
	if err != nil {
		return domain.FileStat{}, err
	}
	key := knownKey{vault: v.ID, path: path, size: stat.Size, mtime: stat.ModTime.UnixNano()}
	if hash, ok := u.known.get(key); ok {
		stat.Hash, stat.Version = hash, u.versions.GetVersion(v)
		return stat, nil
	}
	hashed, err := u.Execute(ctx, v, path)
	if err != nil {
		return domain.FileStat{}, err
	}
	if hashed.Size == stat.Size && hashed.ModTime.Equal(stat.ModTime) {
		u.known.put(key, hashed.Hash)
	}
	return hashed, nil
}
