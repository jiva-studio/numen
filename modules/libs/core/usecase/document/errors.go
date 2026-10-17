package document

import "github.com/jiva-studio/numen/modules/libs/core/port"

// ErrNotADocument is bytes that are not the format at all, ErrEncrypted is a
// document whose text is locked, and ErrNoEntry is an entry the archive of a book
// lacks.
var (
	ErrNotADocument = port.ErrNotADocument
	ErrEncrypted    = port.ErrEncrypted
	ErrNoEntry      = port.ErrNoBookEntry
)
