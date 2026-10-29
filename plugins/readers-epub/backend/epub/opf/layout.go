package opf

// drawn currently.
type Layout string

const (
	Reflowable   Layout = "reflowable"
	PrePaginated Layout = "pre-paginated"
)

// A Direction is the direction a book's pages progress in.
type Direction string

const (
	// DefaultDirection: the book specifies none, and a reader lays it out in the direction
	// of its language.
	DefaultDirection Direction = ""
	LeftToRight      Direction = "ltr"
	RightToLeft      Direction = "rtl"
)
