package sourceport

import "errors"

// ErrNotADocument is bytes that are not a document the reader reads.
var ErrNotADocument = errors.New("not a document this reads")
