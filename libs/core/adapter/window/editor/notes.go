package editor

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ReadNote hands the client the prose of a note.
func (h *noteHandler) ReadNote(
	ctx context.Context, r *connect.Request[v1.ReadNoteRequest],
) (*connect.Response[v1.ReadNoteResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	found, err := h.read.Execute(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ReadNoteResponse{Body: found.Body}
	if reason, ok := wire.ConvertOutcomeToErrorCode(found.Outcome); ok {
		out.Error = &reason
	} else {
		// What the file was when this prose came out of it, for the client to
		// present when it puts prose back.
		out.At = convertFileStatToWire(found.FileStat)
	}
	return connect.NewResponse(out), nil
}

// WriteNote writes prose into a note. If the note still contains the prose the
// client read, it is overwritten; if it contains different content, it is left
// unchanged and the client is notified that the note changed.
func (h *noteHandler) WriteNote(
	ctx context.Context, r *connect.Request[v1.WriteNoteRequest],
) (*connect.Response[v1.WriteNoteResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	// The counter wraps the write, not the response, so closing waits for
	// writes that reach the vault.
	if !h.state.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer h.state.Writing.finish()
	pos, err := h.write.Save(ctx, showing, r.Msg.GetPath(), r.Msg.GetBody(), newBaseline(r.Msg.GetBaseline()))
	// A write that reached the vault has happened, so the response carries the
	// file stat the client uses at its next save. The response also sets
	// IsUnlevelled when the index did not follow, because the prose is on disk
	// but search does not include it.
	unlevelled := h.state.isUnlevelled(err)
	if err == nil || unlevelled {
		// The written prose requires new vectors. The pending chunks are
		// tracked in the index, so several saves are embedded in one pass.
		if h.state.NotifyNoteWritten != nil {
			h.state.NotifyNoteWritten()
		}
		return connect.NewResponse(&v1.WriteNoteResponse{
			At: convertFileStatToWire(pos), IsUnlevelled: unlevelled,
		}), nil
	}
	reason, ok := wire.GetErrorCode(err)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.WriteNoteResponse{Error: &reason}), nil
}

// CreateNote creates a note named after the given title and linked as the
// request specifies.
//
// The index is synchronized before the response, so the note is searchable as
// soon as it is on disk.
func (h *noteHandler) CreateNote(
	ctx context.Context, r *connect.Request[v1.CreateNoteRequest],
) (*connect.Response[v1.CreateNoteResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	links, err := h.getLinks(ctx, r.Msg.GetLinks())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if !h.state.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer h.state.Writing.finish()

	made, err := h.create.Execute(ctx, showing, note.NewNote{
		Title: r.Msg.GetTitle(),
		Path:  r.Msg.GetPath(),
		Links: links,
	})
	unlevelled := h.state.isUnlevelled(err)
	if made.Path != "" {
		// The note is on disk under that name, so the response carries it. The
		// index catches up after the write, and the watcher repeats that
		// update.
		return connect.NewResponse(&v1.CreateNoteResponse{
			Path: made.Path, IsUnlevelled: unlevelled,
		}), nil
	}
	if err == nil {
		return connect.NewResponse(&v1.CreateNoteResponse{}), nil
	}
	reason, ok := wire.GetErrorCode(err)
	if !ok {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.CreateNoteResponse{Error: &reason}), nil
}

// WriteLink writes one relationship into one note. What is already written
// there is carried across, and the note at the other end is left alone.
func (h *noteHandler) WriteLink(
	ctx context.Context, r *connect.Request[v1.WriteLinkRequest],
) (*connect.Response[v1.WriteLinkResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	link, err := h.toDomainLink(ctx, r.Msg.GetLink())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if !h.state.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer h.state.Writing.finish()

	// at the moment it is made.
	if _, err := h.editLinks.Add(
		ctx, showing, r.Msg.GetPath(), domain.FileStat{}, link,
	); err != nil {
		reason, ok := wire.GetErrorCode(err)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.WriteLinkResponse{Error: &reason}), nil
	}
	return connect.NewResponse(&v1.WriteLinkResponse{}), nil
}

// RemoveLink takes one relationship out of one note.
func (h *noteHandler) RemoveLink(
	ctx context.Context, r *connect.Request[v1.RemoveLinkRequest],
) (*connect.Response[v1.RemoveLinkResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	if !h.state.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer h.state.Writing.finish()

	role, _ := wire.ConvertRoleFromWire(r.Msg.GetRole())
	to, err := h.getAddress(ctx, r.Msg.GetTo())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	key := domain.LinkKey{Target: to, Role: role, Type: r.Msg.GetType()}
	if _, err := h.editLinks.Remove(ctx, showing, r.Msg.GetPath(), key, domain.FileStat{}); err != nil {
		reason, ok := wire.GetErrorCode(err)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.RemoveLinkResponse{Error: &reason}), nil
	}
	return connect.NewResponse(&v1.RemoveLinkResponse{}), nil
}

// ReplaceLinks makes the links between two notes the ones the request names.
func (h *noteHandler) ReplaceLinks(
	ctx context.Context, r *connect.Request[v1.ReplaceLinksRequest],
) (*connect.Response[v1.ReplaceLinksResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	links := make([]note.PairLink, 0, len(r.Msg.GetLinks()))
	for _, l := range r.Msg.GetLinks() {
		role, ok := wire.ConvertRoleFromWire(l.GetRole())
		if !ok {
			err := fmt.Errorf("no link carries the role %v", l.GetRole())
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		marker, ok := wire.ConvertMarkerFromWire(l.GetMarker())
		if !ok {
			err := fmt.Errorf("no link carries the marker %v", l.GetMarker())
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		links = append(links, note.PairLink{
			From: l.GetFrom(), To: l.GetTo(), Role: role, Type: l.GetType(), Why: l.GetNote(),
			Label: l.GetLabel(), Marker: marker,
		})
	}
	if !h.state.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer h.state.Writing.finish()

	if err := h.replaceLinks.Execute(
		ctx, showing, r.Msg.GetFirst(), r.Msg.GetSecond(), links,
	); err != nil {
		if errors.Is(err, note.ErrNotAPair) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		reason, ok := wire.GetErrorCode(err)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.ReplaceLinksResponse{Error: &reason}), nil
	}
	return connect.NewResponse(&v1.ReplaceLinksResponse{}), nil
}

func (h *noteHandler) GetOpeningNote(
	ctx context.Context, _ *connect.Request[v1.GetOpeningNoteRequest],
) (*connect.Response[v1.GetOpeningNoteResponse], error) {
	showing := h.state.GetCurrentVault()
	if showing.ID == "" {
		// A window standing on nothing opens on no note.
		return connect.NewResponse(&v1.GetOpeningNoteResponse{}), nil
	}

	ref, found, err := h.getOpeningNote.Execute(ctx, showing)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.GetOpeningNoteResponse{}
	if found {
		out.Note = convertNoteToWire(ref)
	}
	return connect.NewResponse(out), nil
}

func (h *noteHandler) GetNeighbourhood(
	ctx context.Context, r *connect.Request[v1.GetNeighbourhoodRequest],
) (*connect.Response[v1.GetNeighbourhoodResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	found, err := h.showNeighbourhood.Execute(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// The note types are fetched once for the whole graph, so a client renders
	// each plugin-typed note with its type.
	paths := []string{found.Focus.Path}
	for _, related := range found.Related {
		paths = append(paths, related.Path)
	}
	types, err := h.getTypes.Execute(ctx, showing, paths)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.GetNeighbourhoodResponse{
		Focus:     convertNoteToWire(found.Focus),
		FocusType: convertNoteTypeToWire(types[found.Focus.Path]),
	}
	for _, related := range found.Related {
		out.Related = append(out.Related, &v1.Neighbour{
			Note:          convertNoteToWire(related.NoteRef),
			Relation:      convertRelationToWire(related.Relation),
			Label:         related.Label,
			ReverseLabel:  related.ReverseLabel,
			Marker:        wire.ConvertMarkerToWire(related.Marker),
			ReverseMarker: wire.ConvertMarkerToWire(related.ReverseMarker),
			Through:       related.Parent,
			IsMutual:      related.IsMutual,
			Type:          convertNoteTypeToWire(types[related.Path]),
		})
	}
	return connect.NewResponse(out), nil
}

// ResolveAddresses answers where addresses written in one note land, and where
// the wikilinks of a text land, in the vault the window shows.
func (h *noteHandler) ResolveAddresses(
	ctx context.Context, r *connect.Request[v1.ResolveAddressesRequest],
) (*connect.Response[v1.ResolveAddressesResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	if named := r.Msg.GetVault(); named != "" && named != string(showing.ID) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vault %q is not the one this window shows", named))
	}
	out, err := wire.ResolveAddresses(ctx, h.resolveLinks, showing, r.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

// getLinks converts the links of a request into the links a note is written
// with, and fails the whole request if any link cannot be written.
func (h *noteHandler) getLinks(ctx context.Context, links []*v1.Link) ([]domain.Link, error) {
	if len(links) == 0 {
		return nil, nil
	}
	out := make([]domain.Link, 0, len(links))
	for _, l := range links {
		link, err := h.toDomainLink(ctx, l)
		if err != nil {
			return nil, err
		}
		out = append(out, link)
	}
	return out, nil
}

// newDomainLink is one link as the note it is written in declares it.
//
// The window names the note at the other end by the path it is filed under.
// How much of that path the link carries is `note.GetAddress`: a name where it
// means one note, and the path where it would mean another.
func (h *noteHandler) toDomainLink(ctx context.Context, l *v1.Link) (domain.Link, error) {
	role, ok := wire.ConvertRoleFromWire(l.GetRole())
	if !ok {
		return domain.Link{}, fmt.Errorf("no link carries the role %v", l.GetRole())
	}
	if l.GetTo() == "" {
		return domain.Link{}, errors.New("a link needs a note to go to")
	}
	marker, ok := wire.ConvertMarkerFromWire(l.GetMarker())
	if !ok {
		return domain.Link{}, fmt.Errorf("no link carries the marker %v", l.GetMarker())
	}
	target, err := h.getAddress(ctx, l.GetTo())
	if err != nil {
		return domain.Link{}, err
	}
	return domain.Link{
		Target: target, Role: role, Type: l.GetType(), Why: l.GetNote(), Label: l.GetLabel(), Marker: marker,
	}, nil
}

// getAddress is the note at the other end as a link carries it.
func (h *noteHandler) getAddress(ctx context.Context, to string) (domain.Address, error) {
	return h.getLinkAddress.Execute(ctx, h.state.GetCurrentVault(), to)
}

func convertNoteToWire(n domain.NoteRef) *v1.Note {
	return &v1.Note{Path: n.Path, Title: n.Title, Identifier: n.ID}
}

func convertRelationToWire(s domain.Relation) v1.Relation {
	switch s {
	case domain.RelationParent:
		return v1.Relation_RELATION_PARENT
	case domain.RelationChild:
		return v1.Relation_RELATION_CHILD
	case domain.RelationJump:
		return v1.Relation_RELATION_JUMP
	case domain.RelationSibling:
		return v1.Relation_RELATION_SIBLING
	default:
		return v1.Relation_RELATION_UNSPECIFIED
	}
}

// convertFileStatToWire is a file as the schema carries it. Nothing is carried for the
// zero value: a caller is given a file stat only where there is a file behind
// it.
func convertFileStatToWire(ref domain.FileStat) *v1.FileStat {
	if ref.IsZero() {
		return nil
	}
	return &v1.FileStat{Path: ref.Path, Size: ref.Size, Mtime: stamp(ref.ModTime)}
}

// newBaseline converts the baseline a client last read of a note. A nil
// baseline disables the comparison, and the write overwrites the current
// content.
func newBaseline(baseline *v1.ReadBaseline) *note.Baseline {
	if baseline == nil {
		return nil
	}
	return &note.Baseline{Prose: baseline.GetProse(), FileStat: convertFileStatFromWire(baseline.GetAt())}
}

// convertFileStatFromWire converts a wire file stat to the core representation.
// The kind is left empty because it is determined when the file is fetched.
func convertFileStatFromWire(pos *v1.FileStat) domain.FileStat {
	if pos == nil {
		return domain.FileStat{}
	}
	return domain.FileStat{Path: pos.GetPath(), Size: pos.GetSize(), ModTime: instant(pos.GetMtime())}
}

// ListHeadings hands the client the headings of the notes the filter names. A
// window standing on nothing holds no note to divide.
func (h *noteHandler) ListHeadings(
	ctx context.Context,
	r *connect.Request[v1.ListHeadingsRequest],
) (*connect.Response[v1.ListHeadingsResponse], error) {
	showing := h.state.GetCurrentVault()
	if showing.ID == "" {
		return connect.NewResponse(&v1.ListHeadingsResponse{}), nil
	}
	paths, err := eachOnce(r.Msg.GetPaths())
	if err != nil {
		return nil, err
	}
	found, err := h.getHeadings.Execute(ctx, showing, paths)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// In the order the filter named them.
	out := &v1.ListHeadingsResponse{Headings: make([]*v1.NoteHeadings, 0, len(found))}
	for _, path := range paths {
		headings, target := found[path]
		if !target {
			continue
		}
		one := &v1.NoteHeadings{Path: path, Headings: make([]*v1.Heading, 0, len(headings))}
		for _, h := range headings {
			one.Headings = append(one.Headings, &v1.Heading{
				Text:  h.Text,
				Line:  int32(h.Line),
				Level: int32(h.Level),
			})
		}
		out.Headings = append(out.Headings, one)
	}
	return connect.NewResponse(out), nil
}

// WatchEdits streams each change to a note's prose as it happens, until the
// client disconnects.
func (h *noteHandler) WatchEdits(
	ctx context.Context,
	_ *connect.Request[v1.WatchEditsRequest],
	out *connect.ServerStream[v1.WatchEditsResponse],
) error {
	line, isDone := h.state.Edits.subscribe()
	defer isDone()

	// The empty first message confirms the stream is open; without it a stream
	// with no changes cannot be distinguished from one that failed to open.
	if err := out.Send(&v1.WatchEditsResponse{}); err != nil {
		return err
	}

	repeat := time.NewTicker(wire.Again)
	defer repeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-repeat.C:
			if err := out.Send(&v1.WatchEditsResponse{}); err != nil {
				return err
			}
		case response, open := <-line:
			if !open {
				return nil
			}
			if err := out.Send(&v1.WatchEditsResponse{
				Change:  response.Change,
				Path:    response.Path,
				Range:   &v1.Range{Start: int32(response.From), End: int32(response.To)},
				Text:    response.Text,
				IsFinal: response.IsDone,
			}); err != nil {
				return err
			}
		}
	}
}

// RenameNote gives a note a different name.
func (h *noteHandler) RenameNote(
	ctx context.Context, r *connect.Request[v1.RenameNoteRequest],
) (*connect.Response[v1.RenameNoteResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	if !h.state.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer h.state.Writing.finish()

	renamed, err := h.rename.Execute(ctx, showing, r.Msg.GetPath(), r.Msg.GetTitle())
	out := &v1.RenameNoteResponse{
		Path:       renamed.Path,
		Title:      renamed.Title,
		NameSource: newWireNameSource(renamed.By),
	}
	if renamed.Moved != nil {
		out.Moved = newMoveResult(*renamed.Moved)
	}
	out.IsUnlevelled = h.state.isUnlevelled(err)
	if err != nil && !out.GetIsUnlevelled() {
		reason, ok := wire.GetErrorCode(err)
		switch {
		case ok:
			out.Error = &reason
		case out.GetPath() == "":
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		// The note was opened and its name was written, so the response is
		// returned as is. A later failure of the rename does not undo it.
	}
	return connect.NewResponse(out), nil
}

// newWireNameSource is which of the two a rename wrote, as the schema carries it.
func newWireNameSource(delta note.NameSource) v1.NameSource {
	switch delta {
	case note.ByFrontmatter:
		return v1.NameSource_NAME_SOURCE_FRONTMATTER
	case note.ByFilename:
		return v1.NameSource_NAME_SOURCE_FILENAME
	default:
		return v1.NameSource_NAME_SOURCE_UNSPECIFIED
	}
}

// newMoveResult is what the file did, as the schema carries it.
func newMoveResult(moved note.MoveResult) *v1.MoveResult {
	return &v1.MoveResult{
		From: moved.From, To: moved.To,
		Repaired: moved.Repaired, Dangling: moved.Dangling,
	}
}
