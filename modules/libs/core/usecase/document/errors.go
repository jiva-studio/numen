package document

import "github.com/jiva-studio/numen/modules/libs/core/port"

// ErrNotADocument is bytes that are not the format at all, and ErrEncrypted is a
// document whose text is locked.
var (
	ErrNotADocument = port.ErrNotADocument
	ErrEncrypted    = port.ErrEncrypted
)
