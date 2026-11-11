package secret

import (
	"context"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// State reports whether the value of one secret is on this machine.
type State struct {
	ID       string
	HasValue bool
}

// ListStates reads which secrets have a value.
type ListStates struct{ Operation }

// NewListStates makes the ListStates use case over the operation.
func NewListStates(op Operation) *ListStates { return &ListStates{Operation: op} }

// Execute returns the state of every secret in list order. It reads the key store once for each
// secret, so only the page of secrets calls it. A secret whose value cannot be read has none; a
// canceled context fails the call.
func (u *ListStates) Execute(ctx context.Context) ([]State, error) {
	list, err := u.ledger.GetSecrets()
	if err != nil {
		return nil, err
	}
	states := make([]State, 0, len(list))
	for _, one := range list {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		states = append(states, State{ID: one.ID, HasValue: u.hasValue(ctx, one.ID)})
	}
	return states, nil
}

func (u *ListStates) hasValue(ctx context.Context, id string) bool {
	if u.keys == nil {
		return false
	}
	value, err := u.keys.GetKey(ctx, plugin.GetSecretEntryName(id))
	return err == nil && value != ""
}
