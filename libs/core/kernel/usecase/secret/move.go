package secret

import (
	"context"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/port/pluginport"
)

// Move changes the priority of a secret inside its group.
type Move struct{ operation }

// Execute puts the secret before the secret beforeID, or at the end of the list, which is the end
// of its group, where beforeID is empty.
func (u *Move) Execute(ctx context.Context, id, beforeID string) error {
	return u.apply(ctx, func() ([]string, error) {
		return nil, u.ledger.ChangeSecrets(func(list []pluginport.Secret) ([]pluginport.Secret, error) {
			return moveSecret(list, id, beforeID)
		})
	})
}

func moveSecret(list []pluginport.Secret, id, beforeID string) ([]pluginport.Secret, error) {
	index := getIndex(list, id)
	if index < 0 {
		return nil, newNotFoundError(id)
	}
	moved := list[index]
	if beforeID == id {
		return list, nil
	}
	rest := slices.Delete(slices.Clone(list), index, index+1)
	if beforeID == "" {
		return append(rest, moved), nil
	}
	target := getIndex(rest, beforeID)
	if target < 0 {
		return nil, newNotFoundError(beforeID)
	}
	if rest[target].Group != moved.Group {
		return nil, newInvalidError("a secret moves inside its own group")
	}
	return slices.Insert(rest, target, moved), nil
}
