package identity

import "errors"

type Status uint8

const (
	StatusActive Status = iota + 1
	StatusRevoked
)

// State is the record of a device joined to the chain it belongs to.
type State struct {
	Record Record
	Chain  Chain
	Status Status
}

func NewState(record Record, chain Chain) (State, error) {
	return State{}, errors.New("not implemented")
}

// Adopt takes a chain that extends the one of the state, finds this device in it and returns the next state.
func (s State) Adopt(next Chain) (State, error) {
	return State{}, errors.New("not implemented")
}
