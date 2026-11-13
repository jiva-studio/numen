package manifest_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/libs/plugin/catalog/manifest"
	"github.com/jiva-studio/numen/libs/plugin/catalog/manifestjson"
)

func readOperations(operations string) (manifest.Manifest, error) {
	return manifestjson.Read(strings.NewReader(`{"id":"a.b","name":"A","version":"1.0.0","contributes":{"operations":[` + operations + `]}}`))
}

func TestAManifestTitlesOperations(t *testing.T) {
	m, err := readOperations(`{"name":"acp.task","title":"run agent tasks"},{"name":"llm.models","title":"list models"}`)
	if err != nil {
		t.Fatal(err)
	}

	want := []manifest.Operation{{Name: "acp.task", Title: "run agent tasks"}, {Name: "llm.models", Title: "list models"}}
	if len(m.Contributes.Operations) != len(want) || m.Contributes.Operations[0] != want[0] || m.Contributes.Operations[1] != want[1] {
		t.Errorf("operations = %+v, want %+v", m.Contributes.Operations, want)
	}
}

func TestAnOperationTitleIsRejectedWhereItIsWrong(t *testing.T) {
	for name, operations := range map[string]string{
		"a name that is not words joined by dots": `{"name":"Acp_Task","title":"run agent tasks"}`,
		"a name with no title":                    `{"name":"acp.task","title":" "}`,
		"a name declared twice":                   `{"name":"acp.task","title":"run agent tasks"},{"name":"acp.task","title":"run tasks"}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := readOperations(operations); !errors.Is(err, manifest.ErrInvalidContribution) {
				t.Errorf("err = %v, want an invalid contribution", err)
			}
		})
	}
}
