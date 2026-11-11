package usecaseedges

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// No use case imports another use case. Every package under usecase, nested ones included, is a use case
// of its own: it imports neither its parent, nor its child, nor a sibling. A use case imports
// from internal/ only the packages of getSharedPackagesOfUseCases.
//
// Test files are not walked.
func TestNoUseCaseImportsAnotherUseCase(t *testing.T) {
	var violations []string
	var importsRead int
	visited := map[string]bool{}
	err := filepath.WalkDir("../usecase", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		if strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		from := load.GetPackage("..", path)
		if name, ok := getUseCaseName(from); ok {
			visited[name] = true
		}
		for _, spec := range file.Imports {
			to, err := strconv.Unquote(spec.Path.Value)
			if err != nil || !strings.HasPrefix(to, load.Module) {
				continue
			}
			to = strings.TrimPrefix(to, load.Module)
			importsRead++
			if reason := getUseCaseEdgeRefusal(from, to); reason != "" {
				violations = append(violations, from+" reaches "+to+": "+reason)
			}
		}
		return nil
	})
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
