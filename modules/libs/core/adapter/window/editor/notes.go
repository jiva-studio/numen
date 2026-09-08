package editor

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/libs/core/domain"
	"github.com/jiva-studio/numen/modules/libs/core/internal/wire"
	"github.com/jiva-studio/numen/modules/libs/core/usecase/note"
)

// ReadNote hands the client the prose of a note.
func (a *API) ReadNote(
	ctx context.Context, r *connect.Request[v1.ReadNoteRequest],
) (*connect.Response[v1.ReadNoteResponse], error) {
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	found, err := a.Notes.Read.Execute(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ReadNoteResponse{Body: found.Body}
	if reason, refused := wire.RefusalOf(found.Outcome); refused {
		out.Refusal = &reason
	} else {
		// What the file was when this prose came out of it, for the client to
		// present when it puts prose back.
		out.At = fingerprintOf(found.Fingerprint)
	}
	return connect.NewResponse(out), nil
}

// WriteNote puts prose into a note. A note still holding the prose the client
// read is written over; one holding something else is left alone and the client
// is told the note changed.
func (a *API) WriteNote(
	ctx context.Context, r *connect.Request[v1.WriteNoteRequest],
) (*connect.Response[v1.WriteNoteResponse], error) {
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	// Counted around the write and not around the answer, so that closing waits
	// for what reaches the vault.
	if !a.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer a.Writing.done()
	at, err := a.Notes.Write.Save(ctx, showing, r.Msg.GetPath(), r.Msg.GetBody(), seenOf(r.Msg.GetSeen()))
	// A write that reached the vault is a write that happened, so the client is
	// handed the fingerprint it presents at its next save. It is told in the
	// same breath where the index did not follow, because the prose is on disk
	// and search does not hold it.
	behind := a.unlevelled(err)
	if err == nil || behind {
		// What the person typed owes its vectors. Which chunks owe them is not
		// carried: the debt is in the index, so several saves are one pass.
		if a.Wrote != nil {
			a.Wrote()
		}
		return connect.NewResponse(&v1.WriteNoteResponse{
			At: fingerprintOf(at), Unlevelled: behind,
		}), nil
	}
	reason, refused := wire.RefusalBy(err)
	if !refused {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.WriteNoteResponse{Refusal: &reason}), nil
}

// CreateNote makes a note, named after the title it is given and joined to
// whatever the request says it is joined to.
//
// The index is brought level before the answer comes back, so the note is in
// the picture as soon as it is on disk.
func (a *API) CreateNote(
	ctx context.Context, r *connect.Request[v1.CreateNoteRequest],
) (*connect.Response[v1.CreateNoteResponse], error) {
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	links, err := a.written(ctx, r.Msg.GetLinks())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if !a.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer a.Writing.done()

	made, err := a.Notes.Create.Execute(ctx, showing, note.NewNote{
		Title: r.Msg.GetTitle(),
		Path:  r.Msg.GetPath(),
		Links: links,
	})
	behind := a.unlevelled(err)
	if made.Path != "" {
		// The note is on disk under that name, so that is the answer. What comes
		// after the write is the index catching up, and the watcher does it
		// again.
		return connect.NewResponse(&v1.CreateNoteResponse{
			Path: made.Path, Unlevelled: behind,
		}), nil
	}
	if err == nil {
		return connect.NewResponse(&v1.CreateNoteResponse{}), nil
	}
	reason, refused := wire.RefusalBy(err)
	if !refused {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.CreateNoteResponse{Refusal: &reason}), nil
}

// WriteLink writes one relationship into one note. What is already written
// there is carried across, and the note at the other end is left alone.
func (a *API) WriteLink(
	ctx context.Context, r *connect.Request[v1.WriteLinkRequest],
) (*connect.Response[v1.WriteLinkResponse], error) {
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	link, err := a.writes(ctx, r.Msg.GetLink())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if !a.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer a.Writing.done()

	// The request names no fingerprint, so the write is held to what the note is
	// at the moment it is made.
	if _, err := a.Notes.Linking.Add(
		ctx, showing, r.Msg.GetPath(), domain.Fingerprint{}, link,
	); err != nil {
		reason, refused := wire.RefusalBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.WriteLinkResponse{Refusal: &reason}), nil
	}
	return connect.NewResponse(&v1.WriteLinkResponse{}), nil
}

func (a *API) GetOpeningNote(
	ctx context.Context, _ *connect.Request[v1.GetOpeningNoteRequest],
) (*connect.Response[v1.GetOpeningNoteResponse], error) {
	showing := a.Showing()
	if showing.ID == "" {
		// A window standing on nothing opens on no note.
		return connect.NewResponse(&v1.GetOpeningNoteResponse{}), nil
	}

	ref, found, err := a.Notes.Queries.Opening(ctx, showing.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.GetOpeningNoteResponse{}
	if found {
		out.Note = noteOf(ref)
	}
	return connect.NewResponse(out), nil
}

func (a *API) GetNeighbourhood(
	ctx context.Context, r *connect.Request[v1.GetNeighbourhoodRequest],
) (*connect.Response[v1.GetNeighbourhoodResponse], error) {
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	found, err := note.NewShowNeighbourhood(a.Notes.Links, a.Notes.Queries).
		Execute(ctx, showing, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Which of four each note on the picture is, asked once for the whole of
	// it, so a client draws a deck and a stencil as what they are.
	paths := []string{found.Focus.Path}
	for _, related := range found.Related {
		paths = append(paths, related.Path)
	}
	types, err := a.typesAt(ctx, showing, paths)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	out := &v1.GetNeighbourhoodResponse{
		Focus:     noteOf(found.Focus),
		FocusType: typeOf(types[found.Focus.Path]),
	}
	for _, related := range found.Related {
		out.Related = append(out.Related, &v1.Neighbour{
			Note:    noteOf(related.NoteRef),
			Seat:    seatOf(related.Seat),
			Label:   related.Label,
			Through: related.Parent,
			Mutual:  related.Mutual,
			Type:    typeOf(types[related.Path]),
		})
	}
	return connect.NewResponse(out), nil
}

// ResolveAddresses answers where addresses written in one note land. An address
// that reaches nothing is left out of the answer.
func (a *API) ResolveAddresses(
	ctx context.Context, r *connect.Request[v1.ResolveAddressesRequest],
) (*connect.Response[v1.ResolveAddressesResponse], error) {
	showing, err := a.shown()
	if err != nil {
		return nil, err
	}
	found, err := a.Notes.Links.Resolve(ctx, showing.ID, r.Msg.GetFrom(), r.Msg.GetWritten())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// In the order they were asked about, and an address asked about twice is
	// one answer.
	out := &v1.ResolveAddressesResponse{}
	said := make(map[string]bool, len(found))
	for _, written := range r.Msg.GetWritten() {
		one, reached := found[written]
		if !reached || said[written] {
			continue
		}
		said[written] = true
		vault, crossed := one.InVault(showing.ID)
		out.Resolved = append(out.Resolved, &v1.ResolvedAddress{
			Written:   written,
			Path:      one.To,
			Vault:     string(vault),
			Crossed:   crossed,
			Ambiguous: one.Ambiguous,
		})
	}
	return connect.NewResponse(out), nil
}

// written turns the links a request carries into the links a note is written
// with, and refuses the lot where one of them cannot be written.
func (a *API) written(ctx context.Context, links []*v1.Link) ([]domain.Link, error) {
	if len(links) == 0 {
		return nil, nil
	}
	out := make([]domain.Link, 0, len(links))
	for _, l := range links {
		link, err := a.writes(ctx, l)
		if err != nil {
			return nil, err
		}
		out = append(out, link)
	}
	return out, nil
}

// writes is one link as the note it is written in declares it.
//
// The window names the note at the other end by the path it is filed under.
// How much of that path the link carries is `note.Addressed`: a name where it
// means one note, and the path where it would mean another.
func (a *API) writes(ctx context.Context, l *v1.Link) (domain.Link, error) {
	role, ok := roleOf(l.GetRole())
	if !ok {
		return domain.Link{}, fmt.Errorf("no link carries the role %v", l.GetRole())
	}
	if l.GetTo() == "" {
		return domain.Link{}, errors.New("a link needs a note to go to")
	}
	target, err := a.addressed(ctx, l.GetTo())
	if err != nil {
		return domain.Link{}, err
	}
	return domain.Link{Target: target, Role: role, Label: l.GetLabel()}, nil
}

// addressed is the note at the other end as a link carries it. A build with no
// index cannot ask what else is filed under the name, and writes the name.
func (a *API) addressed(ctx context.Context, to string) (domain.Address, error) {
	if a.Notes.Queries == nil {
		return domain.Address{Scheme: domain.SchemeName, Value: domain.Basename(to)}, nil
	}
	return note.Addressed(ctx, a.Notes.Queries, a.Showing().ID, to)
}

// roleOf is the role a link carries, in the core's words. A link is written
// only with a role the schema names.
func roleOf(role v1.Role) (domain.LinkRole, bool) {
	switch role {
	case v1.Role_ROLE_PARENT:
		return domain.RoleParent, true
	case v1.Role_ROLE_CHILD:
		return domain.RoleChild, true
	case v1.Role_ROLE_JUMP:
		return domain.RoleJump, true
	case v1.Role_ROLE_REF:
		return domain.RoleRef, true
	case v1.Role_ROLE_ATTACHMENT:
		return domain.RoleAttachment, true
	default:
		return "", false
	}
}

func noteOf(n domain.NoteRef) *v1.Note {
	return &v1.Note{Path: n.Path, Title: n.Title, Identifier: n.ID}
}

func seatOf(s domain.Relation) v1.Seat {
	switch s {
	case domain.SeatParent:
		return v1.Seat_SEAT_PARENT
	case domain.SeatChild:
		return v1.Seat_SEAT_CHILD
	case domain.SeatJump:
		return v1.Seat_SEAT_JUMP
	case domain.SeatSibling:
		return v1.Seat_SEAT_SIBLING
	default:
		return v1.Seat_SEAT_UNSPECIFIED
	}
}

// fingerprintOf is a file as the schema carries it. Nothing is carried for the
// zero value: a caller is given a fingerprint only where there is a file behind
// it.
func fingerprintOf(ref domain.Fingerprint) *v1.Fingerprint {
	if ref.IsZero() {
		return nil
	}
	return &v1.Fingerprint{Path: ref.Path, Size: ref.Size, Mtime: stamp(ref.ModTime)}
}

// seenOf is what a client says it last saw of a note. Nothing said is nothing
// compared, and the write lands on whatever the note now holds.
func seenOf(seen *v1.LastRead) *note.LastRead {
	if seen == nil {
		return nil
	}
	return &note.LastRead{Prose: seen.GetProse(), Fingerprint: refOf(seen.GetAt())}
}

// refOf is a fingerprint as the core holds one. The kind is left empty: what
// the file is was decided when it was asked for.
func refOf(at *v1.Fingerprint) domain.Fingerprint {
	if at == nil {
		return domain.Fingerprint{}
	}
	return domain.Fingerprint{Path: at.GetPath(), Size: at.GetSize(), ModTime: instant(at.GetMtime())}
}
