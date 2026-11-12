package service_test

import (
	"slices"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/sdk-go/tools"
)

func TestTheVaultToolsRequireOnlyThePermissionsOfWhatTheyCall(t *testing.T) {
	want := map[string][]string{
		"vault_add":      {"vault:manage"},
		"vault_rename":   {"vault:manage"},
		"vault_remove":   {"vault:manage"},
		"vault_open":     {"vault:manage"},
		"library_search": {"index:read"},
		"library_read":   {"index:read"},
	}
	listing := listVaultTools(t)
	for name := range want {
		if !slices.ContainsFunc(listing.Tools, func(tool *tools.ToolInfo) bool { return tool.Name == name }) {
			t.Errorf("the registry lists no %s", name)
		}
	}

	for _, tool := range listing.Tools {
		if got := tool.Requires; !slices.Equal(got, want[tool.Name]) {
			t.Errorf("%s requires %q, want %q", tool.Name, got, want[tool.Name])
		}
	}
}
