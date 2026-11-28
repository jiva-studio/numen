// Package accountdata deletes the data of the account on the sync server and exports what the server holds of it. The
// notes in the vaults of this device stay in every case.
package accountdata

import (
	"context"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/ports"
)

// Deps are what the use cases need. Deletion is kept by the sign-in.
type Deps struct {
	Identity   ports.Identity
	Challenges ports.DeleteChallenges
	Account    ports.AccountData
	Deletion   ports.Deletion
}

// State is whether this device deleted the cloud data of the account.
type State struct {
	Account   string
	IsDeleted bool
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
	return State{Account: public.AccountFingerprint, IsDeleted: u.deps.Deletion.IsDeleted()}, nil
}

// DeleteCloudData asks the server to delete the account. Nothing restores it.
type DeleteCloudData struct{ deps Deps }

// NewDeleteCloudData returns the use case over the dependencies.
func NewDeleteCloudData(deps Deps) *DeleteCloudData { return &DeleteCloudData{deps: deps} }

// Execute signs the delete challenge of the server with the key of this device, sends it, and records that the cloud
// data is deleted. A device with no identity gets session.ErrNoIdentity and the server is not asked.
func (u *DeleteCloudData) Execute(ctx context.Context) error {
	public, err := u.deps.Identity.GetPublicIdentity(ctx)
	if err != nil {
		return err
	}
	challenge, err := u.deps.Challenges.GetAccountDeleteChallenge(ctx, public.DeviceSignKey)
	if err != nil {
		return err
	}
	signature, err := u.deps.Identity.SignAsDevice(ctx, challenge.Message)
	if err != nil {
		return err
	}
	if err := u.deps.Account.DeleteAccount(ctx, challenge.ID, signature); err != nil {
		return err
	}
	u.deps.Deletion.SetDeleted()
	return nil
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
