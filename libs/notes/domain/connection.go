package domain

import (
	"errors"
	"fmt"
)

// Errors a Connection is refused with.
var (
	// ErrInvalidConnection is the base of every refusal of a Connection's shape.
	ErrInvalidConnection = errors.New("invalid connection")
	// ErrNoFreeName is returned when every numbered untitled note name is taken.
	ErrNoFreeName = errors.New("every untitled note name is taken")
	// ErrNotCreatable is returned for a relation a note cannot be created in.
	ErrNotCreatable = errors.New("a note cannot be created in this relation")
	// ErrHierarchyCycle is returned when a connection would make a note its own ancestor.
	ErrHierarchyCycle = errors.New("the hierarchy cannot contain a cycle")
)

// GetInverseRole returns the role the record in the other note carries for the
// same relationship. `parent: B` written in A and `child: A` written in B are
// one edge. The second result is false for a role with no counterpart.
func GetInverseRole(r LinkRole) (LinkRole, bool) {
	switch r {
	case RoleParent:
		return RoleChild, true
	case RoleChild:
		return RoleParent, true
	case RoleJump:
		return RoleJump, true
	case RoleRef:
		return RoleRef, true
	default:
		return "", false
	}
}

// IsConnectionRole reports whether a role can be the record of a connection:
// the roles a person draws between two notes. A reference is the body wikilink
// and an attachment points at no note.
func IsConnectionRole(r LinkRole) bool {
	return r == RoleParent || r == RoleChild || r == RoleJump
}

// Side is one record of a connection: the content of the entry a note writes
// about the other note. The target is not part of it; the connection supplies it.
type Side struct {
	Role   LinkRole
	Type   string
	Why    string
	Label  string
	Marker LinkMarker
}

// Connection is the relationship between two notes as a whole: up to two
// records, AB written in A and BA written in B. Each record has its own label
// and marker. A nil record means that note writes nothing about the other.
type Connection struct {
	A, B string
	AB   *Side
	BA   *Side
}

// GetRole returns what B is to A according to the records: the role of AB, else
// the inverse of the role of BA. It is empty for a connection with no record
// that carries a role.
func (c Connection) GetRole() LinkRole {
	if c.AB != nil && c.AB.Role != "" {
		return c.AB.Role
	}
	if c.BA != nil {
		inverse, _ := GetInverseRole(c.BA.Role)
		return inverse
	}
	return ""
}

// FillRoles returns the connection with the role of every record that has none
// filled in. role is what B is to A; an empty role is taken from the records
// that carry one. The record BA takes the inverse of the role.
func (c Connection) FillRoles(role LinkRole) Connection {
	if role == "" {
		role = c.GetRole()
	}
	inverse, _ := GetInverseRole(role)
	if c.AB != nil && c.AB.Role == "" {
		side := *c.AB
		side.Role = role
		c.AB = &side
	}
	if c.BA != nil && c.BA.Role == "" {
		side := *c.BA
		side.Role = inverse
		c.BA = &side
	}
	return c
}

// IsEmpty reports whether neither note writes a record.
func (c Connection) IsEmpty() bool {
	return c.AB == nil && c.BA == nil
}

// Validate checks the shape of a connection: two distinct notes, known roles
// and markers, and roles that are inverse of each other when both records exist.
func (c Connection) Validate() error {
	if c.A == "" || c.B == "" || c.A == c.B {
		return fmt.Errorf("%w: %q and %q are not two notes", ErrInvalidConnection, c.A, c.B)
	}
	for _, side := range [...]struct {
		from string
		side *Side
	}{{c.A, c.AB}, {c.B, c.BA}} {
		if side.side == nil {
			continue
		}
		if !IsConnectionRole(side.side.Role) {
			return fmt.Errorf("%w: %q is not a role a connection can carry", ErrInvalidConnection, side.side.Role)
		}
		if !IsWritableMarker(side.side.Marker) {
			return fmt.Errorf("%w: %q is not a marker a link can carry", ErrInvalidConnection, side.side.Marker)
		}
	}
	if c.AB == nil || c.BA == nil {
		return nil
	}
	inverse, ok := GetInverseRole(c.AB.Role)
	if !ok || inverse != c.BA.Role {
		return fmt.Errorf("%w: %s in %s and %s in %s are not one relationship",
			ErrInvalidConnection, c.AB.Role, c.A, c.BA.Role, c.B)
	}
	return nil
}

// GetHierarchyEdge returns the parent and the child the connection puts in the
// hierarchy. The third result is false for a connection that takes no part in it.
func (c Connection) GetHierarchyEdge() (parent, child string, ok bool) {
	if c.AB != nil {
		if parent, child, ok = getEdge(c.AB.Role, c.A, c.B); ok {
			return parent, child, true
		}
	}
	if c.BA != nil {
		return getEdge(c.BA.Role, c.B, c.A)
	}
	return "", "", false
}

// getEdge returns the parent and the child of a record written in `from` about
// `to`: `parent` makes `to` the parent, and `child` makes it the child.
func getEdge(role LinkRole, from, to string) (parent, child string, ok bool) {
	if role == RoleParent {
		return to, from, true
	}
	if role == RoleChild {
		return from, to, true
	}
	return "", "", false
}

// CheckAcyclic returns ErrHierarchyCycle when the connection makes a note its own
// ancestor. getParents answers the parents a note has now; the edges between the
// two notes of the connection are ignored, because the connection replaces them.
func (c Connection) CheckAcyclic(getParents func(path string) ([]string, error)) error {
	parent, child, ok := c.GetHierarchyEdge()
	if !ok {
		return nil
	}
	visited := map[string]bool{}
	pending := []string{parent}
	for len(pending) > 0 {
		node := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if node == child {
			return fmt.Errorf("%w: %s would sit above itself", ErrHierarchyCycle, child)
		}
		if visited[node] {
			continue
		}
		visited[node] = true
		parents, err := getParents(node)
		if err != nil {
			return err
		}
		for _, p := range parents {
			if c.isPair(node, p) {
				continue
			}
			pending = append(pending, p)
		}
	}
	return nil
}

func (c Connection) isPair(x, y string) bool {
	return (x == c.A && y == c.B) || (x == c.B && y == c.A)
}

// RelatedNote is a note to create next to an existing one.
type RelatedNote struct {
	// From is the note the new one is related to.
	From string
	// Role is what the new note is to From: its parent, its child, or a jump.
	Role LinkRole
	// Title names the note. Empty takes the first free "Untitled note N".
	Title string
	// Type is the type of the record the new note writes about From.
	Type string
	// Marker is what the record the new note writes draws at From.
	Marker LinkMarker
}
