package editor

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/core/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/core/usecase/note"
	"github.com/jiva-studio/numen/libs/notes/domain"
)

// bindConnections binds the use cases that read and write the relationships
// between notes.
func (h *noteHandler) bindConnections(notes *note.Scenarios) {
	h.getConnection = &notes.Connection
	h.saveConnection = &notes.SaveConnection
	h.removeSide = &notes.RemoveSide
	h.removeConnection = &notes.RemoveConnection
	h.join = &notes.Join
	h.createInRelation = &notes.CreateInRelation
	h.getBacklinks = &notes.Backlinks
}

// connectionFailure is why a connection write did not land, as the schema
// carries it.
type connectionFailure struct {
	code      *v1.ErrorCode
	isInvalid bool
	isCycle   bool
}

// failureOf sorts the error of a connection write. The second result is the
// error to fail the call with when the schema carries no answer for it.
func (h *noteHandler) failureOf(err error) (connectionFailure, error) {
	switch {
	case err == nil, h.state.isUnlevelled(err):
		return connectionFailure{}, nil
	case errors.Is(err, domain.ErrHierarchyCycle):
		return connectionFailure{isCycle: true}, nil
	case errors.Is(err, domain.ErrInvalidConnection):
		return connectionFailure{isInvalid: true}, nil
	}
	reason, ok := wire.GetErrorCode(err)
	if !ok {
		return connectionFailure{}, connect.NewError(wire.GetCode(err), err)
	}
	return connectionFailure{code: &reason}, nil
}

// GetConnection answers the relationship between two notes as a whole.
func (h *noteHandler) GetConnection(
	ctx context.Context, r *connect.Request[v1.GetConnectionRequest],
) (*connect.Response[v1.GetConnectionResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	found, err := h.getConnection.Execute(ctx, showing, r.Msg.GetFirst(), r.Msg.GetSecond(), r.Msg.GetType())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.GetConnectionResponse{Connection: wire.ConvertConnectionToWire(found)}), nil
}

// SaveConnection replaces the records between two notes in both notes or in
// neither.
func (h *noteHandler) SaveConnection(
	ctx context.Context, r *connect.Request[v1.SaveConnectionRequest],
) (*connect.Response[v1.SaveConnectionResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	saving, err := wire.ConvertConnectionFromWire(r.Msg.GetConnection())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if !h.state.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer h.state.Writing.finish()

	failure, err := h.failureOf(h.saveConnection.Execute(ctx, showing, saving))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.SaveConnectionResponse{
		Error: failure.code, IsInvalid: failure.isInvalid, IsCycle: failure.isCycle,
	}), nil
}

// RemoveSide removes the record one note writes about another.
func (h *noteHandler) RemoveSide(
	ctx context.Context, r *connect.Request[v1.RemoveSideRequest],
) (*connect.Response[v1.RemoveSideResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	if !h.state.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer h.state.Writing.finish()

	failure, err := h.failureOf(h.removeSide.Execute(ctx, showing, r.Msg.GetPath(), r.Msg.GetTo(), r.Msg.GetType()))
	if err != nil {
		return nil, err
	}
	if failure.isInvalid || failure.isCycle {
		return nil, connect.NewError(connect.CodeInvalidArgument, domain.ErrInvalidConnection)
	}
	return connect.NewResponse(&v1.RemoveSideResponse{Error: failure.code}), nil
}

// RemoveConnection removes the records two notes write about each other.
func (h *noteHandler) RemoveConnection(
	ctx context.Context, r *connect.Request[v1.RemoveConnectionRequest],
) (*connect.Response[v1.RemoveConnectionResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	if !h.state.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer h.state.Writing.finish()

	failure, err := h.failureOf(
		h.removeConnection.Execute(ctx, showing, r.Msg.GetFirst(), r.Msg.GetSecond(), r.Msg.GetType()))
	if err != nil {
		return nil, err
	}
	if failure.isInvalid || failure.isCycle {
		return nil, connect.NewError(connect.CodeInvalidArgument, domain.ErrInvalidConnection)
	}
	return connect.NewResponse(&v1.RemoveConnectionResponse{Error: failure.code}), nil
}

// JoinNotes makes one note write a record about another.
func (h *noteHandler) JoinNotes(
	ctx context.Context, r *connect.Request[v1.JoinNotesRequest],
) (*connect.Response[v1.JoinNotesResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	side, err := wire.ConvertSideFromWire(r.Msg.GetSide())
	if err != nil || side == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Join(errNoSide, err))
	}
	if !h.state.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer h.state.Writing.finish()

	failure, err := h.failureOf(h.join.Execute(ctx, showing, r.Msg.GetFrom(), r.Msg.GetTo(), *side))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.JoinNotesResponse{
		Error: failure.code, IsInvalid: failure.isInvalid, IsCycle: failure.isCycle,
	}), nil
}

var errNoSide = errors.New("a join needs the record to write")

// CreateRelatedNote makes a note joined to an existing one.
func (h *noteHandler) CreateRelatedNote(
	ctx context.Context, r *connect.Request[v1.CreateRelatedNoteRequest],
) (*connect.Response[v1.CreateRelatedNoteResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	role, ok := wire.ConvertRoleFromWire(r.Msg.GetRole())
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no note is created in this role"))
	}
	marker, ok := wire.ConvertMarkerFromWire(r.Msg.GetMarker())
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no link carries this marker"))
	}
	if !h.state.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer h.state.Writing.finish()

	made, err := h.createInRelation.Execute(ctx, showing, domain.RelatedNote{
		From: r.Msg.GetFrom(), Role: role, Title: r.Msg.GetTitle(), Type: r.Msg.GetType(), Marker: marker,
	})
	unlevelled := h.state.isUnlevelled(err)
	switch {
	case made.Path != "":
		return connect.NewResponse(&v1.CreateRelatedNoteResponse{Path: made.Path, IsUnlevelled: unlevelled}), nil
	case errors.Is(err, domain.ErrNoFreeName):
		return connect.NewResponse(&v1.CreateRelatedNoteResponse{IsNameExhausted: true}), nil
	case errors.Is(err, domain.ErrNotCreatable):
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	reason, ok := wire.GetErrorCode(err)
	if !ok {
		return nil, connect.NewError(wire.GetCode(err), err)
	}
	return connect.NewResponse(&v1.CreateRelatedNoteResponse{Error: &reason}), nil
}

// ListBacklinks answers the links written in other notes that point at a note.
func (h *noteHandler) ListBacklinks(
	ctx context.Context, r *connect.Request[v1.ListBacklinksRequest],
) (*connect.Response[v1.ListBacklinksResponse], error) {
	showing, err := h.state.getCurrentVault()
	if err != nil {
		return nil, err
	}
	found, err := h.getBacklinks.Execute(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ListBacklinksResponse{}
	for _, b := range found {
		out.Backlinks = append(out.Backlinks, &v1.Backlink{
			From:         &v1.Note{Path: b.From},
			Role:         wire.ConvertRoleToWire(b.Role),
			RoleFromHere: wire.ConvertRoleToWire(b.RoleFromHere),
			Type:         b.Type,
			Note:         b.Why,
			Label:        b.Label,
			Marker:       wire.ConvertMarkerToWire(b.Marker),
		})
	}
	return connect.NewResponse(out), nil
}
