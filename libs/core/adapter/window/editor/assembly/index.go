package assembly

import (
	sourceport "github.com/jiva-studio/numen/libs/core/usecase/filetext/sourcecontract"
	indexport "github.com/jiva-studio/numen/libs/core/usecase/indexing/indexcontract"
	noteport "github.com/jiva-studio/numen/libs/core/usecase/note/notecontract"
	problemport "github.com/jiva-studio/numen/libs/core/usecase/note/problemcontract"
	textport "github.com/jiva-studio/numen/libs/core/usecase/textversion/textcontract"
)

type indexNoteAssembly interface {
	GetNoteQueries() noteport.NoteQueries
	GetLinkQueries() noteport.LinkQueries
	GetProblemQueries() problemport.ProblemQueries
	GetIndexProgress() indexport.IndexProgress
}

type indexSourceAssembly interface {
	GetSourceRepository() sourceport.SourceRepository
	GetSourceQueries() sourceport.SourceQueries
	GetPassageQueries() sourceport.PassageQueries
}

// IndexAssembly provides the cache and index queries of an installation to this window.
type IndexAssembly interface {
	indexNoteAssembly
	indexSourceAssembly
	CloseIndex() error
	SetTextVersionListener(listener textport.VersionListener)
	IsRebuildingIndex() bool
}
