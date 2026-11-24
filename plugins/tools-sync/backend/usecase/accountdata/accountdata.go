// Package accountdata deletes the data of the account on the sync server, restores an account that is being deleted,
// and exports what the server holds of it. The notes in the vaults of this device stay in every case.
package accountdata

import (
	"context"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// Deps are what the use cases need. Deletion is kept by the sign-in.
type Deps struct {
	Identity   ports.Identity
	Challenges ports.DeleteChallenges
	Account    ports.AccountData
	Deletion   ports.Deletion
}

// State is whether the account is being deleted and when the server deletes it for good.
type State struct {
	Account    string
	IsDeleting bool
	DeleteAt   time.Time
}

// ReadState reads what the sign-in knows of the deletion of the account. A device with no identity gets
// session.ErrNoIdentity.
type ReadState struct{ deps Deps }

// NewReadState returns the use case over the dependencies.
func NewReadState(deps Deps) *ReadState { return &ReadState{deps: deps} }

// Execute returns the state of the deletion.
func (u *ReadState) Execute(ctx context.Context) (State, error) {
	public, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return State{}, err
	}
	deleteAt, isDeleting := u.deps.Deletion.GetDeletion()
	return State{Account: public.AccountFingerprint, IsDeleting: isDeleting, DeleteAt: deleteAt}, nil
}

// DeleteCloudData asks the server to delete the account after its grace period.
type DeleteCloudData struct{ deps Deps }

// NewDeleteCloudData returns the use case over the dependencies.
func NewDeleteCloudData(deps Deps) *DeleteCloudData { return &DeleteCloudData{deps: deps} }

// Execute signs the delete challenge of the server with the key of this device, sends it, and records that the account
// is deleting. It returns when the server deletes the account for good. A device with no identity gets
// session.ErrNoIdentity and the server is not asked.
func (u *DeleteCloudData) Execute(ctx context.Context) (time.Time, error) {
	public, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return time.Time{}, err
	}
	challenge, err := u.deps.Challenges.GetAccountDeleteChallenge(ctx, public.DeviceSignKey)
	if err != nil {
		return time.Time{}, err
	}
	signature, err := u.deps.Identity.SignAsDevice(ctx, challenge.Message)
	if err != nil {
		return time.Time{}, err
	}
	deleteAt, err := u.deps.Account.DeleteAccount(ctx, challenge.ID, signature)
	if err != nil {
		return time.Time{}, err
	}
	u.deps.Deletion.SetDeletion(deleteAt)
	return deleteAt, nil
}

// RestoreAccount cancels the deletion of the account.
type RestoreAccount struct{ deps Deps }

// NewRestoreAccount returns the use case over the dependencies.
func NewRestoreAccount(deps Deps) *RestoreAccount { return &RestoreAccount{deps: deps} }

// Execute signs in again with the request to restore. An account whose time ran out is a new account then, which has
// no subscription.
func (u *RestoreAccount) Execute(ctx context.Context) error {
	if _, err := u.deps.Identity.GetPublicIdentity(ctx); err != nil {
		return err
	}
	return u.deps.Deletion.Restore(ctx)
}

// ExportAccount reads what the server holds of the account.
type ExportAccount struct{ deps Deps }

// NewExportAccount returns the use case over the dependencies.
func NewExportAccount(deps Deps) *ExportAccount { return &ExportAccount{deps: deps} }

// Execute returns the JSON document of the server. A device with no identity gets session.ErrNoIdentity and the
// server is not asked.
func (u *ExportAccount) Execute(ctx context.Context) ([]byte, error) {
	if _, err := u.deps.Identity.GetPublicIdentity(ctx); err != nil {
		return nil, err
	}
	return u.deps.Account.ExportAccount(ctx)
}
