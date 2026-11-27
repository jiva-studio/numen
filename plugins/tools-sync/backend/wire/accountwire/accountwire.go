// Package accountwire answers the methods of the settings view about the account on the server.
package accountwire

import (
	"context"
	"errors"

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

// Use gives the methods the deletion and the export of the account.
func (a *Account) Use(deps accountdata.Deps) { a.deps = &deps }

func (a *Account) getDeps() (accountdata.Deps, error) {
	if a.deps == nil {
		return accountdata.Deps{}, errNoAccountData
	}
	return *a.deps, nil
}

var errNoAccountData = errors.New("the identity cannot be changed here")

// GetAccountState returns whether this device deleted the cloud data of the account. A device with no identity gets an
// empty answer.
func (a *Account) GetAccountState(ctx context.Context, _ api.GetAccountStateRequest) (api.GetAccountStateResponse, error) {
	deps, err := a.getDeps()
	if err != nil {
		return api.GetAccountStateResponse{}, err
	}
	state, err := accountdata.NewReadState(deps).Execute(ctx)
	if errors.Is(err, sessiondomain.ErrNoIdentity) {
		return api.GetAccountStateResponse{HasIdentity: false, Account: "", IsDeleted: false}, nil
	}
	if err != nil {
		return api.GetAccountStateResponse{}, err
	}
	return api.GetAccountStateResponse{HasIdentity: true, Account: state.Account, IsDeleted: state.IsDeleted}, nil
}

// DeleteCloudData asks the server to delete the account. A device with no identity gets ErrNoIdentity and the server is
// not asked.
func (a *Account) DeleteCloudData(ctx context.Context, _ api.DeleteCloudDataRequest) (api.DeleteCloudDataResponse, error) {
	deps, err := a.getDeps()
	if err != nil {
		return api.DeleteCloudDataResponse{}, err
	}
	return api.DeleteCloudDataResponse{}, accountdata.NewDeleteCloudData(deps).Execute(ctx)
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
