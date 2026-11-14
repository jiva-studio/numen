// Package usecaseedges is the check `go/usecase-edges`, run on the tree with the planted violations that prove it.
package usecaseedges

import (
	"os"
	"slices"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// No use case imports another use case. Every package under usecase, nested ones included, is a use case
// of its own: it imports neither its parent, nor its child, nor a sibling. A use case imports
// from internal/ only the packages of getSharedPackagesOfUseCases. A package under usecase whose directory
// name ends in "test" is test support: it may import any use case, and no production file of the module
// imports it, so the walk reads the whole module.
//
// Test files are not walked.
func TestNoUseCaseImportsAnotherUseCase(t *testing.T) {
	violations, importsRead, visited, err := listEdgeViolations("..", load.Module)
	if err != nil {
		t.Fatal(err)
	}
	for _, violation := range violations {
		t.Error(violation)
	}

	// Each use case is checked by name because a count cannot say which use cases the walk visited.
	if importsRead == 0 {
		t.Fatalf("%d edges of this module read: the walk is not reading the use cases", importsRead)
	}
	for _, name := range getUseCasesWithSource(t) {
		if !visited[name] {
			t.Errorf("usecase/%s has no file the walk read: the rule stops at that use case", name)
		}
	}
}

// The edge rule is called directly on imports between use cases and imports within one.
func TestUseCaseEdgeRule_RejectsImportsBetweenUseCases(t *testing.T) {
	for _, tc := range []struct {
		from, to  string
		isRefused bool
	}{
		// Imports between two use cases are refused.
		{"usecase/vault", "usecase/note", true},
		{"usecase/vault", "usecase/filetext", true},
		{"usecase/vault", "usecase/vectors", true},

		// The same imports reversed.
		{"usecase/note", "usecase/vault", true},
		{"usecase/filetext", "usecase/vault", true},
		{"usecase/filetext", "usecase/run", true},
		{"usecase/vectors", "usecase/vault", true},

		{"usecase/vectors", "usecase/note", true},
		{"usecase/file", "usecase/note", true},
		{"usecase/search", "usecase/note", true},
		{"usecase/note", "usecase/search", true},
		{"usecase/note", "usecase/vectors", true},

		// A package under a use case is a use case of its own: parent, child and sibling are refused, and
		// a package imports itself.
		{"usecase/note/sub", "usecase/vault", true},
		{"usecase/vault/sub", "usecase/note", true},
		{"usecase/note/sub", "usecase/note", true},
		{"usecase/note", "usecase/note/sub", true},
		{"usecase/vault/registry", "usecase/vault/check", true},
		{"usecase/vault", "usecase/vault/registry/deep", true},
		{"usecase/note/query", "usecase/note/query", false},
		{"usecase/note", "usecase/note", false},
		{"usecase/notebook", "usecase/note", true},

		// Shared packages: the listed ones are allowed, every other package under internal/ is refused.
		{"usecase/vectors", "internal/vector", false},
		{"usecase/file", "internal/text", false},
		{"usecase/vault", "internal/vaultreading", true},
		{"usecase/vault", "internal/operation", true},
		{"usecase/vault", "internal/adapter/filesystem/vaultreader", true},

		// A package whose directory name ends in "test" is test support: it imports the use case it sits in
		// and any other use case or test support, and no production package imports it.
		{"usecase/x/xtest", "usecase/x", false},
		{"usecase/modelcall/modelcalltest", "usecase/connection", false},
		{"usecase/modelcall/modelcalltest", "usecase/connection/connectiontest", false},
		{"usecase/x/xtest", "internal/adapterlib/eventbus", true},
		{"usecase/x", "usecase/x/xtest", true},
		{"adapter/window/connection", "usecase/x/xtest", true},
		{"container", "usecase/x/xtest", true},

		// Pairs with a side that is not a use case are not this rule's.
		{"usecase/note", "port", false},
		{"container", "usecase/vault", false},
	} {
		reason := getUseCaseEdgeRefusal(tc.from, tc.to)
		if tc.isRefused && reason == "" {
			t.Errorf("%s reaches %s and is not rejected", tc.from, tc.to)
		}
		if !tc.isRefused && reason != "" {
			t.Errorf("%s reaches %s and fails: %s", tc.from, tc.to, reason)
		}
	}
}

// The walk reads the production files of the whole module: test support that imports the use case it sits in
// passes, a test that imports test support passes, and a production file that imports it fails.
func TestEdgeWalk_RejectsAProductionImportOfTestSupport(t *testing.T) {
	dir := load.WriteTreeFixture(t, map[string]string{
		"usecase/x/x.go":           "package x\n",
		"usecase/x/xtest/xtest.go": "package xtest\n\nimport _ \"fixture/usecase/x\"\n",
		"usecase/x/x_test.go":      "package x_test\n\nimport _ \"fixture/usecase/x/xtest\"\n",
		"adapter/y/y.go":           "package y\n\nimport _ \"fixture/usecase/x/xtest\"\n",
	})
	violations, importsRead, _, err := listEdgeViolations(dir, "fixture/")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"adapter/y reaches usecase/x/xtest: only a test imports test support"}
	if !slices.Equal(violations, want) || importsRead != 2 {
		t.Errorf("the walk read %d edges and refuses %v, want 2 edges and %v", importsRead, violations, want)
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
