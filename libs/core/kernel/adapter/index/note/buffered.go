package note

import (
	"context"
	"sync"

	"github.com/jiva-studio/numen/libs/core/kernel/port/noteport"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// Buffered is the note repository of the index, keeping the paths of the notes written through it while
// the first walk is still reading the vault.
type Buffered struct {
	noteport.NoteRepository
	Writes
}

// Writes is every path written through the index while a walk is reading the vault.
type Writes struct {
	mu       sync.Mutex
	paths    []string
	recorded map[string]bool
	isOver   bool
}

// Save writes the notes and records their paths.
func (r *Buffered) Save(ctx context.Context, vaultID domain.VaultID, notes []domain.NoteChunks) error {
	for _, one := range notes {
		r.Record(one.Note.FileStat.Path)
	}
	return r.NoteRepository.Save(ctx, vaultID, notes)
}

func (r *Buffered) Remove(ctx context.Context, vaultID domain.VaultID, paths []string) error {
	for _, path := range paths {
		r.Record(path)
	}
	return r.NoteRepository.Remove(ctx, vaultID, paths)
}

// StartRecording holds the paths written through this, for the length of one walk.
func (w *Writes) StartRecording() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.isOver = false
	w.paths, w.recorded = nil, nil
}

// Record takes the path before the write it belongs to.
func (w *Writes) Record(path string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isOver || w.recorded[path] {
		return
	}
	if w.recorded == nil {
		w.recorded = map[string]bool{}
	}
	w.recorded[path] = true
	w.paths = append(w.paths, path)
}

// StopRecording is every path held, and the end of the holding.
func (w *Writes) StopRecording() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.isOver = true
	paths := w.paths
	w.paths, w.recorded = nil, nil
	return paths
}
