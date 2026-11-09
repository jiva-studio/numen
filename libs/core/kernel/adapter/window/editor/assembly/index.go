package assembly

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	sourceport "github.com/jiva-studio/numen/libs/core/kernel/usecase/filetext/sourcecontract"
	indexport "github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing/indexcontract"
	noteport "github.com/jiva-studio/numen/libs/core/kernel/usecase/note/notecontract"
	problemport "github.com/jiva-studio/numen/libs/core/kernel/usecase/note/problemcontract"
	textport "github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion/textcontract"
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

// indexTextAssembly is how the window follows the text versions of the vault it shows.
type indexTextAssembly interface {
	SetTextVersionListener(listener textport.VersionListener)
	// ReportTextFilesChanged reports to the text version listener each changed file of the vault whose
	// live text version changed.
	ReportTextFilesChanged(ctx context.Context, v domain.Vault, paths []string) error
}

// IndexAssembly provides the cache and index queries of an installation to this window.
type IndexAssembly interface {
	indexNoteAssembly
	indexSourceAssembly
	indexTextAssembly
	CloseIndex() error
	IsRebuildingIndex() bool
}
