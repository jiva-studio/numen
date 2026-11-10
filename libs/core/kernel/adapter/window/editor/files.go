package editor

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/structpb"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/file"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/tabkind"
	vaults "github.com/jiva-studio/numen/libs/core/kernel/usecase/vault"
	"github.com/jiva-studio/numen/libs/note/domain"
)

var errComingUp = errors.New("the vault is still coming up")

// errTooManyPaths is a filter naming more paths than are answered at once.
var errTooManyPaths = errors.New("that is more paths than are answered at once")

const maxQueryLength = 200

func eachOnce(paths []string) ([]string, error) {
	observed := make(map[string]bool, len(paths))
	out := make([]string, 0, len(paths))
	for _, path := range paths {
		if observed[path] {
			continue
		}
		observed[path] = true
		out = append(out, path)
	}
	if len(out) > maxQueryLength {
		return nil, connect.NewError(connect.CodeInvalidArgument, errTooManyPaths)
	}
	return out, nil
}

// FileHandler serves what the vault holds and the operations on it: its tree and
// its folders.
type FileHandler struct {
	state            *State
	getFolderEntries *file.GetFolderEntries
	getFileKinds     *file.GetFileKinds
	chooseTabKind    *tabkind.Choose
	createFolder     *file.CreateFolder
	move             *vaults.Move
	remove           *note.Remove
	emptyTrash       *note.EmptyTrash
}

// ListFiles is what one folder of the vault holds. The vault settles the order
// the entries come in, and they cross in it.
func (h *FileHandler) ListFiles(
	ctx context.Context, r *connect.Request[v1.ListFilesRequest],
) (*connect.Response[v1.ListFilesResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	if h.getFolderEntries == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errComingUp)
	}
	listing, err := h.getFolderEntries.Execute(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(getListCode(err), err)
	}

	var notes []string
	for _, one := range listing.Entries {
		if one.Kind == domain.KindNote {
			notes = append(notes, one.Path)
		}
	}
	tabKinds, err := h.listTabKinds(ctx, showing, notes)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListFilesResponse{Entries: make([]*v1.Entry, 0, len(listing.Entries))}
	for _, one := range listing.Entries {
		out.Entries = append(out.Entries, &v1.Entry{
			Path:     one.Path,
			Name:     one.Name,
			IsFolder: one.IsFolder,
			IsNote:   one.Kind == domain.KindNote,
			Type:     convertNoteTypeToWire(listing.Types[one.Path]),
			TabKind:  tabKinds[one.Path],
		})
	}
	return connect.NewResponse(out), nil
}

// ListFileKinds returns what the vault holds at each given path, so the client
// opens each path in the editor made for it.
// Whether a note exists is read from the vault and its type from the index, or
// from the note itself when nothing has scanned it yet. A window with no vault
// has no source, and a path with nothing at it is omitted.
func (h *FileHandler) ListFileKinds(
	ctx context.Context, r *connect.Request[v1.ListFileKindsRequest],
) (*connect.Response[v1.ListFileKindsResponse], error) {
	showing := h.state.GetCurrentVault()
	if showing.ID == "" {
		return connect.NewResponse(&v1.ListFileKindsResponse{}), nil
	}
	if h.getFileKinds == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errComingUp)
	}
	paths, err := eachOnce(r.Msg.GetPaths())
	if err != nil {
		return nil, err
	}
	if h.state.EmitEvent != nil {
		for _, p := range paths {
			ext := strings.TrimPrefix(filepath.Ext(p), ".")
			if ext != "" {
				h.state.EmitEvent("onFile:" + ext)
			}
		}
	}
	listing, err := h.getFileKinds.Execute(ctx, showing, paths)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListFileKindsResponse{Kinds: make([]*v1.FileKind, 0, len(listing.Entries))}
	for _, one := range listing.Entries {
		out.Kinds = append(out.Kinds, h.newWireFileKind(one, listing))
	}
	return connect.NewResponse(out), nil
}

func (h *FileHandler) newWireFileKind(one domain.Entry, listing file.Folder) *v1.FileKind {
	kind := &v1.FileKind{
		Path:   one.Path,
		IsNote: one.Kind == domain.KindNote,
		Type:   convertNoteTypeToWire(listing.Types[one.Path]),
	}
	if properties, isNote := listing.Properties[one.Path]; isNote {
		kind.Properties = newWireProperties(properties)
		if h.chooseTabKind != nil {
			kind.TabKind = h.chooseTabKind.Execute(one.Path, listing.Types[one.Path], properties)
		}
	}
	return kind
}

// CreateFolder puts an empty folder in the vault, with the folders above it.
func (h *FileHandler) CreateFolder(
	ctx context.Context, r *connect.Request[v1.CreateFolderRequest],
) (*connect.Response[v1.CreateFolderResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	if h.createFolder == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errComingUp)
	}
	if !h.state.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer h.state.Writing.finish()

	out := &v1.CreateFolderResponse{}
	if err := h.createFolder.Execute(ctx, showing, r.Msg.GetPath()); err != nil {
		reason, ok := wire.GetErrorCode(err)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		out.Error = &reason
	}
	return connect.NewResponse(out), nil
}

// MoveFile puts a file or a folder somewhere else in the vault.
func (h *FileHandler) MoveFile(
	ctx context.Context, r *connect.Request[v1.MoveFileRequest],
) (*connect.Response[v1.MoveFileResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	if !h.state.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer h.state.Writing.finish()

	moved, err := h.move.Execute(ctx, showing, r.Msg.GetFrom(), r.Msg.GetTo())
	out := &v1.MoveFileResponse{}
	if moved.IsMoved {
		out.Moved = newMoveResult(moved)
	}
	out.IsIndexOutdated = h.state.isIndexOutdated(err)
	if err != nil && !out.GetIsIndexOutdated() {
		reason, ok := wire.GetErrorCode(err)
		switch {
		case ok:
			out.Error = &reason
		case !moved.IsMoved:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		// The file is where it was sent, so where it went and which links were
		// repaired is the answer, whatever came apart after it landed.
	}
	return connect.NewResponse(out), nil
}

func getListCode(err error) connect.Code {
	switch {
	case errors.Is(err, file.ErrOutside):
		return connect.CodeInvalidArgument
	case errors.Is(err, fs.ErrNotExist):
		return connect.CodeNotFound
	default:
		return connect.CodeInternal
	}
}

func convertNoteTypeToWire(noteType domain.NoteType) string {
	if noteType == domain.TypeNote {
		return ""
	}
	return string(noteType)
}

// RemoveFile moves a file or a folder of the vault to the trash.
func (h *FileHandler) RemoveFile(
	ctx context.Context, r *connect.Request[v1.RemoveFileRequest],
) (*connect.Response[v1.RemoveFileResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	if !h.state.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer h.state.Writing.finish()

	removed, err := h.remove.Execute(ctx, showing, r.Msg.GetPath())
	indexOutdated := h.state.isIndexOutdated(err)
	if err != nil && !indexOutdated {
		reason, ok := wire.GetErrorCode(err)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.RemoveFileResponse{Error: &reason}), nil
	}
	// The watcher reports only the paths the vault holds a source for. The removal is
	// published here, so the tree drops the row whatever it held.
	h.state.ChangeEvents.publish(window.Change{Paths: []string{removed.Path}})
	return connect.NewResponse(&v1.RemoveFileResponse{
		Trashed:         removed.Trashed,
		Dangling:        removed.Dangling,
		IsIndexOutdated: indexOutdated,
	}), nil
}

// EmptyTrash deletes everything the trash of the vault holds and answers the
// paths it could not delete.
func (h *FileHandler) EmptyTrash(
	ctx context.Context, _ *connect.Request[v1.EmptyTrashRequest],
) (*connect.Response[v1.EmptyTrashResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	if h.emptyTrash == nil {
		return nil, connect.NewError(connect.CodeUnavailable, errComingUp)
	}
	if !h.state.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer h.state.Writing.finish()

	undeleted, err := h.emptyTrash.Execute(ctx, showing)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	h.state.ChangeEvents.publish(window.Change{Paths: []string{note.TrashDir}})
	return connect.NewResponse(&v1.EmptyTrashResponse{Undeleted: undeleted}), nil
}

func newWireProperties(properties map[string]any) *structpb.Struct {
	raw, err := json.Marshal(properties)
	if err != nil {
		return &structpb.Struct{}
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return &structpb.Struct{}
	}
	wire, err := structpb.NewStruct(decoded)
	if err != nil {
		return &structpb.Struct{}
	}
	return wire
}
