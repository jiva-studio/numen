// Package notelibrary is the check `go/note-library`, run on the tree with the planted violations that prove it.
package notelibrary

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/jiva-studio/numen/tools/checks/structure/go/load"
)

// notesModule is what the packages of libs/note are named under.
const notesModule = "github.com/jiva-studio/numen/libs/note/"

// rootPackage names the files of the module folder itself.
const rootPackage = "."

// A package of libs/note that is not listed fails the check.
func getInternalEdges() map[string][]string {
	return map[string][]string{
		rootPackage: nil,
		"domain":    nil,
		"ulid":      nil,
		"markdown":  {"domain", "ulid"},
		"edit":      {"domain", "ulid", "markdown"},
		"textfile":  {"domain"},
	}
}

func getForbiddenInfra() map[string]bool {
	return map[string]bool{
		"os":            true,
		"os/exec":       true,
		"net":           true,
		"net/http":      true,
		"database/sql":  true,
		"encoding/json": true,
		"syscall":       true,
	}
}

func getAllowedInfra() map[string][]string {
	return map[string][]string{
		"textfile": {"encoding/json"},
	}
}

func listThirdParty() []string {
	return []string{"gopkg.in/yaml.v3", "golang.org/x/text/"}
}

// The library reaches only itself, the standard library and the few third-party packages listed,
// performs no I/O and no wire serialization outside the packages that are allowed one, and its
// packages import each other along edit -> markdown -> {domain, ulid}.
func TestNoteLibraryKeepsItsLayersAndItsPurity(t *testing.T) {
	found, err := findNoteLibraryViolations(filepath.Join(load.RepoRootDir, "libs", "note"))
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range found {
		t.Error(one)
	}
}

// TestNoteLibraryWalk_FindsThePlantedViolations verifies on a planted tree that the walk finds a
// package that is not listed, a forbidden import, an import against the layers and a third-party
// import, and passes a package that is allowed its infrastructure import.
func TestNoteLibraryWalk_FindsThePlantedViolations(t *testing.T) {
	root := load.WriteTreeFixture(t, map[string]string{
		"doc.go":                "package notes\n",
		"domain/note.go":        "package domain\n\nimport \"os\"\n\nvar _ = os.Args\n",
		"ulid/ulid.go":          "package ulid\n",
		"markdown/markdown.go":  "package markdown\n\nimport _ \"" + notesModule + "edit\"\n",
		"edit/edit.go":          "package edit\n\nimport _ \"github.com/google/uuid\"\n",
		"textfile/textfile.go":  "package textfile\n\nimport _ \"encoding/json\"\n\nimport _ \"" + notesModule + "domain\"\n",
		"newpackage/package.go": "package newpackage\n",
	})
	found, err := findNoteLibraryViolations(root)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, one := range found {
		got = append(got, strings.TrimPrefix(one, root+"/"))
	}
	want := []string{
		"domain/note.go imports os: libs/note performs no I/O or wire serialization",
		"edit/edit.go names github.com/google/uuid: libs/note reaches only itself, the standard library and the listed third-party packages",
		"markdown/markdown.go (markdown) imports " + notesModule + "edit (edit): violates the layer hierarchy edit -> markdown -> {domain, ulid}",
		"newpackage/package.go: the package newpackage is not listed in getInternalEdges",
	}
	if !slices.Equal(got, want) {
		t.Errorf("the walk found\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// TestNoteLibraryWalk_FailsOnAnEmptyTree verifies that a root without a Go file is an error, not a pass.
func TestNoteLibraryWalk_FailsOnAnEmptyTree(t *testing.T) {
	if _, err := findNoteLibraryViolations(t.TempDir()); err == nil {
		t.Error("a walk over a tree without a Go file passed")
	}
}

// Fails when it reads no file.
func findNoteLibraryViolations(root string) ([]string, error) {
	var found []string
	var read int
	reported := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		pkg := rootPackage
		if first, _, nested := strings.Cut(rel, "/"); nested {
			pkg = first
		}
		edges, isListed := getInternalEdges()[pkg]
		if !isListed {
			if !reported[pkg] {
				reported[pkg] = true
				found = append(found, fmt.Sprintf("%s: the package %s is not listed in getInternalEdges", path, pkg))
			}
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		read++
		for _, imp := range file.Imports {
			to, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return err
			}
			found = append(found, getImportViolations(path, pkg, edges, to)...)
		}
		return nil
	})
	if err == nil && read == 0 {
		err = fmt.Errorf("no Go file of a listed package read under %s: the walk checks nothing", root)
	}
	slices.Sort(found)
	return found, err
}

func getImportViolations(path, pkg string, edges []string, to string) []string {
	if after, ok := strings.CutPrefix(to, notesModule); ok {
		target, _, _ := strings.Cut(after, "/")
		if target != pkg && !slices.Contains(edges, target) {
			return []string{fmt.Sprintf("%s (%s) imports %s (%s): violates the layer hierarchy edit -> markdown -> {domain, ulid}", path, pkg, to, target)}
		}
		return nil
	}
	if getForbiddenInfra()[to] {
		if slices.Contains(getAllowedInfra()[pkg], to) {
			return nil
		}
		return []string{fmt.Sprintf("%s imports %s: libs/note performs no I/O or wire serialization", path, to)}
	}
	if !strings.Contains(strings.Split(to, "/")[0], ".") ||
		slices.ContainsFunc(listThirdParty(), func(allowed string) bool { return to == allowed || strings.HasPrefix(to, allowed) }) {
		return nil
	}
	return []string{fmt.Sprintf("%s names %s: libs/note reaches only itself, the standard library and the listed third-party packages", path, to)}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
