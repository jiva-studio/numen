package settings

import "github.com/jiva-studio/numen/libs/note/domain"

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
