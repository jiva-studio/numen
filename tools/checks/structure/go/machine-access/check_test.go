// Package machineaccess is the check `go/machine-access`, run on the tree with the planted violations that prove it.
package machineaccess

import (
	"fmt"
	"go/ast"
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

// TestCorePackagesDoNotAccessMachineDirectly checks that a scenario queries a port about the machine and
// never the machine itself.
// Resolving a path through every link is one such query and has no answer for a
// folder passed as a content URI. A file, a socket and a system call are the
// same category. Taking a path apart and joining it is string arithmetic and is allowed.
func TestCorePackagesDoNotAccessMachineDirectly(t *testing.T) {
	wrong, err := findMachineAccess("..")
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one + ": the core asks a port what is on the machine")
	}
}

// TestMachineAccessWalk_FindsThePlantedFiles runs the walk over a planted tree: a scenario that
// imports os and resolves a path is found, an adapter that does the same is not.
func TestMachineAccessWalk_FindsThePlantedFiles(t *testing.T) {
	root := load.WriteTreeFixture(t, map[string]string{
		"usecase/note/note.go":   "package note\n\nimport (\n\t\"os\"\n\t\"path/filepath\"\n)\n\nvar _ = os.Args\n\nfunc run() { _, _ = filepath.Abs(\".\") }\n",
		"adapter/files/files.go": "package files\n\nimport \"os\"\n\nvar _ = os.Args\n",
	})
	wrong, err := findMachineAccess(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(wrong) != 2 || !strings.Contains(wrong[0], "usecase/note/note.go") || !strings.Contains(wrong[1], "usecase/note/note.go") {
		t.Errorf("the walk found %v, want an os import and a filepath.Abs call in usecase/note/note.go only", wrong)
	}
}

// TestMachineAccessWalk_FailsOnAnEmptyTree verifies that a root without a core file is an error, not a pass.
func TestMachineAccessWalk_FailsOnAnEmptyTree(t *testing.T) {
	if _, err := findMachineAccess(t.TempDir()); err == nil {
		t.Error("a walk over a tree without a core file passed")
	}
}

func findMachineAccess(root string) ([]string, error) {
	// Fails when it reads no file.
	var wrong []string
	var read int
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		pkg := load.GetPackage(root, path)
		if strings.HasSuffix(path, "_test.go") || load.IsAdapter(pkg) || load.IsShared(pkg) || load.IsMachinery(pkg) {
			return nil
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			return err
		}
		read++
		for _, one := range file.Imports {
			to, err := strconv.Unquote(one.Path.Value)
			if err != nil {
				return err
			}
			if isMachinePackage(to) {
				wrong = append(wrong, path+" is compiled from "+to)
			}
		}
		ast.Inspect(file, func(node ast.Node) bool {
			pos, is := node.(*ast.SelectorExpr)
			if !is {
				return true
			}
			from, is := pos.X.(*ast.Ident)
			if is && from.Name == "filepath" && getFilesystemCalls()[pos.Sel.Name] {
				wrong = append(wrong, path+" runs filepath."+pos.Sel.Name)
			}
			return true
		})
		return nil
	})
	if err == nil && read == 0 {
		err = fmt.Errorf("no core file read under %s: the walk checks nothing", root)
	}
	return wrong, err
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
