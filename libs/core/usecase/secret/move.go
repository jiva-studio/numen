package secret

import (
	"context"
	"slices"
)

// Move changes the priority of a secret inside its group.
type Move struct{ operation }

// Execute puts the secret before the secret beforeID, or at the end of the list, which is the end
// of its group, where beforeID is empty.
func (u *Move) Execute(ctx context.Context, id, beforeID string) error {
	return u.plugins.ApplySecretChange(ctx, func() ([]string, error) {
		return nil, u.ledger.ChangeSecrets(func(list []Secret) ([]Secret, error) {
			return moveSecret(list, id, beforeID)
		})
	})
}

func moveSecret(list []Secret, id, beforeID string) ([]Secret, error) {
	index := getIndex(list, id)
	if index < 0 {
		return nil, notFound(id)
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
		return nil, notFound(beforeID)
	}
	if rest[target].Group != moved.Group {
		return nil, invalid("a secret moves inside its own group")
	}
	return slices.Insert(rest, target, moved), nil
}
