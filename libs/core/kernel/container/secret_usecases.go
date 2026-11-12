package container

import "github.com/jiva-studio/numen/libs/core/kernel/usecase/secret"

// SecretUseCases is every use case on the list of secrets.
type SecretUseCases struct {
	Items  SecretItems
	Groups SecretGroups
	Access SecretAccess
	List   *secret.List
	Prune  *secret.Prune
}

// SecretItems are the use cases on one secret.
type SecretItems struct {
	Create *secret.Create
	Rename *secret.Rename
	Write  *secret.Write
	Delete *secret.Delete
	Move   *secret.Move
}

// SecretGroups are the use cases on a group of secrets.
type SecretGroups struct {
	Rename *secret.RenameGroup
	Delete *secret.DeleteGroup
}

// SecretAccess are the use cases on what each plugin may read.
type SecretAccess struct {
	Allow      *secret.Allow
	Deny       *secret.Deny
	ListStates *secret.ListStates
}
