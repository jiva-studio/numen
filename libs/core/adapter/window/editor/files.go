package editor

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/usecase/file"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	vaults "github.com/jiva-studio/numen/libs/core/usecase/vault"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

var errComingUp = errors.New("the vault is still coming up")

// errTooManyPaths is a filter naming more paths than are answered at once.
var errTooManyPaths = errors.New("that is more paths than are answered at once")

const mostAsked = 200

// eachOnce is the paths a client asked about, each named once.
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
	if len(out) > mostAsked {
		return nil, connect.NewError(connect.CodeInvalidArgument, errTooManyPaths)
	}
	return out, nil
}

// fileHandler answers what the vault holds and what a person does to it: its tree,
// and its folders.
type fileHandler struct {
	state            *State
	getFolderEntries *file.GetFolderEntries
	getFileKinds     *file.GetFileKinds
	createFolder     *file.CreateFolder
	move             *vaults.Move
	remove           *note.Remove
}

// ListFiles is what one folder of the vault holds. The vault settles the order
// the entries come in, and they cross in it.
func (h *fileHandler) ListFiles(
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

	out := &v1.ListFilesResponse{Entries: make([]*v1.Entry, 0, len(listing.Entries))}
	for _, one := range listing.Entries {
		out.Entries = append(out.Entries, &v1.Entry{
			Path:     one.Path,
			Name:     one.Name,
			IsFolder: one.IsFolder,
			IsNote:   one.Kind == domain.KindNote,
			Type:     convertNoteTypeToWire(listing.Types[one.Path]),
		})
	}
	return connect.NewResponse(out), nil
}

// ListFileKinds hands the client what the vault holds at each of those paths,
// so that a client holding a path opens what stands there in the editor made
// for it.
//
// Whether a note is there is read off the vault and the type off the index, or off the note
// where nothing has scanned it yet, so a file a plugin has just written opens
// in the editor made for it. A window standing
// on nothing holds no source, and a path with nothing at it is left out.
func (h *fileHandler) ListFileKinds(
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
		out.Kinds = append(out.Kinds, &v1.FileKind{
			Path:   one.Path,
			IsNote: one.Kind == domain.KindNote,
			Type:   convertNoteTypeToWire(listing.Types[one.Path]),
		})
	}
	return connect.NewResponse(out), nil
}

// CreateFolder puts an empty folder in the vault, with the folders above it.
func (h *fileHandler) CreateFolder(
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
func (h *fileHandler) MoveFile(
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
	if moved.IsLanded {
		out.Moved = newMoveResult(moved)
	}
	out.IsUnlevelled = h.state.isUnlevelled(err)
	if err != nil && !out.GetIsUnlevelled() {
		reason, ok := wire.GetErrorCode(err)
		switch {
		case ok:
			out.Error = &reason
		case !moved.IsLanded:
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		// The file is where it was sent, so where it went and which links were
		// repaired is the answer, whatever came apart after it landed.
	}
	return connect.NewResponse(out), nil
}

// getListCode is the code a folder that could not be listed is answered with.
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

// convertNoteTypeToWire is what a note is, as the schema carries it: the word its frontmatter
// holds. A note carrying no type of its own is an ordinary note, and is carried as no word.
func convertNoteTypeToWire(noteType domain.NoteType) string {
	if noteType == domain.TypeNote {
		return ""
	}
	return string(noteType)
}

// RemoveFile takes a file or a folder out of the vault. It goes to the trash,
// and a request that says so destroys a note.
func (h *fileHandler) RemoveFile(
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

	removed, err := h.removal(ctx, showing, r.Msg.GetPath(), r.Msg.GetDestroy())
	unlevelled := h.state.isUnlevelled(err)
	if err != nil && !unlevelled {
		reason, ok := wire.GetErrorCode(err)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.RemoveFileResponse{Error: &reason}), nil
	}
	// The watcher reports only the paths the vault holds a source for. What
	// went is said here, so the tree drops the row whatever stood on it.
	h.state.ChangeEvents.publish(change{paths: []string{removed.Path}})
	return connect.NewResponse(&v1.RemoveFileResponse{
		Trashed:      removed.Trashed,
		Dangling:     removed.Dangling,
		IsUnlevelled: unlevelled,
	}), nil
}

// removal is the two ways something leaves the vault.
func (h *fileHandler) removal(
	ctx context.Context,
	v domain.Vault,
	path string,
	destroy bool,
) (note.RemoveResult, error) {
	if destroy {
		return h.remove.Destroy(ctx, v, path)
	}
	return h.remove.Execute(ctx, v, path)
}
