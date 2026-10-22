package editor

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/internal/adapterlib/wire"
	vaults "github.com/jiva-studio/numen/modules/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/modules/libs/notes/domain"
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

// ListVaults is every vault the installation holds. Which of them the window
// has in front of the person is asked of the window.
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

// ChooseFolder puts this machine's own folder dialog in front of the person.
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
	// A person who closed the dialog chose nothing, and that is an answer.
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

// RenameVault is what a person calls a vault. The folder keeps the name the
// filesystem gives it.
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

// RemoveVault takes a vault off the list and out of the index, and its folder
// to the trash when the request asks for it.
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
	// A window that is closing, or already settling what it owes, is what
	// stopped this, and the vault asked for is as it was.
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

// vaultsErrorCodeBy says which code an error about the list carries, and
// whether it carries one at all. Anything else is the list, the index or the
// machine being out of reach.
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

// workspaceHandler answers where the person is in the window: what they have open,
// and what is put in front of them.
type workspaceHandler struct{ state *State }

// WriteOpenTabs takes what currently open, as the window last said it.
//
// It is the other direction to WatchFocus: a place is put in front of the
// person there, and here the window says what is in front of them now.
func (h *workspaceHandler) WriteOpenTabs(
	_ context.Context, r *connect.Request[v1.WriteOpenTabsRequest],
) (*connect.Response[v1.WriteOpenTabsResponse], error) {
	tabs := r.Msg.GetTabs()
	open := domain.OpenTabs{Tabs: make([]domain.Tab, 0, len(tabs)), FrontID: r.Msg.GetFront()}
	for _, one := range tabs {
		tab := domain.Tab{
			ID:    one.GetId(),
			Kind:  one.GetKind(),
			Path:  one.GetPath(),
			Title: one.GetTitle(),
		}
		if doc := one.GetDocument(); doc != nil {
			tab.Document = &domain.DocumentProgress{
				Page:      int(doc.GetPage()),
				PageCount: int(doc.GetPageCount()),
			}
		}
		if book := one.GetBook(); book != nil {
			tab.Book = &domain.BookProgress{
				Offset:    int(book.GetOffset()),
				Page:      int(book.GetPage()),
				PageCount: int(book.GetPageCount()),
			}
		}
		if rec := one.GetRecording(); rec != nil {
			tab.Recording = &domain.RecordingProgress{
				TranscribedDuration: int(rec.GetTranscribedDurationMs()),
				Duration:            int(rec.GetDurationMs()),
			}
		}
		open.Tabs = append(open.Tabs, tab)
	}
	h.state.openTabs.Store(&open)
	if h.state.NotifyOpenTabs != nil {
		h.state.NotifyOpenTabs(open)
	}
	return connect.NewResponse(&v1.WriteOpenTabsResponse{}), nil
}

// DetachTab asks that a tab be opened in a new window.
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

// GetOpenTabs is what currently open, as the window last said. A window
// that has said nothing has nothing open as far as anyone here knows.
func (s *State) GetOpenTabs() domain.OpenTabs {
	if open := s.openTabs.Load(); open != nil {
		return *open
	}
	return domain.OpenTabs{}
}

// WatchFocus reports what something else asked to be put in front of the
// person, for as long as the client listens.
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
			// A place nobody asked for is empty, and is here to fail when
			// the client has gone.
			if err := out.Send(&v1.WatchFocusResponse{}); err != nil {
				return err
			}
		case pos, open := <-line:
			if !open {
				return nil
			}
			if err := out.Send(&v1.WatchFocusResponse{
				Path:  pos.Path,
				Spans: convertByteSpansToWire(pos.Spans),
			}); err != nil {
				return err
			}
		}
	}
}

func convertByteSpansToWire(pos []domain.ByteSpan) []*v1.Span {
	out := make([]*v1.Span, 0, len(pos))
	for _, span := range pos {
		out = append(out, &v1.Span{From: int32(span.From), To: int32(span.To)})
	}
	return out
}
