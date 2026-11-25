// Package accountwire answers the methods of the settings view about the account on the server.
package accountwire

import (
	"context"
	"errors"
	"time"

	"github.com/jiva-studio/numen/plugins/tools-sync/backend/api"
	sessiondomain "github.com/jiva-studio/numen/plugins/tools-sync/backend/domain/session"
	"github.com/jiva-studio/numen/plugins/tools-sync/backend/usecase/accountdata"
)

// Account answers the methods of the settings view about the account on the server.
type Account struct {
	deps *accountdata.Deps
}

// New returns the methods with no account data until Use gives them.
func New() *Account { return &Account{deps: nil} }

// Use gives the methods the deletion, the restore and the export of the account.
func (a *Account) Use(deps accountdata.Deps) { a.deps = &deps }

func (a *Account) getDeps() (accountdata.Deps, error) {
	if a.deps == nil {
		return accountdata.Deps{}, errNoAccountData
	}
	return *a.deps, nil
}

var errNoAccountData = errors.New("the identity cannot be changed here")

// GetAccountState returns whether the account is being deleted and when the server deletes it. A device with no
// identity gets an empty answer.
func (a *Account) GetAccountState(ctx context.Context, _ api.GetAccountStateRequest) (api.GetAccountStateResponse, error) {
	deps, err := a.getDeps()
	if err != nil {
		return api.GetAccountStateResponse{}, err
	}
	state, err := accountdata.NewReadState(deps).Execute(ctx)
	if errors.Is(err, sessiondomain.ErrNoIdentity) {
		return api.GetAccountStateResponse{HasIdentity: false, Account: "", IsDeleting: false, DeleteAt: ""}, nil
	}
	if err != nil {
		return api.GetAccountStateResponse{}, err
	}
	return api.GetAccountStateResponse{HasIdentity: true, Account: state.Account, IsDeleting: state.IsDeleting, DeleteAt: formatTime(state.DeleteAt)}, nil
}

// DeleteCloudData asks the server to delete the account after its grace period. A device with no identity gets
// ErrNoIdentity and the server is not asked.
func (a *Account) DeleteCloudData(ctx context.Context, _ api.DeleteCloudDataRequest) (api.DeleteCloudDataResponse, error) {
	deps, err := a.getDeps()
	if err != nil {
		return api.DeleteCloudDataResponse{}, err
	}
	deleteAt, err := accountdata.NewDeleteCloudData(deps).Execute(ctx)
	if err != nil {
		return api.DeleteCloudDataResponse{}, err
	}
	return api.DeleteCloudDataResponse{DeleteAt: formatTime(deleteAt)}, nil
}

// RestoreAccount cancels the deletion of the account.
func (a *Account) RestoreAccount(ctx context.Context, _ api.RestoreAccountRequest) (api.RestoreAccountResponse, error) {
	deps, err := a.getDeps()
	if err != nil {
		return api.RestoreAccountResponse{}, err
	}
	return api.RestoreAccountResponse{}, accountdata.NewRestoreAccount(deps).Execute(ctx)
}

// ExportAccountData returns what the server holds of the account as a JSON document.
func (a *Account) ExportAccountData(ctx context.Context, _ api.ExportAccountDataRequest) (api.ExportAccountDataResponse, error) {
	deps, err := a.getDeps()
	if err != nil {
		return api.ExportAccountDataResponse{}, err
	}
	document, err := accountdata.NewExportAccount(deps).Execute(ctx)
	if err != nil {
		return api.ExportAccountDataResponse{}, err
	}
	return api.ExportAccountDataResponse{Document: string(document)}, nil
}

func formatTime(at time.Time) string {
	if at.IsZero() {
		return ""
	}
	return at.UTC().Format(time.RFC3339)
}
