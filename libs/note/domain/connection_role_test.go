package domain_test

import (
	"testing"

	"github.com/jiva-studio/numen/libs/note/domain"
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

func TestGetRoleAndFillRolesHandleOneSidedAndPrepopulatedRecords(t *testing.T) {
	abOnly := domain.Connection{A: "a", B: "b", AB: &domain.Side{Role: domain.RoleChild}}
	if got := abOnly.GetRole(); got != domain.RoleChild {
		t.Errorf("GetRole = %s, want child", got)
	}
	filledAB := abOnly.FillRoles(domain.RoleParent)
	if filledAB.AB.Role != domain.RoleChild || filledAB.BA != nil {
		t.Errorf("FillRoles overwrote AB or created BA: %+v", filledAB)
	}

	baOnly := domain.Connection{A: "a", B: "b", BA: &domain.Side{Role: domain.RoleParent}}
	filledBA := baOnly.FillRoles(domain.RoleParent)
	if filledBA.AB != nil || filledBA.BA.Role != domain.RoleParent {
		t.Errorf("FillRoles created AB or overwrote BA: %+v", filledBA)
	}

	empty := domain.Connection{A: "a", B: "b"}
	if got := empty.GetRole(); got != "" {
		t.Errorf("GetRole of empty = %q, want empty", got)
	}
	if !empty.IsEmpty() || abOnly.IsEmpty() || baOnly.IsEmpty() {
		t.Error("IsEmpty reported wrong emptiness")
	}
}
