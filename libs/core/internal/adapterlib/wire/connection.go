package wire

import (
	"errors"
	"fmt"

	v1 "github.com/jiva-studio/numen/libs/protocol/gen/numen/v1"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

// ConvertRoleFromWire is the role a link carries, in the core's words. The
// second result is false for a value the schema does not name as a role.
func ConvertRoleFromWire(role v1.Role) (domain.LinkRole, bool) {
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

// ConvertRoleToWire is a role as the schema carries it. A role the schema does
// not name is written as ROLE_UNSPECIFIED.
func ConvertRoleToWire(role domain.LinkRole) v1.Role {
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

// ConvertMarkerFromWire is the marker a link draws, in the core's words. The
// unspecified marker draws nothing. The second result is false for a value the
// schema does not name as a marker.
func ConvertMarkerFromWire(marker v1.Marker) (domain.LinkMarker, bool) {
	switch marker {
	case v1.Marker_MARKER_UNSPECIFIED:
		return domain.MarkerNone, true
	case v1.Marker_MARKER_ARROW:
		return domain.MarkerArrow, true
	default:
		return domain.MarkerNone, false
	}
}

// ConvertMarkerToWire is a marker as the schema carries it.
func ConvertMarkerToWire(marker domain.LinkMarker) v1.Marker {
	if marker == domain.MarkerArrow {
		return v1.Marker_MARKER_ARROW
	}
	return v1.Marker_MARKER_UNSPECIFIED
}

// ErrBadWireValue is returned for a role or a marker the schema does not name.
var ErrBadWireValue = errors.New("the schema names no such value")

// ConvertSideFromWire is the record of a connection in the core's words. A nil
// side is no record.
func ConvertSideFromWire(side *v1.ConnectionSide) (*domain.Side, error) {
	return convertSide(side, false)
}

// convertSide converts a record. A record without a role is accepted when
// isRoleOptional is set, and its role is left empty for the connection to fill.
func convertSide(side *v1.ConnectionSide, isRoleOptional bool) (*domain.Side, error) {
	if side == nil {
		return nil, nil
	}
	role, ok := ConvertRoleFromWire(side.GetRole())
	if !ok && !(isRoleOptional && side.GetRole() == v1.Role_ROLE_UNSPECIFIED) {
		return nil, fmt.Errorf("%w: role %v", ErrBadWireValue, side.GetRole())
	}
	marker, ok := ConvertMarkerFromWire(side.GetMarker())
	if !ok {
		return nil, fmt.Errorf("%w: marker %v", ErrBadWireValue, side.GetMarker())
	}
	return &domain.Side{
		Role: role, Type: side.GetType(), Why: side.GetNote(), Label: side.GetLabel(), Marker: marker,
	}, nil
}

// ConvertSideToWire is a record of a connection as the schema carries it. No
// record is nil.
func ConvertSideToWire(side *domain.Side) *v1.ConnectionSide {
	if side == nil {
		return nil
	}
	return &v1.ConnectionSide{
		Role: ConvertRoleToWire(side.Role), Type: side.Type, Note: side.Why,
		Label: side.Label, Marker: ConvertMarkerToWire(side.Marker),
	}
}

// ConvertConnectionFromWire is a connection in the core's words. A record
// without a role takes the role of the connection, or the inverse of the
// other record's role.
func ConvertConnectionFromWire(c *v1.Connection) (domain.Connection, error) {
	first, err := convertSide(c.GetFirstSide(), true)
	if err != nil {
		return domain.Connection{}, err
	}
	second, err := convertSide(c.GetSecondSide(), true)
	if err != nil {
		return domain.Connection{}, err
	}
	role, _ := ConvertRoleFromWire(c.GetRole())
	joined := domain.Connection{A: c.GetFirst(), B: c.GetSecond(), AB: first, BA: second}
	joined = joined.FillRoles(role)
	for _, side := range [...]*domain.Side{joined.AB, joined.BA} {
		if side != nil && side.Role == "" {
			return domain.Connection{}, fmt.Errorf("%w: a record carries no role and the connection gives none", ErrBadWireValue)
		}
	}
	return joined, nil
}

// ConvertConnectionToWire is a connection as the schema carries it.
func ConvertConnectionToWire(c domain.Connection) *v1.Connection {
	return &v1.Connection{
		First: c.A, Second: c.B, Role: ConvertRoleToWire(c.GetRole()),
		FirstSide: ConvertSideToWire(c.AB), SecondSide: ConvertSideToWire(c.BA),
	}
}
