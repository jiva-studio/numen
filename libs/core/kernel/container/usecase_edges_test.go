package container

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func getUseCaseName(pkg string) (name string, isUseCase bool) {
	rest, found := strings.CutPrefix(pkg, "usecase/")
	if !found {
		return "", false
	}
	name, _, _ = strings.Cut(rest, "/")
	return name, true
}

func getUseCaseEdgeRefusal(from, to string) string {
	source, isSource := getUseCaseName(from)
	target, isTarget := getUseCaseName(to)
	if !isSource || !isTarget || source == target {
		return ""
	}
	return "a use case imports no other use case"
}

// No use case imports another use case; a package below a use case is that use case.
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
	folders, err := os.ReadDir("../usecase")
	if err != nil {
		t.Fatal(err)
	}
	for _, folder := range folders {
		if name := folder.Name(); folder.IsDir() && !visited[name] {
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
