package host

import (
	"context"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"
)

// ConnectionReader reads the relationships between the notes of the vault.
type ConnectionReader interface {
	// GetConnection returns the records two notes write about each other. A note
	// that writes none has a nil side. The empty type selects the records that
	// carry none.
	GetConnection(ctx context.Context, first, second, kind string) (Connection, error)
	// ListBacklinks lists the links written in other notes that point at a note.
	ListBacklinks(ctx context.Context, to string) ([]Backlink, error)
}

// ConnectionWriter writes the relationships between the notes of the vault.
type ConnectionWriter interface {
	// SaveConnection makes the records between two notes exactly the ones given,
	// in both notes or in neither.
	SaveConnection(ctx context.Context, c Connection) error
	// RemoveSide removes the records of the type that one note writes about
	// another and leaves the other note alone.
	RemoveSide(ctx context.Context, from, to, kind string) error
	// RemoveConnection removes the records two notes write about each other.
	RemoveConnection(ctx context.Context, first, second, kind string) error
	// Join makes one note write a record about another and keeps the record the
	// other note already writes.
	Join(ctx context.Context, from, to string, side Side) error
}

// Relations reads and writes the relationships between the notes of the vault.
// The host enforces the rules of a connection: the inverse role of the record in
// the other note, a hierarchy without cycles, and an atomic write of both notes.
type Relations interface {
	ConnectionReader
	ConnectionWriter
	// CreateInRelation creates a note joined to an existing one and returns its path.
	CreateInRelation(ctx context.Context, in RelatedNote) (string, error)
}

var _ Relations = (*Client)(nil)

func writeRole(role LinkRole) v1.Role {
	switch role {
	case RoleParent:
		return v1.Role_ROLE_PARENT
	case RoleChild:
		return v1.Role_ROLE_CHILD
	case RoleJump:
		return v1.Role_ROLE_JUMP
	case RoleRef:
		return v1.Role_ROLE_REF
	case RoleAttachment:
		return v1.Role_ROLE_ATTACHMENT
	default:
		return v1.Role_ROLE_UNSPECIFIED
	}
}

func readRole(role v1.Role) LinkRole {
	switch role {
	case v1.Role_ROLE_PARENT:
		return RoleParent
	case v1.Role_ROLE_CHILD:
		return RoleChild
	case v1.Role_ROLE_JUMP:
		return RoleJump
	case v1.Role_ROLE_REF:
		return RoleRef
	case v1.Role_ROLE_ATTACHMENT:
		return RoleAttachment
	default:
		return ""
	}
}

func writeMarker(marker LinkMarker) v1.Marker {
	if marker == MarkerArrow {
		return v1.Marker_MARKER_ARROW
	}
	return v1.Marker_MARKER_UNSPECIFIED
}

func readMarker(marker v1.Marker) LinkMarker {
	if marker == v1.Marker_MARKER_ARROW {
		return MarkerArrow
	}
	return MarkerNone
}

func writeSide(side *Side) *v1.ConnectionSide {
	if side == nil {
		return nil
	}
	return &v1.ConnectionSide{
		Role: writeRole(side.Role), Type: side.Type, Note: side.Why,
		Label: side.Label, Marker: writeMarker(side.Marker),
	}
}

func readSide(side *v1.ConnectionSide) *Side {
	if side == nil {
		return nil
	}
	return &Side{
		Role: readRole(side.GetRole()), Type: side.GetType(), Why: side.GetNote(),
		Label: side.GetLabel(), Marker: readMarker(side.GetMarker()),
	}
}

func (c *Client) GetConnection(ctx context.Context, first, second, kind string) (Connection, error) {
	res, err := c.rpc.GetConnection(ctx, request(c, &v1.HostVaultServiceGetConnectionRequest{
		First: first, Second: second, Type: kind,
	}))
	if err != nil {
		return Connection{}, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return Connection{}, err
	}
	got := res.Msg.GetConnection()
	return Connection{
		A: got.GetFirst(), B: got.GetSecond(),
		AB: readSide(got.GetFirstSide()), BA: readSide(got.GetSecondSide()),
	}, nil
}

func (c *Client) SaveConnection(ctx context.Context, conn Connection) error {
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

func (c *Client) Join(ctx context.Context, from, to string, side Side) error {
	res, err := c.rpc.JoinNotes(ctx, request(c, &v1.HostVaultServiceJoinNotesRequest{
		From: from, To: to, Side: writeSide(&side),
	}))
	if err != nil {
		return readError(err)
	}
	return fail(res.Msg.GetError())
}

func (c *Client) CreateInRelation(ctx context.Context, in RelatedNote) (string, error) {
	res, err := c.rpc.CreateRelatedNote(ctx, request(c, &v1.HostVaultServiceCreateRelatedNoteRequest{
		From: in.From, Role: writeRole(in.Role), Title: in.Title, Type: in.Type,
	}))
	if err != nil {
		return "", readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return res.Msg.GetPath(), err
	}
	return res.Msg.GetPath(), nil
}

func (c *Client) ListBacklinks(ctx context.Context, to string) ([]Backlink, error) {
	res, err := c.rpc.GetBacklinks(ctx, request(c, &v1.GetBacklinksRequest{Path: to}))
	if err != nil {
		return nil, readError(err)
	}
	if err := fail(res.Msg.GetError()); err != nil {
		return nil, err
	}
	out := make([]Backlink, 0, len(res.Msg.GetLinks()))
	for _, one := range res.Msg.GetLinks() {
		out = append(out, Backlink{ResolvedLink: readLink(one), RoleFromHere: readRole(one.GetRoleFromHere())})
	}
	return out, nil
}
