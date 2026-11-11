// Package machineaccess is the check `go/machine-access`, run on the tree with the planted violations that prove it.
package machineaccess

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
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
	var wrong []string
	var read int
	err := filepath.WalkDir("..", func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		pkg := load.GetPackage("..", path)
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
	if err != nil {
		t.Fatal(err)
	}
	for _, one := range wrong {
		t.Error(one + ": the core asks a port what is on the machine")
	}

	// A walk that read no file of the core is a rule checked against nothing,
	// and it passes.
	if read < 100 {
		t.Fatalf("%d files of the core read: the walk is not reading it", read)
	}
}

func TestMain(m *testing.M) {
	os.Exit(load.Main(m))
}
