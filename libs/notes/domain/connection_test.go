package domain_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

func TestInverseRolesPairUp(t *testing.T) {
	for role, want := range map[domain.LinkRole]domain.LinkRole{
		domain.RoleParent: domain.RoleChild,
		domain.RoleChild:  domain.RoleParent,
		domain.RoleJump:   domain.RoleJump,
		domain.RoleRef:    domain.RoleRef,
	} {
		if got, ok := domain.GetInverseRole(role); !ok || got != want {
			t.Errorf("inverse of %s = %s, %v; want %s", role, got, ok, want)
		}
	}
	if _, ok := domain.GetInverseRole(domain.RoleAttachment); ok {
		t.Error("an attachment has no inverse")
	}
}

func TestConnectionValidation(t *testing.T) {
	arrow := domain.MarkerArrow
	cases := map[string]struct {
		c  domain.Connection
		ok bool
	}{
		"one side": {domain.Connection{A: "a", B: "b", AB: &domain.Side{Role: domain.RoleJump}}, true},
		"two inverse sides": {domain.Connection{A: "a", B: "b",
			AB: &domain.Side{Role: domain.RoleParent}, BA: &domain.Side{Role: domain.RoleChild}}, true},
		"an arrow with no label": {domain.Connection{A: "a", B: "b",
			AB: &domain.Side{Role: domain.RoleJump, Marker: arrow}}, true},
		"empty":      {domain.Connection{A: "a", B: "b"}, true},
		"same note":  {domain.Connection{A: "a", B: "a"}, false},
		"no note":    {domain.Connection{A: "a"}, false},
		"bad role":   {domain.Connection{A: "a", B: "b", AB: &domain.Side{Role: "sideways"}}, false},
		"bad marker": {domain.Connection{A: "a", B: "b", AB: &domain.Side{Role: domain.RoleJump, Marker: "circle"}}, false},
		"same role on both ends": {domain.Connection{A: "a", B: "b",
			AB: &domain.Side{Role: domain.RoleParent}, BA: &domain.Side{Role: domain.RoleParent}}, false},
		"attachment pair": {domain.Connection{A: "a", B: "b",
			AB: &domain.Side{Role: domain.RoleAttachment}, BA: &domain.Side{Role: domain.RoleAttachment}}, false},
		"a reference is not drawn": {domain.Connection{A: "a", B: "b",
			AB: &domain.Side{Role: domain.RoleRef}}, false},
	}
	for name, tc := range cases {
		err := tc.c.Validate()
		if (err == nil) != tc.ok {
			t.Errorf("%s: err = %v, want ok = %v", name, err, tc.ok)
		}
		if err != nil && !errors.Is(err, domain.ErrInvalidConnection) {
			t.Errorf("%s: %v is not ErrInvalidConnection", name, err)
		}
	}
}

func TestTheHierarchyEdgeIsReadFromEitherSide(t *testing.T) {
	ab := domain.Connection{A: "a", B: "b", AB: &domain.Side{Role: domain.RoleParent}}
	if p, c, ok := ab.GetHierarchyEdge(); !ok || p != "b" || c != "a" {
		t.Errorf("parent in A: %s, %s, %v", p, c, ok)
	}
	ba := domain.Connection{A: "a", B: "b", BA: &domain.Side{Role: domain.RoleParent}}
	if p, c, ok := ba.GetHierarchyEdge(); !ok || p != "a" || c != "b" {
		t.Errorf("parent in B: %s, %s, %v", p, c, ok)
	}
	jump := domain.Connection{A: "a", B: "b", AB: &domain.Side{Role: domain.RoleJump}}
	if _, _, ok := jump.GetHierarchyEdge(); ok {
		t.Error("a jump takes no part in the hierarchy")
	}
}

func parentsOf(edges map[string][]string) func(string) ([]string, error) {
	return func(path string) ([]string, error) { return edges[path], nil }
}

func TestACycleThroughAnAncestorIsRefused(t *testing.T) {
	// c's parent is b, b's parent is a. Making a a child of c closes the loop.
	edges := map[string][]string{"c": {"b"}, "b": {"a"}}
	c := domain.Connection{A: "a", B: "c", AB: &domain.Side{Role: domain.RoleParent}}
	if err := c.CheckAcyclic(parentsOf(edges)); !errors.Is(err, domain.ErrHierarchyCycle) {
		t.Errorf("err = %v, want ErrHierarchyCycle", err)
	}
}

func TestASelfContainedEdgeBetweenThePairIsReplaced(t *testing.T) {
	// b is a's parent now; the connection makes a b's parent. The old edge is the
	// one being replaced, and does not count.
	edges := map[string][]string{"a": {"b"}}
	c := domain.Connection{A: "a", B: "b", AB: &domain.Side{Role: domain.RoleChild}}
	if err := c.CheckAcyclic(parentsOf(edges)); err != nil {
		t.Errorf("err = %v", err)
	}
}

func TestADiamondIsNotACycle(t *testing.T) {
	edges := map[string][]string{"d": {"b", "c"}, "b": {"a"}, "c": {"a"}}
	c := domain.Connection{A: "x", B: "d", AB: &domain.Side{Role: domain.RoleParent}}
	if err := c.CheckAcyclic(parentsOf(edges)); err != nil {
		t.Errorf("err = %v", err)
	}
}

func TestAJumpIsNeverACycle(t *testing.T) {
	edges := map[string][]string{"a": {"b"}}
	c := domain.Connection{A: "b", B: "a", AB: &domain.Side{Role: domain.RoleJump}}
	if err := c.CheckAcyclic(parentsOf(edges)); err != nil {
		t.Errorf("err = %v", err)
	}
}
