package secret

import (
	"context"
	"slices"
)

// RenameGroup changes the group of every secret of a group the person made.
type RenameGroup struct{ operation }

// Execute renames the group in one write. It fails where an installed plugin asks for the group,
// where it has no secret, or where the new name is the name of another group.
func (u *RenameGroup) Execute(ctx context.Context, group, name string) error {
	if err := checkName("group", name); err != nil {
		return err
	}
	return u.apply(ctx, func() ([]string, error) {
		requested := ListAskedGroups(u.plugins.ListDeclarations())
		if slices.Contains(requested, group) {
			return nil, newInvalidError("a group an installed plugin asks for cannot be renamed")
		}
		if slices.Contains(requested, name) {
			return nil, newInvalidError("the name is the name of a group an installed plugin asks for")
		}
		return nil, u.ledger.ChangeSecrets(func(list []Secret) ([]Secret, error) {
			return renameGroup(list, group, name)
		})
	})
}

func renameGroup(list []Secret, group, name string) ([]Secret, error) {
	if !slices.ContainsFunc(list, func(one Secret) bool { return one.Group == group }) {
		return nil, newNotFoundError(group)
	}
	if group == name {
		return list, nil
	}
	if slices.ContainsFunc(list, func(one Secret) bool { return one.Group == name }) {
		return nil, newInvalidError("the name is the name of another group")
	}
	regrouped := slices.Clone(list)
	for index := range regrouped {
		if regrouped[index].Group == group {
			regrouped[index].Group = name
		}
	}
	return regrouped, nil
}
