package editor

import (
	"github.com/jiva-studio/numen/modules/libs/core/port"
)

// IndexAssembly is what this window asks the cache and index queries of an installation.
type IndexAssembly interface {
	// The cache, as the ports this window asks it, and the close that hands the
	// file back.
	GetNoteQueries() port.NoteQueries
	GetLinkQueries() port.LinkQueries
	GetProblemQueries() port.ProblemQueries
	GetIndexProgress() port.IndexProgress
	GetSourceRepository() port.SourceRepository
	GetSourceQueries() port.SourceQueries
	GetPassageQueries() port.PassageQueries
	CloseIndex() error

	// IsRebuildingIndex is whether this launch was asked to read every file
	// into the cache again. It belongs to the vault the window opened on and to
	// no vault shown later.
	IsRebuildingIndex() bool
}
