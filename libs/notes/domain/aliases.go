package domain

import (
	"github.com/jiva-studio/numen/libs/notes/domain/name"
)

type (
	// From name subpackage
	SourceKind = name.SourceKind
	FileStat   = name.FileStat
)

const (
	// From name subpackage
	KindNote      = name.KindNote
	KindFile      = name.KindFile
	NoteExtension = name.NoteExtension
)

var (
	// From name subpackage
	ErrInvalidName     = name.ErrInvalidName
	Filename           = name.Filename
	GetReducedFilename = name.GetReducedFilename
	ChooseMatch        = name.ChooseMatch
	FoldName           = name.FoldName
	Basename           = name.Basename
	LinkName           = name.LinkName
)
