// Package jsonkeys is the check `go/json-keys`, run on the tree with the planted violations that prove it.
package jsonkeys

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// The rule rejects snake_case and PascalCase JSON struct tags and untagged boundary struct fields,
// and accepts camelCase tags and ignored fields.
func TestJSONKeysRule_RejectsSnakeCaseAndUntaggedBoundaryFields(t *testing.T) {
	dir := load.WriteFiles(t, map[string]string{
		"sdk/sdk.go": `package sdk

func Handle[Req, Res any](fn func(Req) Res) {}

func Cache[V any]() {}
`,
		"app/shapes.go": `package app

import "fixture/sdk"

type ValidConfig struct {
	MaxTokens int    ` + "`json:\"maxTokens\"`" + `
	FloorHint string ` + "`json:\"floorHint,omitempty\"`" + `
	Internal  string ` + "`json:\"-\"`" + `
}

type BadExported struct {
	MaxTokens int ` + "`json:\"max_tokens\"`" + `
	Count     int ` + "`json:\"Count\"`" + `
}

type badUnexported struct {
	QueryPrefix string ` + "`json:\"query_prefix\"`" + `
}

type cachedModel struct {
	Name  string
	Title string ` + "`json:\"title\"`" + `
}

func register() {
	sdk.Handle(func(in ValidConfig) ValidConfig { return in })
	sdk.Cache[[]cachedModel]()
}
`,
	})
	pkgs := load.LoadPackages(t, dir, []string{"GOWORK=off"}, "./...")
	violations, checked := getJSONKeyViolations(pkgs)
	var got []string
	for _, one := range violations {
		got = append(got, filepath.Base(one.position)+" "+one.detail)
	}
	want := []string{
		`shapes.go:12 MaxTokens has non-camelCase json tag "max_tokens"`,
		`shapes.go:13 Count has non-camelCase json tag "Count"`,
		`shapes.go:17 QueryPrefix has non-camelCase json tag "query_prefix"`,
		`shapes.go:21 Name has no json tag`,
	}
	if !slices.Equal(got, want) {
		t.Errorf("violations = %v, want %v", got, want)
	}
	if checked < 5 {
		t.Errorf("checked = %d, want at least 5", checked)
	}
}

func TestProjectJSONKeysAreCamelCase(t *testing.T) {
	if testing.Short() {
		t.Skip("loads and type-checks the modules of the workspace")
	}
	checkProjectJSONKeysAreCamelCase(t, load.Workspace(t))
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
