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
	Path []string
	JSON string
}

// Model is one model a setting that names a model can be set to.
type Model struct {
	Path []string

	Name string

	Title string
	Group string

	IsDefault bool

	Writes []Setting

	Presence Presence

	ReasoningLevels []domain.ReasoningLevel

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

// ChangeSubscriber starts listening to the settings that change. Each batch holds the settings that
// changed together, with the value each now holds, named by its declared path and delivered once
// to every subscriber. A path no owner declares is not delivered. The returned function stops the
// delivery and closes the channel.
type ChangeSubscriber func() (changes <-chan []Setting, stop func())
