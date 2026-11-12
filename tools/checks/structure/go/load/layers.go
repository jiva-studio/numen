package load

import (
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

func GetDriving() map[string]bool {
	return map[string]bool{
		"adapter/cli":                 true,
		"adapter/window/agent":        true,
		"adapter/window/catalog":      true,
		"adapter/window/command":      true,
		"adapter/window/connection":   true,
		"adapter/window/model":        true,
		"adapter/window/editor":       true,
		"adapter/window/plugin":       true,
		"adapter/window/search":       true,
		"adapter/window/settings":     true,
		"adapter/window/shell":        true,
		"adapter/window/text":         true,
		"adapter/window/vault":        true,
		"adapter/window/vaults":       true,
		"adapter/window/windowstate":  true,
		"adapter/window/workspace":    true,
		"adapter/plugin/hostendpoint": true,
		"adapter/plugin/hostsearch":   true,
		"adapter/plugin/hosttext":     true,
		"adapter/plugin/hostvault":    true,
		"adapter/window/theme":        true,
		"internal/adapterlib/wire":    true,
	}
}

func IsMachinery(pkg string) bool {
	return IsUnder(pkg, "container") || strings.HasPrefix(pkg, "internal/test")
}

func IsShared(pkg string) bool {
	return strings.HasPrefix(pkg, "internal/adapterlib/")
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

func NamesHandler(file *ast.File) bool {
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
		return filepath.Join("..", "..", "..", "note", "domain")
	}
	return filepath.Join("..", pos)
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

func IsDriving(pkg string) bool {
	return IsAmong(slices.Collect(maps.Keys(GetDriving())), pkg)
}
