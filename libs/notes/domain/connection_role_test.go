package domain_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/notes/domain"
)

func TestFillRolesSetsTheInverseRoleOnTheOtherRecord(t *testing.T) {
	c := domain.Connection{A: "a", B: "b", AB: &domain.Side{}, BA: &domain.Side{}}.FillRoles(domain.RoleParent)

	if c.AB.Role != domain.RoleParent || c.BA.Role != domain.RoleChild {
		t.Errorf("roles = %s and %s", c.AB.Role, c.BA.Role)
	}
	if err := c.Validate(); err != nil {
		t.Error(err)
	}
}

func TestFillRolesReadsTheRoleFromTheRecordThatHasOne(t *testing.T) {
	c := domain.Connection{A: "a", B: "b", AB: &domain.Side{}, BA: &domain.Side{Role: domain.RoleChild}}.FillRoles("")

	if c.AB.Role != domain.RoleParent {
		t.Errorf("role = %s, want parent", c.AB.Role)
	}
	if got := c.GetRole(); got != domain.RoleParent {
		t.Errorf("GetRole = %s", got)
	}
}

func TestGetRoleOfAConnectionWithOnlyTheSecondRecordIsItsInverse(t *testing.T) {
	c := domain.Connection{A: "a", B: "b", BA: &domain.Side{Role: domain.RoleChild}}

	if got := c.GetRole(); got != domain.RoleParent {
		t.Errorf("GetRole = %s, want parent", got)
	}
}
