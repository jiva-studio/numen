package settingsport

import (
	"errors"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ErrNotASetting is a value the settings cannot be read out of again: one that
// is not JSON, one of the wrong shape, or a number past what its setting goes
// to. Nothing is written, and the value is the caller's to correct.
var ErrNotASetting = errors.New("the settings cannot be read out of that value")

// Setting is one setting of the file in which this installation is configured,
// with its value.
type Setting struct {
	// Path is the setting, as a path through the file.
	Path []string
	// JSON is the value stored at the path.
	JSON string
}

// Model is one model a setting that names a model can be set to.
type Model struct {
	// Path is the setting the model is read from. The model in force is the one whose
	// Name is stored there.
	Path []string

	// Name is the value stored there while this model is in force.
	Name string

	// Title is what is drawn on the row, and Group what the rows around it
	// stand under.
	Title string
	Group string

	// IsDefault marks the model an unconfigured installation runs on.
	IsDefault bool

	// Writes is what choosing it writes. A model that decides more than its own
	// name writes more than one setting.
	Writes []Setting

	// Presence is what this model's files are on this machine.
	Presence Presence

	// ReasoningLevels are the reasoning depths the model supports, from least to
	// most. Empty where it offers no choice.
	ReasoningLevels []domain.ReasoningLevel

	// DefaultReasoning is the level the model works at unasked, empty where it names none.
	DefaultReasoning string
}

// Presence describes where a model's files are on this machine and nothing else.
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
