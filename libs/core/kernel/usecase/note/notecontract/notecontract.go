// Package notecontract re-exports the note port contract.
package notecontract

import "github.com/jiva-studio/numen/libs/core/kernel/port/noteport"

type (
	LinkQueries    = noteport.LinkQueries
	NoteRepository = noteport.NoteRepository
)
