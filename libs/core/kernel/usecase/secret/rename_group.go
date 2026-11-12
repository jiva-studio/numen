package secret

import (
	"context"
	"slices"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/plugin"
)

// RenameGroup changes the group name for all secrets in a custom group.
type RenameGroup struct{ Operation }

// NewRenameGroup makes the RenameGroup use case over the operation.
func NewRenameGroup(op Operation) *RenameGroup { return &RenameGroup{Operation: op} }

// Execute renames the group in one write.
func (u *RenameGroup) Execute(ctx context.Context, group, name string) error {
	if err := checkName("group", name); err != nil {
		return err
	}
	return u.apply(ctx, func() ([]string, error) {
		requested := plugin.ListAskedSecretGroups(u.plugins.ListDeclarations())
		if slices.Contains(requested, group) {
			return nil, newInvalidError("a group an installed plugin asks for cannot be renamed")
		}
		if slices.Contains(requested, name) {
			return nil, newInvalidError("the name is the name of a group an installed plugin asks for")
		}
		return nil, u.ledger.ChangeSecrets(func(list []plugin.Secret) ([]plugin.Secret, error) {
			return renameGroup(list, group, name)
		})
	})
}

func renameGroup(list []plugin.Secret, group, name string) ([]plugin.Secret, error) {
	if !slices.ContainsFunc(list, func(one plugin.Secret) bool { return one.Group == group }) {
		return nil, newNotFoundError(group)
	}
	if group == name {
		return list, nil
	}
	if slices.ContainsFunc(list, func(one plugin.Secret) bool { return one.Group == name }) {
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
