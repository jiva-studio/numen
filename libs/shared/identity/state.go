package identity

import (
	"fmt"
)

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

// NewState checks the chain and finds the device of the record in it: active, or revoked.
func NewState(record Record, chain Chain) (State, error) {
	if err := chain.Verify(); err != nil {
		return State{}, err
	}
	status, err := getStatus(chain, record.Device)
	if err != nil {
		return State{}, err
	}
	return State{Record: record, Chain: chain, Status: status}, nil
}

func getStatus(chain Chain, device DeviceKey) (Status, error) {
	sign := device.SignerKey()
	switch {
	case chain.IsActiveDevice(sign):
		return StatusActive, nil
	case chain.IsKnownDevice(sign):
		return StatusRevoked, nil
	default:
		return 0, fmt.Errorf("%w: the device of the record is not in its chain", ErrMalformed)
	}
}

// Adopt takes a chain that extends the one of the state, finds this device in it and returns the next state.
// An active device takes the wrapped keys of the chain when the epoch grew; a revoked device keeps its record.
func (s State) Adopt(next Chain) (State, error) {
	if err := next.Verify(); err != nil {
		return State{}, err
	}
	if !next.IsExtensionOf(s.Chain) {
		return State{}, ErrNotExtension
	}
	status, err := getStatus(next, s.Record.Device)
	if err != nil {
		return State{}, err
	}
	record := s.Record
	if status == StatusActive && next.Epoch() > s.Chain.Epoch() {
		wrapped, ok := next.GetWrappedFor(record.Device.SignerKey())
		if !ok {
			return State{}, fmt.Errorf("%w: the chain holds no keys for this device", ErrUnwrap)
		}
		record.Wrapped = wrapped
	}
	return State{Record: record, Chain: next, Status: status}, nil
}

// KeyEpoch returns the epoch of the account keys this device holds: the epoch of the chain for an active device, the last
// epoch it was given for a revoked one.
func (s State) KeyEpoch() uint32 { return s.Chain.heldEpoch(s.Record.Device.SignerKey()) }

// OpenAccountKeys unwraps the account keys of the record and loads the key encryption key of every epoch before theirs.
func (s State) OpenAccountKeys() (AccountKeys, error) {
	keys, err := s.Record.Device.UnwrapAccountKeys(s.Record.Wrapped)
	if err != nil {
		return AccountKeys{}, err
	}
	return s.Chain.openHistoryFrom(keys, s.KeyEpoch())
}
