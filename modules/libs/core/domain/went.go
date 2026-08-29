package domain

// Went is a note that is no longer where it was, and where it now is.
//
// The bytes do not change on the way, so this says nothing about what the note
// holds. It says only that a name stopped naming it.
type Went struct {
	From string
	To   string
}
