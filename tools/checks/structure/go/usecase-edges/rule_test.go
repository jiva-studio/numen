package usecaseedges

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func getUseCaseName(pkg string) (name string, isUseCase bool) {
	return strings.CutPrefix(pkg, "usecase/")
}

func getSharedPackagesOfUseCases() map[string]string {
	return map[string]string{
		"internal/text":   "cuts a byte range on character boundaries and imports only the standard library",
		"internal/vector": "quantizes embedding vectors and imports only the embedding domain",
	}
}

func getUseCaseEdgeRefusal(from, to string) string {
	if load.IsUseCaseTestSupport(to) && !load.IsUseCaseTestSupport(from) {
		return "only a test imports test support"
	}
	source, isSource := getUseCaseName(from)
	if !isSource {
		return ""
	}
	if load.IsUnder(to, "internal") {
		if _, isAllowed := getSharedPackagesOfUseCases()[to]; !isAllowed {
			return "a use case imports from internal/ only the listed shared packages; the rule of a use case lives in the domain and what it depends on in a port"
		}
		return ""
	}
	target, isTarget := getUseCaseName(to)
	if !isTarget || source == target || load.IsUseCaseTestSupport(from) {
		return ""
	}
	return "a use case imports no other use case"
}

func getUseCasesWithSource(t *testing.T) []string {
	t.Helper()
	var names []string
	err := filepath.WalkDir("../usecase", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.IsDir() || path == "../usecase" {
			return err
		}
		files, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, file := range files {
			if name := file.Name(); !file.IsDir() && strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
				names = append(names, strings.TrimPrefix(filepath.ToSlash(path), "../usecase/"))
				break
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return names
}

// listEdgeViolations reads the imports of the production files of the module at root, whose import path is
// module with a trailing slash, and returns the refused edges, the number of edges read and the use cases
// with a file read.
func listEdgeViolations(root, module string) (violations []string, importsRead int, visited map[string]bool, err error) {
	visited = map[string]bool{}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && entry.Name() == "testdata" {
			return fs.SkipDir
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		from := load.GetPackage(root, path)
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
	return violations, importsRead, visited, err
}
