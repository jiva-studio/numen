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
	// DefaultDirection: the book says nothing, and a reader lays it out the way
	// its language is written.
	DefaultDirection Direction = ""
	LeftToRight      Direction = "ltr"
	RightToLeft      Direction = "rtl"
)
