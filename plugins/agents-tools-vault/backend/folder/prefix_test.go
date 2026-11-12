package folder_test

import (
	"testing"

	"github.com/jiva-studio/numen/plugins/agents-tools-vault/backend/folder"
)

func TestGetPrefixNormalizesTheFolderName(t *testing.T) {
	for name, want := range map[string]string{
		"":              "",
		".":             "",
		"/":             "",
		"./":            "",
		"Rivers":        "Rivers/",
		"/Rivers/":      "Rivers/",
		"./Rivers/Nile": "Rivers/Nile/",
	} {
		if got := folder.GetPrefix(name); got != want {
			t.Errorf("GetPrefix(%q) = %q, want %q", name, got, want)
		}
	}
}
