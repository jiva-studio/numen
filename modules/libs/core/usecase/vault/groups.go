package vault

import (
	"context"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
)

// A group of notes is one write. The count is where the gain flattens out; the
// size bounds what a vault of long files holds in memory before writing any of
// it.
const (
	notesPerWrite = 500
	bytesPerWrite = 8 << 20
)

// grouping collects parsed notes and writes them a group at a time, so that
// neither what is held in memory nor the length of one write grows with the
// number of notes handed to it.
//
// A walk of a whole vault and a handful of named files are the same job at
// different sizes: a folder dropped into a watched vault arrives as one event
// naming thousands of notes.
type grouping struct {
	write func(context.Context, []domain.IndexedNote) error

	notes []domain.IndexedNote
	bytes int
}

// add takes one note and its size on disk, writing the group when either bound
// is reached.
func (g *grouping) add(ctx context.Context, note domain.IndexedNote, size int) error {
	g.notes = append(g.notes, note)
	g.bytes += size
	if len(g.notes) >= notesPerWrite || g.bytes >= bytesPerWrite {
		return g.flush(ctx)
	}
	return nil
}

// flush writes whatever is held, and does nothing when nothing is.
func (g *grouping) flush(ctx context.Context) error {
	if len(g.notes) == 0 {
		return nil
	}
	if err := g.write(ctx, g.notes); err != nil {
		return err
	}
	g.notes, g.bytes = g.notes[:0], 0
	return nil
}
