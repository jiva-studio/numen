package mcp_test

import (
	"testing"

	"github.com/jiva-studio/numen/modules/libs/core/adapter/mcp"
)

// A vault tool is served only where what it works through is here.
func TestVaultToolsAreServedOnlyWhereTheirUseCasesAre(t *testing.T) {
	session := sessionOf(t, mcp.New(mcp.Core{}))

	tools, err := session.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range tools.Tools {
		if one.Name == "vault_get" || one.Name == "vault_problems" {
			t.Errorf("%s is served with nothing behind it", one.Name)
		}
	}
}
