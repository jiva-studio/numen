package indexing

import (
	"context"

	"github.com/jiva-studio/numen/libs/shared/note/domain"
)

// A group of notes is one write. The count is where the gain flattens out; the
// size bounds what a vault of long files holds in memory before writing any of
// it.
const (
	notesPerWrite = 500
	bytesPerWrite = 8 << 20
)

// grouper collects parsed notes and writes them one group at a time, so neither
// memory use nor the length of one write grows with the number of notes supplied.
//
// A walk of a whole vault and a handful of named files are the same job at
// different sizes: a folder dropped into a watched vault arrives as one event
// naming thousands of notes.
type grouper struct {
	write func(context.Context, []domain.Note) error

	notes []domain.Note
	bytes int
}

func (g *grouper) add(ctx context.Context, note domain.Note, size int) error {
	g.notes = append(g.notes, note)
	g.bytes += size
	if len(g.notes) >= notesPerWrite || g.bytes >= bytesPerWrite {
		return g.flush(ctx)
	}
	return nil
}

func (g *grouper) flush(ctx context.Context) error {
	if len(g.notes) == 0 {
		return nil
	}
	if err := g.write(ctx, g.notes); err != nil {
		return err
	}
	clear(g.notes)
	g.notes, g.bytes = g.notes[:0], 0
	return nil
}
