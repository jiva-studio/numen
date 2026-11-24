package file

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
	"github.com/jiva-studio/numen/libs/shared/note/edit"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// StoreFiles opens a file outside the vault by its name.
type StoreFiles interface {
	Open(ctx context.Context, name string) (io.ReadCloser, error)
}

// EvictWriters opens a vault for deleting the bytes of its files.
type EvictWriters interface {
	OpenFileMover(v domain.Vault) (vaultport.FileMover, error)
}

// PutAbsentFiles lists files whose bytes are not on this device. It refuses a path that holds a file
// of another hash.
type PutAbsentFiles struct {
	readers vaultport.VaultReaders
	absent  vaultport.AbsentFiles
}

func NewPutAbsentFiles(readers vaultport.VaultReaders, absent vaultport.AbsentFiles) *PutAbsentFiles {
	return &PutAbsentFiles{readers: readers, absent: absent}
}

// Execute lists the files, or lists none and returns vault.ErrFilePresent when a path holds a file
// with other content. A path that holds the same content is not listed.
func (u *PutAbsentFiles) Execute(ctx context.Context, v domain.Vault, files []vault.AbsentFile) error {
	reader, err := u.readers.Open(v)
	if err != nil {
		return err
	}
	listed := make([]vault.AbsentFile, 0, len(files))
	for _, entry := range files {
		isHeld, err := checkHeld(ctx, reader, entry)
		if err != nil {
			return err
		}
		if !isHeld {
			listed = append(listed, entry)
		}
	}
	return u.absent.PutAbsentFiles(ctx, v, listed)
}

func checkHeld(ctx context.Context, reader vaultport.VaultReader, entry vault.AbsentFile) (bool, error) {
	stat, err := reader.StatFile(ctx, entry.Path)
	if edit.IsNoNote(err) || err == nil && stat.IsAbsent {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	hash, err := hashFile(ctx, reader, entry.Path)
	if err != nil {
		return false, err
	}
	if !strings.EqualFold(hash, entry.Hash) {
		return false, fmt.Errorf("list %s as absent: %w", entry.Path, vault.ErrFilePresent)
	}
	return true, nil
}

func hashFile(ctx context.Context, reader vaultport.VaultReader, path string) (_ string, err error) {
	file, err := reader.Open(ctx, path)
	if err != nil {
		return "", err
	}
	defer func() { err = errors.Join(err, file.Close()) }()
	return domain.HashReader(file)
}

// FillFile places the bytes of a store file at an absent path once their SHA-256 equals the hash of
// the entry.
type FillFile struct {
	absent  vaultport.AbsentFiles
	writers vaultport.FolderWriters
	store   StoreFiles
	index   edit.IndexUpdater
}

func NewFillFile(
	absent vaultport.AbsentFiles, writers vaultport.FolderWriters, store StoreFiles, index edit.IndexUpdater,
) *FillFile {
	return &FillFile{absent: absent, writers: writers, store: store, index: index}
}

// Execute checks the bytes of the store file, writes them to the vault path in one atomic step, and
// removes the entry. A file of other bytes returns vault.ErrHashMismatch and places nothing; a path
// that is not listed returns edit.ErrNotExist.
func (u *FillFile) Execute(ctx context.Context, v domain.Vault, path, storeName string) error {
	entry, isListed, err := u.absent.GetAbsentFile(ctx, v, path)
	if err != nil {
		return err
	}
	if !isListed {
		return fmt.Errorf("fill %s: not listed as absent: %w", path, edit.ErrNotExist)
	}
	if err := u.check(ctx, storeName, entry); err != nil {
		return err
	}
	if err := u.place(ctx, v, path, storeName); err != nil {
		return err
	}
	if err := u.absent.RemoveAbsentFiles(ctx, v, []string{path}); err != nil {
		return err
	}
	return edit.WrapIndexOutdated(u.index(ctx, v, []string{path}), path)
}

func (u *FillFile) check(ctx context.Context, storeName string, entry vault.AbsentFile) (err error) {
	source, err := u.store.Open(ctx, storeName)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, source.Close()) }()
	hash, err := domain.HashReader(source)
	if err != nil {
		return err
	}
	if !strings.EqualFold(hash, entry.Hash) {
		return fmt.Errorf("fill %s: %w", entry.Path, vault.ErrHashMismatch)
	}
	return nil
}

func (u *FillFile) place(ctx context.Context, v domain.Vault, path, storeName string) (err error) {
	writer, err := u.writers.OpenFolderWriter(v)
	if err != nil {
		return err
	}
	source, err := u.store.Open(ctx, storeName)
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, source.Close()) }()
	return writer.Import(ctx, path, source)
}

// EvictFile deletes the bytes of a file whose hash is the expected one and lists the file as absent.
type EvictFile struct {
	readers vaultport.VaultReaders
	absent  vaultport.AbsentFiles
	writers EvictWriters
}

func NewEvictFile(readers vaultport.VaultReaders, absent vaultport.AbsentFiles, writers EvictWriters) *EvictFile {
	return &EvictFile{readers: readers, absent: absent, writers: writers}
}

// Execute lists the file at path for the provider and deletes its bytes. The listing comes first, so a
// crash between the two leaves a listed file with bytes and loses nothing. A file of another hash
// returns vault.ErrChanged and is kept. The entry carries the size and time of the file, so the index
// sees no change.
func (u *EvictFile) Execute(ctx context.Context, v domain.Vault, path, expectedHash, provider string) error {
	reader, err := u.readers.Open(v)
	if err != nil {
		return err
	}
	stat, err := reader.StatFile(ctx, path)
	if err != nil {
		return err
	}
	if stat.IsAbsent {
		return nil
	}
	hash, err := hashFile(ctx, reader, path)
	if err != nil {
		return err
	}
	if !strings.EqualFold(hash, expectedHash) {
		return fmt.Errorf("drop %s: %w", path, vault.ErrChanged)
	}
	entry := vault.AbsentFile{Path: path, Size: stat.Size, Mtime: stat.ModTime, Hash: hash, Provider: provider}
	if err := u.absent.PutAbsentFiles(ctx, v, []vault.AbsentFile{entry}); err != nil {
		return err
	}
	mover, err := u.writers.OpenFileMover(v)
	if err != nil {
		return err
	}
	return mover.Remove(ctx, path)
}

// ListAbsentFiles lists the files the vault shows and does not hold.
type ListAbsentFiles struct{ absent vaultport.AbsentFiles }

func NewListAbsentFiles(absent vaultport.AbsentFiles) *ListAbsentFiles {
	return &ListAbsentFiles{absent: absent}
}

func (u *ListAbsentFiles) Execute(ctx context.Context, v domain.Vault) ([]vault.AbsentFile, error) {
	return u.absent.ListAbsentFiles(ctx, v)
}
