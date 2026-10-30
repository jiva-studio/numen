package note

import "github.com/jiva-studio/numen/libs/core/usecase/note/query"

const (
	ByFrontmatter = query.ByFrontmatter
	ByFilename    = query.ByFilename
)

type (
	FocusNote            = query.FocusNote
	GetBacklinks         = query.GetBacklinks
	GetConnection        = query.GetConnection
	GetHeadings          = query.GetHeadings
	GetNamedPaths        = query.GetNamedPaths
	GetOpeningNote       = query.GetOpeningNote
	GetTitle             = query.GetTitle
	GetTitles            = query.GetTitles
	GetTypes             = query.GetTypes
	IsVaultIndexed       = query.IsVaultIndexed
	ShowLinks            = query.ShowLinks
	NoteLinks            = query.NoteLinks
	ShowNeighbourhood    = query.ShowNeighbourhood
	RefQueries           = query.RefQueries
	NameSource           = query.NameSource
	SyncTitleAndFilename = query.SyncTitleAndFilename
	SyncSetting          = query.SyncSetting
	CutNotes             = query.CutNotes
)

var (
	NewCutNotes          = query.NewCutNotes
	NewFocusNote         = query.NewFocusNote
	NewGetBacklinks      = query.NewGetBacklinks
	NewGetConnection     = query.NewGetConnection
	NewGetHeadings       = query.NewGetHeadings
	NewGetNamedPaths     = query.NewGetNamedPaths
	NewGetOpeningNote    = query.NewGetOpeningNote
	NewGetTitle          = query.NewGetTitle
	NewGetTitles         = query.NewGetTitles
	NewGetTypes          = query.NewGetTypes
	NewIsVaultIndexed    = query.NewIsVaultIndexed
	NewShowLinks         = query.NewShowLinks
	NewShowNeighbourhood = query.NewShowNeighbourhood

	ErrInvalidName = query.ErrInvalidName
)
