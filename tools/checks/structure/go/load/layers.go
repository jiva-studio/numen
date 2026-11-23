package load

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Module is the base import path prefix for core packages.
const Module = "github.com/jiva-studio/numen/libs/core/kernel/"

// GetDriving returns the driving adapters of the kernel by package path under the kernel, read from the tree.
func GetDriving(tb testing.TB) map[string]bool {
	tb.Helper()
	driving, err := ListDriving("..")
	if err != nil {
		tb.Fatal(err)
	}
	return driving
}

// ListDriving returns the driving adapters under the kernel folder root: the packages of adapter/,
// internal/adapter/ and internal/adapterlib/ with a file that imports a use case or names a handler of the
// schema. Test files and testdata are not read.
func ListDriving(root string) (map[string]bool, error) {
	driving := map[string]bool{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if entry.Name() == "testdata" {
				return fs.SkipDir
			}
			return nil
		}
		pkg := GetPackage(root, path)
		if driving[pkg] {
			return nil
		}
		is, err := isDrivingFile(pkg, path)
		if is {
			driving[pkg] = true
		}
		return err
	})
	if err == nil && len(driving) == 0 {
		err = fmt.Errorf("no driving adapter under %s: the walk reads no adapter", root)
	}
	return driving, err
}

func isDrivingFile(pkg, path string) (bool, error) {
	if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") || (!IsAdapter(pkg) && !IsShared(pkg)) {
		return false, nil
	}
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return false, err
	}
	return isImportingUseCase(file) || HasHandlerName(file), nil
}

func isImportingUseCase(file *ast.File) bool {
	for _, one := range file.Imports {
		if to, err := strconv.Unquote(one.Path.Value); err == nil && strings.HasPrefix(to, Module) && IsUnder(strings.TrimPrefix(to, Module), "usecase") {
			return true
		}
	}
	return false
}

func IsMachinery(pkg string) bool {
	return IsUnder(pkg, "container") || strings.HasPrefix(pkg, "internal/test")
}

func IsShared(pkg string) bool {
	return strings.HasPrefix(pkg, "internal/adapterlib/")
}

// IsHelper reports whether pkg is a helper of the kernel: a package directly under internal/ that is not one of the
// folders internal/adapter, internal/adapterlib and internal/testsupport.
func IsHelper(pkg string) bool {
	name, isInternal := strings.CutPrefix(pkg, "internal/")
	return isInternal && name != "" && !strings.Contains(name, "/") && !slices.Contains([]string{"adapter", "adapterlib", "testsupport"}, name)
}

// Identity records an ID field: containing type, field name, and field type.
type Identity struct{ In, Field, Spelled string }

func GetTypeName(pos ast.Expr) string {
	switch target := pos.(type) {
	case *ast.Ident:
		return target.Name
	case *ast.SelectorExpr:
		if from, is := target.X.(*ast.Ident); is {
			return from.Name + "." + target.Sel.Name
		}
	}
	return ""
}

// Schema is the generated connect package. An adapter that names one of its handler types serves
// something outside the core, whether over the wire or to another adapter, which makes it a driving
// adapter. An adapter that only builds its messages or clients calls out and is driven.
const Schema = Wire + "/gen/numen/v1/numenv1connect"

func HasHandlerName(file *ast.File) bool {
	name := ""
	for _, one := range file.Imports {
		if to, err := strconv.Unquote(one.Path.Value); err == nil && to == Schema {
			name = "numenv1connect"
			if one.Name != nil {
				name = one.Name.Name
			}
		}
	}
	if name == "" {
		return false
	}
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		pos, is := node.(*ast.SelectorExpr)
		if !is {
			return true
		}
		if from, is := pos.X.(*ast.Ident); is && from.Name == name && strings.HasSuffix(pos.Sel.Name, "Handler") {
			found = true
		}
		return true
	})
	return found
}

func GetPackage(root, path string) string {
	target, err := filepath.Rel(root, filepath.Dir(path))
	if err != nil {
		return path
	}
	return filepath.ToSlash(target)
}

func IsAdapter(pkg string) bool {
	return strings.HasPrefix(pkg, "adapter/") ||
		strings.HasPrefix(pkg, "internal/adapter/")
}

func IsAmong(these []string, pkg string) bool {
	for _, one := range these {
		if pkg == one || strings.HasPrefix(pkg, one+"/") {
			return true
		}
	}
	return false
}

func ListGrouping() []string {
	return []string{"adapter", "internal/adapter", "adapter/window", "internal/adapterlib"}
}

func GetFamily(pkg string) string {
	depth := 2
	for _, one := range ListGrouping() {
		if strings.HasPrefix(pkg, one+"/") {
			depth = max(depth, strings.Count(one, "/")+2)
		}
	}
	target := strings.Split(pkg, "/")
	if len(target) < depth {
		return pkg
	}
	return strings.Join(target[:depth], "/")
}

func GetPackageDir(pos string) string {
	if pos == "domain" {
		return filepath.Join("..", "..", "..", "shared", "note", "domain")
	}
	return filepath.Join("..", pos)
}

// Root is a folder a check walks, with the name its findings are reported under.
type Root struct{ Name, Dir string }

// ListTypeRoots returns the folders whose types go/identities and go/instants read.
func ListTypeRoots() []Root {
	return []Root{
		{Name: "kernel/domain", Dir: filepath.Join("..", "domain")},
		{Name: "kernel/port", Dir: filepath.Join("..", "port")},
		{Name: "note/domain", Dir: filepath.Join("..", "..", "..", "shared", "note", "domain")},
	}
}

// TreeEntry is one file of a folder tree, named by its path under the folder.
type TreeEntry struct{ Path string }

// Name returns the path of the file under the folder that was read.
func (e TreeEntry) Name() string { return e.Path }

// IsDir reports false: a tree entry is a file.
func (e TreeEntry) IsDir() bool { return false }

func ReadTree(dir string) ([]TreeEntry, error) {
	var found []TreeEntry
	err := filepath.WalkDir(dir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		found = append(found, TreeEntry{Path: filepath.ToSlash(rel)})
		return nil
	})
	return found, err
}

func IsUnder(pkg, folder string) bool {
	return pkg == folder || strings.HasPrefix(pkg, folder+"/")
}

func IsPortName(name string) bool {
	return slices.Contains(GetPortNames(), name)
}

func GetPortNames() []string {
	entries, err := os.ReadDir(GetPackageDir("port"))
	if err != nil {
		return nil
	}
	var names []string
	for _, one := range entries {
		if one.IsDir() {
			names = append(names, one.Name())
		}
	}
	return names
}

func GetFieldsNamed(file *ast.File, suffix string) []Identity {
	var found []Identity
	ast.Inspect(file, func(node ast.Node) bool {
		spec, is := node.(*ast.TypeSpec)
		if !is {
			return true
		}
		target, is := spec.Type.(*ast.StructType)
		if !is {
			return true
		}
		for _, one := range target.Fields.List {
			for _, name := range one.Names {
				if name.IsExported() && strings.HasSuffix(name.Name, suffix) {
					found = append(found, Identity{spec.Name.Name, name.Name, GetTypeName(one.Type)})
				}
			}
		}
		return true
	})
	return found
}

func ParseSourceFiles(t *testing.T, dir string) []*ast.File {
	t.Helper()
	entries, err := ReadTree(dir)
	if err != nil {
		t.Fatal(err)
	}
	var files []*ast.File
	for _, one := range entries {
		if one.IsDir() || !strings.HasSuffix(one.Name(), ".go") || strings.HasSuffix(one.Name(), "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(dir, one.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, file)
	}
	return files
}

func IsDriving(driving map[string]bool, pkg string) bool {
	return IsAmong(slices.Collect(maps.Keys(driving)), pkg)
}

// IsUseCaseTestSupport reports a package under usecase whose directory name ends in "test" (connectiontest).
// It builds its parent use case for the tests of several packages, as net/http/httptest does for net/http,
// and is not a use case. pkg is a path relative to the core or a full import path.
func IsUseCaseTestSupport(pkg string) bool {
	base := pkg[strings.LastIndex(pkg, "/")+1:]
	return (strings.HasPrefix(pkg, "usecase/") || strings.Contains(pkg, "/usecase/")) && strings.HasSuffix(base, "test")
}
