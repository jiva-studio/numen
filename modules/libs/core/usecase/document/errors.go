package document

import "github.com/jiva-studio/numen/modules/libs/core/port"

// ErrNotADocument is bytes that are not the format at all, ErrEncrypted is a
// document whose text is locked, ErrNoBook is a path that holds no book, ErrNoEntry
// is an entry the archive of a book lacks, and ErrBookChanged is a book whose
// file is no longer the bytes an ask named. ErrUnavailable is a plugin that is
// not there to answer.
var (
	ErrNotADocument = port.ErrNotADocument
	ErrEncrypted    = port.ErrEncrypted
	ErrNoBook       = port.ErrNoBook
	ErrNoEntry      = port.ErrNoBookEntry
	ErrBookChanged  = port.ErrBookChanged
	ErrUnavailable  = port.ErrUnavailable
)
