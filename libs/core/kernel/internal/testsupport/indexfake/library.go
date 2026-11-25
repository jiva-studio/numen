package indexfake

import (
	"bytes"
	"context"
	"fmt"
	"github.com/jiva-studio/numen/libs/shared/note/edit"
	"io"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/jiva-studio/numen/libs/shared/note/domain"

	vaultdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/vault"
	"github.com/jiva-studio/numen/libs/core/kernel/port/vaultport"
)

// GetFirst and GetSecond are two vaults whose books share no words, so a
// chunk read from the wrong one is detected.
func GetFirst() domain.Vault { return domain.Vault{ID: "01FIRST", Name: "first", Path: "/first"} }

func GetSecond() domain.Vault { return domain.Vault{ID: "01SECOND", Name: "second", Path: "/second"} }

// GetSanskrit and GetLatin are the vocabularies of the two vaults.
func GetSanskrit() []string {
	return []string{"udyana", "vrksa", "bija", "jala", "pathin", "grama", "nadi", "parvata"}
}

func GetLatin() []string {
	return []string{"aqua", "terra", "ventus", "ignis", "silva", "mons", "flumen", "campus"}
}

// GetWords returns n words of one vocabulary. Each vault's books use their own
// vocabulary, so a chunk identifies its vault.
func GetWords(vocabulary []string, n int) string {
	out := make([]string, 0, n)
	for i := range n {
		out = append(out, vocabulary[i%len(vocabulary)])
	}
	return strings.Join(out, " ")
}

// Library is one vault as a set of files. It counts the reads per path, so a
// test can check that an unchanged file was not opened.
type Library struct {
	vaultport.VaultReader
	Files map[string]*File
	Reads map[string]int
}

// File is one file of a Library.
type File struct {
	Kind  domain.SourceKind
	Raw   []byte
	MTime int64
	// IsAbsent lists the file with its size and no bytes.
	IsAbsent bool
	Size     int64
}

// NewLibrary creates an empty vault.
func NewLibrary() *Library {
	return &Library{Files: map[string]*File{}, Reads: map[string]int{}}
}

// Put adds one file to the vault, with the modification time given.
func (l *Library) Put(path string, kind domain.SourceKind, raw []byte, mtime int64) {
	l.Files[path] = &File{Kind: kind, Raw: raw, MTime: mtime}
}

// PutAbsent lists a file whose bytes are not on this device.
func (l *Library) PutAbsent(path string, kind domain.SourceKind, size, mtime int64) {
	l.Files[path] = &File{Kind: kind, MTime: mtime, IsAbsent: true, Size: size}
}

// Fill gives the absent file at path its bytes.
func (l *Library) Fill(path string, raw []byte) {
	l.Files[path].Raw, l.Files[path].IsAbsent = raw, false
}

func (l *Library) WalkFiles(_ context.Context, fn func(domain.FileStat) error) error {
	for _, path := range slices.Sorted(maps.Keys(l.Files)) {
		if err := fn(l.getStat(path)); err != nil {
			return err
		}
	}
	return nil
}

func (l *Library) Read(_ context.Context, path string) ([]byte, error) {
	target, ok := l.Files[path]
	if !ok {
		return nil, fmt.Errorf("read %s: %w", path, edit.ErrNotExist)
	}
	if target.IsAbsent {
		return nil, fmt.Errorf("read %s: %w", path, vaultdomain.ErrFileAbsent)
	}
	l.Reads[path]++
	return target.Raw, nil
}

func (l *Library) Open(_ context.Context, path string) (io.ReadSeekCloser, error) {
	target, ok := l.Files[path]
	if !ok {
		return nil, fmt.Errorf("open %s: %w", path, edit.ErrNotExist)
	}
	if target.IsAbsent {
		return nil, fmt.Errorf("open %s: %w", path, vaultdomain.ErrFileAbsent)
	}
	l.Reads[path]++
	return nopCloser{bytes.NewReader(target.Raw)}, nil
}

// nopCloser is a file held in memory, with nothing to release.
type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }

func (l *Library) StatFile(_ context.Context, path string) (domain.FileStat, error) {
	if _, ok := l.Files[path]; !ok {
		return domain.FileStat{}, fmt.Errorf("stat %s: %w", path, edit.ErrNotExist)
	}
	return l.getStat(path), nil
}

func (l *Library) getStat(path string) domain.FileStat {
	target := l.Files[path]
	size := int64(len(target.Raw))
	if target.IsAbsent {
		size = target.Size
	}
	return domain.FileStat{
		Path: path, Kind: target.Kind, Size: size, ModTime: time.Unix(0, target.MTime), IsAbsent: target.IsAbsent,
	}
}

// Vaults opens the reader of each vault a test set up.
type Vaults map[domain.VaultID]*Library

func (v Vaults) Open(vault domain.Vault) (vaultport.VaultReader, error) {
	reader, ok := v[vault.ID]
	if !ok {
		return nil, fmt.Errorf("no vault %s", vault.ID)
	}
	return reader, nil
}
