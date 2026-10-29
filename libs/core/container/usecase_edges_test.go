package container

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// useCaseEdges are the use cases each use case may build on. Under strict Clean
// Architecture, use cases are independent application scenarios that compose
// via ports or in the composition root, never by importing sibling usecase packages.
var useCaseEdges = map[string][]string{
	"note":          nil,
	"settings":      nil,
	"run":           nil,
	"filetext":      nil,
	"vectors":       nil,
	"file":          nil,
	"search":        nil,
	"vault":         nil,
	"indexing":      nil,
	"completion":    nil,
	"plugincommand": nil,
	"textversion":   nil,
}

// getUseCaseName is the use case a package belongs to, and whether it is one.
func getUseCaseName(pkg string) (name string, isUseCase bool) {
	rest, found := strings.CutPrefix(pkg, "usecase/")
	if !found {
		return "", false
	}
	name, _, _ = strings.Cut(rest, "/")
	return name, true
}

// getUseCaseEdgeRefusal says why one package may not reach another, and nothing
// where it may.
func getUseCaseEdgeRefusal(from, to string) string {
	source, isSource := getUseCaseName(from)
	target, isTarget := getUseCaseName(to)
	if !isSource || !isTarget || source == target {
		return ""
	}
	if slices.Contains(useCaseEdges[source], target) {
		return ""
	}
	return "a use case builds only on the use cases useCaseEdges lists for it"
}

// A use case reaches another only along an edge useCaseEdges declares.
//
// Test files are not walked.
func TestUseCaseImportsFollowDeclaredEdges(t *testing.T) {
	var violations []string
	var importsRead, useCaseImportsRead int
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
		from := getPackage("..", path)
		if name, ok := getUseCaseName(from); ok {
			visited[name] = true
		}
		for _, spec := range file.Imports {
			to, err := strconv.Unquote(spec.Path.Value)
			if err != nil || !strings.HasPrefix(to, module) {
				continue
			}
			to = strings.TrimPrefix(to, module)
			importsRead++
			if _, ok := getUseCaseName(to); ok {
				useCaseImportsRead++
			}
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

	// The floors sit under what the tree holds; each use case is checked by
	// name because a count cannot say which use cases the walk visited.
	if importsRead < 50 {
		t.Fatalf("%d edges of this module read: the walk is not reading the use cases", importsRead)
	}
	_ = useCaseImportsRead
	for name := range useCaseEdges {
		if !visited[name] {
			t.Errorf("usecase/%s has no file the walk read: the rule stops at that use case", name)
		}
	}
}

// The edge rule is called directly on declared and undeclared edges.
func TestUseCaseEdgeRuleRefusesUndeclaredEdges(t *testing.T) {
	for _, tc := range []struct {
		from, to  string
		isRefused bool
	}{
		// Cross-use-case edges are refused.
		{"usecase/vault", "usecase/note", true},
		{"usecase/vault", "usecase/filetext", true},
		{"usecase/vault", "usecase/vectors", true},

		// The same edges reversed, and the edges skipping the declaration.
		{"usecase/note", "usecase/vault", true},
		{"usecase/filetext", "usecase/vault", true},
		{"usecase/filetext", "usecase/run", true},
		{"usecase/vectors", "usecase/vault", true},

		// A use case that builds on none.
		{"usecase/vectors", "usecase/note", true},
		{"usecase/file", "usecase/note", true},
		{"usecase/search", "usecase/note", true},
		{"usecase/note", "usecase/search", true},
		{"usecase/note", "usecase/vectors", true},

		// A package under a use case is that use case.
		{"usecase/note/sub", "usecase/vault", true},
		{"usecase/vault/sub", "usecase/note", true},
		{"usecase/note/sub", "usecase/note", false},

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
