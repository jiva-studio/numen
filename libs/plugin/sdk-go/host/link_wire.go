package host

import (
	v1 "github.com/jiva-studio/numen/libs/core/protocol/gen/numen/v1"
	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
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

func readMarker(marker v1.Marker) domain.LinkMarker {
	if marker == v1.Marker_MARKER_ARROW {
		return domain.MarkerArrow
	}
	return domain.MarkerNone
}
