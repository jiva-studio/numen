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

// NewState checks the chain, finds the device of the record in it, active or revoked, and checks that the wrapped keys of the record are keys of the account of the chain, of any of its epochs.
func NewState(record Record, chain Chain) (State, error) {
	if err := chain.Verify(); err != nil {
		return State{}, err
	}
	status, err := getStatus(chain, record.Device)
	if err != nil {
		return State{}, err
	}
	if err := checkAccount(record, chain); err != nil {
		return State{}, err
	}
	return State{Record: record, Chain: chain, Status: status}, nil
}

func checkAccount(record Record, chain Chain) error {
	opened, err := record.Device.UnwrapAccountKeys(record.Wrapped)
	if err != nil {
		return fmt.Errorf("%w: the wrapped account keys do not open for the device: %w", ErrMalformed, err)
	}
	if !chain.hasAccountBoxKey(opened.BoxPublicKey()) {
		return fmt.Errorf("%w: the wrapped account keys are not the account of the chain", ErrMalformed)
	}
	return nil
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

// Adopt takes a chain that extends the one of the state, or that outranks it at a fork, finds this device in it and returns the next
// state. An active device takes the wrapped keys of the chain when the chain gave it new ones or voided the ones it held, and the keys
// must be the keys of the chain; a revoked device keeps its record.
func (s State) Adopt(next Chain) (State, error) {
	if err := next.Verify(); err != nil {
		return State{}, err
	}
	isFork := !next.IsExtensionOf(s.Chain)
	if isFork && !next.IsPreferredOver(s.Chain) {
		return State{}, ErrNotExtension
	}
	status, err := getStatus(next, s.Record.Device)
	switch {
	case isFork && err != nil:
		return State{}, fmt.Errorf("%w: %w", ErrDeviceVoided, err)
	case err != nil:
		return State{}, err
	}
	record := s.Record
	if status == StatusActive {
		if record, err = s.takeKeys(next, isFork); err != nil {
			return State{}, err
		}
	}
	return State{Record: record, Chain: next, Status: status}, nil
}

func (s State) takeKeys(next Chain, isFork bool) (Record, error) {
	sign := s.Record.Device.SignerKey()
	shared := len(s.Chain.entries)
	if isFork {
		shared = next.getForkIndex(s.Chain)
	}
	isNew := next.getKeyIndex(sign) >= shared
	if !isNew && s.Chain.getKeyIndex(sign) < shared {
		return s.Record, nil
	}
	wrapped, ok := next.GetWrappedFor(sign)
	if !ok {
		return Record{}, fmt.Errorf("%w: the chain holds no keys for this device", ErrUnwrap)
	}
	record := s.Record
	record.Wrapped = wrapped
	keys, err := record.Device.UnwrapAccountKeys(wrapped)
	if err != nil {
		return Record{}, fmt.Errorf("%w: %w", ErrWrapMismatch, err)
	}
	if _, err := next.OpenKeyHistory(keys); err != nil {
		return Record{}, fmt.Errorf("%w: %w", ErrWrapMismatch, err)
	}
	return record, nil
}

// KeyEpoch returns the epoch of the account keys this device holds: the epoch of the chain for an active device, the last
// epoch it was given for a revoked one.
func (s State) KeyEpoch() uint32 { return s.Chain.getKeyEpoch(s.Record.Device.SignerKey()) }

// OpenAccountKeys unwraps the account keys of the record and loads the key encryption key of every epoch before theirs.
func (s State) OpenAccountKeys() (AccountKeys, error) {
	keys, err := s.Record.Device.UnwrapAccountKeys(s.Record.Wrapped)
	if err != nil {
		return AccountKeys{}, err
	}
	return s.Chain.openHistoryFrom(keys, s.KeyEpoch())
}
