package webui

import (
	"context"
	"errors"
	"fmt"

	"connectrpc.com/connect"

	v1 "github.com/jiva-studio/numen/modules/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/domain"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/port"
	"github.com/jiva-studio/numen/modules/apps/desktop/internal/core/usecase/note"
)

// errNoEditing is what a build with no write path answers.
var errNoEditing = errors.New("this build cannot edit notes")

// Read hands the client the prose of a note.
func (a *API) Read(ctx context.Context, r *connect.Request[v1.ReadRequest]) (*connect.Response[v1.ReadResponse], error) {
	if a.Reads == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoEditing)
	}
	found, err := a.Reads.Execute(ctx, a.Vault, r.Msg.GetPath())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	out := &v1.ReadResponse{Body: found.Body}
	if refusal, refused := refusalOf(found.Outcome); refused {
		out.Refusal = &refusal
	} else {
		// What the file was when this prose came out of it, for the client to
		// present when it puts prose back.
		out.At = fingerprintOf(found.Ref)
	}
	return connect.NewResponse(out), nil
}

// Write puts prose into a note. A note still holding the prose the client read
// is written over; one holding something else is left alone and the client is
// told the note changed.
func (a *API) Write(ctx context.Context, r *connect.Request[v1.WriteRequest]) (*connect.Response[v1.WriteResponse], error) {
	if a.Saves == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoEditing)
	}
	// Counted around the write and not around the answer, so that closing waits
	// for what reaches the vault.
	if !a.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer a.Writing.done()
	at, err := a.Saves.Save(ctx, a.Vault, r.Msg.GetPath(), r.Msg.GetBody(), seenOf(r.Msg.GetSeen()))
	if err == nil {
		// What the person typed owes its vectors. Which chunks owe them is not
		// carried: the debt is in the index, so several saves are one pass.
		if a.Wrote != nil {
			a.Wrote()
		}
		return connect.NewResponse(&v1.WriteResponse{At: fingerprintOf(at)}), nil
	}
	// A note holding prose this client has not read is its own answer. A
	// refusal is something the client can do nothing about; this one is a
	// question, and the person answers it.
	if errors.Is(err, port.ErrChanged) {
		return connect.NewResponse(&v1.WriteResponse{Changed: true}), nil
	}
	refusal, refused := refusedBy(err)
	if !refused {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.WriteResponse{Refusal: &refusal}), nil
}

// Create makes a note, named after the title it is given and joined to whatever
// the request says it is joined to.
//
// The index is brought level before the answer comes back, so the note is in
// the picture as soon as it is on disk.
func (a *API) Create(ctx context.Context, r *connect.Request[v1.CreateRequest]) (*connect.Response[v1.CreateResponse], error) {
	if a.Makes == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoEditing)
	}
	links, err := written(r.Msg.GetLinks())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if !a.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer a.Writing.done()

	made, err := a.Makes.Execute(ctx, a.Vault, note.NewNote{
		Title:  r.Msg.GetTitle(),
		Folder: r.Msg.GetFolder(),
		Links:  links,
	})
	if made.Path != "" {
		// The note is on disk under that name, so that is the answer. What comes
		// after the write is the index catching up, and the watcher does it
		// again.
		return connect.NewResponse(&v1.CreateResponse{Path: made.Path}), nil
	}
	if err == nil {
		return connect.NewResponse(&v1.CreateResponse{}), nil
	}
	refusal, refused := refusedBy(err)
	if !refused {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&v1.CreateResponse{Refusal: &refusal}), nil
}

// Join writes one relationship into one note. What is already written there is
// carried across, and the note at the other end is left alone.
func (a *API) Join(ctx context.Context, r *connect.Request[v1.JoinRequest]) (*connect.Response[v1.JoinResponse], error) {
	if a.Joins == nil {
		return nil, connect.NewError(connect.CodeUnimplemented, errNoEditing)
	}
	link, err := writes(r.Msg.GetLink())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if !a.Writing.begin() {
		return nil, connect.NewError(connect.CodeUnavailable, errClosing)
	}
	defer a.Writing.done()

	if err := a.Joins.Add(ctx, a.Vault, r.Msg.GetPath(), link); err != nil {
		// A join reads a note, splices its frontmatter and puts it back. A note
		// that moved in between is left alone, and the client reads it again
		// before it asks for this.
		if errors.Is(err, port.ErrChanged) {
			return connect.NewResponse(&v1.JoinResponse{Changed: true}), nil
		}
		refusal, refused := refusedBy(err)
		if !refused {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		return connect.NewResponse(&v1.JoinResponse{Refusal: &refusal}), nil
	}
	return connect.NewResponse(&v1.JoinResponse{}), nil
}

// written turns the links a request carries into the links a note is written
// with, and refuses the lot where one of them cannot be written.
func written(links []*v1.NewLink) ([]domain.Link, error) {
	if len(links) == 0 {
		return nil, nil
	}
	out := make([]domain.Link, 0, len(links))
	for _, l := range links {
		link, err := writes(l)
		if err != nil {
			return nil, err
		}
		out = append(out, link)
	}
	return out, nil
}

// writes is one link as the note it is written in declares it.
//
// A link is written by the other note's name, which is what the path it is
// filed under is called without its folder or its extension.
func writes(l *v1.NewLink) (domain.Link, error) {
	role, ok := roleOf(l.GetSeat())
	if !ok {
		return domain.Link{}, fmt.Errorf("no link seats a note as %v", l.GetSeat())
	}
	name := domain.Basename(l.GetTo())
	if name == "" {
		return domain.Link{}, errors.New("a link needs a note to go to")
	}
	return domain.Link{
		Target: domain.Address{Scheme: domain.SchemeName, Value: name},
		Role:   role,
		Label:  l.GetLabel(),
	}, nil
}

// roleOf is the role a link carries to put the note at its other end in a seat.
func roleOf(seat v1.Seat) (domain.LinkRole, bool) {
	switch seat {
	case v1.Seat_SEAT_PARENT:
		return domain.RoleParent, true
	case v1.Seat_SEAT_CHILD:
		return domain.RoleChild, true
	case v1.Seat_SEAT_JUMP:
		return domain.RoleJump, true
	default:
		return "", false
	}
}

// fingerprintOf is a file as the schema carries it. Nothing is carried for the
// zero value: a caller is given a fingerprint only where there is a file behind
// it.
func fingerprintOf(ref domain.FileRef) *v1.Fingerprint {
	if ref == (domain.FileRef{}) {
		return nil
	}
	return &v1.Fingerprint{Path: ref.Path, Size: ref.Size, Mtime: ref.MTime}
}

// seenOf is what a client says it last saw of a note. Nothing said is nothing
// compared, and the write lands on whatever the note now holds.
func seenOf(seen *v1.Seen) *note.Seen {
	if seen == nil {
		return nil
	}
	return &note.Seen{Prose: seen.GetProse(), At: refOf(seen.GetAt())}
}

// refOf is a fingerprint as the core holds one. The kind is left empty: what
// the file is was decided when it was asked for.
func refOf(at *v1.Fingerprint) domain.FileRef {
	if at == nil {
		return domain.FileRef{}
	}
	return domain.FileRef{Path: at.GetPath(), Size: at.GetSize(), MTime: at.GetMtime()}
}

// refusedBy says which refusal a write's error is, and whether it is one at all.
// Anything else is the vault being out of reach.
//
// A note that changed is not among them. It is answered on its own, because a
// refusal is something the client can do nothing about and that one is a
// question for the person.
func refusedBy(err error) (v1.Refusal, bool) {
	switch {
	case errors.Is(err, note.ErrTooLarge):
		return v1.Refusal_REFUSAL_TOO_LARGE, true
	case errors.Is(err, note.ErrUnreadable):
		return v1.Refusal_REFUSAL_UNREADABLE, true
	case errors.Is(err, note.ErrBodyRefused):
		return v1.Refusal_REFUSAL_BODY_REFUSED, true
	case errors.Is(err, port.ErrNotANote):
		return v1.Refusal_REFUSAL_NOT_A_NOTE, true
	case errors.Is(err, port.ErrOccupied):
		return v1.Refusal_REFUSAL_OCCUPIED, true
	default:
		return v1.Refusal_REFUSAL_UNSPECIFIED, false
	}
}

// refusalOf says which refusal an outcome is, and whether it is one at all.
func refusalOf(o note.Outcome) (v1.Refusal, bool) {
	switch o {
	case note.Missing:
		return v1.Refusal_REFUSAL_MISSING, true
	case note.NotANote:
		return v1.Refusal_REFUSAL_NOT_A_NOTE, true
	case note.NotText:
		return v1.Refusal_REFUSAL_NOT_TEXT, true
	case note.TooLarge:
		return v1.Refusal_REFUSAL_TOO_LARGE, true
	case note.Unreadable:
		return v1.Refusal_REFUSAL_UNREADABLE, true
	default:
		return v1.Refusal_REFUSAL_UNSPECIFIED, false
	}
}
