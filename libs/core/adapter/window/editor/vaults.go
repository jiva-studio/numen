package editor

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// errNoVaults is what a build that serves no list of vaults answers.
var errNoVaults = errors.New("this build serves no list of vaults")

// vaultsHandler answers about the vaults this installation holds.
type vaultsHandler struct {
	state       *State
	list        *vaults.List
	find        *vaults.Find
	folderCheck *vaults.FolderCheck
	add         *vaults.Add
	rename      *vaults.Rename
	remove      *vaults.Remove

	// chooseFolder is nil where the application has no window to put a dialog in.
	chooseFolder *vaults.ChooseFolder
}

// ListVaults returns every vault of the installation. The active vault is
// queried from the window.
func (s *vaultsHandler) ListVaults(
	_ context.Context,
	_ *connect.Request[v1.ListVaultsRequest],
) (*connect.Response[v1.ListVaultsResponse], error) {
	if s.list == nil || s.folderCheck == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoVaults)
	}
	target, err := s.list.Execute()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListVaultsResponse{Vaults: make([]*v1.Vault, 0, len(target))}
	for _, v := range target {
		out.Vaults = append(out.Vaults, s.convertVaultToWire(v))
	}
	return connect.NewResponse(out), nil
}

// ChooseFolder opens this machine's native folder dialog.
func (s *vaultsHandler) ChooseFolder(
	ctx context.Context,
	r *connect.Request[v1.ChooseFolderRequest],
) (*connect.Response[v1.ChooseFolderResponse], error) {
	if s.chooseFolder == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoVaults)
	}
	selected, err := s.chooseFolder.Execute(ctx, r.Msg.GetTitle(), r.Msg.GetStartingAt())
	if vaults.IsFolderDialogUnavailable(err) {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	// A closed dialog is a valid response with IsChosen false.
	return connect.NewResponse(&v1.ChooseFolderResponse{Path: selected.Path, IsChosen: selected.IsChosen}), nil
}

// AddVault turns a folder into a vault on the list. A name another vault has
// gets a number appended, and is not an error here.
func (s *vaultsHandler) AddVault(
	_ context.Context,
	r *connect.Request[v1.AddVaultRequest],
) (*connect.Response[v1.AddVaultResponse], error) {
	if s.add == nil || s.folderCheck == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoVaults)
	}
	added, err := s.add.Execute(r.Msg.GetPath(), r.Msg.GetName())
	if err != nil {
		reason, ok := vaultsErrorCodeBy(err)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.AddVaultResponse{Error: &reason}), nil
	}
	return connect.NewResponse(&v1.AddVaultResponse{Vault: s.convertVaultToWire(added)}), nil
}

// RenameVault sets the display name of a vault. The folder name on the
// filesystem is unchanged.
func (s *vaultsHandler) RenameVault(
	ctx context.Context,
	r *connect.Request[v1.RenameVaultRequest],
) (*connect.Response[v1.RenameVaultResponse], error) {
	if s.find == nil || s.rename == nil || s.folderCheck == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoVaults)
	}
	v, err := s.find.Execute(r.Msg.GetId())
	if err == nil {
		v, err = s.rename.Execute(ctx, v, r.Msg.GetName())
	}
	if err != nil {
		reason, ok := vaultsErrorCodeBy(err)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.RenameVaultResponse{Error: &reason}), nil
	}
	return connect.NewResponse(&v1.RenameVaultResponse{Vault: s.convertVaultToWire(v)}), nil
}

// RemoveVault removes a vault from the list and the index, and moves its folder
// to the trash when the request sets the trash flag.
func (s *vaultsHandler) RemoveVault(
	ctx context.Context,
	r *connect.Request[v1.RemoveVaultRequest],
) (*connect.Response[v1.RemoveVaultResponse], error) {
	if s.remove == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoVaults)
	}
	err := s.remove.Execute(ctx, r.Msg.GetId(), s.state.GetCurrentVault().ID, r.Msg.GetTrash())
	if err != nil {
		reason, ok := vaultsErrorCodeBy(err)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.RemoveVaultResponse{Error: &reason}), nil
	}
	return connect.NewResponse(&v1.RemoveVaultResponse{}), nil
}

// OpenVault shows another vault in this window.
func (s *vaultsHandler) OpenVault(
	ctx context.Context,
	r *connect.Request[v1.OpenVaultRequest],
) (*connect.Response[v1.OpenVaultResponse], error) {
	if s.find == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoVaults)
	}
	v, err := s.find.Execute(r.Msg.GetId())
	if err == nil {
		err = s.state.OpenVault(ctx, v)
	}
	if err == nil {
		return connect.NewResponse(&v1.OpenVaultResponse{}), nil
	}
	// A closing window or a shutdown in progress stops the operation; the requested
	// vault is unchanged.
	if errors.Is(err, errWindowClosing) || errors.Is(err, errShutdownInProgress) {
		return nil, connect.NewError(connect.CodeUnavailable, err)
	}
	reason, ok := vaultsErrorCodeBy(err)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.OpenVaultResponse{Error: &reason}), nil
}

// convertVaultToWire is one vault as the schema carries it. A folder that is not
// there to be found is marked, and the vault stays on the list.
func (s *vaultsHandler) convertVaultToWire(v domain.Vault) *v1.Vault {
	return &v1.Vault{
		Id: string(v.ID), Name: v.Name, Path: v.Path,
		IsMissing: s.folderCheck.Execute(v),
	}
}

// vaultsErrorCodeBy returns the error code for a vault-list error and whether
// the error maps to one. Any other error is a failure of the list, the index or
// the host.
func vaultsErrorCodeBy(err error) (v1.VaultsErrorCode, bool) {
	switch {
	case errors.Is(err, vaults.ErrUnreadable):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNREADABLE, true
	case errors.Is(err, vaults.ErrCopy):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_COPY, true
	case errors.Is(err, domain.ErrOverlaps):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_OVERLAPS, true
	case errors.Is(err, vaults.ErrNameTaken):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_NAME_TAKEN, true
	case errors.Is(err, vaults.ErrLastVault):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_LAST_VAULT, true
	case errors.Is(err, vaults.ErrShowing):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_SHOWING, true
	case errors.Is(err, vaults.ErrUnknown), errors.Is(err, vaults.ErrNoName):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNKNOWN, true
	case errors.Is(err, vaults.ErrNoTrash):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_NO_TRASH, true
	case errors.Is(err, errAnswerPending):
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_ASKING, true
	default:
		return v1.VaultsErrorCode_VAULTS_ERROR_CODE_UNSPECIFIED, false
	}
}

// workspaceHandler serves the window's workspace state: the open tabs and the
// focus requests.
type workspaceHandler struct{ state *State }

// WriteOpenTabs stores the currently open tabs, as last reported by the window,
// for the vault in the window.
//
// It is the reverse direction of WatchFocus: WatchFocus delivers a focus request
// to the window, and WriteOpenTabs reports the window's current tabs.
func (h *workspaceHandler) WriteOpenTabs(
	ctx context.Context, r *connect.Request[v1.WriteOpenTabsRequest],
) (*connect.Response[v1.WriteOpenTabsResponse], error) {
	tabs := r.Msg.GetTabs()
	open := domain.OpenTabs{Tabs: make([]domain.Tab, 0, len(tabs)), ActiveID: r.Msg.GetActiveId()}
	for _, one := range tabs {
		open.Tabs = append(open.Tabs, domain.Tab{
			ID:    one.GetId(),
			Kind:  one.GetKind(),
			Path:  one.GetPath(),
			Title: one.GetTitle(),
			State: one.GetState(),
		})
	}
	h.state.openTabs.Store(&open)
	if h.state.NotifyOpenTabs != nil {
		h.state.NotifyOpenTabs(open)
	}
	if err := h.state.writeOpenTabs(ctx, open); err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.WriteOpenTabsResponse{}), nil
}

// ReadOpenTabs returns the tabs kept for the vault in the window.
func (h *workspaceHandler) ReadOpenTabs(
	ctx context.Context, _ *connect.Request[v1.ReadOpenTabsRequest],
) (*connect.Response[v1.ReadOpenTabsResponse], error) {
	open, err := h.state.readOpenTabs(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	if len(open.Tabs) == 0 && open.ActiveID == "" {
		return connect.NewResponse(&v1.ReadOpenTabsResponse{}), nil
	}
	out := &v1.ReadOpenTabsResponse{Tabs: make([]*v1.Tab, 0, len(open.Tabs)), ActiveId: open.ActiveID}
	for _, one := range open.Tabs {
		out.Tabs = append(out.Tabs, &v1.Tab{
			Id: one.ID, Kind: one.Kind, Path: one.Path, Title: one.Title, State: one.State,
		})
	}
	return connect.NewResponse(out), nil
}

// DetachTab requests that a tab be opened in a new window.
func (h *workspaceHandler) DetachTab(
	ctx context.Context, r *connect.Request[v1.DetachTabRequest],
) (*connect.Response[v1.DetachTabResponse], error) {
	if h.state.DetachTab == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errors.New("tab tear-off not supported"))
	}
	if err := h.state.DetachTab(ctx, r.Msg.GetTabId(), int(r.Msg.GetX()), int(r.Msg.GetY())); err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.DetachTabResponse{}), nil
}

// GetOpenTabs returns the open tabs as last reported by the window, or the
// empty set when the window has not reported any.
func (s *State) GetOpenTabs() domain.OpenTabs {
	if open := s.openTabs.Load(); open != nil {
		return *open
	}
	return domain.OpenTabs{}
}

// WatchFocus streams focus requests to the client until it disconnects.
func (h *workspaceHandler) WatchFocus(
	ctx context.Context,
	_ *connect.Request[v1.WatchFocusRequest],
	out *connect.ServerStream[v1.WatchFocusResponse],
) error {
	line, isDone := h.state.Places.subscribe()
	defer isDone()

	repeat := time.NewTicker(wire.Again)
	defer repeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-repeat.C:
			// An empty response is a keepalive that fails when the client has disconnected.
			if err := out.Send(&v1.WatchFocusResponse{}); err != nil {
				return err
			}
		case pos, open := <-line:
			if !open {
				return nil
			}
			if err := out.Send(&v1.WatchFocusResponse{
				Path:   pos.Path,
				Ranges: convertByteRangesToWire(pos.Ranges),
			}); err != nil {
				return err
			}
		}
	}
}

func convertByteRangesToWire(pos []domain.ByteRange) []*v1.Range {
	out := make([]*v1.Range, 0, len(pos))
	for _, span := range pos {
		out = append(out, &v1.Range{Start: int32(span.Start), End: int32(span.End)})
	}
	return out
}
