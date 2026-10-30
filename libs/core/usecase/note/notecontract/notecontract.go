// Package notecontract re-exports the note port contract.
package notecontract

import "github.com/jiva-studio/numen/libs/core/port/noteport"

type (
	LinkQueries    = noteport.LinkQueries
	NoteQueries    = noteport.NoteQueries
	NoteRepository = noteport.NoteRepository
)
