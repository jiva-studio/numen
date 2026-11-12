package host

import (
	"context"

	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// ConnectionReader reads the relationships between the notes of the vault.
type ConnectionReader interface {
	// GetConnection returns the records two notes write about each other. A note
	// that writes none has a nil side. The empty type selects the records that
	// carry none.
	GetConnection(ctx context.Context, first, second, kind string) (domain.Connection, error)
	// ListBacklinks lists the links written in other notes that point at a note.
	ListBacklinks(ctx context.Context, to string) ([]domain.Backlink, error)
}

// ConnectionWriter writes the relationships between the notes of the vault. The host enforces the
// rules of a connection: the inverse role of the record in the other note, a hierarchy without
// cycles, and an atomic write of both notes.
type ConnectionWriter interface {
	// SaveConnection makes the records between two notes exactly the ones given,
	// in both notes or in neither.
	SaveConnection(ctx context.Context, c domain.Connection) error
	// RemoveSide removes the records of the type that one note writes about
	// another and leaves the other note alone.
	RemoveSide(ctx context.Context, from, to, kind string) error
	// RemoveConnection removes the records two notes write about each other.
	RemoveConnection(ctx context.Context, first, second, kind string) error
	// Join makes one note write a record about another and keeps the record the
	// other note already writes.
	Join(ctx context.Context, from, to string, side domain.Side) error
}

// RelatedNoteCreator creates a note joined to an existing one.
type RelatedNoteCreator interface {
	// CreateInRelation creates a note joined to an existing one and returns its path.
	CreateInRelation(ctx context.Context, in domain.NewRelatedNote) (string, error)
}

var (
	_ ConnectionReader   = (*Client)(nil)
	_ ConnectionWriter   = (*Client)(nil)
	_ RelatedNoteCreator = (*Client)(nil)
)

func writeRole(role domain.LinkRole) v1.Role {
	switch role {
	case domain.RoleParent:
		return v1.Role_ROLE_PARENT
	case domain.RoleChild:
		return v1.Role_ROLE_CHILD
	case domain.RoleJump:
		return v1.Role_ROLE_JUMP
	case domain.RoleRef:
		return v1.Role_ROLE_REF
	case domain.RoleAttachment:
		return v1.Role_ROLE_ATTACHMENT
	default:
		return v1.Role_ROLE_UNSPECIFIED
	}
}

func readRole(role v1.Role) domain.LinkRole {
	switch role {
	case v1.Role_ROLE_PARENT:
		return domain.RoleParent
	case v1.Role_ROLE_CHILD:
		return domain.RoleChild
	case v1.Role_ROLE_JUMP:
		return domain.RoleJump
	case v1.Role_ROLE_REF:
		return domain.RoleRef
	case v1.Role_ROLE_ATTACHMENT:
		return domain.RoleAttachment
	default:
		return ""
	}
}

func writeMarker(marker domain.LinkMarker) v1.Marker {
	if marker == domain.MarkerArrow {
		return v1.Marker_MARKER_ARROW
	}
	return v1.Marker_MARKER_UNSPECIFIED
}

func readMarker(marker v1.Marker) domain.LinkMarker {
	if marker == v1.Marker_MARKER_ARROW {
		return domain.MarkerArrow
	}
	return domain.MarkerNone
}

func writeSide(side *domain.Side) *v1.ConnectionSide {
	if side == nil {
		return nil
	}
	return &v1.ConnectionSide{
		Role: writeRole(side.Role), Type: side.Type, Note: side.Why,
		Label: side.Label, Marker: writeMarker(side.Marker),
	}
}

func readSide(side *v1.ConnectionSide) *domain.Side {
	if side == nil {
		return nil
	}
	return &domain.Side{
		Role: readRole(side.GetRole()), Type: side.GetType(), Why: side.GetNote(),
		Label: side.GetLabel(), Marker: readMarker(side.GetMarker()),
	}
}

func (c *Client) GetConnection(ctx context.Context, first, second, kind string) (domain.Connection, error) {
	res, err := c.rpc.GetConnection(ctx, request(c, &v1.HostVaultServiceGetConnectionRequest{
		First: first, Second: second, Type: kind,
	}))
	if err != nil {
		return domain.Connection{}, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return domain.Connection{}, err
	}
	got := res.Msg.GetConnection()
	return domain.Connection{
		A: got.GetFirst(), B: got.GetSecond(),
		AB: readSide(got.GetFirstSide()), BA: readSide(got.GetSecondSide()),
	}, nil
}

func (c *Client) SaveConnection(ctx context.Context, conn domain.Connection) error {
	res, err := c.rpc.SaveConnection(ctx, request(c, &v1.HostVaultServiceSaveConnectionRequest{
		Connection: &v1.Connection{
			First: conn.A, Second: conn.B,
			FirstSide: writeSide(conn.AB), SecondSide: writeSide(conn.BA),
		},
	}))
	if err != nil {
		return readError(err)
	}
	return fail(res.Msg.GetError())
}

func (c *Client) RemoveSide(ctx context.Context, from, to, kind string) error {
	res, err := c.rpc.RemoveSide(ctx, request(c, &v1.HostVaultServiceRemoveSideRequest{
		Path: from, To: to, Type: kind,
	}))
	if err != nil {
		return readError(err)
	}
	return fail(res.Msg.GetError())
}

func (c *Client) RemoveConnection(ctx context.Context, first, second, kind string) error {
	res, err := c.rpc.RemoveConnection(ctx, request(c, &v1.HostVaultServiceRemoveConnectionRequest{
		First: first, Second: second, Type: kind,
	}))
	if err != nil {
		return readError(err)
	}
	return fail(res.Msg.GetError())
}

func (c *Client) Join(ctx context.Context, from, to string, side domain.Side) error {
	res, err := c.rpc.JoinNotes(ctx, request(c, &v1.HostVaultServiceJoinNotesRequest{
		From: from, To: to, Side: writeSide(&side),
	}))
	if err != nil {
		return readError(err)
	}
	return fail(res.Msg.GetError())
}

func (c *Client) CreateInRelation(ctx context.Context, in domain.NewRelatedNote) (string, error) {
	res, err := c.rpc.CreateRelatedNote(ctx, request(c, &v1.HostVaultServiceCreateRelatedNoteRequest{
		From: in.From, Role: writeRole(in.Role), Title: in.Title, Type: in.Type,
		Marker: writeMarker(in.Marker),
	}))
	if err != nil {
		return "", readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return res.Msg.GetPath(), err
	}
	return res.Msg.GetPath(), nil
}

func (c *Client) ListBacklinks(ctx context.Context, to string) ([]domain.Backlink, error) {
	res, err := c.rpc.GetBacklinks(ctx, request(c, &v1.GetBacklinksRequest{Path: to}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	out := make([]domain.Backlink, 0, len(res.Msg.GetLinks()))
	for _, one := range res.Msg.GetLinks() {
		out = append(out, domain.Backlink{ResolvedLink: readLink(one), SourceRole: readRole(one.GetSourceRole())})
	}
	return out, nil
}
