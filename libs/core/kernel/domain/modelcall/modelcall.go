// Package modelcall holds the vocabulary of the calls plugins ask the core to carry out through their
// connections. The core knows an operation by its name; the payload and the frames are bytes of the
// contract the operation belongs to.
package modelcall

// Priority says whose action a call serves. The zero value is background, so a call that sets none
// waits behind the calls of a person's actions.
type Priority int

const (
	// PriorityBackground is work that serves no person's action, such as indexing.
	PriorityBackground Priority = iota
	// PriorityInteractive is a person's action, such as a search or a check of a line.
	PriorityInteractive
)

// Request is one call a plugin asks for: the name of the operation, the payload of its contract and whose
// action it serves.
type Request struct {
	Operation string
	Payload   []byte
	Priority  Priority
}

// Send receives the data frames of a call in order. An error of send ends the call.
type Send func(data []byte) error

// ProviderCall is the connection one call to a provider is made for: a snapshot read at the start of
// the call. Type is the id the manifest of the plugin gives the type, without the plugin id. Fields are the
// fields that are not secret, with the fixed values of the type, Secrets are the values of the secret
// fields by key, and Options are the values of the row fields the row of the call holds. The provider
// holds it for the length of the call.
type ProviderCall struct {
	ConnectionID string
	Type         string
	Fields       map[string]string
	Secrets      map[string]string
	Options      map[string]string
}

// Option is one value a row field can take. An empty title shows the value, and an empty group shows no
// heading.
type Option struct{ Value, Title, Group string }
