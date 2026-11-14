package usecaseedges

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

func getUseCaseName(pkg string) (name string, isUseCase bool) {
	return strings.CutPrefix(pkg, "usecase/")
}

func getUseCaseEdgeRefusal(from, to string) string {
	source, isSource := getUseCaseName(from)
	if !isSource {
		return ""
	}
	if load.IsUnder(to, "internal") {
		if !load.IsHelper(to) {
			return "a use case imports from internal/ only a helper, a package directly under internal/ outside its adapter folders; the rule of a use case lives in the domain and what it depends on in a port"
		}
		return ""
	}
	target, isTarget := getUseCaseName(to)
	if !isTarget || source == target {
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
