package note

import (
	"github.com/jiva-studio/numen/libs/core/internal/chunking"
	"github.com/jiva-studio/numen/libs/core/port/noteport"
	"github.com/jiva-studio/numen/libs/core/port/windowport"
	"github.com/jiva-studio/numen/libs/core/usecase/note/query"
	"github.com/jiva-studio/numen/libs/notes/edit"
)

const (
	ByFrontmatter = query.ByFrontmatter
	ByFilename    = query.ByFilename
)

type (
	FocusNote            = query.FocusNote
	GetBacklinks         = query.GetBacklinks
	GetConnection        = query.GetConnection
	GetHeadings          = query.GetHeadings
	GetLinkAddress       = query.GetLinkAddress
	ResolveLinks         = query.ResolveLinks
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
	ErrInvalidName = query.ErrInvalidName
	ErrNoNotePath  = query.ErrNoNotePath
)

func NewCutNotes(maxTokens int) chunking.CutNotes { return query.NewCutNotes(maxTokens) }

func NewFocusNote(queries noteport.NoteQueries, view windowport.Window) *query.FocusNote {
	return query.NewFocusNote(queries, view)
}

func NewGetBacklinks(links noteport.LinkQueries) query.GetBacklinks {
	return query.NewGetBacklinks(links)
}

func NewGetConnection(links noteport.LinkQueries) query.GetConnection {
	return query.NewGetConnection(links)
}

func NewGetHeadings(queries noteport.NoteQueries) *query.GetHeadings {
	return query.NewGetHeadings(queries)
}

func NewGetLinkAddress(names edit.NameQueries) *query.GetLinkAddress {
	return query.NewGetLinkAddress(names)
}

func NewResolveLinks(links noteport.LinkQueries) *query.ResolveLinks {
	return query.NewResolveLinks(links)
}

func NewGetNamedPaths(queries noteport.NoteQueries) *query.GetNamedPaths {
	return query.NewGetNamedPaths(queries)
}

func NewGetOpeningNote(queries noteport.NoteQueries) *query.GetOpeningNote {
	return query.NewGetOpeningNote(queries)
}

func NewGetTitle(queries noteport.NoteQueries) *query.GetTitle { return query.NewGetTitle(queries) }

func NewGetTitles(queries noteport.NoteQueries) query.GetTitles { return query.NewGetTitles(queries) }

func NewGetTypes(queries noteport.NoteQueries) *query.GetTypes { return query.NewGetTypes(queries) }

func NewIsVaultIndexed(queries noteport.NoteQueries) *query.IsVaultIndexed {
	return query.NewIsVaultIndexed(queries)
}

func NewShowLinks(links noteport.LinkQueries) query.ShowLinks { return query.NewShowLinks(links) }

func NewShowNeighbourhood(links noteport.LinkQueries, notes query.RefQueries) query.ShowNeighbourhood {
	return query.NewShowNeighbourhood(links, notes)
}
