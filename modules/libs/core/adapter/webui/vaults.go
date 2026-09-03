package webui

import (
	"context"
	"errors"
	"os"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/port"
	usecase "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
)

// vaults answers about the vaults this installation holds, over the API this
// window serves.
type vaults struct{ api *API }

// errNoVaults is what a build that holds no list of vaults answers.
var errNoVaults = errors.New("this build holds no list of vaults")

// errNoChanging is what a build that cannot change that list answers.
var errNoChanging = errors.New("this build cannot change the vaults this installation holds")

// errNoPicker is what a build with no window to put a picker in front of
// answers.
var errNoPicker = errors.New("this build has no folder picker")

// errNoOpening is what a build that cannot move the window to another vault
// answers.
var errNoOpening = errors.New("this build cannot show another vault")

// List is every vault the installation holds, and which of them the window has
// in front of the person.
func (s vaults) List(
	_ context.Context,
	_ *connect.Request[v1.VaultsServiceListRequest],
) (*connect.Response[v1.VaultsServiceListResponse], error) {
	if s.api.Vaults == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoVaults)
	}
	held, err := usecase.List{Registry: s.api.Vaults}.Execute()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.VaultsServiceListResponse{
		Vaults:  make([]*v1.Known, 0, len(held)),
		Showing: string(s.api.Showing().ID),
	}
	for _, v := range held {
		out.Vaults = append(out.Vaults, knownOf(v))
	}
	return connect.NewResponse(out), nil
}

// Choose puts this machine's own folder picker in front of the person.
func (s vaults) Choose(
	ctx context.Context,
	r *connect.Request[v1.VaultsServiceChooseRequest],
) (*connect.Response[v1.VaultsServiceChooseResponse], error) {
	if s.api.Picker == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoPicker)
	}
	path, chose, err := s.api.Picker.Choose(ctx, r.Msg.GetTitle(), r.Msg.GetStartingAt())
	if errors.Is(err, port.ErrChoosing) || errors.Is(err, port.ErrNoFolderDialog) {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// A person who closed the picker chose nothing, and that is an answer.
	return connect.NewResponse(&v1.VaultsServiceChooseResponse{Path: path, Chose: chose}), nil
}

// Add turns a folder into a vault on the list. A name another vault has gets a
// number appended, and is not a refusal here.
func (s vaults) Add(
	_ context.Context,
	r *connect.Request[v1.VaultsServiceAddRequest],
) (*connect.Response[v1.VaultsServiceAddResponse], error) {
	if s.api.Adding == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoChanging)
	}
	added, err := s.api.Adding.Execute(r.Msg.GetPath(), r.Msg.GetName())
	if err != nil {
		refusal, refused := vaultRefusedBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.VaultsServiceAddResponse{Refusal: &refusal}), nil
	}
	return connect.NewResponse(&v1.VaultsServiceAddResponse{Vault: knownOf(added)}), nil
}

// Rename is what a person calls a vault. The folder keeps the name the
// filesystem gives it.
func (s vaults) Rename(
	ctx context.Context,
	r *connect.Request[v1.VaultsServiceRenameRequest],
) (*connect.Response[v1.VaultsServiceRenameResponse], error) {
	if s.api.Vaults == nil || s.api.Renaming == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoChanging)
	}
	v, err := s.found(r.Msg.GetId())
	if err == nil {
		v, err = s.api.Renaming.Execute(ctx, v, r.Msg.GetName())
	}
	if err != nil {
		refusal, refused := vaultRefusedBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.VaultsServiceRenameResponse{Refusal: &refusal}), nil
	}
	return connect.NewResponse(&v1.VaultsServiceRenameResponse{Vault: knownOf(v)}), nil
}

// Forget takes a vault off the list and out of the index. The folder stays
// where it is.
func (s vaults) Forget(
	ctx context.Context,
	r *connect.Request[v1.VaultsServiceForgetRequest],
) (*connect.Response[v1.VaultsServiceForgetResponse], error) {
	if s.api.Vaults == nil || s.api.Forgetting == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoChanging)
	}
	v, err := s.offTheList(r.Msg.GetId())
	if err == nil {
		err = s.api.Forgetting.Execute(ctx, v)
	}
	if err != nil {
		refusal, refused := vaultRefusedBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.VaultsServiceForgetResponse{Refusal: &refusal}), nil
	}
	return connect.NewResponse(&v1.VaultsServiceForgetResponse{}), nil
}

// Erase is Forget, and the folder goes to the place this machine keeps what a
// person deleted.
func (s vaults) Erase(
	ctx context.Context,
	r *connect.Request[v1.VaultsServiceEraseRequest],
) (*connect.Response[v1.VaultsServiceEraseResponse], error) {
	if s.api.Vaults == nil || s.api.Erasing == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoChanging)
	}
	v, err := s.offTheList(r.Msg.GetId())
	if err == nil {
		err = s.api.Erasing.Execute(ctx, v)
	}
	if err != nil {
		refusal, refused := vaultRefusedBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.VaultsServiceEraseResponse{Refusal: &refusal}), nil
	}
	return connect.NewResponse(&v1.VaultsServiceEraseResponse{}), nil
}

// Open shows another vault in this window.
func (s vaults) Open(
	ctx context.Context,
	r *connect.Request[v1.VaultsServiceOpenRequest],
) (*connect.Response[v1.VaultsServiceOpenResponse], error) {
	if s.api.Vaults == nil || s.api.Opens == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoOpening)
	}
	v, err := s.found(r.Msg.GetId())
	if err == nil {
		err = s.api.Opens(ctx, v)
	}
	if err == nil {
		return connect.NewResponse(&v1.VaultsServiceOpenResponse{}), nil
	}
	// A window that is closing, or already settling what it owes, is what
	// stopped this, and the vault asked for is as it was.
	if errors.Is(err, errGoing) || errors.Is(err, errSettling) {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	refusal, refused := vaultRefusedBy(err)
	if !refused {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.VaultsServiceOpenResponse{Refusal: &refusal}), nil
}

// offTheList is the vault an identity names, asked to leave. The vault the
// window has in front of the person stays: the use cases are not told which one
// that is, and this is.
func (s vaults) offTheList(id string) (domain.Vault, error) {
	v, err := s.found(id)
	if err != nil {
		return domain.Vault{}, err
	}
	if v.ID == s.api.Showing().ID {
		return domain.Vault{}, errShowing
	}
	return v, nil
}

// errShowing is the vault the window has in front of the person, asked to go.
var errShowing = errors.New("this vault is the one the window is showing")

// found is the vault an identity names.
func (s vaults) found(id string) (domain.Vault, error) {
	return usecase.Find{Registry: s.api.Vaults}.Execute(id)
}

// knownOf is one vault as the schema carries it. A folder that is not there to
// be found is marked, and the vault stays on the list.
func knownOf(v domain.Vault) *v1.Known {
	_, err := os.Stat(v.Path)
	return &v1.Known{Id: string(v.ID), Name: v.Name, Path: v.Path, Missing: err != nil}
}

// vaultRefusedBy says which refusal an error about the list is, and whether it
// is one at all. Anything else is the list, the index or the machine being out
// of reach.
func vaultRefusedBy(err error) (v1.VaultsRefusal, bool) {
	switch {
	case errors.Is(err, usecase.ErrUnreadable):
		return v1.VaultsRefusal_VAULTS_REFUSAL_UNREADABLE, true
	case errors.Is(err, usecase.ErrCopy):
		return v1.VaultsRefusal_VAULTS_REFUSAL_COPY, true
	case errors.Is(err, domain.ErrOverlaps):
		return v1.VaultsRefusal_VAULTS_REFUSAL_OVERLAPS, true
	case errors.Is(err, usecase.ErrNameTaken):
		return v1.VaultsRefusal_VAULTS_REFUSAL_NAME_TAKEN, true
	case errors.Is(err, usecase.ErrLastVault):
		return v1.VaultsRefusal_VAULTS_REFUSAL_LAST_VAULT, true
	case errors.Is(err, errShowing):
		return v1.VaultsRefusal_VAULTS_REFUSAL_SHOWING, true
	case errors.Is(err, usecase.ErrUnknown):
		return v1.VaultsRefusal_VAULTS_REFUSAL_UNKNOWN, true
	case errors.Is(err, port.ErrNoTrash):
		return v1.VaultsRefusal_VAULTS_REFUSAL_NO_TRASH, true
	case errors.Is(err, errAsking):
		return v1.VaultsRefusal_VAULTS_REFUSAL_ASKING, true
	default:
		return v1.VaultsRefusal_VAULTS_REFUSAL_UNSPECIFIED, false
	}
}
