package assembly

import (
	"github.com/jiva-studio/numen/libs/core/port/indexport"
	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/recognitionport"
	"github.com/jiva-studio/numen/libs/core/port/sourceport"
)

// IndexAssembly is what this window asks the cache and index queries of an installation.
type IndexAssembly interface {
	// The cache, as the ports this window asks it, and the close that hands the
	// file back.
	GetNoteQueries() noteport.NoteQueries
	GetLinkQueries() noteport.LinkQueries
	GetProblemQueries() recognitionport.ProblemQueries
	GetIndexProgress() indexport.IndexProgress
	GetSourceRepository() sourceport.SourceRepository
	GetSourceQueries() sourceport.SourceQueries
	GetPassageQueries() sourceport.PassageQueries
	CloseIndex() error

	// IsRebuildingIndex is whether this launch was asked to read every file
	// into the cache again. It belongs to the vault the window opened on and to
	// no vault shown later.
	IsRebuildingIndex() bool
}
