package port

import "errors"

// ErrNotASetting is a value the settings cannot be read out of again: one that
// is not JSON, one of the wrong shape, or a number past what its setting goes
// to. Nothing is written, and the value is the caller's to correct.
var ErrNotASetting = errors.New("the settings cannot be read out of that value")

// Setting is one setting of the file a person configures this installation in,
// and what to put there.
type Setting struct {
	// Path is the setting, as a path through the file.
	Path []string
	// JSON is what stands there.
	JSON string
}

// Model is one model a setting that names a model can be set to.
type Model struct {
	// Path is the setting the model is read from. The one in force is the
	// model whose Name stands there.
	Path []string

	// Name is what stands there while this model is the one in force.
	Name string

	// Title is what is drawn on the row, and Shelf what the rows around it
	// stand under.
	Title string
	Shelf string

	// IsDefault marks the model an installation nobody has configured runs on.
	IsDefault bool

	// Writes is what choosing it writes. A model that decides more than its own
	// name writes more than one setting.
	Writes []Setting

	// Presence is what this model's files are on this machine.
	Presence Presence
}

// Presence is what a model's files are on this machine. It says where files
// stand and nothing else.
type Presence int

const (
	// NothingToFetch is a model reached over the network. Nothing of it is
	// fetched.
	NothingToFetch Presence = iota
	// Present is a model whose files stand where a fetch puts them.
	Present
	// NotFetched is a model whose files are not on this machine. They are
	// fetched when the model is next needed.
	NotFetched
)
