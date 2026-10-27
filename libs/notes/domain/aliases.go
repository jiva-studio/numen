package domain

import (
	"github.com/jiva-studio/numen/libs/notes/domain/name"
)

type (
	// From name subpackage
	SourceKind  = name.SourceKind
	Fingerprint = name.Fingerprint
)

const (
	// From name subpackage
	KindNote      = name.KindNote
	KindBook      = name.KindBook
	KindRecording = name.KindRecording
	KindURL       = name.KindURL
	NoteExtension = name.NoteExtension
	URLExtension  = name.URLExtension
)

var (
	// From name subpackage
	ErrUnnameable      = name.ErrUnnameable
	Filename           = name.Filename
	GetReducedFilename = name.GetReducedFilename
	ChooseMatch        = name.ChooseMatch
	FoldName           = name.FoldName
	Basename           = name.Basename
	LinkName           = name.LinkName
)
