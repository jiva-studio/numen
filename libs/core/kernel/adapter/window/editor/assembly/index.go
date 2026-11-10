package assembly

import (
	"context"

	"github.com/jiva-studio/numen/libs/note/domain"

	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	sourceport "github.com/jiva-studio/numen/libs/core/kernel/usecase/filetext/sourcecontract"
	indexport "github.com/jiva-studio/numen/libs/core/kernel/usecase/indexing/indexcontract"
	problemport "github.com/jiva-studio/numen/libs/core/kernel/usecase/note/problemcontract"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note/query"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/search"
	textport "github.com/jiva-studio/numen/libs/core/kernel/usecase/textversion/textcontract"
	vaultport "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/vaultcontract"
	windowport "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault/windowcontract"
)

type indexNoteAssembly interface {
	OpenGetOpeningNote() *query.GetOpeningNote
	OpenGetTypes() *query.GetTypes
	OpenGetHeadings() *query.GetHeadings
	OpenResolveLinks() *query.ResolveLinks
	OpenGetTitles() query.GetTitles
	OpenGetNameMatches() *search.GetNameMatches
	OpenGetFolderEntries(readers vaultport.VaultReaders) *file.GetFolderEntries
	OpenGetFileKinds(readers vaultport.VaultReaders) *file.GetFileKinds
	OpenFocusNote(view windowport.Window) *query.FocusNote
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
