// Package linkrule holds what a link written into the frontmatter of a note must satisfy.
package linkrule

import (
	"fmt"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/host/domain"
)

// CheckWritable rejects a link whose role is unknown or whose marker cannot be written.
func CheckWritable(link domain.Link) error {
	if !domain.IsKnownRole(link.Role) {
		return roleError(link.Role)
	}
	if !domain.IsWritableMarker(link.Marker) {
		return fmt.Errorf("invalid marker %q", link.Marker)
	}
	return nil
}

// CheckRole rejects a role that is neither empty nor known.
func CheckRole(role domain.LinkRole) error {
	if role != "" && !domain.IsKnownRole(role) {
		return roleError(role)
	}
	return nil
}

func roleError(role domain.LinkRole) error {
	return fmt.Errorf("invalid role %q: use parent, child, jump, ref or attachment", role)
}
