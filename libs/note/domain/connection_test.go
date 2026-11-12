package domain_test

import (
	"errors"
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
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
		"empty":       {domain.Connection{A: "a", B: "b"}, true},
		"same note":   {domain.Connection{A: "a", B: "a"}, false},
		"no note":     {domain.Connection{A: "a"}, false},
		"no note A":   {domain.Connection{B: "b"}, false},
		"only BA":     {domain.Connection{A: "a", B: "b", BA: &domain.Side{Role: domain.RoleParent}}, true},
		"bad role":    {domain.Connection{A: "a", B: "b", AB: &domain.Side{Role: "sideways"}}, false},
		"bad role BA": {domain.Connection{A: "a", B: "b", BA: &domain.Side{Role: "sideways"}}, false},
		"bad marker":  {domain.Connection{A: "a", B: "b", AB: &domain.Side{Role: domain.RoleJump, Marker: "circle"}}, false},
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
	abChild := domain.Connection{A: "a", B: "b", AB: &domain.Side{Role: domain.RoleChild}}
	if p, c, ok := abChild.GetHierarchyEdge(); !ok || p != "a" || c != "b" {
		t.Errorf("child in A: %s, %s, %v", p, c, ok)
	}
	ba := domain.Connection{A: "a", B: "b", BA: &domain.Side{Role: domain.RoleParent}}
	if p, c, ok := ba.GetHierarchyEdge(); !ok || p != "a" || c != "b" {
		t.Errorf("parent in B: %s, %s, %v", p, c, ok)
	}
	baChild := domain.Connection{A: "a", B: "b", BA: &domain.Side{Role: domain.RoleChild}}
	if p, c, ok := baChild.GetHierarchyEdge(); !ok || p != "b" || c != "a" {
		t.Errorf("child in B: %s, %s, %v", p, c, ok)
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

func TestCheckAcyclicHandlesVisitedErrorsAndPairDirections(t *testing.T) {
	// Reverse pair direction: b currently has parent a, and connection makes b parent of a.
	reverseEdges := map[string][]string{"b": {"a"}}
	reverseConn := domain.Connection{A: "a", B: "b", AB: &domain.Side{Role: domain.RoleParent}}
	if err := reverseConn.CheckAcyclic(parentsOf(reverseEdges)); err != nil {
		t.Errorf("reverse pair replacement failed: %v", err)
	}

	// Pair edge followed by another parent that creates a cycle.
	pairPlusCycle := map[string][]string{"a": {"b", "c"}, "c": {"b"}}
	childConn := domain.Connection{A: "a", B: "b", AB: &domain.Side{Role: domain.RoleChild}}
	if err := childConn.CheckAcyclic(parentsOf(pairPlusCycle)); !errors.Is(err, domain.ErrHierarchyCycle) {
		t.Errorf("got %v, want ErrHierarchyCycle", err)
	}

	// Non-pair edges touching A or B individually still form a cycle.
	threeCycle := map[string][]string{"a": {"c"}, "c": {"b"}}
	if err := childConn.CheckAcyclic(parentsOf(threeCycle)); !errors.Is(err, domain.ErrHierarchyCycle) {
		t.Errorf("got %v, want ErrHierarchyCycle", err)
	}

	// Visited node is queried only once, and remaining pending branches are still explored.
	diamondThenCycle := map[string][]string{
		"d": {"x", "b", "c"},
		"b": {"a"},
		"c": {"a"},
		"x": {"child"},
	}
	calls := map[string]int{}
	err := (domain.Connection{A: "child", B: "d", AB: &domain.Side{Role: domain.RoleParent}}).CheckAcyclic(func(path string) ([]string, error) {
		calls[path]++
		if calls[path] > 1 {
			return nil, errors.New("visited node queried twice")
		}
		return diamondThenCycle[path], nil
	})
	if !errors.Is(err, domain.ErrHierarchyCycle) {
		t.Errorf("got %v, want ErrHierarchyCycle", err)
	}

	// Error from getParents is propagated.
	wantErr := errors.New("storage failure")
	err = reverseConn.CheckAcyclic(func(string) ([]string, error) {
		return nil, wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Errorf("got %v, want %v", err, wantErr)
	}
}

func TestFindSide(t *testing.T) {
	links := []domain.ResolvedLink{
		{Link: domain.Link{Role: domain.RoleRef}, ToVault: "v1", To: "Delta.md"},
		{Link: domain.Link{Role: domain.RoleParent, Type: "river", Why: "flows into"}, ToVault: "v2", To: "Delta.md"},
		{Link: domain.Link{Role: domain.RoleParent, Type: "lake"}, ToVault: "v1", To: "Delta.md"},
		{Link: domain.Link{Role: domain.RoleParent, Type: "river", Why: "flows into", Label: "feeds"}, ToVault: "v1", To: "Delta.md"},
	}
	got := domain.FindSide(links, "v1", "Delta.md", "river")
	if got == nil || got.Role != domain.RoleParent || got.Why != "flows into" || got.Label != "feeds" {
		t.Errorf("side = %+v", got)
	}
	if side := domain.FindSide(links, "v1", "Delta.md", ""); side != nil {
		t.Errorf("a body reference is not a side: %+v", side)
	}
	if side := domain.FindSide(links, "v1", "Marsh.md", "river"); side != nil {
		t.Errorf("another note: %+v", side)
	}
}
