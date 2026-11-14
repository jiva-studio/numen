package cards

import (
	"context"
	"crypto/rand"
	"errors"
	"sync"
	"time"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/edit"
	"github.com/jiva-studio/numen/plugins/views-flashcards-studio/backend/ports"
)

// newEdit is what a change to an existing note is made through: the vault it is read and written
// through, what updates the index for it, and what time it is. The function returned with it
// returns the failures to release the write lock of the changes made through it, which edit.Files
// releases without an error.
func newEdit(readers ports.VaultReaders, writers ports.VaultWriters, index edit.IndexUpdater, now ports.Clock) (edit.Edit, func() error) {
	failures := &releaseFailures{mu: sync.Mutex{}, err: nil}
	return edit.Edit{
		Open: func(v domain.Vault) (edit.Files, error) {
			reader, err := readers.Open(v)
			if err != nil {
				return nil, err
			}
			writer, err := writers.Open(v)
			if err != nil {
				return nil, err
			}
			return files{reader: reader, writer: writer, writers: writers, vault: v, failures: failures}, nil
		},
		Index:           index,
		Clock:           now,
		Entropy:         rand.Reader,
		FileStat:        domain.FileStat{Path: "", Kind: "", Size: 0, ModTime: time.Time{}},
		ShouldOverwrite: false,
		Bound:           0,
		Baseline:        nil,
	}, failures.getError
}

// files are the files of one vault, read and written through the ports.
type files struct {
	reader   ports.VaultReader
	writer   ports.VaultWriter
	writers  ports.VaultWriters
	vault    domain.Vault
	failures *releaseFailures
}

// releaseFailures joins the failures to release a write lock.
type releaseFailures struct {
	mu  sync.Mutex
	err error
}

func (r *releaseFailures) addError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.err = errors.Join(r.err, err)
}

func (r *releaseFailures) getError() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

func (f files) StatFile(ctx context.Context, path string) (domain.FileStat, error) {
	return f.reader.StatFile(ctx, path)
}

func (f files) Read(ctx context.Context, path string) ([]byte, error) {
	return f.reader.Read(ctx, path)
}

func (f files) Write(ctx context.Context, path string, content []byte, stat domain.FileStat) (domain.FileStat, error) {
	return f.writer.Write(ctx, path, content, stat)
}

func (f files) Lock(ctx context.Context) (func(), error) {
	release, err := f.writers.Lock(ctx, f.vault)
	if err != nil {
		return nil, err
	}
	return func() { f.failures.addError(release()) }, nil
}
