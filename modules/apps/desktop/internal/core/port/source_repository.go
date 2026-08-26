package port

import (
	"context"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
)

// Source is one file with text that the index holds: what the file was when it
// was read, and what read it.
//
// Hash addresses the content, and Recipe names the extractor and the sizes that
// produced the source's chunks. Both are empty on a source nothing has taken
// text out of yet.
type Source struct {
	Ref    domain.FileRef
	Hash   string
	Recipe string

	// TextFrom names the producer of the text this source's chunks are places
	// in. Empty where the source's own bytes are the text, which is the
	// ordinary case.
	//
	// It is cleared by a write that records a fingerprint alone, because a file
	// that changed is a file whose reading was of other bytes.
	TextFrom string
}

// Chunk is one cut of a source's text, as it is handed to storage. Location is
// where it sits in the terms the source's own numbering uses, and is empty where
// the format named none.
//
// Text is indexed for the words it holds and is not kept: it is the text at
// Start for Length in the source, so a chunk whose text says something the
// source does not is a chunk that cannot be read back.
//
// Small are the chunks inside this one. A Chunk with none of its own is a large
// chunk all the same: what makes it large is that nothing encloses it.
type Chunk struct {
	Start    int
	Length   int
	Location string
	Text     string
	// Opens are the parts of the source that begin exactly where this chunk
	// does: what a section starting here is called. Empty for a chunk that
	// opens none, which is most of them.
	Opens []string
	Small []Chunk
}

// An Extraction is one source as reading it left it: the file, the recipe that
// read it, and the chunks its text was cut into.
type Extraction struct {
	Source Source
	Chunks []Chunk
}

// SourceRepository holds the sources of one vault. Every method takes a vault:
// one database holds them all, and a call that forgets its vault touches another
// vault's sources.
type SourceRepository interface {
	// SaveSource records what a file is now. A source saved with no recipe owes
	// its text: the file it names has changed, or nothing has read it yet.
	SaveSource(ctx context.Context, vaultID string, s Source) error

	// SaveExtraction records a source and replaces its chunks with the ones its
	// text was cut into. Both arrive in one write, so a recipe is never
	// recorded for chunks that are not there.
	SaveExtraction(ctx context.Context, vaultID string, e Extraction) error

	// RemoveSources takes out the sources of one kind at the paths given, and
	// everything derived from them. A source the vault no longer holds cannot be
	// read, so a passage naming it can never be shown: leaving it in the index
	// leaves a row that answers a search and then cannot be looked at.
	RemoveSources(ctx context.Context, vaultID string, kind domain.SourceKind, paths []string) error

	// MoveSources files what the vault held at one path under another, with
	// everything under it. The path of a source is this port's, whatever kind of
	// file it is, so a folder of notes and books travels in one write.
	//
	// A note is called by the filename it lands under where nothing inside the
	// file names it, and where the new filename is the name its own title is
	// filed under. A note carrying a name the filename is not takes that name
	// with it.
	MoveSources(ctx context.Context, vaultID, from, to string) error
}
