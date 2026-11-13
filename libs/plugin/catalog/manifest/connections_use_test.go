package manifest_test

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

func readPermissions(permissions, settings string) (manifest.Manifest, error) {
	return manifestjson.Read(strings.NewReader(
		`{"id":"a.b","name":"A","version":"1.0.0","permissions":[` + permissions + `],"settings":[` + settings + `]}`,
	))
}

func TestAPermissionToUseConnectionsNamesItsOperations(t *testing.T) {
	m, err := readPermissions(`"vault:read","connections:use:llm.chat","connections:use:llm.profile"`, "")
	if err != nil {
		t.Fatal(err)
	}

	if got := m.ListConnectionsUse(); !slices.Equal(got, []string{"llm.chat", "llm.profile"}) {
		t.Errorf("ListConnectionsUse() = %v, want [llm.chat llm.profile]", got)
	}
}

func TestAPermissionToUseConnectionsWithoutAnOperationIsRefused(t *testing.T) {
	for _, permission := range []string{`"connections:use"`, `"connections:use:"`, `"connections:use:LLM Chat"`} {
		if _, err := readPermissions(permission, ""); !errors.Is(err, manifest.ErrInvalidPermission) {
			t.Errorf("%s: got %v, want ErrInvalidPermission", permission, err)
		}
	}
}

func TestAPermissionToUseConnectionsForAnOperationASettingOfRowsListsIsRefused(t *testing.T) {
	_, err := readPermissions(`"connections:use:llm.chat"`,
		`{"key":"chat","type":"connections","title":"Chat","operations":["llm.chat"],"max":1}`)

	if !errors.Is(err, manifest.ErrInvalidPermission) {
		t.Errorf("got %v, want ErrInvalidPermission", err)
	}
}
