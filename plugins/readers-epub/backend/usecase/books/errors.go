package books

// Kind is the class of a failed book request: the client tells the classes apart.
type Kind int

const (
	// KindInvalidArgument is a request that names no book, no entry or no CFI, or a file that is no EPUB.
	KindInvalidArgument Kind = iota + 1
	// KindNotFound is a book, a document, an entry or a place the book does not hold.
	KindNotFound
	// KindPermissionDenied is a path outside the vault or a vault the host does not show.
	KindPermissionDenied
	// KindFailedPrecondition is an encrypted book or a file that changed since the baseline.
	KindFailedPrecondition
	// KindUnavailable is a request after Close.
	KindUnavailable
)

// Error is the failure of a book request. Message is the text the client shows.
type Error struct {
	Kind    Kind
	Message string
}

func (e *Error) Error() string {
	return e.Message
}

func newError(kind Kind, message string) error {
	return &Error{Kind: kind, Message: message}
}
