package note

import (
	"context"
	"errors"
	"fmt"
	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"strings"
	"time"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/kernel/domain/naming"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/kernel/usecase/note"
	"github.com/jiva-studio/numen/libs/note/domain"
	"github.com/jiva-studio/numen/libs/note/edit"
)

// ReadNote hands the client the prose of a note.
func (h *Prose) ReadNote(
	ctx context.Context, r *connect.Request[v1.ReadNoteRequest],
) (*connect.Response[v1.ReadNoteResponse], error) {
	showing, err := getShownVault(h.vault)
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
func (h *Prose) WriteNote(
	ctx context.Context, r *connect.Request[v1.WriteNoteRequest],
) (*connect.Response[v1.WriteNoteResponse], error) {
	showing, err := getShownVault(h.vault)
	if err != nil {
		return nil, err
	}
	pos, err := h.save.Save(ctx, showing, r.Msg.GetPath(), r.Msg.GetBody(), newBaseline(r.Msg.GetBaseline()))
	indexOutdated := h.writes.IsIndexOutdated(err)
	if err == nil || indexOutdated {
		// The written prose requires new vectors. The pending chunks are tracked in the index, so several saves
		// are embedded in one pass.
		h.writes.NoteWritten()
	}
	if errors.Is(err, windowdomain.ErrClosing) {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	// A write that reached the vault has happened, so the response carries the
	// file stat the client uses at its next save. The response also sets
	// IsIndexOutdated when the index did not follow, because the prose is on disk
	// but search does not include it.
	if err == nil || indexOutdated {
		return connect.NewResponse(&v1.WriteNoteResponse{
			At: convertFileStatToWire(pos), IsIndexOutdated: indexOutdated,
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
func (h *Prose) CreateNote(
	ctx context.Context, r *connect.Request[v1.CreateNoteRequest],
) (*connect.Response[v1.CreateNoteResponse], error) {
	showing, err := getShownVault(h.vault)
	if err != nil {
		return nil, err
	}
	links, err := newNamedLinks(r.Msg.GetLinks())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	creating := note.NewNote{Title: r.Msg.GetTitle(), Path: r.Msg.GetPath(), Named: links}
	made, err := h.createNote(ctx, showing, creating)
	indexOutdated := h.writes.IsIndexOutdated(err)
	switch {
	case errors.Is(err, note.ErrInvalidLink):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	case errors.Is(err, windowdomain.ErrClosing):
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	case errors.Is(err, domain.ErrNoFreeName):
		return connect.NewResponse(&v1.CreateNoteResponse{IsNameExhausted: true}), nil
	}
	if made.Path != "" {
		// The note is on disk under that name, so the response carries it. The
		// index catches up after the write, and the watcher repeats that
		// update.
		return connect.NewResponse(&v1.CreateNoteResponse{
			Path: made.Path, IsIndexOutdated: indexOutdated,
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
func (h *Links) WriteLink(
	ctx context.Context, r *connect.Request[v1.WriteLinkRequest],
) (*connect.Response[v1.WriteLinkResponse], error) {
	showing, err := getShownVault(h.vault)
	if err != nil {
		return nil, err
	}
	link, err := newNamedLink(r.Msg.GetLink())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if err := h.writeLink.Execute(ctx, showing, r.Msg.GetPath(), link); err != nil {
		switch {
		case errors.Is(err, note.ErrInvalidLink):
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		case errors.Is(err, windowdomain.ErrClosing):
			return nil, connect.NewError(connect.CodeUnavailable, errClosing)
		}
		reason, ok := wire.GetErrorCode(err)
		if !ok {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.WriteLinkResponse{Error: &reason}), nil
	}
	return connect.NewResponse(&v1.WriteLinkResponse{}), nil
}

func (h *Outline) GetOpeningNote(
	ctx context.Context, _ *connect.Request[v1.GetOpeningNoteRequest],
) (*connect.Response[v1.GetOpeningNoteResponse], error) {
	showing := h.vault.GetCurrentVault()
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

func (h *Outline) GetNeighborhood(
	ctx context.Context, r *connect.Request[v1.GetNeighborhoodRequest],
) (*connect.Response[v1.GetNeighborhoodResponse], error) {
	showing, err := getShownVault(h.vault)
	if err != nil {
		return nil, err
	}
	found, err := h.getNeighborhood.Execute(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	types, tabKinds := found.Types, found.TabKinds

	out := &v1.GetNeighborhoodResponse{
		Focus:        convertNoteToWire(found.Focus),
		FocusType:    convertNoteTypeToWire(types[found.Focus.Path]),
		FocusTabKind: tabKinds[found.Focus.Path],
	}
	for _, related := range found.Related {
		out.Related = append(out.Related, &v1.Neighbor{
			Note:     convertNoteToWire(related.NoteRef),
			Relation: convertRelationToWire(related.Relation),
			Type:     convertNoteTypeToWire(types[related.Path]),
			TabKind:  tabKinds[related.Path],
		})
	}
	for _, connection := range found.Connections {
		out.Connections = append(out.Connections, wire.ConvertConnectionToWire(connection))
	}
	return connect.NewResponse(out), nil
}

// ResolveAddresses answers where addresses written in one note land, and where
// the wikilinks of a text land, in the vault the window shows.
func (h *Links) ResolveAddresses(
	ctx context.Context, r *connect.Request[v1.ResolveAddressesRequest],
) (*connect.Response[v1.ResolveAddressesResponse], error) {
	showing, err := getShownVault(h.vault)
	if err != nil {
		return nil, err
	}
	if named := r.Msg.GetVault(); named != "" && named != string(showing.ID) {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("vault %q is not the one this window shows", named))
	}
	out, err := wire.ResolveAddresses(ctx, h.resolveLinks, showing.ID, r.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(out), nil
}

func newNamedLinks(links []*v1.Link) ([]note.NamedLink, error) {
	out := make([]note.NamedLink, 0, len(links))
	for _, l := range links {
		link, err := newNamedLink(l)
		if err != nil {
			return nil, err
		}
		out = append(out, link)
	}
	return out, nil
}

func newNamedLink(l *v1.Link) (note.NamedLink, error) {
	role, ok := wire.ConvertRoleFromWire(l.GetRole())
	if !ok {
		return note.NamedLink{}, fmt.Errorf("no link has the role %v", l.GetRole())
	}
	if l.GetTo() == "" {
		return note.NamedLink{}, errors.New("a link needs a note to go to")
	}
	marker, ok := wire.ConvertMarkerFromWire(l.GetMarker())
	if !ok {
		return note.NamedLink{}, fmt.Errorf("no link has the marker %v", l.GetMarker())
	}
	return note.NamedLink{To: l.GetTo(), Role: role, Type: l.GetType(), Why: l.GetNote(), Label: l.GetLabel(), Marker: marker}, nil
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

func convertFileStatToWire(ref domain.FileStat) *v1.FileStat {
	if ref.IsZero() {
		return nil
	}
	return &v1.FileStat{Path: ref.Path, Size: ref.Size, Mtime: stamp(ref.ModTime)}
}

func newBaseline(baseline *v1.ReadBaseline) *edit.Baseline {
	if baseline == nil {
		return nil
	}
	return &edit.Baseline{Prose: baseline.GetProse(), FileStat: convertFileStatFromWire(baseline.GetAt())}
}

func convertFileStatFromWire(pos *v1.FileStat) domain.FileStat {
	if pos == nil {
		return domain.FileStat{}
	}
	return domain.FileStat{Path: pos.GetPath(), Size: pos.GetSize(), ModTime: instant(pos.GetMtime())}
}

// ListHeadings hands the client the headings of the notes the filter names. A
// window standing on nothing holds no note to divide.
func (h *Outline) ListHeadings(
	ctx context.Context,
	r *connect.Request[v1.ListHeadingsRequest],
) (*connect.Response[v1.ListHeadingsResponse], error) {
	showing := h.vault.GetCurrentVault()
	if showing.ID == "" {
		return connect.NewResponse(&v1.ListHeadingsResponse{}), nil
	}
	paths, err := eachOnce(r.Msg.GetPaths())
	if err != nil {
		return nil, err
	}
	found, err := h.listHeadings.Execute(ctx, showing, paths)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.ListHeadingsResponse{Headings: make([]*v1.NoteHeadings, 0, len(found))}
	for _, entry := range found {
		one := &v1.NoteHeadings{Path: entry.Path, Headings: make([]*v1.Heading, 0, len(entry.Headings))}
		for _, h := range entry.Headings {
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
func (h *Handler) WatchEdits(
	ctx context.Context,
	_ *connect.Request[v1.WatchEditsRequest],
	out *connect.ServerStream[v1.WatchEditsResponse],
) error {
	line, isDone := h.edits.SubscribeEdits()
	defer isDone()

	// The empty first message confirms the stream is open; without it a stream
	// with no changes cannot be distinguished from one that failed to open.
	if err := out.Send(&v1.WatchEditsResponse{}); err != nil {
		return err
	}

	repeat := time.NewTicker(wire.ResendInterval)
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
func (h *Prose) RenameNote(
	ctx context.Context, r *connect.Request[v1.RenameNoteRequest],
) (*connect.Response[v1.RenameNoteResponse], error) {
	showing, err := getShownVault(h.vault)
	if err != nil {
		return nil, err
	}
	renamed, err := h.rename.Execute(ctx, showing, r.Msg.GetPath(), r.Msg.GetTitle())
	isIndexOutdated := h.writes.IsIndexOutdated(err)
	if errors.Is(err, windowdomain.ErrClosing) {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	out := &v1.RenameNoteResponse{
		Path:            renamed.Path,
		Title:           renamed.Title,
		NameSource:      newWireNameSource(renamed.By),
		IsIndexOutdated: isIndexOutdated,
	}
	if renamed.Moved != nil {
		out.Moved = newMoveResult(*renamed.Moved)
	}
	if err != nil && !isIndexOutdated {
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

func newWireNameSource(delta naming.Source) v1.NameSource {
	switch delta {
	case naming.ByFrontmatter:
		return v1.NameSource_NAME_SOURCE_FRONTMATTER
	case naming.ByFilename:
		return v1.NameSource_NAME_SOURCE_FILENAME
	default:
		return v1.NameSource_NAME_SOURCE_UNSPECIFIED
	}
}

func newMoveResult(moved domain.MoveResult) *v1.MoveResult {
	return &v1.MoveResult{
		From: moved.From, To: moved.To,
		Repaired: moved.Repaired, Dangling: moved.Dangling,
	}
}

func (h *Prose) createNote(ctx context.Context, v domain.Vault, in note.NewNote) (note.CreateResult, error) {
	if strings.TrimSpace(in.Title) == "" {
		return h.create.ExecuteUntitled(ctx, v, in)
	}
	return h.create.Execute(ctx, v, in)
}
