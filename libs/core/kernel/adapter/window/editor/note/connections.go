package note

import (
	"context"
	"errors"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"

	windowdomain "github.com/jiva-studio/numen/libs/core/kernel/domain/window"
	"github.com/jiva-studio/numen/libs/core/kernel/internal/adapterlib/wire"
	"github.com/jiva-studio/numen/libs/note/domain"
)

// connectionFailure is why a connection write did not land, as the schema
// carries it.
type connectionFailure struct {
	code      *v1.ErrorCode
	isInvalid bool
	isCycle   bool
}

func failureOf(err error) (connectionFailure, error) {
	switch {
	case err == nil:
		return connectionFailure{}, nil
	case errors.Is(err, windowdomain.ErrClosing):
		return connectionFailure{}, connect.NewError(connect.CodeUnavailable, errClosing)
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
func (h *Connections) GetConnection(
	ctx context.Context, r *connect.Request[v1.GetConnectionRequest],
) (*connect.Response[v1.GetConnectionResponse], error) {
	showing, err := getShownVault(h.vault)
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
func (h *Connections) SaveConnection(
	ctx context.Context, r *connect.Request[v1.SaveConnectionRequest],
) (*connect.Response[v1.SaveConnectionResponse], error) {
	showing, err := getShownVault(h.vault)
	if err != nil {
		return nil, err
	}
	saving, err := wire.ConvertConnectionFromWire(r.Msg.GetConnection())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	failure, err := failureOf(h.saveConnection.Execute(ctx, showing, saving))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.SaveConnectionResponse{
		Error: failure.code, IsInvalid: failure.isInvalid, IsCycle: failure.isCycle,
	}), nil
}

// RemoveSide removes the record one note writes about another.
func (h *Connections) RemoveSide(
	ctx context.Context, r *connect.Request[v1.RemoveSideRequest],
) (*connect.Response[v1.RemoveSideResponse], error) {
	showing, err := getShownVault(h.vault)
	if err != nil {
		return nil, err
	}
	failure, err := failureOf(h.removeSide.Execute(ctx, showing, r.Msg.GetPath(), r.Msg.GetTo(), r.Msg.GetType()))
	if err != nil {
		return nil, err
	}
	if failure.isInvalid || failure.isCycle {
		return nil, connect.NewError(connect.CodeInvalidArgument, domain.ErrInvalidConnection)
	}
	return connect.NewResponse(&v1.RemoveSideResponse{Error: failure.code}), nil
}

// RemoveConnection removes the records two notes write about each other.
func (h *Connections) RemoveConnection(
	ctx context.Context, r *connect.Request[v1.RemoveConnectionRequest],
) (*connect.Response[v1.RemoveConnectionResponse], error) {
	showing, err := getShownVault(h.vault)
	if err != nil {
		return nil, err
	}
	failure, err := failureOf(
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
func (h *Connections) JoinNotes(
	ctx context.Context, r *connect.Request[v1.JoinNotesRequest],
) (*connect.Response[v1.JoinNotesResponse], error) {
	showing, err := getShownVault(h.vault)
	if err != nil {
		return nil, err
	}
	side, err := wire.ConvertSideFromWire(r.Msg.GetSide())
	if err != nil || side == nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.Join(errNoSide, err))
	}
	failure, err := failureOf(h.join.Execute(ctx, showing, r.Msg.GetFrom(), r.Msg.GetTo(), *side))
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&v1.JoinNotesResponse{
		Error: failure.code, IsInvalid: failure.isInvalid, IsCycle: failure.isCycle,
	}), nil
}

var errNoSide = errors.New("a join needs the record to write")

// CreateRelatedNote makes a note joined to an existing one.
func (h *Links) CreateRelatedNote(
	ctx context.Context, r *connect.Request[v1.CreateRelatedNoteRequest],
) (*connect.Response[v1.CreateRelatedNoteResponse], error) {
	showing, err := getShownVault(h.vault)
	if err != nil {
		return nil, err
	}
	role, ok := wire.ConvertRoleFromWire(r.Msg.GetRole())
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no note is created in this role"))
	}
	marker, ok := wire.ConvertMarkerFromWire(r.Msg.GetMarker())
	if !ok {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("no link has this marker"))
	}
	made, indexOutdated, err := h.createRelated.Execute(ctx, showing, domain.RelatedNote{
		From: r.Msg.GetFrom(), Role: role, Title: r.Msg.GetTitle(), Type: r.Msg.GetType(), Marker: marker,
	})
	switch {
	case errors.Is(err, windowdomain.ErrClosing):
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	case made.Path != "":
		return connect.NewResponse(&v1.CreateRelatedNoteResponse{Path: made.Path, IsIndexOutdated: indexOutdated}), nil
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
func (h *Links) ListBacklinks(
	ctx context.Context, r *connect.Request[v1.ListBacklinksRequest],
) (*connect.Response[v1.ListBacklinksResponse], error) {
	showing, err := getShownVault(h.vault)
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
			From:       &v1.Note{Path: b.From},
			Role:       wire.ConvertRoleToWire(b.Role),
			SourceRole: wire.ConvertRoleToWire(b.SourceRole),
			Type:       b.Type,
			Note:       b.Why,
			Label:      b.Label,
			Marker:     wire.ConvertMarkerToWire(b.Marker),
		})
	}
	return connect.NewResponse(out), nil
}
